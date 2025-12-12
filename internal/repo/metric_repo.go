package repo

import (
	"context"

	"github.com/dushixiang/pika/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MetricRepo struct {
	db *gorm.DB
}

func NewMetricRepo(db *gorm.DB) *MetricRepo {
	return &MetricRepo{
		db: db,
	}
}

// SaveHostMetric 保存主机信息指标（按 agent 覆盖，避免先删后插的空窗）
func (r *MetricRepo) SaveHostMetric(ctx context.Context, metric *models.HostMetric) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "agent_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"os", "platform", "platform_version", "kernel_version", "kernel_arch", "uptime", "boot_time", "procs", "timestamp"}),
		}).
		Create(metric).Error
}

// GetAvailableNetworkInterfaces 获取探针的可用网卡列表
// TODO: 需要改为从 VictoriaMetrics 查询 labels
func (r *MetricRepo) GetAvailableNetworkInterfaces(ctx context.Context, agentID string) ([]string, error) {
	// 暂时返回空列表，后续改为从 VictoriaMetrics 查询
	// 使用 GET /api/v1/label/interface/values?match[]=pika_network_sent_bytes_rate{agent_id="xxx"}
	return []string{}, nil
}

// DeleteAgentMetrics 删除指定探针的所有指标数据
// TODO: 需要改为调用 VictoriaMetrics 的 DELETE API
func (r *MetricRepo) DeleteAgentMetrics(ctx context.Context, agentID string) error {
	// 1. 删除 PostgreSQL 中的主机信息
	if err := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Delete(&models.HostMetric{}).Error; err != nil {
		return err
	}

	// 2. TODO: 调用 VictoriaMetrics 删除时间序列数据
	// DELETE /api/v1/admin/tsdb/delete_series?match[]=pika_*{agent_id="xxx"}

	return nil
}

// 以下是兼容性数据结构定义，用于返回查询结果
// TODO: 后续从 VictoriaMetrics 查询时使用这些结构

// AggregatedCPUMetric CPU聚合指标
type AggregatedCPUMetric struct {
	Timestamp    int64   `json:"timestamp"`
	MaxUsage     float64 `json:"maxUsage"`
	LogicalCores int     `json:"logicalCores"`
}

// AggregatedMemoryMetric 内存聚合指标
type AggregatedMemoryMetric struct {
	Timestamp int64   `json:"timestamp"`
	MaxUsage  float64 `json:"maxUsage"`
	Total     uint64  `json:"total"`
}

// AggregatedDiskMetric 磁盘聚合指标
type AggregatedDiskMetric struct {
	Timestamp  int64   `json:"timestamp"`
	MountPoint string  `json:"mountPoint"`
	MaxUsage   float64 `json:"maxUsage"`
	Total      uint64  `json:"total"`
}

// AggregatedNetworkMetric 网络聚合指标
type AggregatedNetworkMetric struct {
	Timestamp   int64   `json:"timestamp"`
	Interface   string  `json:"interface"`
	MaxSentRate float64 `json:"maxSentRate"`
	MaxRecvRate float64 `json:"maxRecvRate"`
}

// AggregatedNetworkConnectionMetric 网络连接统计聚合指标
type AggregatedNetworkConnectionMetric struct {
	Timestamp      int64  `json:"timestamp"`
	MaxEstablished uint32 `json:"maxEstablished"`
	MaxSynSent     uint32 `json:"maxSynSent"`
	MaxSynRecv     uint32 `json:"maxSynRecv"`
	MaxFinWait1    uint32 `json:"maxFinWait1"`
	MaxFinWait2    uint32 `json:"maxFinWait2"`
	MaxTimeWait    uint32 `json:"maxTimeWait"`
	MaxClose       uint32 `json:"maxClose"`
	MaxCloseWait   uint32 `json:"maxCloseWait"`
	MaxLastAck     uint32 `json:"maxLastAck"`
	MaxListen      uint32 `json:"maxListen"`
	MaxClosing     uint32 `json:"maxClosing"`
	MaxTotal       uint32 `json:"maxTotal"`
}

// AggregatedDiskIOMetric 磁盘IO聚合指标
type AggregatedDiskIOMetric struct {
	Timestamp         int64   `json:"timestamp"`
	MaxReadRate       float64 `json:"maxReadRate"`
	MaxWriteRate      float64 `json:"maxWriteRate"`
	TotalReadBytes    uint64  `json:"totalReadBytes"`
	TotalWriteBytes   uint64  `json:"totalWriteBytes"`
	MaxIopsInProgress uint64  `json:"maxIopsInProgress"`
}

// AggregatedGPUMetric GPU聚合指标
type AggregatedGPUMetric struct {
	Timestamp      int64   `json:"timestamp"`
	Index          int     `json:"index"`
	Name           string  `json:"name"`
	MaxUtilization float64 `json:"maxUtilization"`
	MaxMemoryUsed  uint64  `json:"maxMemoryUsed"`
	MaxTemperature float64 `json:"maxTemperature"`
	MaxPowerDraw   float64 `json:"maxPowerDraw"`
	MemoryTotal    uint64  `json:"memoryTotal"`
}

