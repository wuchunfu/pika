package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dushixiang/pika/internal/models"
	"github.com/dushixiang/pika/internal/repo"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TrafficService struct {
	logger          *zap.Logger
	agentRepo       *repo.AgentRepo
	alertRecordRepo *repo.AlertRecordRepo
}

func NewTrafficService(logger *zap.Logger, db *gorm.DB) *TrafficService {
	return &TrafficService{
		logger:          logger,
		agentRepo:       repo.NewAgentRepo(db),
		alertRecordRepo: repo.NewAlertRecordRepo(db),
	}
}

func (s *TrafficService) updateById(ctx context.Context, agentID string, stats *models.TrafficStatsData) error {
	return s.agentRepo.UpdateById(ctx, &models.Agent{
		ID:           agentID,
		TrafficStats: datatypes.NewJSONType(*stats),
	})
}

// UpdateAgentTraffic 更新探针流量统计(每次上报网络指标时调用)
func (s *TrafficService) UpdateAgentTraffic(ctx context.Context, agentID string, currentRecvTotal, currentSentTotal uint64) error {
	agent, err := s.agentRepo.FindById(ctx, agentID)
	if err != nil {
		return err
	}

	// 获取流量统计数据
	stats := agent.TrafficStats.Data()

	// 如果流量统计未启用,跳过更新
	if !stats.Enabled {
		return nil
	}

	// 设置默认类型
	if stats.Type == "" {
		stats.Type = "recv" // 默认只统计进站流量，向后兼容
	}

	now := time.Now().UnixMilli()
	baselineChanged := false

	// 初始化基线(首次统计或基线丢失)
	if stats.BaselineRecv == 0 && stats.BaselineSend == 0 {
		stats.BaselineRecv = currentRecvTotal
		stats.BaselineSend = currentSentTotal
		if stats.PeriodStart == 0 {
			stats.PeriodStart = now
		}
		baselineChanged = true
	}

	// 检测计数器重置(探针重启)
	recvReset := currentRecvTotal < stats.BaselineRecv
	sendReset := currentSentTotal < stats.BaselineSend

	if recvReset || sendReset {
		s.logger.Warn("检测到流量计数器重置",
			zap.String("agentId", agentID),
			zap.Uint64("baselineRecv", stats.BaselineRecv),
			zap.Uint64("currentRecv", currentRecvTotal),
			zap.Uint64("baselineSend", stats.BaselineSend),
			zap.Uint64("currentSend", currentSentTotal))

		if recvReset {
			stats.BaselineRecv = currentRecvTotal
		}
		if sendReset {
			stats.BaselineSend = currentSentTotal
		}
		baselineChanged = true
	}

	baseDelta := calculateBaseDelta(stats.Type, currentRecvTotal, currentSentTotal, stats.BaselineRecv, stats.BaselineSend)
	if baselineChanged {
		stats.UsedOffset = calculateOffset(stats.Used, baseDelta)
	}
	stats.Used = applyOffset(baseDelta, stats.UsedOffset)

	// 检查告警(如果配置了限额)
	if stats.Limit > 0 {
		s.checkTrafficAlerts(ctx, &agent, &stats)
	}

	// 保存更新后的统计数据
	agent.TrafficStats = datatypes.NewJSONType(stats)

	// 更新数据库
	return s.updateById(ctx, agentID, &stats)
}

// checkTrafficAlerts 检查并发送流量告警
func (s *TrafficService) checkTrafficAlerts(ctx context.Context, agent *models.Agent, stats *models.TrafficStatsData) {
	usagePercent := float64(stats.Used) / float64(stats.Limit) * 100

	// 100% 告警
	if usagePercent >= 100 && !stats.AlertSent100 {
		s.sendTrafficAlert(ctx, agent, stats, 100, usagePercent)
		stats.AlertSent100 = true
	}
	// 90% 告警
	if usagePercent >= 90 && !stats.AlertSent90 {
		s.sendTrafficAlert(ctx, agent, stats, 90, usagePercent)
		stats.AlertSent90 = true
	}
	// 80% 告警
	if usagePercent >= 80 && !stats.AlertSent80 {
		s.sendTrafficAlert(ctx, agent, stats, 80, usagePercent)
		stats.AlertSent80 = true
	}
}

