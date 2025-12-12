package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dushixiang/pika/internal/models"
	"github.com/dushixiang/pika/internal/protocol"
	"github.com/dushixiang/pika/internal/repo"
	"github.com/dushixiang/pika/internal/vmclient"

	"github.com/go-orz/cache"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MetricDataPoint 统一的指标数据点结构
type MetricDataPoint struct {
	Timestamp int64   `json:"timestamp"` // 毫秒时间戳
	Value     float64 `json:"value"`
}

// MetricSeries 指标系列（支持多系列，如多网卡、多传感器）
type MetricSeries struct {
	Name   string            `json:"name"`             // 系列名称
	Labels map[string]string `json:"labels,omitempty"` // 额外标签
	Data   []MetricDataPoint `json:"data"`             // 数据点列表
}

// GetMetricsResponse 统一的查询响应格式
type GetMetricsResponse struct {
	AgentID string         `json:"agentId"`
	Type    string         `json:"type"`
	Range   string         `json:"range"`
	Series  []MetricSeries `json:"series"`
}

// QueryDefinition 查询定义（用于构建多个查询）
type QueryDefinition struct {
	Name   string            // 系列名称
	Query  string            // PromQL 查询语句
	Labels map[string]string // 额外标签
}

// MetricService 指标服务
type MetricService struct {
	logger           *zap.Logger
	metricRepo       *repo.MetricRepo
	monitorStatsRepo *repo.MonitorStatsRepo
	propertyService  *PropertyService
	trafficService   *TrafficService // 流量统计服务
	vmClient         *vmclient.VMClient

	latestCache cache.Cache[string, *LatestMetrics]
}

// NewMetricService 创建指标服务
func NewMetricService(logger *zap.Logger, db *gorm.DB, propertyService *PropertyService, trafficService *TrafficService, vmClient *vmclient.VMClient) *MetricService {
	return &MetricService{
		logger:           logger,
		metricRepo:       repo.NewMetricRepo(db),
		monitorStatsRepo: repo.NewMonitorStatsRepo(db),
		propertyService:  propertyService,
		trafficService:   trafficService,
		vmClient:         vmClient,
		latestCache:      cache.New[string, *LatestMetrics](time.Minute),
	}
}

