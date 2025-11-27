package models

// CPUMetric CPU指标
type CPUMetric struct {
	ID            uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID       string  `gorm:"index:idx_cpu_agent_ts,priority:1" json:"agentId"`                    // 探针ID
	UsagePercent  float64 `json:"usagePercent"`                                                        // CPU使用率
	LogicalCores  int     `json:"logicalCores"`                                                        // 逻辑核心数
	PhysicalCores int     `json:"physicalCores"`                                                       // 物理核心数
	ModelName     string  `json:"modelName"`                                                           // CPU型号
	Timestamp     int64   `gorm:"index:idx_cpu_agent_ts,priority:2;index:idx_cpu_ts" json:"timestamp"` // 时间戳（毫秒）
}

func (CPUMetric) TableName() string {
	return "cpu_metrics"
}

// MemoryMetric 内存指标
type MemoryMetric struct {
	ID           uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID      string  `gorm:"index:idx_mem_agent_ts,priority:1" json:"agentId"`                    // 探针ID
	Total        uint64  `json:"total"`                                                               // 总内存(字节)
	Used         uint64  `json:"used"`                                                                // 已使用(字节)
	Free         uint64  `json:"free"`                                                                // 空闲(字节)
	UsagePercent float64 `json:"usagePercent"`                                                        // 使用率
	SwapTotal    uint64  `json:"swapTotal"`                                                           // 总交换空间(字节)
	SwapUsed     uint64  `json:"swapUsed"`                                                            // 已使用交换空间(字节)
	SwapFree     uint64  `json:"swapFree"`                                                            // 空闲交换空间(字节)
	Timestamp    int64   `gorm:"index:idx_mem_agent_ts,priority:2;index:idx_mem_ts" json:"timestamp"` // 时间戳（毫秒）
}

func (MemoryMetric) TableName() string {
	return "memory_metrics"
}

// DiskMetric 磁盘指标
type DiskMetric struct {
	ID           uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID      string  `gorm:"index:idx_disk_agent_ts,priority:1" json:"agentId"`                     // 探针ID
	MountPoint   string  `json:"mountPoint"`                                                            // 挂载点
	Total        uint64  `json:"total"`                                                                 // 总容量(字节)
	Used         uint64  `json:"used"`                                                                  // 已使用(字节)
	Free         uint64  `json:"free"`                                                                  // 空闲(字节)
	UsagePercent float64 `json:"usagePercent"`                                                          // 使用率
	Timestamp    int64   `gorm:"index:idx_disk_agent_ts,priority:2;index:idx_disk_ts" json:"timestamp"` // 时间戳（毫秒）
}

func (DiskMetric) TableName() string {
	return "disk_metrics"
}

// NetworkMetric 网络指标
type NetworkMetric struct {
	ID             uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID        string `gorm:"index:idx_net_agent_ts,priority:1" json:"agentId"`                    // 探针ID
	Interface      string `json:"interface"`                                                           // 网卡名称
	BytesSentRate  uint64 `json:"bytesSentRate"`                                                       // 发送速率(字节/秒)
	BytesRecvRate  uint64 `json:"bytesRecvRate"`                                                       // 接收速率(字节/秒)
	BytesSentTotal uint64 `json:"bytesSentTotal"`                                                      // 累计发送字节数
	BytesRecvTotal uint64 `json:"bytesRecvTotal"`                                                      // 累计接收字节数
	Timestamp      int64  `gorm:"index:idx_net_agent_ts,priority:2;index:idx_net_ts" json:"timestamp"` // 时间戳（毫秒）
}

func (NetworkMetric) TableName() string {
	return "network_metrics"
}

// LoadMetric 系统负载指标
type LoadMetric struct {
	ID        uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID   string  `gorm:"index:idx_load_agent_ts,priority:1" json:"agentId"`                     // 探针ID
	Load1     float64 `json:"load1"`                                                                 // 1分钟负载
	Load5     float64 `json:"load5"`                                                                 // 5分钟负载
	Load15    float64 `json:"load15"`                                                                // 15分钟负载
	Timestamp int64   `gorm:"index:idx_load_agent_ts,priority:2;index:idx_load_ts" json:"timestamp"` // 时间戳（毫秒）
}

func (LoadMetric) TableName() string {
	return "load_metrics"
}

// DiskIOMetric 磁盘IO指标
type DiskIOMetric struct {
	ID             uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID        string `gorm:"index:idx_diskio_agent_ts,priority:1" json:"agentId"`                       // 探针ID
	Device         string `json:"device"`                                                                    // 设备名称
	ReadCount      uint64 `json:"readCount"`                                                                 // 读取次数
	WriteCount     uint64 `json:"writeCount"`                                                                // 写入次数
	ReadBytes      uint64 `json:"readBytes"`                                                                 // 读取字节数
	WriteBytes     uint64 `json:"writeBytes"`                                                                // 写入字节数
	ReadBytesRate  uint64 `json:"readBytesRate"`                                                             // 读取速率(字节/秒)
	WriteBytesRate uint64 `json:"writeBytesRate"`                                                            // 写入速率(字节/秒)
	ReadTime       uint64 `json:"readTime"`                                                                  // 读取时间(毫秒)
	WriteTime      uint64 `json:"writeTime"`                                                                 // 写入时间(毫秒)
	IoTime         uint64 `json:"ioTime"`                                                                    // IO时间(毫秒)
	IopsInProgress uint64 `json:"iopsInProgress"`                                                            // 正在进行的IO操作数
	Timestamp      int64  `gorm:"index:idx_diskio_agent_ts,priority:2;index:idx_diskio_ts" json:"timestamp"` // 时间戳（毫秒）
}