// sendTrafficAlert 发送流量告警
func (s *TrafficService) sendTrafficAlert(ctx context.Context, agent *models.Agent, stats *models.TrafficStatsData, threshold int, actualPercent float64) {
	level := "info"
	if threshold == 100 {
		level = "critical"
	} else if threshold == 90 {
		level = "warning"
	}

	now := time.Now().UnixMilli()
	record := &models.AlertRecord{
		AgentID:   agent.ID,
		AgentName: agent.Name,
		AlertType: "traffic",
		Message: fmt.Sprintf("流量使用已达到%d%%，当前使用%.2f%%（%s/%s）",
			threshold, actualPercent,
			formatBytes(stats.Used),
			formatBytes(stats.Limit)),
		Threshold:   float64(threshold),
		ActualValue: actualPercent,
		Level:       level,
		Status:      "firing",
		FiredAt:     now,
		CreatedAt:   now,
	}

	// 创建告警记录
	if err := s.alertRecordRepo.CreateAlertRecord(ctx, record); err != nil {
		s.logger.Error("创建流量告警记录失败", zap.Error(err))
		return
	}

	s.logger.Info("流量告警记录已创建",
		zap.String("agentId", agent.ID),
		zap.String("agentName", agent.Name),
		zap.Int("threshold", threshold),
		zap.Float64("actualPercent", actualPercent))

	// 注意: 告警通知需要通过告警系统的统一通知机制发送,
	// 这里只创建记录,通知由其他机制处理
}

// formatBytes 格式化字节数为人类可读的格式
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// UpdateTrafficConfig 更新流量配置
// used: 如果大于0，则使用该值作为已使用流量；如果为0，则保持当前值或重置
func (s *TrafficService) UpdateTrafficConfig(ctx context.Context, agentID string, enabled bool, trafficType string, limit uint64, resetDay int, used uint64) error {
	if resetDay < 0 || resetDay > 31 {
		return fmt.Errorf("重置日期必须在0-31之间")
	}

	// 验证流量类型
	if trafficType != "" && trafficType != "recv" && trafficType != "send" && trafficType != "both" {
		return fmt.Errorf("流量类型必须是 recv、send 或 both")
	}

	agent, err := s.agentRepo.FindById(ctx, agentID)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	stats := agent.TrafficStats.Data()
	oldResetDay := stats.ResetDay
	oldEnabled := stats.Enabled
	oldType := stats.Type

	// 设置配置
	stats.Enabled = enabled
	stats.Limit = limit
	stats.ResetDay = resetDay

	// 设置流量类型（默认为 recv）
	if trafficType != "" {
		stats.Type = trafficType
	} else if stats.Type == "" {
		stats.Type = "recv" // 默认只统计进站流量
	}

	// 如果是首次启用、禁用后重新启用、修改重置日期或修改流量类型，重置流量统计
	if (!oldEnabled && stats.Enabled) || (stats.Enabled && resetDay != oldResetDay) || (stats.Enabled && stats.PeriodStart == 0) || (stats.Enabled && oldType != stats.Type && oldType != "") {
		stats.Used = 0
		stats.UsedOffset = 0
		stats.PeriodStart = now
		stats.BaselineRecv = 0 // 下次上报时会设置正确的基线
		stats.BaselineSend = 0
		stats.AlertSent80 = false
		stats.AlertSent90 = false
		stats.AlertSent100 = false
	}

	// 如果提供了 used 参数且大于 0，则使用该值
	if used > 0 {
		stats.UsedOffset = addOffset(stats.UsedOffset, calculateOffset(used, stats.Used))
		stats.Used = used
	}

	// 如果禁用流量统计，清空相关数据
	if !stats.Enabled {
		stats.Used = 0
		stats.UsedOffset = 0
		stats.PeriodStart = 0
		stats.BaselineRecv = 0
		stats.BaselineSend = 0
		stats.AlertSent80 = false
		stats.AlertSent90 = false
		stats.AlertSent100 = false
	}

	return s.updateById(ctx, agentID, &stats)
}