// HandleMetricData 处理指标数据
func (s *MetricService) HandleMetricData(ctx context.Context, agentID string, metricType string, data json.RawMessage) error {
	now := time.Now().UnixMilli()

	// 更新内存缓存
	latestMetrics, ok := s.latestCache.Get(agentID)
	if !ok {
		latestMetrics = &LatestMetrics{}
		s.latestCache.Set(agentID, latestMetrics, time.Hour)
	}

	// 解析数据并写入 VictoriaMetrics
	switch protocol.MetricType(metricType) {
	case protocol.MetricTypeCPU:
		var cpuData protocol.CPUData
		if err := json.Unmarshal(data, &cpuData); err != nil {
			return err
		}
		metric := &CPUMetric{
			AgentID:       agentID,
			UsagePercent:  cpuData.UsagePercent,
			LogicalCores:  cpuData.LogicalCores,
			PhysicalCores: cpuData.PhysicalCores,
			ModelName:     cpuData.ModelName,
			Timestamp:     now,
		}
		latestMetrics.CPU = metric
		metrics := s.convertToMetrics(agentID, metricType, &cpuData, now)
		return s.vmClient.Write(ctx, metrics)

	case protocol.MetricTypeMemory:
		var memData protocol.MemoryData
		if err := json.Unmarshal(data, &memData); err != nil {
			return err
		}
		metric := &MemoryMetric{
			AgentID:      agentID,
			Total:        memData.Total,
			Used:         memData.Used,
			Free:         memData.Free,
			Available:    memData.Available,
			UsagePercent: memData.UsagePercent,
			SwapTotal:    memData.SwapTotal,
			SwapUsed:     memData.SwapUsed,
			SwapFree:     memData.SwapFree,
			Timestamp:    now,
		}
		latestMetrics.Memory = metric
		metrics := s.convertToMetrics(agentID, metricType, &memData, now)
		return s.vmClient.Write(ctx, metrics)

	case protocol.MetricTypeDisk:
		var diskDataList []protocol.DiskData
		if err := json.Unmarshal(data, &diskDataList); err != nil {
			return err
		}
		// 计算汇总数据用于缓存
		var totalTotal, totalUsed, totalFree uint64
		for _, diskData := range diskDataList {
			totalTotal += diskData.Total
			totalUsed += diskData.Used
			totalFree += diskData.Free
		}
		var usagePercent float64
		if totalTotal > 0 {
			usagePercent = float64(totalUsed) / float64(totalTotal) * 100
		}
		latestMetrics.Disk = &DiskSummary{
			UsagePercent: usagePercent,
			TotalDisks:   len(diskDataList),
			Total:        totalTotal,
			Used:         totalUsed,
			Free:         totalFree,
		}
		metrics := s.convertToMetrics(agentID, metricType, diskDataList, now)
		return s.vmClient.Write(ctx, metrics)

	case protocol.MetricTypeNetwork:
		var networkDataList []protocol.NetworkData
		if err := json.Unmarshal(data, &networkDataList); err != nil {
			return err
		}
		// 计算汇总数据用于缓存
		var totalSentRate, totalRecvRate uint64
		var totalSentTotal, totalRecvTotal uint64
		for _, netData := range networkDataList {
			totalSentRate += netData.BytesSentRate
			totalRecvRate += netData.BytesRecvRate
			totalSentTotal += netData.BytesSentTotal
			totalRecvTotal += netData.BytesRecvTotal
		}
		latestMetrics.Network = &NetworkSummary{
			TotalBytesSentRate:  totalSentRate,
			TotalBytesRecvRate:  totalRecvRate,
			TotalBytesSentTotal: totalSentTotal,
			TotalBytesRecvTotal: totalRecvTotal,
			TotalInterfaces:     len(networkDataList),
		}
		// 更新流量统计
		if err := s.trafficService.UpdateAgentTraffic(ctx, agentID, totalRecvTotal); err != nil {
			s.logger.Error("更新探针流量统计失败",
				zap.String("agentId", agentID),
				zap.Error(err))
		}
		metrics := s.convertToMetrics(agentID, metricType, networkDataList, now)
		return s.vmClient.Write(ctx, metrics)

	case protocol.MetricTypeNetworkConnection:
		var connData protocol.NetworkConnectionData
		if err := json.Unmarshal(data, &connData); err != nil {
			return err
		}
		metric := &NetworkConnectionMetric{
			AgentID:     agentID,
			Established: connData.Established,
			SynSent:     connData.SynSent,
			SynRecv:     connData.SynRecv,
			FinWait1:    connData.FinWait1,
			FinWait2:    connData.FinWait2,
			TimeWait:    connData.TimeWait,
			Close:       connData.Close,
			CloseWait:   connData.CloseWait,
			LastAck:     connData.LastAck,
			Listen:      connData.Listen,
			Closing:     connData.Closing,
			Total:       connData.Total,
			Timestamp:   now,
		}
		latestMetrics.NetworkConnection = metric
		metrics := s.convertToMetrics(agentID, metricType, &connData, now)
		return s.vmClient.Write(ctx, metrics)

	case protocol.MetricTypeDiskIO:
		var diskIODataList []*protocol.DiskIOData
		if err := json.Unmarshal(data, &diskIODataList); err != nil {
			return err
		}
		metrics := s.convertToMetrics(agentID, metricType, diskIODataList, now)
		return s.vmClient.Write(ctx, metrics)

	case protocol.MetricTypeHost:
		var hostData protocol.HostInfoData
		if err := json.Unmarshal(data, &hostData); err != nil {
			return err
		}
		// Host 信息仍然保存到 PostgreSQL（静态信息，不频繁变化）
		metric := &models.HostMetric{
			AgentID:         agentID,
			OS:              hostData.OS,
			Platform:        hostData.Platform,
			PlatformVersion: hostData.PlatformVersion,
			KernelVersion:   hostData.KernelVersion,
			KernelArch:      hostData.KernelArch,
			Uptime:          hostData.Uptime,
			BootTime:        hostData.BootTime,
			Procs:           hostData.Procs,
			Timestamp:       now,
		}
		latestMetrics.Host = metric
		return s.metricRepo.SaveHostMetric(ctx, metric)

	case protocol.MetricTypeGPU:
		var gpuDataList []protocol.GPUData
		if err := json.Unmarshal(data, &gpuDataList); err != nil {
			return err
		}
		// 更新缓存
		var gpuMetrics []GPUMetric
		for _, gpuData := range gpuDataList {
			gpuMetrics = append(gpuMetrics, GPUMetric{
				AgentID:          agentID,
				Index:            gpuData.Index,
				Name:             gpuData.Name,
				Utilization:      gpuData.Utilization,
				MemoryTotal:      gpuData.MemoryTotal,
				MemoryUsed:       gpuData.MemoryUsed,
				MemoryFree:       gpuData.MemoryFree,
				Temperature:      gpuData.Temperature,
				PowerDraw:        gpuData.PowerUsage,
				FanSpeed:         gpuData.FanSpeed,
				PerformanceState: "",
				Timestamp:        now,
			})
		}
		latestMetrics.GPU = gpuMetrics
		metrics := s.convertToMetrics(agentID, metricType, gpuDataList, now)
		return s.vmClient.Write(ctx, metrics)

	case protocol.MetricTypeTemperature:
		var tempDataList []protocol.TemperatureData
		if err := json.Unmarshal(data, &tempDataList); err != nil {
			return err
		}
		// 更新缓存
		var tempMetrics []TemperatureMetric
		for _, tempData := range tempDataList {
			sensorLabel := tempData.Type
			if sensorLabel == "" {
				sensorLabel = tempData.SensorKey
			}
			tempMetrics = append(tempMetrics, TemperatureMetric{
				AgentID:     agentID,
				SensorKey:   tempData.SensorKey,
				SensorLabel: sensorLabel,
				Temperature: tempData.Temperature,
				Timestamp:   now,
			})
		}
		latestMetrics.Temp = tempMetrics
		metrics := s.convertToMetrics(agentID, metricType, tempDataList, now)
		return s.vmClient.Write(ctx, metrics)

	case protocol.MetricTypeMonitor:
		var monitorDataList []protocol.MonitorData
		if err := json.Unmarshal(data, &monitorDataList); err != nil {
			return err
		}
		metrics := s.convertToMetrics(agentID, metricType, monitorDataList, now)
		return s.vmClient.Write(ctx, metrics)

	default:
		s.logger.Warn("unknown metric type", zap.String("type", metricType))
		return nil
	}
}

