package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dushixiang/pika/internal/models"
	"github.com/dushixiang/pika/internal/protocol"
	"github.com/dushixiang/pika/internal/repo"
	ws "github.com/dushixiang/pika/internal/websocket"
	"github.com/go-orz/cache"
	"github.com/go-orz/orz"
	"github.com/go-orz/toolkit"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MonitorService struct {
	logger *zap.Logger
	*repo.MonitorRepo
	*orz.Service
	agentRepo        *repo.AgentRepo
	metricRepo       *repo.MetricRepo
	metricService    *MetricService
	monitorStatsRepo *repo.MonitorStatsRepo
	wsManager        *ws.Manager

	// 监控概览缓存：缓存监控任务列表（使用不同的 key 区分 public 和 private）
	overviewCache cache.Cache[string, []PublicMonitorOverview]
	// 监控统计缓存：缓存单个监控任务的统计数据
	statsCache cache.Cache[string, []models.MonitorStats]

	// 调度器引用（用于动态管理任务）
	scheduler MonitorScheduler
}

// MonitorScheduler 调度器接口（避免循环依赖）
type MonitorScheduler interface {
	AddTask(monitorID string, interval int) error
	UpdateTask(monitorID string, interval int) error
	RemoveTask(monitorID string)
}

func NewMonitorService(logger *zap.Logger, db *gorm.DB, metricService *MetricService, wsManager *ws.Manager) *MonitorService {
	return &MonitorService{
		logger:           logger,
		Service:          orz.NewService(db),
		MonitorRepo:      repo.NewMonitorRepo(db),
		agentRepo:        repo.NewAgentRepo(db),
		metricRepo:       repo.NewMetricRepo(db),
		metricService:    metricService,
		monitorStatsRepo: repo.NewMonitorStatsRepo(db),
		wsManager:        wsManager,

		// 缓存 5 分钟，避免频繁查询
		overviewCache: cache.New[string, []PublicMonitorOverview](5 * time.Minute),
		statsCache:    cache.New[string, []models.MonitorStats](5 * time.Minute),
	}
}

// SetScheduler 设置调度器（由外部注入，避免循环依赖）
func (s *MonitorService) SetScheduler(scheduler MonitorScheduler) {
	s.scheduler = scheduler
}

type MonitorTaskRequest struct {
	Name             string                     `json:"name"`
	Type             string                     `json:"type"`
	Target           string                     `json:"target"`
	Description      string                     `json:"description"`
	Enabled          bool                       `json:"enabled,omitempty"`
	ShowTargetPublic bool                       `json:"showTargetPublic,omitempty"` // 在公开页面是否显示目标地址
	Visibility       string                     `json:"visibility,omitempty"`       // 可见性: public-匿名可见, private-登录可见
	Interval         int                        `json:"interval"`                   // 检测频率（秒）
	HTTPConfig       protocol.HTTPMonitorConfig `json:"httpConfig,omitempty"`
	TCPConfig        protocol.TCPMonitorConfig  `json:"tcpConfig,omitempty"`
	ICMPConfig       protocol.ICMPMonitorConfig `json:"icmpConfig,omitempty"`
	AgentIds         []string                   `json:"agentIds,omitempty"`
	Tags             []string                   `json:"tags"`
}

// PublicMonitorOverview 用于公开展示的监控配置及汇总数据
type PublicMonitorOverview struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	Target           string   `json:"target"`
	ShowTargetPublic bool     `json:"showTargetPublic"` // 在公开页面是否显示目标地址
	Description      string   `json:"description"`
	Enabled          bool     `json:"enabled"`
	Interval         int      `json:"interval"`
	AgentIds         []string `json:"agentIds"`
	AgentCount       int      `json:"agentCount"`
	LastCheckStatus  string   `json:"lastCheckStatus"`
	LastCheckError   string   `json:"lastCheckError"`
	CurrentResponse  int64    `json:"currentResponse"`
	AvgResponse24h   int64    `json:"avgResponse24h"`
	Uptime24h        float64  `json:"uptime24h"`
	Uptime7d         float64  `json:"uptime7d"`
	CertExpiryDate   int64    `json:"certExpiryDate"`
	CertExpiryDays   int      `json:"certExpiryDays"`
	LastCheckTime    int64    `json:"lastCheckTime"`
}

