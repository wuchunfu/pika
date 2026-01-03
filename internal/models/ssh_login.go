package models

import (
	"time"

	"gorm.io/gorm"
)

// SSHLoginConfig SSH登录监控配置
type SSHLoginConfig struct {
	ID            string `gorm:"primaryKey" json:"id"`                  // 配置ID (UUID)
	AgentID       string `gorm:"uniqueIndex;not null" json:"agentId"`   // 探针ID（唯一）
	Enabled       bool   `json:"enabled"`                               // 是否启用
	RecordFailed  bool   `json:"recordFailed"`                          // 是否记录失败登录
	ApplyStatus   string `json:"applyStatus,omitempty"`                 // 配置应用状态: success/failed/pending
	ApplyMessage  string `json:"applyMessage,omitempty"`                // 应用结果消息
	ApplyError    string `json:"applyError,omitempty"`                  // 应用错误信息
	LastAppliedAt int64  `json:"lastAppliedAt,omitempty"`               // 最后应用时间（毫秒）
	CreatedAt     int64  `json:"createdAt"`                             // 创建时间（毫秒）
	UpdatedAt     int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"` // 更新时间（毫秒）
}

func (SSHLoginConfig) TableName() string {
	return "ssh_login_configs"
}

// BeforeCreate GORM钩子：设置创建时间
func (s *SSHLoginConfig) BeforeCreate(tx *gorm.DB) error {
	if s.CreatedAt == 0 {
		s.CreatedAt = time.Now().UnixMilli()
	}
	return nil
}

// SSHLoginEvent SSH登录事件
type SSHLoginEvent struct {
	ID        string `gorm:"primaryKey" json:"id"`          // 事件ID (UUID)
	AgentID   string `gorm:"index;not null" json:"agentId"` // 探针ID
	Username  string `gorm:"index" json:"username"`         // 用户名
	IP        string `gorm:"index" json:"ip"`               // 来源IP
	Port      string `json:"port,omitempty"`                // 来源端口
	Status    string `gorm:"index" json:"status"`           // 状态: success/failed
	Method    string `json:"method,omitempty"`              // 认证方式: password/publickey
	TTY       string `json:"tty,omitempty"`                 // 终端
	SessionID string `json:"sessionId,omitempty"`           // 会话ID
	Timestamp int64  `gorm:"index" json:"timestamp"`        // 登录时间（毫秒时间戳）
	CreatedAt int64  `json:"createdAt"`                     // 记录创建时间（毫秒）
}

func (SSHLoginEvent) TableName() string {
	return "ssh_login_events"
}

// BeforeCreate GORM钩子：设置创建时间
func (s *SSHLoginEvent) BeforeCreate(tx *gorm.DB) error {
	if s.CreatedAt == 0 {
		s.CreatedAt = time.Now().UnixMilli()
	}
	return nil
}