// GetMetrics 获取聚合指标数据（从 VictoriaMetrics 查询）
// 返回统一的 GetMetricsResponse 格式
func (s *MetricService) GetMetrics(ctx context.Context, agentID, metricType string, start, end int64, interfaceName string) (*GetMetricsResponse, error) {
	// 构造 PromQL 查询（返回多个查询以支持多系列）
	queries := s.buildPromQLQueries(agentID, metricType, interfaceName)
	if len(queries) == 0 {
		return nil, fmt.Errorf("unsupported metric type: %s", metricType)
	}

	// 执行查询并转换结果
	// step 设为 0，让 VictoriaMetrics 自动选择合适的步长
	var series []MetricSeries

	for _, q := range queries {
		result, err := s.vmClient.QueryRange(ctx, q.Query,
			time.UnixMilli(start),
			time.UnixMilli(end),
			0)
		if err != nil {
			s.logger.Error("查询 VictoriaMetrics 失败",
				zap.String("query", q.Query),
				zap.Error(err))
			continue // 跳过失败的查询，继续处理其他查询
		}

		// 转换查询结果为 MetricSeries
		convertedSeries := s.convertQueryResultToSeries(result, q.Name, q.Labels)
		series = append(series, convertedSeries...)
	}

	return &GetMetricsResponse{
		AgentID: agentID,
		Type:    metricType,
		Range:   fmt.Sprintf("%d-%d", start, end),
		Series:  series,
	}, nil
}

// alignTimeRangeToBucket 将时间范围对齐到桶边界，确保不同时间框架的桶数一致
func alignTimeRangeToBucket(start, end int64, bucketMs int64) (int64, int64) {
	if bucketMs <= 0 {
		return start, end
	}
	alignedStart := (start / bucketMs) * bucketMs
	endBucket := ((end - 1) / bucketMs) * bucketMs
	alignedEnd := endBucket + bucketMs - 1
	if alignedEnd < alignedStart {
		alignedEnd = alignedStart
	}
	return alignedStart, alignedEnd
}

// GetLatestMetrics 获取最新指标
func (s *MetricService) GetLatestMetrics(ctx context.Context, agentID string) (*LatestMetrics, error) {
	metrics, _ := s.latestCache.Get(agentID)
	return metrics, nil
}

// GetMonitorMetrics 获取监控指标历史数据
// TODO: 需要重写为从 VictoriaMetrics 查询
func (s *MetricService) GetMonitorMetrics(ctx context.Context, agentID, monitorName string, start, end int64) ([]MonitorMetric, error) {
	// 暂时返回空数据，后续从 VictoriaMetrics 查询
	return []MonitorMetric{}, nil
}