func (s *MonitorService) CreateMonitor(ctx context.Context, req *MonitorTaskRequest) (*models.MonitorTask, error) {
	// 设置默认检测频率
	interval := req.Interval
	if interval <= 0 {
		interval = 60 // 默认 60 秒
	}

	// 设置默认可见性
	visibility := req.Visibility
	if visibility == "" {
		visibility = "public" // 默认公开可见
	}

	task := &models.MonitorTask{
		ID:               uuid.NewString(),
		Name:             strings.TrimSpace(req.Name),
		Type:             req.Type,
		Target:           strings.TrimSpace(req.Target),
		Description:      req.Description,
		Enabled:          req.Enabled,
		ShowTargetPublic: req.ShowTargetPublic,
		Visibility:       visibility,
		Interval:         interval,
		AgentIds:         datatypes.JSONSlice[string](req.AgentIds),
		Tags:             datatypes.JSONSlice[string](req.Tags),
		HTTPConfig:       datatypes.NewJSONType(req.HTTPConfig),
		TCPConfig:        datatypes.NewJSONType(req.TCPConfig),
		ICMPConfig:       datatypes.NewJSONType(req.ICMPConfig),
		CreatedAt:        0,
		UpdatedAt:        0,
	}

	if err := s.MonitorRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	// 清理缓存
	s.clearCache(task.ID)

	// 如果任务启用，添加到调度器
	if task.Enabled && s.scheduler != nil {
		if err := s.scheduler.AddTask(task.ID, task.Interval); err != nil {
			s.logger.Error("添加监控任务到调度器失败",
				zap.String("taskID", task.ID),
				zap.Error(err))
		}
	}

	return task, nil
}

func (s *MonitorService) UpdateMonitor(ctx context.Context, id string, req *MonitorTaskRequest) (*models.MonitorTask, error) {
	task, err := s.MonitorRepo.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	// 记录旧状态，用于判断是否需要更新调度器
	oldEnabled := task.Enabled
	oldInterval := task.Interval

	task.Enabled = req.Enabled
	task.Name = strings.TrimSpace(req.Name)
	task.Type = req.Type
	task.Target = strings.TrimSpace(req.Target)
	task.Description = req.Description
	task.ShowTargetPublic = req.ShowTargetPublic
	task.Visibility = req.Visibility
	task.Tags = req.Tags

	// 更新检测频率
	interval := req.Interval
	if interval <= 0 {
		interval = 60 // 默认 60 秒
	}
	task.Interval = interval

	task.AgentIds = req.AgentIds
	task.HTTPConfig = datatypes.NewJSONType(req.HTTPConfig)
	task.TCPConfig = datatypes.NewJSONType(req.TCPConfig)
	task.ICMPConfig = datatypes.NewJSONType(req.ICMPConfig)

	if err := s.MonitorRepo.Save(ctx, &task); err != nil {
		return nil, err
	}

	// 清理缓存
	s.clearCache(task.ID)

	// 更新调度器
	if s.scheduler != nil {
		// 如果从禁用变为启用，或者间隔时间改变
		if !oldEnabled && task.Enabled {
			// 添加任务到调度器
			if err := s.scheduler.AddTask(task.ID, task.Interval); err != nil {
				s.logger.Error("添加监控任务到调度器失败",
					zap.String("taskID", task.ID),
					zap.Error(err))
			}
		} else if oldEnabled && !task.Enabled {
			// 从调度器中移除任务
			s.scheduler.RemoveTask(task.ID)
		} else if task.Enabled && oldInterval != task.Interval {
			// 更新任务间隔
			if err := s.scheduler.UpdateTask(task.ID, task.Interval); err != nil {
				s.logger.Error("更新监控任务调度器失败",
					zap.String("taskID", task.ID),
					zap.Error(err))
			}
		}
	}

	return &task, nil
}

