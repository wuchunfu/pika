package audit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// Logger 日志接口
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// defaultLogger 默认日志实现
type defaultLogger struct{}

func (l *defaultLogger) Debug(format string, args ...interface{}) {
	slog.Debug(fmt.Sprintf(format, args...))
}

func (l *defaultLogger) Info(format string, args ...interface{}) {
	slog.Info(fmt.Sprintf(format, args...))
}

func (l *defaultLogger) Warn(format string, args ...interface{}) {
	slog.Warn(fmt.Sprintf(format, args...))
}

func (l *defaultLogger) Error(format string, args ...interface{}) {
	slog.Error(fmt.Sprintf(format, args...))
}

var globalLogger Logger = &defaultLogger{}

// SetLogger 设置全局日志器
func SetLogger(logger Logger) {
	if logger != nil {
		globalLogger = logger
	}
}

// ProcessCache 进程缓存
type ProcessCache struct {
	processes []*process.Process
	timestamp time.Time
	mu        sync.RWMutex
	ttl       time.Duration
}

// NewProcessCache 创建进程缓存
func NewProcessCache(ttl time.Duration) *ProcessCache {
	return &ProcessCache{
		ttl: ttl,
	}
}

// Get 获取进程列表（带缓存）
func (pc *ProcessCache) Get() ([]*process.Process, error) {
	pc.mu.RLock()
	if time.Since(pc.timestamp) < pc.ttl && pc.processes != nil {
		processes := pc.processes
		pc.mu.RUnlock()
		return processes, nil
	}
	pc.mu.RUnlock()

	// 缓存过期，重新获取
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// 双重检查
	if time.Since(pc.timestamp) < pc.ttl && pc.processes != nil {
		return pc.processes, nil
	}

	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	pc.processes = processes
	pc.timestamp = time.Now()
	globalLogger.Debug("进程缓存已更新，共 %d 个进程", len(processes))

	return processes, nil
}

// Clear 清除缓存
func (pc *ProcessCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.processes = nil
	pc.timestamp = time.Time{}
}

// CommandExecutor 命令执行器
type CommandExecutor struct {
	timeout time.Duration
}

// NewCommandExecutor 创建命令执行器
func NewCommandExecutor(timeout time.Duration) *CommandExecutor {
	return &CommandExecutor{
		timeout: timeout,
	}
}

// Execute 执行命令
func (ce *CommandExecutor) Execute(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ce.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// 检查是否超时
		if ctx.Err() == context.DeadlineExceeded {
			globalLogger.Warn("命令执行超时(%v): %s %v", ce.timeout, name, args)
			return "", fmt.Errorf("命令执行超时(%v): %s", ce.timeout, name)
		}

		// 记录错误但返回输出
		if stderr.Len() > 0 {
			globalLogger.Debug("命令执行失败: %s %v, stderr: %s", name, args, stderr.String())
		}
		return stdout.String(), err
	}

	return stdout.String(), nil
}

// FileHashCache 文件哈希缓存
type FileHashCache struct {
	cache map[string]cachedHash
	mu    sync.RWMutex
}

type cachedHash struct {
	hash    string
	modTime time.Time
}

// NewFileHashCache 创建文件哈希缓存
func NewFileHashCache() *FileHashCache {
	return &FileHashCache{
		cache: make(map[string]cachedHash),
	}
}

// GetSHA256 获取文件 SHA256 哈希（带缓存）
func (fhc *FileHashCache) GetSHA256(filePath string) string {
	info, err := os.Stat(filePath)
	if err != nil {
		globalLogger.Debug("无法获取文件信息: %s, err: %v", filePath, err)
		return ""
	}

	modTime := info.ModTime()

	// 检查缓存
	fhc.mu.RLock()
	if cached, ok := fhc.cache[filePath]; ok {
		if cached.modTime.Equal(modTime) {
			fhc.mu.RUnlock()
			return cached.hash
		}
	}
	fhc.mu.RUnlock()

	// 计算哈希
	hash := calculateSHA256(filePath)
	if hash == "" {
		return ""
	}

	// 更新缓存
	fhc.mu.Lock()
	fhc.cache[filePath] = cachedHash{
		hash:    hash,
		modTime: modTime,
	}
	fhc.mu.Unlock()

	return hash
}

// calculateSHA256 计算文件 SHA256 哈希
func calculateSHA256(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// StringUtils 字符串工具
type StringUtils struct{}

// TruncateString 截断字符串
func (su *StringUtils) Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// ContainsAny 检查字符串是否包含任意关键词
func (su *StringUtils) ContainsAny(s string, keywords []string) bool {
	lower := strings.ToLower(s)
	for _, keyword := range keywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// IsLocalIP 判断是否是本地 IP
func IsLocalIP(ip string) bool {
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return true
	}

	// 简化版本，实际应该使用 net.ParseIP
	privateRanges := []string{
		"10.", "192.168.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.",
		"172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31.",
	}

	for _, prefix := range privateRanges {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}

	return false
}

// BatchProcessor 批处理器
type BatchProcessor struct {
	batchSize int
}

// NewBatchProcessor 创建批处理器
func NewBatchProcessor(batchSize int) *BatchProcessor {
	return &BatchProcessor{
		batchSize: batchSize,
	}
}

// Process 批量处理
func (bp *BatchProcessor) Process(items []string, handler func(batch []string) error) error {
	for i := 0; i < len(items); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		if err := handler(batch); err != nil {
			return fmt.Errorf("批处理失败 (批次 %d-%d): %w", i, end, err)
		}
	}
	return nil
}

// RetryExecutor 重试执行器
type RetryExecutor struct {
	maxRetries int
	delay      time.Duration
}

// NewRetryExecutor 创建重试执行器
func NewRetryExecutor(maxRetries int, delay time.Duration) *RetryExecutor {
	return &RetryExecutor{
		maxRetries: maxRetries,
		delay:      delay,
	}
}

// Execute 执行函数（带重试）
func (re *RetryExecutor) Execute(fn func() error) error {
	var lastErr error
	for i := 0; i <= re.maxRetries; i++ {
		if i > 0 {
			globalLogger.Debug("重试第 %d 次...", i)
			time.Sleep(re.delay)
		}

		if err := fn(); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("重试 %d 次后仍失败: %w", re.maxRetries, lastErr)
}