// GetMonitorMetricsByName 获取指定监控项的历史数据
// TODO: 需要重写为从 VictoriaMetrics 查询
func (s *MetricService) GetMonitorMetricsByName(ctx context.Context, agentID, monitorName string, start, end int64, limit int) ([]MonitorMetric, error) {
	// 暂时返回空数据，后续从 VictoriaMetrics 查询
	return []MonitorMetric{}, nil
}

// DeleteAgentMetrics 删除探针的所有指标数据
func (s *MetricService) DeleteAgentMetrics(ctx context.Context, agentID string) error {
	// 1. 删除 PostgreSQL 中的主机信息
	if err := s.metricRepo.DeleteAgentMetrics(ctx, agentID); err != nil {
		s.logger.Error("删除 PostgreSQL 中的探针数据失败",
			zap.String("agentID", agentID),
			zap.Error(err))
		// 继续删除 VictoriaMetrics 中的数据
	}

	// 2. 删除 VictoriaMetrics 中的时间序列数据
	match := []string{fmt.Sprintf(`pika_.*{agent_id="%s"}`, agentID)}
	if err := s.vmClient.DeleteSeries(ctx, match); err != nil {
		s.logger.Error("删除 VictoriaMetrics 中的探针数据失败",
			zap.String("agentID", agentID),
			zap.Error(err))
		return err
	}

	s.logger.Info("成功删除探针的所有指标数据",
		zap.String("agentID", agentID))
	return nil
}

// DeleteMonitorMetrics 删除指定监控任务的所有指标数据
func (s *MetricService) DeleteMonitorMetrics(ctx context.Context, monitorID string) error {
	// 删除 VictoriaMetrics 中的监控指标数据
	match := []string{fmt.Sprintf(`pika_monitor_.*{monitor_id="%s"}`, monitorID)}
	if err := s.vmClient.DeleteSeries(ctx, match); err != nil {
		s.logger.Error("删除 VictoriaMetrics 中的监控数据失败",
			zap.String("monitorID", monitorID),
			zap.Error(err))
		return err
	}

	s.logger.Info("成功删除监控任务的所有指标数据",
		zap.String("monitorID", monitorID))
	return nil
}

// GetAvailableNetworkInterfaces 获取探针的可用网卡列表（从 VictoriaMetrics 查询）
func (s *MetricService) GetAvailableNetworkInterfaces(ctx context.Context, agentID string) ([]string, error) {
	// 查询 interface label 的所有值，排除空字符串（汇总数据）
	match := []string{fmt.Sprintf(`pika_network_sent_bytes_rate{agent_id="%s"}`, agentID)}
	allInterfaces, err := s.vmClient.GetLabelValues(ctx, "interface", match)
	if err != nil {
		s.logger.Error("查询网卡列表失败",
			zap.String("agentID", agentID),
			zap.Error(err))
		return []string{}, nil // 返回空列表而不是错误
	}

	// 过滤掉空字符串（汇总数据）
	interfaces := make([]string, 0, len(allInterfaces))
	for _, iface := range allInterfaces {
		if iface != "" {
			interfaces = append(interfaces, iface)
		}
	}

	return interfaces, nil
}

// ===== 内存缓存使用的本地模型定义 =====
// 注意：这些模型仅用于内存缓存，不再保存到 PostgreSQL

// CPUMetric CPU指标（内存缓存）
type CPUMetric struct {
	AgentID       string  `json:"agentId"`
	UsagePercent  float64 `json:"usagePercent"`
	LogicalCores  int     `json:"logicalCores"`
	PhysicalCores int     `json:"physicalCores"`
	ModelName     string  `json:"modelName"`
	Timestamp     int64   `json:"timestamp"`
}

// MemoryMetric 内存指标（内存缓存）
type MemoryMetric struct {
	AgentID      string  `json:"agentId"`
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	Available    uint64  `json:"available"`
	UsagePercent float64 `json:"usagePercent"`
	SwapTotal    uint64  `json:"swapTotal"`
	SwapUsed     uint64  `json:"swapUsed"`
	SwapFree     uint64  `json:"swapFree"`
	Timestamp    int64   `json:"timestamp"`
}