// AggregatedTemperatureMetric 温度聚合指标
type AggregatedTemperatureMetric struct {
	Timestamp      int64   `json:"timestamp"`
	SensorKey      string  `json:"sensorKey"`
	SensorLabel    string  `json:"sensorLabel"`
	MaxTemperature float64 `json:"maxTemperature"`
}

// AggregatedMonitorMetric 聚合的监控指标
type AggregatedMonitorMetric struct {
	Timestamp    int64   `json:"timestamp"`
	AgentID      string  `json:"agentId"`
	AvgResponse  float64 `json:"avgResponse"`
	MaxResponse  int64   `json:"maxResponse"`
	MinResponse  int64   `json:"minResponse"`
	SuccessCount int64   `json:"successCount"`
	TotalCount   int64   `json:"totalCount"`
	SuccessRate  float64 `json:"successRate"`
	LastStatus   string  `json:"lastStatus"`
	LastErrorMsg string  `json:"lastErrorMsg"`
}

// GetMonitorMetrics 获取监控指标历史数据
// TODO: 需要改为从 VictoriaMetrics 查询
func (r *MetricRepo) GetMonitorMetrics(ctx context.Context, agentID, monitorID string, start, end int64) ([]MonitorMetric, error) {
	// 暂时返回空列表，后续改为从 VictoriaMetrics 查询
	return []MonitorMetric{}, nil
}

// GetMonitorMetricsByName 获取指定监控项的历史数据
// TODO: 需要改为从 VictoriaMetrics 查询
func (r *MetricRepo) GetMonitorMetricsByName(ctx context.Context, agentID, monitorID string, start, end int64, limit int) ([]MonitorMetric, error) {
	// 暂时返回空列表，后续改为从 VictoriaMetrics 查询
	return []MonitorMetric{}, nil
}

// GetLatestMonitorMetricsByType 获取指定类型的最新监控指标（所有探针）
// TODO: 需要改为从 VictoriaMetrics 查询
func (r *MetricRepo) GetLatestMonitorMetricsByType(ctx context.Context, monitorType string) ([]*MonitorMetric, error) {
	// 暂时返回空列表，后续改为从 VictoriaMetrics 查询
	// 使用 GET /api/v1/query?query=pika_monitor_status{type="xxx"}
	return []*MonitorMetric{}, nil
}

// GetAllLatestMonitorMetrics 获取所有最新的监控指标（所有探针的所有监控项）
// TODO: 需要改为从 VictoriaMetrics 查询
func (r *MetricRepo) GetAllLatestMonitorMetrics(ctx context.Context) ([]*MonitorMetric, error) {
	// 暂时返回空列表，后续改为从 VictoriaMetrics 查询
	// 使用 GET /api/v1/query?query=pika_monitor_status
	return []*MonitorMetric{}, nil
}

// DeleteMonitorMetrics 删除指定监控任务的所有指标数据
// TODO: 需要改为调用 VictoriaMetrics 删除 API
func (r *MetricRepo) DeleteMonitorMetrics(ctx context.Context, monitorID string) error {
	// 暂时不删除数据，后续调用 VictoriaMetrics 删除 API
	// DELETE /api/v1/admin/tsdb/delete_series?match[]=pika_monitor_*{monitor_id="xxx"}
	return nil
}

// GetMonitorMetricsAgg 获取聚合后的监控指标
// TODO: 需要改为从 VictoriaMetrics 查询
func (r *MetricRepo) GetMonitorMetricsAgg(ctx context.Context, monitorID string, start, end int64, bucketSeconds int) ([]AggregatedMonitorMetric, error) {
	// 暂时返回空列表，后续从 VictoriaMetrics 查询
	return []AggregatedMonitorMetric{}, nil
}

// MonitorMetric 监控指标（与service层定义保持一致）
type MonitorMetric struct {
	ID             uint   `json:"id"`
	AgentId        string `json:"agentId"`
	MonitorId      string `json:"monitorId"`
	Type           string `json:"type"`
	Target         string `json:"target"`
	Status         string `json:"status"`
	StatusCode     int    `json:"statusCode"`
	ResponseTime   int64  `json:"responseTime"`
	Error          string `json:"error"`
	Message        string `json:"message"`
	ContentMatch   bool   `json:"contentMatch"`
	CertExpiryTime int64  `json:"certExpiryTime"`
	CertDaysLeft   int    `json:"certDaysLeft"`
	Timestamp      int64  `json:"timestamp"`
}

// 注意：所有指标数据已迁移到 VictoriaMetrics
// 查询方法已移至 MetricService，通过 VMClient 访问