func (DiskIOMetric) TableName() string {
	return "disk_io_metrics"
}

// GPUMetric GPU指标
type GPUMetric struct {
	ID               uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID          string  `gorm:"index:idx_gpu_agent_idx_ts,priority:1" json:"agentId"`                    // 探针ID
	Index            int     `gorm:"index:idx_gpu_agent_idx_ts,priority:2" json:"index"`                      // GPU索引
	Name             string  `json:"name"`                                                                    // GPU名称
	Utilization      float64 `json:"utilization"`                                                             // GPU使用率(%)
	MemoryTotal      uint64  `json:"memoryTotal"`                                                             // 总显存(字节)
	MemoryUsed       uint64  `json:"memoryUsed"`                                                              // 已使用显存(字节)
	MemoryFree       uint64  `json:"memoryFree"`                                                              // 空闲显存(字节)
	Temperature      float64 `json:"temperature"`                                                             // 温度(℃)
	PowerDraw        float64 `json:"powerDraw"`                                                               // 功耗(瓦)
	FanSpeed         float64 `json:"fanSpeed"`                                                                // 风扇转速(%)
	PerformanceState string  `json:"performanceState"`                                                        // 性能状态
	Timestamp        int64   `gorm:"index:idx_gpu_agent_idx_ts,priority:3;index:idx_gpu_ts" json:"timestamp"` // 时间戳（毫秒）
}

func (GPUMetric) TableName() string {
	return "gpu_metrics"
}

// TemperatureMetric 温度指标
type TemperatureMetric struct {
	ID          uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID     string  `gorm:"index:idx_temp_agent_sensor_ts,priority:1" json:"agentId"`                     // 探针ID
	SensorKey   string  `gorm:"index:idx_temp_agent_sensor_ts,priority:2" json:"sensorKey"`                   // 传感器标识
	SensorLabel string  `json:"sensorLabel"`                                                                  // 传感器标签
	Temperature float64 `json:"temperature"`                                                                  // 温度(℃)
	Timestamp   int64   `gorm:"index:idx_temp_agent_sensor_ts,priority:3;index:idx_temp_ts" json:"timestamp"` // 时间戳（毫秒）
}

func (TemperatureMetric) TableName() string {
	return "temperature_metrics"
}

// HostMetric 主机信息指标
type HostMetric struct {
	ID              uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID         string `gorm:"uniqueIndex:ux_host_agent" json:"agentId"` // 探针ID（唯一约束用于 upsert）
	OS              string `json:"os"`                                       // 操作系统
	Platform        string `json:"platform"`                                 // 平台
	PlatformVersion string `json:"platformVersion"`                          // 平台版本
	KernelVersion   string `json:"kernelVersion"`                            // 内核版本
	KernelArch      string `json:"kernelArch"`                               // 内核架构
	Uptime          uint64 `json:"uptime"`                                   // 运行时间(秒)
	BootTime        uint64 `json:"bootTime"`                                 // 启动时间(Unix时间戳-秒)
	Procs           uint64 `json:"procs"`                                    // 进程数
	Timestamp       int64  `gorm:"index:idx_host_ts" json:"timestamp"`       // 时间戳（毫秒）
}

func (HostMetric) TableName() string {
	return "host_metrics"
}

// MonitorMetric 监控指标
type MonitorMetric struct {
	ID             uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentId        string `gorm:"index:idx_mon_agent_monitor_ts,priority:1" json:"agentId"`                                                             // 探针ID
	MonitorId      string `gorm:"index:idx_mon_agent_monitor_ts,priority:2;index:idx_mon_type_monitor_ts,priority:2" json:"monitorId"`                  // 监控项ID
	Type           string `gorm:"index:idx_mon_type_monitor_ts,priority:1" json:"type"`                                                                 // 监控类型: http, tcp
	Target         string `json:"target"`                                                                                                               // 监控目标
	Status         string `json:"status"`                                                                                                               // 状态: up, down
	StatusCode     int    `json:"statusCode"`                                                                                                           // HTTP状态码
	ResponseTime   int64  `json:"responseTime"`                                                                                                         // 响应时间(毫秒)
	Error          string `json:"error"`                                                                                                                // 错误信息
	Message        string `json:"message"`                                                                                                              // 附加信息
	ContentMatch   bool   `json:"contentMatch"`                                                                                                         // 内容匹配结果
	CertExpiryTime int64  `json:"certExpiryTime"`                                                                                                       // 证书过期时间(毫秒时间戳), 0表示无证书
	CertDaysLeft   int    `json:"certDaysLeft"`                                                                                                         // 证书剩余天数
	Timestamp      int64  `gorm:"index:idx_mon_agent_monitor_ts,priority:3;index:idx_mon_type_monitor_ts,priority:3;index:idx_mon_ts" json:"timestamp"` // 时间戳（毫秒）
}

func (MonitorMetric) TableName() string {
	return "monitor_metrics"
}