// NetworkConnectionMetric 网络连接统计指标（内存缓存）
type NetworkConnectionMetric struct {
	AgentID     string `json:"agentId"`
	Established uint32 `json:"established"`
	SynSent     uint32 `json:"synSent"`
	SynRecv     uint32 `json:"synRecv"`
	FinWait1    uint32 `json:"finWait1"`
	FinWait2    uint32 `json:"finWait2"`
	TimeWait    uint32 `json:"timeWait"`
	Close       uint32 `json:"close"`
	CloseWait   uint32 `json:"closeWait"`
	LastAck     uint32 `json:"lastAck"`
	Listen      uint32 `json:"listen"`
	Closing     uint32 `json:"closing"`
	Total       uint32 `json:"total"`
	Timestamp   int64  `json:"timestamp"`
}

// GPUMetric GPU指标（内存缓存）
type GPUMetric struct {
	AgentID          string  `json:"agentId"`
	Index            int     `json:"index"`
	Name             string  `json:"name"`
	Utilization      float64 `json:"utilization"`
	MemoryTotal      uint64  `json:"memoryTotal"`
	MemoryUsed       uint64  `json:"memoryUsed"`
	MemoryFree       uint64  `json:"memoryFree"`
	Temperature      float64 `json:"temperature"`
	PowerDraw        float64 `json:"powerDraw"`
	FanSpeed         float64 `json:"fanSpeed"`
	PerformanceState string  `json:"performanceState"`
	Timestamp        int64   `json:"timestamp"`
}

// TemperatureMetric 温度指标（内存缓存）
type TemperatureMetric struct {
	AgentID     string  `json:"agentId"`
	SensorKey   string  `json:"sensorKey"`
	SensorLabel string  `json:"sensorLabel"`
	Temperature float64 `json:"temperature"`
	Timestamp   int64   `json:"timestamp"`
}

// MonitorMetric 监控指标
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

// DiskSummary 磁盘汇总数据
type DiskSummary struct {
	UsagePercent float64 `json:"usagePercent"` // 平均使用率
	TotalDisks   int     `json:"totalDisks"`   // 磁盘数量
	Total        uint64  `json:"total"`        // 总容量(字节)
	Used         uint64  `json:"used"`         // 已使用(字节)
	Free         uint64  `json:"free"`         // 空闲(字节)
}

// NetworkSummary 网络汇总数据
type NetworkSummary struct {
	TotalBytesSentRate  uint64 `json:"totalBytesSentRate"`  // 总发送速率(字节/秒)
	TotalBytesRecvRate  uint64 `json:"totalBytesRecvRate"`  // 总接收速率(字节/秒)
	TotalBytesSentTotal uint64 `json:"totalBytesSentTotal"` // 累计总发送流量
	TotalBytesRecvTotal uint64 `json:"totalBytesRecvTotal"` // 累计总接收流量
	TotalInterfaces     int    `json:"totalInterfaces"`     // 网卡数量
}

// LatestMetrics 最新指标数据（用于API响应）
type LatestMetrics struct {
	CPU               *CPUMetric               `json:"cpu,omitempty"`
	Memory            *MemoryMetric            `json:"memory,omitempty"`
	Disk              *DiskSummary             `json:"disk,omitempty"`
	Network           *NetworkSummary          `json:"network,omitempty"`
	NetworkConnection *NetworkConnectionMetric `json:"networkConnection,omitempty"`
	Host              *models.HostMetric       `json:"host,omitempty"`
	GPU               []GPUMetric              `json:"gpu,omitempty"`
	Temp              []TemperatureMetric      `json:"temperature,omitempty"`
}

