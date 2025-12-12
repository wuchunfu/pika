package models

// HostMetric 主机信息指标（静态信息，保存在 PostgreSQL）
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

// 注意：所有其他指标数据已迁移到 VictoriaMetrics
// 包括：CPU, Memory, Disk, Network, NetworkConnection, DiskIO, GPU, Temperature, Monitor
