package protocol

// PublicIPConfigData 公网 IP 采集配置（服务端下发给客户端）
type PublicIPConfigData struct {
	Enabled         bool     `json:"enabled"`         // 是否启用采集
	IntervalSeconds int      `json:"intervalSeconds"` // 采集间隔（秒）
	IPv4Enabled     bool     `json:"ipv4Enabled"`     // 是否采集 IPv4
	IPv6Enabled     bool     `json:"ipv6Enabled"`     // 是否采集 IPv6
	IPv4APIs        []string `json:"ipv4Apis"`        // IPv4 API 列表
	IPv6APIs        []string `json:"ipv6Apis"`        // IPv6 API 列表
}

// PublicIPReportData 公网 IP 采集结果（客户端上报）
type PublicIPReportData struct {
	IPv4 string `json:"ipv4,omitempty"` // IPv4 地址
	IPv6 string `json:"ipv6,omitempty"` // IPv6 地址
}
