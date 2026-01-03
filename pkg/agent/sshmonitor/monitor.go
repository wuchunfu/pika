package sshmonitor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/dushixiang/pika/internal/protocol"
	"github.com/fsnotify/fsnotify"
)

const (
	DefaultLogFile = "/var/log/pika/ssh_login.log"
	DefaultLogDir  = "/var/log/pika"
)

// Monitor SSH登录监控器
type Monitor struct {
	mu          sync.RWMutex
	enabled     bool
	logFile     string
	watcher     *fsnotify.Watcher
	ctx         context.Context
	cancel      context.CancelFunc
	eventCh     chan SSHLoginEvent
	parser      *Parser
	hookManager *HookManager
	watcherOnce sync.Once
	lastOffset  int64 // 记录上次读取的位置，避免重复读取
}

// NewMonitor 创建监控器
func NewMonitor() *Monitor {
	return &Monitor{
		enabled:     false,
		logFile:     DefaultLogFile,
		eventCh:     make(chan SSHLoginEvent, 100),
		parser:      NewParser(),
		hookManager: NewHookManager(),
	}
}

// Start 启动监控（根据配置启用/禁用）
func (m *Monitor) Start(ctx context.Context, config protocol.SSHLoginConfig) error {
	// 检查操作系统
	if runtime.GOOS != "linux" {
		return fmt.Errorf("SSH登录监控仅支持 Linux 系统")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已启动，先停止
	if m.enabled {
		m.stopInternal()
	}

	if !config.Enabled {
		slog.Info("SSH登录监控已禁用")
		return nil
	}

	// 确保日志目录存在
	if err := os.MkdirAll(DefaultLogDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 确保日志文件存在
	if _, err := os.Stat(m.logFile); os.IsNotExist(err) {
		f, err := os.Create(m.logFile)
		if err != nil {
			return fmt.Errorf("创建日志文件失败: %w", err)
		}
		f.Close()
		os.Chmod(m.logFile, 0644)
	}

	// 安装 PAM Hook
	if err := m.hookManager.Install(); err != nil {
		if os.IsPermission(err) {
			slog.Warn("权限不足，无法自动安装 PAM Hook", "error", err)
			slog.Info("如需启用 SSH 登录监控，请手动执行安装")
			// 不返回错误，继续监控（假设已手动安装）
		} else {
			slog.Warn("安装 PAM Hook 失败", "error", err)
			return err
		}
	}

	// 初始化 watcher
	if err := m.initWatcher(ctx); err != nil {
		return err
	}

	// 读取现有日志（避免丢失 Agent 重启期间的登录）
	if err := m.readExistingLogs(); err != nil {
		slog.Warn("读取现有日志失败", "error", err)
	}

	m.enabled = true
	slog.Info("SSH登录监控已启动", "logFile", m.logFile)
	return nil
}

// Stop 停止监控
func (m *Monitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopInternal()
}

// stopInternal 内部停止方法（不加锁）
func (m *Monitor) stopInternal() error {
	if !m.enabled {
		return nil
	}

	// 取消 context
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	// 关闭 watcher
	if m.watcher != nil {
		m.watcher.Close()
		m.watcher = nil
		m.watcherOnce = sync.Once{}
	}

	// 卸载 PAM Hook
	if err := m.hookManager.Uninstall(); err != nil {
		slog.Warn("卸载 PAM Hook 失败", "error", err)
	}

	m.enabled = false
	slog.Info("SSH登录监控已停止")
	return nil
}

// GetEvents 获取事件通道
func (m *Monitor) GetEvents() <-chan SSHLoginEvent {
	return m.eventCh
}

// initWatcher 初始化文件监控器
func (m *Monitor) initWatcher(ctx context.Context) error {
	var err error
	m.watcherOnce.Do(func() {
		m.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			err = fmt.Errorf("创建文件监控器失败: %w", err)
			return
		}

		// 添加日志文件到监控
		if err = m.watcher.Add(m.logFile); err != nil {
			err = fmt.Errorf("添加文件监控失败: %w", err)
			return
		}

		// 创建 context
		m.ctx, m.cancel = context.WithCancel(ctx)

		// 启动监控循环
		go m.watchLoop()

		slog.Info("文件监控器已启动", "file", m.logFile)
	})
	return err
}

// watchLoop 监控循环
func (m *Monitor) watchLoop() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			// 检测文件被删除或重命名（logrotate）
			if event.Op&fsnotify.Remove == fsnotify.Remove ||
				event.Op&fsnotify.Rename == fsnotify.Rename {
				slog.Warn("日志文件被删除或轮转，尝试重新监控")

				// 等待新文件创建
				time.Sleep(1 * time.Second)

				// 重新添加监控
				if err := m.watcher.Add(m.logFile); err != nil {
					slog.Error("重新添加文件监控失败", "error", err)
				} else {
					m.lastOffset = 0 // 重置偏移量
				}
			}

			// 只处理写入事件
			if event.Op&fsnotify.Write == fsnotify.Write {
				m.handleFileWrite()
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("文件监控错误", "error", err)
		}
	}
}

// handleFileWrite 处理文件写入事件
func (m *Monitor) handleFileWrite() {
	// 打开文件，从上次读取位置继续读
	f, err := os.Open(m.logFile)
	if err != nil {
		slog.Warn("打开日志文件失败", "error", err)
		return
	}
	defer f.Close()

	// 移动到上次读取位置
	if _, err := f.Seek(m.lastOffset, 0); err != nil {
		slog.Warn("移动文件位置失败", "error", err)
		return
	}

	// 解析新增的日志行
	events, newOffset, err := m.parser.ParseFromReader(f, m.lastOffset)
	if err != nil {
		slog.Warn("解析日志失败", "error", err)
		return
	}

	// 更新偏移量
	m.lastOffset = newOffset

	// 发送事件
	for _, event := range events {
		select {
		case m.eventCh <- event:
			slog.Info("检测到SSH登录", "user", event.Username, "ip", event.IP, "status", event.Status)
		default:
			slog.Warn("事件队列已满，丢弃事件")
		}
	}
}

// readExistingLogs 读取现有日志（Agent重启时）
func (m *Monitor) readExistingLogs() error {
	f, err := os.Open(m.logFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// 从文件开头读取所有日志
	events, newOffset, err := m.parser.ParseFromReader(f, 0)
	if err != nil {
		return err
	}

	m.lastOffset = newOffset

	// 只上报最近的登录（例如最近100条），避免重复上报历史数据
	startIdx := len(events) - 100
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(events); i++ {
		select {
		case m.eventCh <- events[i]:
		default:
			break
		}
	}

	return nil
}
