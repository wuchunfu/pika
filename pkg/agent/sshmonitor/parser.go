package sshmonitor

import (
	"bufio"
	"encoding/json"
	"io"
)

// SSHLoginEvent SSH登录事件（内部结构）
type SSHLoginEvent struct {
	Username  string `json:"username"`
	IP        string `json:"ip"`
	Port      string `json:"port,omitempty"`
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status"`
	Method    string `json:"method,omitempty"`
	TTY       string `json:"tty,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}

// Parser 日志解析器
type Parser struct{}

// NewParser 创建解析器
func NewParser() *Parser {
	return &Parser{}
}

// ParseFromReader 从 Reader 解析日志
// 返回：事件列表、新的文件偏移量、错误
func (p *Parser) ParseFromReader(r io.Reader, startOffset int64) ([]SSHLoginEvent, int64, error) {
	var events []SSHLoginEvent
	scanner := bufio.NewScanner(r)

	currentOffset := startOffset

	for scanner.Scan() {
		line := scanner.Text()
		lineSize := int64(len(line) + 1) // +1 for newline

		// 跳过空行
		if line == "" {
			currentOffset += lineSize
			continue
		}

		// 解析 JSON
		var event SSHLoginEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// 跳过格式错误的行
			currentOffset += lineSize
			continue
		}

		events = append(events, event)
		currentOffset += lineSize
	}

	if err := scanner.Err(); err != nil {
		return events, currentOffset, err
	}

	return events, currentOffset, nil
}