func (s *MonitorService) DeleteMonitor(ctx context.Context, id string) error {
	err := s.Transaction(ctx, func(ctx context.Context) error {
		// 删除监控任务
		if err := s.MonitorRepo.DeleteById(ctx, id); err != nil {
			return err
		}

		// 删除监控统计数据
		if err := s.monitorStatsRepo.DeleteByMonitorId(ctx, id); err != nil {
			s.logger.Error("删除监控统计数据失败", zap.String("monitorId", id), zap.Error(err))
			return err
		}

		// 删除监控指标数据（从 VictoriaMetrics）
		if err := s.metricService.DeleteMonitorMetrics(ctx, id); err != nil {
			s.logger.Error("删除监控指标数据失败", zap.String("monitorId", id), zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 清理缓存
	s.clearCache(id)

	// 从调度器中移除
	if s.scheduler != nil {
		s.scheduler.RemoveTask(id)
	}

	return nil
}

// ListByAuth 返回公开展示所需的监控配置和汇总统计
func (s *MonitorService) ListByAuth(ctx context.Context, isAuthenticated bool) ([]PublicMonitorOverview, error) {
	// 构建缓存键：根据认证状态使用不同的 key
	cacheKey := "overview:public"
	if isAuthenticated {
		cacheKey = "overview:private"
	}

	// 尝试从缓存获取
	if cachedResult, ok := s.overviewCache.Get(cacheKey); ok {
		return cachedResult, nil
	}

	// 缓存未命中，查询数据库
	// 获取符合权限的监控任务列表
	monitors, err := s.FindByAuth(ctx, isAuthenticated)
	if err != nil {
		return nil, err
	}

	if len(monitors) == 0 {
		emptyResult := []PublicMonitorOverview{}
		// 缓存空结果
		s.overviewCache.Set(cacheKey, emptyResult, 5*time.Minute)
		return emptyResult, nil
	}

	// 提取监控任务ID列表
	monitorIds := make([]string, 0, len(monitors))
	for _, monitor := range monitors {
		monitorIds = append(monitorIds, monitor.ID)
	}

	// 批量获取统计数据（已经由后台任务预计算好）
	statsList, err := s.monitorStatsRepo.FindByMonitorIdIn(ctx, monitorIds)
	if err != nil {
		return nil, err
	}

	// 按监控任务ID分组统计数据
	statsMap := make(map[string][]models.MonitorStats, len(monitors))
	for _, stats := range statsList {
		statsMap[stats.MonitorId] = append(statsMap[stats.MonitorId], stats)
	}

	// 构建监控概览列表
	items := make([]PublicMonitorOverview, 0, len(monitors))
	for _, monitor := range monitors {
		// 直接聚合预计算的统计数据
		summary := aggregateMonitorStats(statsMap[monitor.ID])

		// 构建监控概览对象
		item := s.buildMonitorOverview(monitor, summary)
		items = append(items, item)
	}

	// 缓存结果
	s.overviewCache.Set(cacheKey, items, 5*time.Minute)

	return items, nil
}

// buildMonitorOverview 构建监控概览对象
func (s *MonitorService) buildMonitorOverview(monitor models.MonitorTask, summary monitorOverviewSummary) PublicMonitorOverview {
	// 根据 ShowTargetPublic 字段决定是否返回真实的 Target
	target := monitor.Target
	if !monitor.ShowTargetPublic {
		target = "******"
	}

	return PublicMonitorOverview{
		ID:               monitor.ID,
		Name:             monitor.Name,
		Type:             monitor.Type,
		Target:           target,
		ShowTargetPublic: monitor.ShowTargetPublic,
		Description:      monitor.Description,
		Enabled:          monitor.Enabled,
		Interval:         monitor.Interval,
		AgentIds:         cloneAgentIDs(monitor.AgentIds),
		AgentCount:       summary.AgentCount,
		LastCheckStatus:  summary.LastCheckStatus,
		LastCheckError:   summary.LastCheckError,
		CurrentResponse:  summary.CurrentResponse,
		AvgResponse24h:   summary.AvgResponse24h,
		Uptime24h:        summary.Uptime24h,
		Uptime7d:         summary.Uptime7d,
		CertExpiryDate:   summary.CertExpiryDate,
		CertExpiryDays:   summary.CertExpiryDays,
		LastCheckTime:    summary.LastCheckTime,
	}
}

type monitorOverviewSummary struct {
	AgentCount      int
	LastCheckStatus string
	LastCheckError  string
	CurrentResponse int64
	AvgResponse24h  int64
	Uptime24h       float64
	Uptime7d        float64
	CertExpiryDate  int64
	CertExpiryDays  int
	LastCheckTime   int64
}

func aggregateMonitorStats(stats []models.MonitorStats) monitorOverviewSummary {
	summary := monitorOverviewSummary{
		LastCheckStatus: "unknown",
	}

	if len(stats) == 0 {
		return summary
	}

	var totalCurrentResponse int64
	var totalAvgResponse24h int64
	var totalUptime24h float64
	var totalUptime7d float64
	var lastCheckTime int64
	var lastCheckError string
	var certExpiryDate int64
	var certExpiryDays int
	hasCert := false
	hasUp := false
	hasDown := false

	for _, stat := range stats {
		totalCurrentResponse += stat.CurrentResponse
		totalAvgResponse24h += stat.AvgResponse24h
		totalUptime24h += stat.Uptime24h
		totalUptime7d += stat.Uptime7d

		// 记录最新的检测时间和对应的错误信息
		if stat.LastCheckTime > lastCheckTime {
			lastCheckTime = stat.LastCheckTime
			// 如果这是最新的检测且状态为 down，记录错误信息
			if stat.LastCheckStatus == "down" {
				lastCheckError = stat.LastCheckError
			} else {
				lastCheckError = ""
			}
		}

		switch stat.LastCheckStatus {
		case "up":
			hasUp = true
		case "down":
			hasDown = true
		}

		if stat.CertExpiryDate > 0 {
			if !hasCert || stat.CertExpiryDate < certExpiryDate {
				certExpiryDate = stat.CertExpiryDate
				certExpiryDays = stat.CertExpiryDays
				hasCert = true
			}
		}
	}

	count := len(stats)
	summary.AgentCount = count
	if count > 0 {
		summary.CurrentResponse = totalCurrentResponse / int64(count)
		summary.AvgResponse24h = totalAvgResponse24h / int64(count)
		summary.Uptime24h = totalUptime24h / float64(count)
		summary.Uptime7d = totalUptime7d / float64(count)
	}
	summary.LastCheckTime = lastCheckTime
	summary.LastCheckError = lastCheckError

	switch {
	case hasUp:
		summary.LastCheckStatus = "up"
	case hasDown:
		summary.LastCheckStatus = "down"
	default:
		summary.LastCheckStatus = "unknown"
	}

	if hasCert {
		summary.CertExpiryDate = certExpiryDate
		summary.CertExpiryDays = certExpiryDays
	}

	return summary
}

func cloneAgentIDs(ids datatypes.JSONSlice[string]) []string {
	if len(ids) == 0 {
		return []string{}
	}

	copied := make([]string, len(ids))
	copy(copied, []string(ids))
	return copied
}

// resolveTargetAgents 计算监控任务对应的目标探针范围
// 规则：
// 1. 如果既没有指定 AgentIds 也没有指定 Tags，返回所有传入的探针（全部节点）
// 2. 如果指定了 AgentIds 或 Tags（或两者都指定），则返回匹配的探针（自动去重）
//   - AgentIds: 直接匹配探针 ID
//   - Tags: 匹配探针标签中包含任意一个指定标签的探针
//   - 两者结果取并集
func (s *MonitorService) resolveTargetAgents(monitor models.MonitorTask, availableAgents []models.Agent) []models.Agent {
	// 如果既没有指定 AgentIds 也没有指定 Tags，使用所有可用探针
	if len(monitor.AgentIds) == 0 && len(monitor.Tags) == 0 {
		return availableAgents
	}

	// 使用 map 来去重
	targetAgentIDSet := make(map[string]struct{})

	// 1. 处理通过 AgentIds 指定的探针
	if len(monitor.AgentIds) > 0 {
		for _, agentID := range monitor.AgentIds {
			targetAgentIDSet[agentID] = struct{}{}
		}
	}

	// 2. 处理通过 Tags 指定的探针
	if len(monitor.Tags) > 0 {
		for _, agent := range availableAgents {
			if agent.Tags != nil && len(agent.Tags) > 0 {
				// 检查探针的标签中是否包含任何一个指定的标签
				for _, agentTag := range agent.Tags {
					for _, monitorTag := range monitor.Tags {
						if agentTag == monitorTag {
							targetAgentIDSet[agent.ID] = struct{}{}
							break
						}
					}
				}
			}
		}
	}

	// 3. 根据去重后的 ID 集合筛选探针
	targetAgents := make([]models.Agent, 0, len(targetAgentIDSet))
	for _, agent := range availableAgents {
		if _, ok := targetAgentIDSet[agent.ID]; ok {
			targetAgents = append(targetAgents, agent)
		}
	}

	return targetAgents
}

// sendMonitorConfigToAgent 向指定探针发送监控配置（内部方法）
func (s *MonitorService) sendMonitorConfigToAgent(agentID string, payload protocol.MonitorConfigPayload) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := protocol.Message{
		Type: protocol.MessageTypeMonitorConfig,
		Data: payloadData,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return s.wsManager.SendToClient(agentID, msgData)
}

// SendMonitorTaskToAgents 向指定探针发送单个监控任务（公开方法）
func (s *MonitorService) SendMonitorTaskToAgents(ctx context.Context, monitor models.MonitorTask) error {
	// 实时获取所有在线探针，避免依赖数据库状态
	onlineIDs := s.wsManager.GetAllClients()
	if len(onlineIDs) == 0 {
		return nil
	}

	// 查询在线探针的详细信息
	onlineAgents, err := s.agentRepo.ListByIDs(ctx, onlineIDs)
	if err != nil {
		s.logger.Error("获取在线探针信息失败", zap.Error(err))
		return err
	}
	if len(onlineAgents) == 0 {
		return nil
	}

	// 使用统一的方法计算目标探针
	targetAgents := s.resolveTargetAgents(monitor, onlineAgents)
	if len(targetAgents) == 0 {
		return nil
	}

	// 构建监控项
	item := protocol.MonitorItem{
		ID:     monitor.ID,
		Type:   monitor.Type,
		Target: monitor.Target,
	}

	if monitor.Type == "http" || monitor.Type == "https" {
		var httpConfig protocol.HTTPMonitorConfig
		if err := monitor.HTTPConfig.Scan(&httpConfig); err == nil {
			item.HTTPConfig = &httpConfig
		}
	} else if monitor.Type == "tcp" {
		var tcpConfig protocol.TCPMonitorConfig
		if err := monitor.TCPConfig.Scan(&tcpConfig); err == nil {
			item.TCPConfig = &tcpConfig
		}
	} else if monitor.Type == "icmp" || monitor.Type == "ping" {
		var icmpConfig protocol.ICMPMonitorConfig
		if err := monitor.ICMPConfig.Scan(&icmpConfig); err == nil {
			item.ICMPConfig = &icmpConfig
		}
	}

	// 构建 payload
	payload := protocol.MonitorConfigPayload{
		Interval: 0,
		Items:    []protocol.MonitorItem{item},
	}

	// 向每个目标探针发送
	for _, agent := range targetAgents {
		if err := s.sendMonitorConfigToAgent(agent.ID, payload); err != nil {
			s.logger.Error("发送监控配置失败",
				zap.String("taskID", monitor.ID),
				zap.String("taskName", monitor.Name),
				zap.String("agentID", agent.ID),
				zap.Error(err))
		}
	}

	return nil
}

// CalculateMonitorStats 计算监控统计数据
func (s *MonitorService) CalculateMonitorStats(ctx context.Context) error {
	now := time.Now()

	// 获取所有启用的监控任务
	monitors, err := s.MonitorRepo.FindByEnabled(ctx, true)
	if err != nil {
		return err
	}

	// 获取所有探针（包括在线和离线）
	agents, err := s.agentRepo.FindAll(ctx)
	if err != nil {
		return err
	}

	// 收集所有有效的监控任务ID
	validStatsIDs := make(map[string]bool)

	// 为每个监控任务的每个探针计算统计数据
	for _, monitor := range monitors {
		// 使用统一的方法计算目标探针
		targetAgents := s.resolveTargetAgents(monitor, agents)

		for _, agent := range targetAgents {
			stats, err := s.calculateStatsForAgentMonitor(ctx, agent.ID, monitor.ID, monitor.Type, monitor.Target, now)
			if err != nil {
				s.logger.Error("计算监控统计失败",
					zap.String("agentID", agent.ID),
					zap.String("monitorName", monitor.Name),
					zap.Error(err))
				continue
			}

			// 记录有效的监控任务ID
			validStatsIDs[stats.ID] = true

			if err := s.monitorStatsRepo.Save(ctx, stats); err != nil {
				s.logger.Error("保存监控统计失败",
					zap.String("agentID", agent.ID),
					zap.String("monitorName", monitor.Name),
					zap.Error(err))
			}
		}
	}

	// 清理无效监控任务的统计数据
	if err := s.cleanupInvalidStats(ctx, validStatsIDs); err != nil {
		s.logger.Error("清理无效统计数据失败", zap.Error(err))
		// 不返回错误，继续运行
	}

	return nil
}

// calculateStatsForAgentMonitor 计算单个探针单个监控任务的统计数据
func (s *MonitorService) calculateStatsForAgentMonitor(ctx context.Context, agentID, monitorId, monitorType, target string, now time.Time) (*models.MonitorStats, error) {
	stats := &models.MonitorStats{
		ID:          toolkit.Sign("monitor_stats", agentID, monitorId, monitorType, target),
		AgentID:     agentID,
		MonitorId:   monitorId,
		MonitorType: monitorType,
		Target:      target,
	}

	// 计算24小时数据
	start24h := now.Add(-24 * time.Hour).UnixMilli()
	end := now.UnixMilli()
	metrics24h, err := s.metricRepo.GetMonitorMetrics(ctx, agentID, monitorId, start24h, end)
	if err != nil {
		return nil, err
	}

	// 计算7天数据
	start7d := now.Add(-7 * 24 * time.Hour).UnixMilli()
	metrics7d, err := s.metricRepo.GetMonitorMetrics(ctx, agentID, monitorId, start7d, end)
	if err != nil {
		return nil, err
	}

	// 计算24小时统计
	if len(metrics24h) > 0 {
		var totalResponse int64
		var successCount int64
		lastMetric := metrics24h[len(metrics24h)-1]

		for _, metric := range metrics24h {
			if metric.Status == "up" {
				successCount++
				totalResponse += metric.ResponseTime
			}
		}

		stats.TotalChecks24h = int64(len(metrics24h))
		stats.SuccessChecks24h = successCount
		if successCount > 0 {
			stats.AvgResponse24h = totalResponse / successCount
		}
		if stats.TotalChecks24h > 0 {
			stats.Uptime24h = float64(successCount) / float64(stats.TotalChecks24h) * 100
		}

		// 最后一次检测数据
		stats.CurrentResponse = lastMetric.ResponseTime
		stats.LastCheckTime = lastMetric.Timestamp
		stats.LastCheckStatus = lastMetric.Status
		stats.LastCheckError = lastMetric.Error

		// 从最新的检测结果中获取证书信息
		if lastMetric.CertExpiryTime > 0 {
			stats.CertExpiryDate = lastMetric.CertExpiryTime
			stats.CertExpiryDays = lastMetric.CertDaysLeft
		}
	}

	// 计算7天统计
	if len(metrics7d) > 0 {
		var successCount int64
		for _, metric := range metrics7d {
			if metric.Status == "up" {
				successCount++
			}
		}

		stats.TotalChecks7d = int64(len(metrics7d))
		stats.SuccessChecks7d = successCount
		if stats.TotalChecks7d > 0 {
			stats.Uptime7d = float64(successCount) / float64(stats.TotalChecks7d) * 100
		}
	}

	return stats, nil
}

// GetMonitorStatsByID 获取监控任务的统计数据（聚合后的单个监控详情）
func (s *MonitorService) GetMonitorStatsByID(ctx context.Context, monitorID string) (*PublicMonitorOverview, error) {
	// 查询监控任务
	monitor, err := s.MonitorRepo.FindById(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	// 获取统计数据
	statsList, err := s.monitorStatsRepo.FindByMonitorId(ctx, monitor.ID)
	if err != nil {
		return nil, err
	}

	// 聚合统计数据
	summary := aggregateMonitorStats(statsList)

	// 构建监控概览对象
	overview := s.buildMonitorOverview(monitor, summary)

	return &overview, nil
}

// GetMonitorAgentStats 获取监控任务各探针的统计数据（详细列表）
func (s *MonitorService) GetMonitorAgentStats(ctx context.Context, monitorID string) ([]models.MonitorStats, error) {
	// 构建缓存键
	cacheKey := fmt.Sprintf("agents:%s", monitorID)

	// 尝试从缓存获取
	if cachedResult, ok := s.statsCache.Get(cacheKey); ok {
		return cachedResult, nil
	}

	// 缓存未命中，查询数据库
	monitor, err := s.MonitorRepo.FindById(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	statsList, err := s.monitorStatsRepo.FindByMonitorId(ctx, monitor.ID)
	if err != nil {
		return nil, err
	}

	// 获取探针列表
	agents, err := s.agentRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	// 当前监控任务的关联的探针
	targetAgents := s.resolveTargetAgents(monitor, agents)
	var targetAgentsMap = make(map[string]string)
	for _, agent := range targetAgents {
		targetAgentsMap[agent.ID] = agent.Name
	}

	// 填充监控名称、探针名称和隐私设置
	var filteredStatsList []models.MonitorStats
	for _, stats := range statsList {
		agentName, ok := targetAgentsMap[stats.AgentID]
		if !ok {
			continue
		}

		stats.MonitorName = monitor.Name

		// 根据 ShowTargetPublic 字段决定是否隐藏 Target
		if !monitor.ShowTargetPublic {
			stats.Target = "******"
		}

		stats.AgentName = agentName
		filteredStatsList = append(filteredStatsList, stats)
	}

	// 缓存结果
	s.statsCache.Set(cacheKey, filteredStatsList, 5*time.Minute)

	return filteredStatsList, nil
}

// GetMonitorHistory 获取监控任务的历史响应时间数据
// 根据前端时间范围选择，直接使用聚合表数据
// 支持时间范围：15m, 30m, 1h, 3h, 6h, 12h, 1d, 3d, 7d
func (s *MonitorService) GetMonitorHistory(ctx context.Context, monitorID, timeRange string) ([]repo.AggregatedMonitorMetric, error) {
	monitor, err := s.MonitorRepo.FindById(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	// 根据前端时间范围，选择最优的聚合粒度
	var duration time.Duration
	var bucketSeconds int // 使用的聚合 bucket（秒）

	switch timeRange {
	case "15m":
		duration = 15 * time.Minute
		bucketSeconds = 60 // 1分钟聚合，约15个点
	case "30m":
		duration = 30 * time.Minute
		bucketSeconds = 60 // 1分钟聚合，约30个点
	case "1h":
		duration = 1 * time.Hour
		bucketSeconds = 300 // 5分钟聚合，约12个点
	case "3h":
		duration = 3 * time.Hour
		bucketSeconds = 300 // 5分钟聚合，约36个点
	case "6h":
		duration = 6 * time.Hour
		bucketSeconds = 900 // 15分钟聚合，约24个点
	case "12h":
		duration = 12 * time.Hour
		bucketSeconds = 900 // 15分钟聚合，约48个点
	case "1d", "24h":
		duration = 24 * time.Hour
		bucketSeconds = 1800 // 30分钟聚合，约48个点
	case "3d":
		duration = 3 * 24 * time.Hour
		bucketSeconds = 3600 // 1小时聚合，约72个点
	case "7d":
		duration = 7 * 24 * time.Hour
		bucketSeconds = 7200 // 2小时聚合，约84个点
	default:
		// 默认 15 分钟
		duration = 15 * time.Minute
		bucketSeconds = 60
	}

	now := time.Now()
	end := now.UnixMilli()
	start := now.Add(-duration).UnixMilli()

	// 对齐时间范围到 bucket 边界
	bucketMs := int64(bucketSeconds * 1000)
	start, end = alignTimeRangeToBucket(start, end, bucketMs)

	// 直接使用聚合表查询
	return s.metricRepo.GetMonitorMetricsAgg(ctx, monitor.ID, start, end, bucketSeconds)
}

// GetMonitorByAuth 根据认证状态获取监控任务（已登录返回全部，未登录返回公开可见）
func (s *MonitorService) GetMonitorByAuth(ctx context.Context, id string, isAuthenticated bool) (*models.MonitorTask, error) {
	if isAuthenticated {
		monitor, err := s.MonitorRepo.FindById(ctx, id)
		if err != nil {
			return nil, err
		}
		if !monitor.Enabled {
			return nil, fmt.Errorf("monitor is disabled")
		}
		return &monitor, nil
	}
	monitor, err := s.MonitorRepo.FindPublicMonitorByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !monitor.Enabled {
		return nil, fmt.Errorf("monitor is disabled")
	}
	return monitor, nil
}

// clearCache 清理监控任务相关的所有缓存
func (s *MonitorService) clearCache(monitorID string) {
	// 清理概览缓存
	s.overviewCache.Delete("overview:public")
	s.overviewCache.Delete("overview:private")

	// 清理统计缓存
	s.statsCache.Delete(fmt.Sprintf("agents:%s", monitorID))
}

// cleanupInvalidStats 按监控任务维度清理无效的统计数据
// 删除不在有效监控任务列表中的所有统计数据（监控任务被禁用或删除）
func (s *MonitorService) cleanupInvalidStats(ctx context.Context, validStatsIDs map[string]bool) error {
	// 获取所有现有的统计数据
	allStats, err := s.monitorStatsRepo.FindAll(ctx)
	if err != nil {
		return err
	}

	// 收集需要删除的统计数据ID
	idsToDelete := make([]string, 0)
	for _, stats := range allStats {
		// 如果该统计数据对应的监控任务不在有效列表中，删除该统计数据
		if !validStatsIDs[stats.ID] {
			idsToDelete = append(idsToDelete, stats.ID)
		}
	}

	// 批量删除无效的统计数据
	if len(idsToDelete) > 0 {
		s.logger.Info("清理无效的监控统计数据",
			zap.Int("count", len(idsToDelete)))
		if err := s.monitorStatsRepo.DeleteByIDs(ctx, idsToDelete); err != nil {
			return err
		}
	}

	return nil
}
