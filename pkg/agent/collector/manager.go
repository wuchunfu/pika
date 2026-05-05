package collector

import (
	"errors"
	"time"

	"github.com/dushixiang/pika/internal/protocol"
	"github.com/dushixiang/pika/pkg/agent/config"
)

// ErrNoData 表示采集成功但当前周期无有效数据可上报（如 GPU 不存在、磁盘列表为空）。
// 调用方应静默跳过，不视为错误。
var ErrNoData = errors.New("no data")

// Manager 采集器管理器：纯采集，不做 I/O
type Manager struct {
	cpuCollector               *CPUCollector
	memoryCollector            *MemoryCollector
	diskCollector              *DiskCollector
	diskIOCollector            *DiskIOCollector
	networkCollector           *NetworkCollector
	networkConnectionCollector *NetworkConnectionCollector
	hostCollector              *HostCollector
	temperatureCollector       *TemperatureCollector
	gpuCollector               *GPUCollector
	monitorCollector           *MonitorCollector
}

// NewManager 创建采集器管理器
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cpuCollector:               NewCPUCollector(),
		memoryCollector:            NewMemoryCollector(),
		diskCollector:              NewDiskCollector(cfg),
		diskIOCollector:            NewDiskIOCollector(),
		networkCollector:           NewNetworkCollector(cfg),
		networkConnectionCollector: NewNetworkConnectionCollector(),
		hostCollector:              NewHostCollector(),
		temperatureCollector:       NewTemperatureCollector(),
		gpuCollector:               NewGPUCollector(),
		monitorCollector:           NewMonitorCollector(),
	}
}

// makeSample 构造一条 MetricSample
func makeSample(t protocol.MetricType, data interface{}) protocol.MetricSample {
	return protocol.MetricSample{
		Type:      t,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
}

// CollectCPU 采集 CPU 指标
func (m *Manager) CollectCPU() (protocol.MetricSample, error) {
	data, err := m.cpuCollector.Collect()
	if err != nil {
		return protocol.MetricSample{}, err
	}
	return makeSample(protocol.MetricTypeCPU, data), nil
}

// CollectMemory 采集内存指标
func (m *Manager) CollectMemory() (protocol.MetricSample, error) {
	data, err := m.memoryCollector.Collect()
	if err != nil {
		return protocol.MetricSample{}, err
	}
	return makeSample(protocol.MetricTypeMemory, data), nil
}

// CollectDisk 采集磁盘指标。无有效数据时返回 ErrNoData。
func (m *Manager) CollectDisk() (protocol.MetricSample, error) {
	data, err := m.diskCollector.Collect()
	if err != nil {
		return protocol.MetricSample{}, err
	}
	if len(data) == 0 {
		return protocol.MetricSample{}, ErrNoData
	}
	return makeSample(protocol.MetricTypeDisk, data), nil
}

// CollectDiskIO 采集磁盘 IO 指标
func (m *Manager) CollectDiskIO() (protocol.MetricSample, error) {
	data, err := m.diskIOCollector.Collect()
	if err != nil {
		return protocol.MetricSample{}, err
	}
	return makeSample(protocol.MetricTypeDiskIO, data), nil
}

// CollectNetwork 采集网络指标。无有效数据时返回 ErrNoData。
func (m *Manager) CollectNetwork() (protocol.MetricSample, error) {
	data, err := m.networkCollector.Collect()
	if err != nil {
		return protocol.MetricSample{}, err
	}
	if len(data) == 0 {
		return protocol.MetricSample{}, ErrNoData
	}
	return makeSample(protocol.MetricTypeNetwork, data), nil
}

// CollectNetworkConnection 采集网络连接统计
func (m *Manager) CollectNetworkConnection() (protocol.MetricSample, error) {
	data, err := m.networkConnectionCollector.Collect()
	if err != nil {
		return protocol.MetricSample{}, err
	}
	return makeSample(protocol.MetricTypeNetworkConnection, data), nil
}

// CollectHost 采集主机信息（含 Load）
func (m *Manager) CollectHost() (protocol.MetricSample, error) {
	data, err := m.hostCollector.Collect()
	if err != nil {
		return protocol.MetricSample{}, err
	}
	return makeSample(protocol.MetricTypeHost, data), nil
}

// CollectGPU 采集 GPU 指标。无 GPU 时返回 ErrNoData（GPU 不是必备）
func (m *Manager) CollectGPU() (protocol.MetricSample, error) {
	data, err := m.gpuCollector.Collect()
	if err != nil {
		return protocol.MetricSample{}, err
	}
	if len(data) == 0 {
		return protocol.MetricSample{}, ErrNoData
	}
	return makeSample(protocol.MetricTypeGPU, data), nil
}

// CollectTemperature 采集温度信息。无传感器时返回 ErrNoData（温度不是必备）
func (m *Manager) CollectTemperature() (protocol.MetricSample, error) {
	data, err := m.temperatureCollector.Collect()
	if err != nil {
		return protocol.MetricSample{}, err
	}
	if len(data) == 0 {
		return protocol.MetricSample{}, ErrNoData
	}
	return makeSample(protocol.MetricTypeTemperature, data), nil
}

// CollectMonitor 采集服务监控数据（HTTP/TCP 探活）
func (m *Manager) CollectMonitor(items []protocol.MonitorItem) protocol.MetricSample {
	data := m.monitorCollector.Collect(items)
	return makeSample(protocol.MetricTypeMonitor, data)
}

// GetPublicIP 通过 API 获取公网 IP 地址
func (m *Manager) GetPublicIP(apiURL string, isIPv6 bool) (string, error) {
	collector := NewDDNSCollector(&protocol.DDNSConfigData{
		Enabled: true,
	})
	return collector.GetIPFromAPI(apiURL, isIPv6)
}

// GetInterfaceIP 从网络接口获取 IP 地址
func (m *Manager) GetInterfaceIP(interfaceName string, isIPv6 bool) (string, error) {
	collector := NewDDNSCollector(&protocol.DDNSConfigData{
		Enabled: true,
	})
	return collector.GetIPFromInterface(interfaceName, isIPv6)
}