// GetTrafficStats 获取流量统计信息
func (s *TrafficService) GetTrafficStats(ctx context.Context, agentID string) (*TrafficStats, error) {
	agent, err := s.agentRepo.FindById(ctx, agentID)
	if err != nil {
		return nil, err
	}

	trafficData := agent.TrafficStats.Data()

	// 设置默认类型
	trafficType := trafficData.Type
	if trafficType == "" {
		trafficType = "recv" // 默认只统计进站流量
	}

	stats := &TrafficStats{
		Enabled:     trafficData.Enabled,
		Type:        trafficType,
		Limit:       trafficData.Limit,
		Used:        trafficData.Used,
		ResetDay:    trafficData.ResetDay,
		PeriodStart: trafficData.PeriodStart,
		Alerts: TrafficAlerts{
			Sent80:  trafficData.AlertSent80,
			Sent90:  trafficData.AlertSent90,
			Sent100: trafficData.AlertSent100,
		},
	}

	// 计算使用百分比
	if trafficData.Limit > 0 {
		stats.UsedPercent = float64(trafficData.Used) / float64(trafficData.Limit) * 100
		if trafficData.Used < trafficData.Limit {
			stats.Remaining = trafficData.Limit - trafficData.Used
		} else {
			stats.Remaining = 0
		}
	}

	// 计算下次重置日期和剩余天数
	if trafficData.ResetDay > 0 && trafficData.PeriodStart > 0 {
		periodStart := time.UnixMilli(trafficData.PeriodStart)
		nextReset := calculateNextResetDate(periodStart, trafficData.ResetDay)
		stats.PeriodEnd = nextReset.UnixMilli()
		stats.DaysUntilReset = int(time.Until(nextReset).Hours() / 24)
		if stats.DaysUntilReset < 0 {
			stats.DaysUntilReset = 0
		}
	}

	return stats, nil
}

// ResetAgentTraffic 重置探针流量
func (s *TrafficService) ResetAgentTraffic(ctx context.Context, agentID string) error {
	agent, err := s.agentRepo.FindById(ctx, agentID)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	stats := agent.TrafficStats.Data()

	stats.Used = 0
	stats.UsedOffset = 0
	stats.BaselineRecv = 0 // 下次上报时会设置正确的基线
	stats.BaselineSend = 0
	stats.PeriodStart = now
	stats.AlertSent80 = false
	stats.AlertSent90 = false
	stats.AlertSent100 = false

	s.logger.Info("探针流量已重置",
		zap.String("agentId", agentID),
		zap.String("agentName", agent.Name))

	return s.updateById(ctx, agentID, &stats)
}