// buildPromQLQueries 构造 PromQL 查询列表（支持多系列）
func (s *MetricService) buildPromQLQueries(agentID, metricType string, interfaceName string) []QueryDefinition {
	var queries []QueryDefinition

	switch metricType {
	case "cpu":
		queries = []QueryDefinition{{
			Name:  "usage",
			Query: fmt.Sprintf(`pika_cpu_usage_percent{agent_id="%s"}`, agentID),
		}}

	case "memory":
		queries = []QueryDefinition{{
			Name:  "usage",
			Query: fmt.Sprintf(`pika_memory_usage_percent{agent_id="%s"}`, agentID),
		}}

	case "disk":
		queries = []QueryDefinition{{
			Name:  "usage",
			Query: fmt.Sprintf(`pika_disk_usage_percent{agent_id="%s",mount_point=""}`, agentID),
		}}

	case "network":
		// 网络流量：上行和下行
		if interfaceName != "" && interfaceName != "all" {
			// 指定网卡
			queries = []QueryDefinition{
				{
					Name:   "upload",
					Query:  fmt.Sprintf(`pika_network_sent_bytes_rate{agent_id="%s",interface="%s"}`, agentID, interfaceName),
					Labels: map[string]string{"interface": interfaceName},
				},
				{
					Name:   "download",
					Query:  fmt.Sprintf(`pika_network_recv_bytes_rate{agent_id="%s",interface="%s"}`, agentID, interfaceName),
					Labels: map[string]string{"interface": interfaceName},
				},
			}
		} else {
			// 所有网卡汇总
			queries = []QueryDefinition{
				{
					Name:  "upload",
					Query: fmt.Sprintf(`sum(pika_network_sent_bytes_rate{agent_id="%s"}) by (agent_id)`, agentID),
				},
				{
					Name:  "download",
					Query: fmt.Sprintf(`sum(pika_network_recv_bytes_rate{agent_id="%s"}) by (agent_id)`, agentID),
				},
			}
		}

	case "network_connection":
		// 网络连接统计：多个状态
		queries = []QueryDefinition{
			{Name: "established", Query: fmt.Sprintf(`pika_network_conn_established{agent_id="%s"}`, agentID)},
			{Name: "time_wait", Query: fmt.Sprintf(`pika_network_conn_time_wait{agent_id="%s"}`, agentID)},
			{Name: "close_wait", Query: fmt.Sprintf(`pika_network_conn_close_wait{agent_id="%s"}`, agentID)},
			{Name: "listen", Query: fmt.Sprintf(`pika_network_conn_total{agent_id="%s"}`, agentID)},
		}

	case "disk_io":
		// 磁盘 IO：读和写
		queries = []QueryDefinition{
			{Name: "read", Query: fmt.Sprintf(`pika_disk_read_bytes_rate{agent_id="%s"}`, agentID)},
			{Name: "write", Query: fmt.Sprintf(`pika_disk_write_bytes_rate{agent_id="%s"}`, agentID)},
		}

	case "gpu":
		// GPU：利用率和温度（按 GPU 分组）
		queries = []QueryDefinition{
			{
				Name:  "utilization",
				Query: fmt.Sprintf(`pika_gpu_utilization_percent{agent_id="%s"}`, agentID),
			},
			{
				Name:  "temperature",
				Query: fmt.Sprintf(`pika_gpu_temperature_celsius{agent_id="%s"}`, agentID),
			},
		}

	case "temperature":
		// 温度：按传感器类型分组
		queries = []QueryDefinition{{
			Name:  "temperature",
			Query: fmt.Sprintf(`pika_temperature_celsius{agent_id="%s"}`, agentID),
		}}
	}

	return queries
}

// convertQueryResultToSeries 将 VictoriaMetrics 查询结果转换为 MetricSeries
func (s *MetricService) convertQueryResultToSeries(result *vmclient.QueryResult, seriesName string, extraLabels map[string]string) []MetricSeries {
	if result == nil || len(result.Data.Result) == 0 {
		return []MetricSeries{}
	}

	var allSeries []MetricSeries

	// 遍历所有时间序列
	for _, timeSeries := range result.Data.Result {
		// 提取数据点
		var dataPoints []MetricDataPoint
		for _, valueArray := range timeSeries.Values {
			if len(valueArray) != 2 {
				continue
			}

			// valueArray: [timestamp(float64), value(string)]
			timestamp, ok := valueArray[0].(float64)
			if !ok {
				continue
			}
			valueStr, ok := valueArray[1].(string)
			if !ok {
				continue
			}

			var value float64
			fmt.Sscanf(valueStr, "%f", &value)

			dataPoints = append(dataPoints, MetricDataPoint{
				Timestamp: int64(timestamp * 1000), // 转换为毫秒
				Value:     value,
			})
		}

		// 合并标签
		labels := make(map[string]string)
		for k, v := range timeSeries.Metric {
			// 排除内部标签
			if k != "__name__" && k != "agent_id" {
				labels[k] = v
			}
		}
		// 添加额外标签
		for k, v := range extraLabels {
			labels[k] = v
		}

		// 构建系列名称（如果有特定标签如 GPU index 或 sensor_label，添加到名称中）
		finalName := seriesName
		if sensorLabel, ok := labels["sensor_label"]; ok {
			finalName = sensorLabel
			delete(labels, "sensor_label") // 已合并到名称中，从标签中删除
		} else if gpuIndex, ok := labels["gpu_index"]; ok {
			finalName = fmt.Sprintf("GPU_%s", gpuIndex)
			delete(labels, "gpu_index")
		}

		allSeries = append(allSeries, MetricSeries{
			Name:   finalName,
			Labels: labels,
			Data:   dataPoints,
		})
	}

	return allSeries
}
