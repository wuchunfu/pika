package config

// AppConfig 应用配置
type AppConfig struct {
	JWT             JWTConfig          `json:"JWT"`
	Users           map[string]string  `json:"Users"`           // 用户名 -> bcrypt加密的密码
	OIDC            *OIDCConfig        `json:"OIDC"`            // OIDC配置（可选）
	GitHub          *GitHubOAuthConfig `json:"GitHub"`          // GitHub OAuth配置（可选）
	GeoIP           *GeoIPConfig       `json:"GeoIP"`           // GeoIP配置（可选）
	VictoriaMetrics *VMConfig          `json:"VictoriaMetrics"` // VictoriaMetrics配置（可选）
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret       string `json:"Secret"`
	ExpiresHours int    `json:"ExpiresHours"`
}

// OIDCConfig OIDC认证配置
type OIDCConfig struct {
	Enabled      bool   `json:"Enabled"`      // 是否启用OIDC
	Issuer       string `json:"Issuer"`       // OIDC Provider的Issuer URL
	ClientID     string `json:"ClientID"`     // Client ID
	ClientSecret string `json:"ClientSecret"` // Client Secret
	RedirectURL  string `json:"RedirectURL"`  // 回调URL
}

// GitHubOAuthConfig GitHub OAuth认证配置
type GitHubOAuthConfig struct {
	Enabled      bool     `json:"Enabled"`      // 是否启用GitHub登录
	ClientID     string   `json:"ClientID"`     // GitHub OAuth App Client ID
	ClientSecret string   `json:"ClientSecret"` // GitHub OAuth App Client Secret
	RedirectURL  string   `json:"RedirectURL"`  // 回调URL
	AllowedUsers []string `json:"AllowedUsers"` // 允许登录的GitHub用户名白名单（为空则允许所有用户）
}

// GeoIPConfig GeoIP配置
type GeoIPConfig struct {
	Enabled    bool   `json:"Enabled"`    // 是否启用GeoIP查询
	DBPath     string `json:"DBPath"`     // GeoIP数据库文件路径（如：GeoLite2-City.mmdb）
	DBLanguage string `json:"DBLanguage"` // 数据库语言（如：zh-CN、en）
}

// VMConfig VictoriaMetrics配置
type VMConfig struct {
	Enabled       bool   `json:"Enabled"`       // 是否启用VictoriaMetrics
	URL           string `json:"URL"`           // VictoriaMetrics地址
	RetentionDays int    `json:"RetentionDays"` // 数据保留天数（用于文档说明）
	WriteTimeout  int    `json:"WriteTimeout"`  // 写入超时（秒）
	QueryTimeout  int    `json:"QueryTimeout"`  // 查询超时（秒）
}