// CheckAndResetTraffic 检查并重置所有到期的探针流量(定时任务调用)
func (s *TrafficService) CheckAndResetTraffic(ctx context.Context) error {
	agents, err := s.agentRepo.FindAgentsWithTrafficReset(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	resetCount := 0

	for _, agent := range agents {
		if s.shouldResetTraffic(&agent, now) {
			if err := s.ResetAgentTraffic(ctx, agent.ID); err != nil {
				s.logger.Error("重置探针流量失败",
					zap.String("agentId", agent.ID),
					zap.Error(err))
				continue
			}
			resetCount++
		}
	}

	if resetCount > 0 {
		s.logger.Info("流量重置检查完成", zap.Int("重置数量", resetCount))
	}

	return nil
}

// shouldResetTraffic 判断是否需要重置流量
func (s *TrafficService) shouldResetTraffic(agent *models.Agent, now time.Time) bool {
	stats := agent.TrafficStats.Data()
	// 流量统计未启用、未配置重置日期或周期未开始时不重置
	if !stats.Enabled || stats.ResetDay == 0 || stats.PeriodStart == 0 {
		return false
	}

	periodStart := time.UnixMilli(stats.PeriodStart)
	nextReset := calculateNextResetDate(periodStart, stats.ResetDay)

	return now.After(nextReset) || now.Equal(nextReset)
}

// calculateNextResetDate 计算下次重置日期
func calculateNextResetDate(periodStart time.Time, resetDay int) time.Time {
	year, month, day := periodStart.Date()
	location := periodStart.Location()

	// 先计算当月的重置日期
	currentMonthLastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, location).Day()
	currentMonthResetDay := resetDay
	if resetDay > currentMonthLastDay {
		currentMonthResetDay = currentMonthLastDay
	}
	currentMonthReset := time.Date(year, month, currentMonthResetDay, 0, 0, 0, 0, location)

	// 如果当月的重置日期还没到（在 periodStart 之后或同一天），返回当月重置日期
	if day < currentMonthResetDay || (day == currentMonthResetDay && periodStart.Before(currentMonthReset)) {
		return currentMonthReset
	}

	// 当月重置日期已过，计算下个月的重置日期
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}

	// 处理月末日期(如2月没有31号)
	nextMonthLastDay := time.Date(nextYear, nextMonth+1, 0, 0, 0, 0, 0, location).Day()
	nextMonthResetDay := resetDay
	if resetDay > nextMonthLastDay {
		nextMonthResetDay = nextMonthLastDay
	}

	return time.Date(nextYear, nextMonth, nextMonthResetDay, 0, 0, 0, 0, location)
}

func calculateBaseDelta(trafficType string, currentRecvTotal, currentSentTotal, baselineRecv, baselineSend uint64) uint64 {
	switch trafficType {
	case "recv":
		return currentRecvTotal - baselineRecv
	case "send":
		return currentSentTotal - baselineSend
	case "both":
		recvUsed := currentRecvTotal - baselineRecv
		sendUsed := currentSentTotal - baselineSend
		return recvUsed + sendUsed
	default:
		return currentRecvTotal - baselineRecv
	}
}

func calculateOffset(targetUsed, baseDelta uint64) int64 {
	const maxInt64 = int64(^uint64(0) >> 1)
	const minInt64 = -maxInt64 - 1

	if targetUsed >= baseDelta {
		diff := targetUsed - baseDelta
		if diff > uint64(maxInt64) {
			return maxInt64
		}
		return int64(diff)
	}

	diff := baseDelta - targetUsed
	if diff > uint64(maxInt64) {
		return minInt64
	}
	return -int64(diff)
}

func applyOffset(baseDelta uint64, offset int64) uint64 {
	if offset >= 0 {
		return baseDelta + uint64(offset)
	}
	const minInt64 = -1 - (int64(^uint64(0) >> 1))
	if offset == minInt64 {
		return 0
	}
	neg := uint64(-offset)
	if neg > baseDelta {
		return 0
	}
	return baseDelta - neg
}

func addOffset(base, delta int64) int64 {
	const maxInt64 = int64(^uint64(0) >> 1)
	const minInt64 = -maxInt64 - 1

	if delta > 0 && base > maxInt64-delta {
		return maxInt64
	}
	if delta < 0 && base < minInt64-delta {
		return minInt64
	}
	return base + delta
}

// TrafficStats 流量统计信息
type TrafficStats struct {
	Enabled        bool          `json:"enabled"`
	Type           string        `json:"type"` // 统计类型: recv/send/both
	Limit          uint64        `json:"limit"`
	Used           uint64        `json:"used"`
	UsedPercent    float64       `json:"usedPercent"`
	Remaining      uint64        `json:"remaining"`
	ResetDay       int           `json:"resetDay"`
	PeriodStart    int64         `json:"periodStart"`
	PeriodEnd      int64         `json:"periodEnd"`
	DaysUntilReset int           `json:"daysUntilReset"`
	Alerts         TrafficAlerts `json:"alerts"`
}

// TrafficAlerts 流量告警状态
type TrafficAlerts struct {
	Sent80  bool `json:"sent80"`
	Sent90  bool `json:"sent90"`
	Sent100 bool `json:"sent100"`
}
