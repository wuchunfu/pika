package protocol

// MonitorConfigPayload 监控配置 payload
type MonitorConfigPayload struct {
	Interval int           `json:"interval"`
	Items    []MonitorItem `json:"items"`
}

// MonitorItem 监控项配置
type MonitorItem struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Target     string             `json:"target"`
	HTTPConfig *HTTPMonitorConfig `json:"httpConfig,omitempty"`
	TCPConfig  *TCPMonitorConfig  `json:"tcpConfig,omitempty"`
	ICMPConfig *ICMPMonitorConfig `json:"icmpConfig,omitempty"`
}

// HTTPMonitorConfig HTTP 监控配置
type HTTPMonitorConfig struct {
	Method             string            `json:"method"`
	ExpectedStatusCode int               `json:"expectedStatusCode"`
	ExpectedContent    string            `json:"expectedContent,omitempty"`
	Timeout            int               `json:"timeout"`
	Headers            map[string]string `json:"headers,omitempty"`
	Body               string            `json:"body,omitempty"`
}

// TCPMonitorConfig TCP 监控配置
type TCPMonitorConfig struct {
	Timeout int `json:"timeout"`
}

// ICMPMonitorConfig ICMP 监控配置
type ICMPMonitorConfig struct {
	Timeout int `json:"timeout"` // 超时时间（秒）
	Count   int `json:"count"`   // Ping 次数
}
