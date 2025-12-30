package tamper

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TamperEvent 防篡改事件
type TamperEvent struct {
	Path      string    `json:"path"`      // 被修改的路径
	Operation string    `json:"operation"` // 操作类型: write, remove, rename, chmod
	Timestamp time.Time `json:"timestamp"` // 事件时间
	Details   string    `json:"details"`   // 详细信息
}

// AttributeTamperAlert 属性篡改告警
type AttributeTamperAlert struct {
	Path      string    `json:"path"`      // 被篡改的路径
	Timestamp time.Time `json:"timestamp"` // 检测时间
	Details   string    `json:"details"`   // 详细信息(如: "不可变属性被移除")
	Restored  bool      `json:"restored"`  // 是否已自动恢复
}

// UpdateResult 更新结果
type UpdateResult struct {
	Added   []string // 新增保护的目录
	Removed []string // 移除保护的目录
	Current []string // 当前所有保护的目录
}

// Protector 防篡改保护器
type Protector struct {
	mu          sync.RWMutex
	paths       map[string]bool // 当前保护的目录集合(使用 map 便于查找)
	watcher     *fsnotify.Watcher
	ctx         context.Context
	cancel      context.CancelFunc
	eventCh     chan TamperEvent
	alertCh     chan AttributeTamperAlert // 属性篡改告警通道
	watcherOnce sync.Once                 // 确保 watcher 只创建一次
	checkTicker *time.Ticker              // 属性检查定时器
}

// NewProtector 创建防篡改保护器
func NewProtector() *Protector {
	return &Protector{
		paths:   make(map[string]bool),
		eventCh: make(chan TamperEvent, 100),
		alertCh: make(chan AttributeTamperAlert, 50),
	}
}

// ApplyIncrementalUpdate 应用增量更新（服务端已计算好新增和移除）
// 参数 toAdd: 需要新增保护的目录列表
// 参数 toRemove: 需要移除保护的目录列表
// 返回: 更新结果和错误
func (p *Protector) ApplyIncrementalUpdate(ctx context.Context, toAdd, toRemove []string) (*UpdateResult, error) {
	// 检查操作系统
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("防篡改功能仅支持 Linux 系统")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 如果没有变化,直接返回
	if len(toAdd) == 0 && len(toRemove) == 0 {
		slog.Info("防篡改保护目录列表无变化")
		return &UpdateResult{
			Added:   []string{},
			Removed: []string{},
			Current: p.getCurrentPaths(),
		}, nil
	}

	// 初始化 watcher(如果还没创建且有新增目录)
	if len(toAdd) > 0 {
		if err := p.initWatcher(ctx); err != nil {
			return nil, err
		}
	}

	// 处理需要移除的目录
	var removeFailed []string
	for _, path := range toRemove {
		if !p.paths[path] {
			slog.Info("目录未被保护，跳过移除", "path", path)
			continue
		}
		if err := p.removePath(path); err != nil {
			slog.Warn("移除目录保护失败", "path", path, "error", err)
			removeFailed = append(removeFailed, path)
		} else {
			delete(p.paths, path)
			slog.Info("已取消保护目录", "path", path)
		}
	}

	// 处理需要新增的目录
	var addFailed []string
	for _, path := range toAdd {
		if p.paths[path] {
			slog.Info("目录已被保护，跳过新增", "path", path)
			continue
		}
		if err := p.addPath(path); err != nil {
			slog.Warn("添加目录保护失败", "path", path, "error", err)
			addFailed = append(addFailed, path)
		} else {
			p.paths[path] = true
			slog.Info("已保护目录", "path", path)
		}
	}

	// 构建结果
	result := &UpdateResult{
		Added:   filterFailed(toAdd, addFailed),
		Removed: filterFailed(toRemove, removeFailed),
		Current: p.getCurrentPaths(),
	}

	// 如果有失败的操作,返回错误
	if len(addFailed) > 0 || len(removeFailed) > 0 {
		return result, fmt.Errorf("部分操作失败: 添加失败 %d 个, 移除失败 %d 个", len(addFailed), len(removeFailed))
	}

	slog.Info("防篡改保护已更新",
		"新增", len(result.Added),
		"移除", len(result.Removed),
		"当前保护", len(result.Current))

	return result, nil
}

// UpdatePaths 更新保护的目录列表（保留此方法用于完整配置更新）
// 参数 newPaths: 新的完整目录列表
// 返回: 更新结果(新增/移除的目录)和错误
func (p *Protector) UpdatePaths(ctx context.Context, newPaths []string) (*UpdateResult, error) {
	// 检查操作系统
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("防篡改功能仅支持 Linux 系统")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 转换为 map 便于比较
	newPathsMap := make(map[string]bool)
	for _, path := range newPaths {
		newPathsMap[path] = true
	}

	// 计算需要新增的目录
	var toAdd []string
	for path := range newPathsMap {
		if !p.paths[path] {
			toAdd = append(toAdd, path)
		}
	}

	// 计算需要移除的目录
	var toRemove []string
	for path := range p.paths {
		if !newPathsMap[path] {
			toRemove = append(toRemove, path)
		}
	}

	// 如果没有变化,直接返回
	if len(toAdd) == 0 && len(toRemove) == 0 {
		slog.Info("防篡改保护目录列表无变化")
		return &UpdateResult{
			Added:   []string{},
			Removed: []string{},
			Current: p.getCurrentPaths(),
		}, nil
	}

	// 初始化 watcher(如果还没创建)
	if err := p.initWatcher(ctx); err != nil {
		return nil, err
	}

	// 处理需要移除的目录
	var removeFailed []string
	for _, path := range toRemove {
		if err := p.removePath(path); err != nil {
			slog.Warn("移除目录保护失败", "path", path, "error", err)
			removeFailed = append(removeFailed, path)
		} else {
			delete(p.paths, path)
			slog.Info("已取消保护目录", "path", path)
		}
	}

	// 处理需要新增的目录
	var addFailed []string
	for _, path := range toAdd {
		if err := p.addPath(path); err != nil {
			slog.Warn("添加目录保护失败", "path", path, "error", err)
			addFailed = append(addFailed, path)
		} else {
			p.paths[path] = true
			slog.Info("已保护目录", "path", path)
		}
	}

	// 构建结果
	result := &UpdateResult{
		Added:   filterFailed(toAdd, addFailed),
		Removed: filterFailed(toRemove, removeFailed),
		Current: p.getCurrentPaths(),
	}

	// 如果有失败的操作,返回错误
	if len(addFailed) > 0 || len(removeFailed) > 0 {
		return result, fmt.Errorf("部分操作失败: 添加失败 %d 个, 移除失败 %d 个", len(addFailed), len(removeFailed))
	}

	slog.Info("防篡改保护已更新",
		"新增", len(result.Added),
		"移除", len(result.Removed),
		"当前保护", len(result.Current))

	return result, nil
}

// StopAll 停止所有防篡改保护
func (p *Protector) StopAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.paths) == 0 {
		slog.Info("没有正在保护的目录")
		return nil
	}

	var lastErr error

	// 取消 context
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}

	// 停止定时器
	if p.checkTicker != nil {
		p.checkTicker.Stop()
		p.checkTicker = nil
	}

	// 关闭监控器
	if p.watcher != nil {
		if err := p.watcher.Close(); err != nil {
			slog.Warn("关闭文件监控器失败", "error", err)
			lastErr = err
		}
		p.watcher = nil
		p.watcherOnce = sync.Once{} // 重置,允许下次重新创建
	}

	// 移除所有目录的不可变属性
	for path := range p.paths {
		if err := p.setImmutable(path, false); err != nil {
			slog.Warn("移除目录不可变属性失败", "path", path, "error", err)
			lastErr = err
		} else {
			slog.Info("已取消保护目录", "path", path)
		}
	}

	// 清空路径列表
	p.paths = make(map[string]bool)

	slog.Info("已停止所有防篡改保护")
	return lastErr
}

// GetEvents 获取事件通道
func (p *Protector) GetEvents() <-chan TamperEvent {
	return p.eventCh
}

// GetAlerts 获取属性篡改告警通道
func (p *Protector) GetAlerts() <-chan AttributeTamperAlert {
	return p.alertCh
}

// GetProtectedPaths 获取受保护的路径列表
func (p *Protector) GetProtectedPaths() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.getCurrentPaths()
}

// IsProtected 检查路径是否受保护
func (p *Protector) IsProtected(path string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.paths[path]
}

// getCurrentPaths 获取当前路径列表(内部方法,不加锁)
func (p *Protector) getCurrentPaths() []string {
	paths := make([]string, 0, len(p.paths))
	for path := range p.paths {
		paths = append(paths, path)
	}
	return paths
}

// initWatcher 初始化文件监控器(只会创建一次)
func (p *Protector) initWatcher(ctx context.Context) error {
	var err error
	p.watcherOnce.Do(func() {
		// 创建文件监控器
		p.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			err = fmt.Errorf("创建文件监控器失败: %w", err)
			return
		}

		// 创建 context
		p.ctx, p.cancel = context.WithCancel(ctx)

		// 启动监控循环
		go p.watchLoop()

		// 启动定期属性检查
		p.checkTicker = time.NewTicker(5 * time.Second)
		go p.periodicAttributeCheck()

		slog.Info("文件监控器已启动")
	})
	return err
}

// addPath 添加目录保护(内部方法,不加锁)
func (p *Protector) addPath(path string) error {
	// 设置不可变属性
	if err := p.setImmutable(path, true); err != nil {
		return fmt.Errorf("设置目录不可变属性失败: %w", err)
	}

	// 添加到监控
	if p.watcher != nil {
		if err := p.watcher.Add(path); err != nil {
			// 如果添加监控失败,尝试回滚不可变属性
			_ = p.setImmutable(path, false)
			return fmt.Errorf("添加目录到监控失败: %w", err)
		}
	}

	return nil
}

// removePath 移除目录保护(内部方法,不加锁)
func (p *Protector) removePath(path string) error {
	// 从监控中移除
	if p.watcher != nil {
		if err := p.watcher.Remove(path); err != nil {
			slog.Warn("从监控中移除目录失败", "error", err)
			// 继续执行,不返回错误
		}
	}

	// 移除不可变属性
	if err := p.setImmutable(path, false); err != nil {
		return fmt.Errorf("移除目录不可变属性失败: %w", err)
	}

	return nil
}

// watchLoop 监控循环
func (p *Protector) watchLoop() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case event, ok := <-p.watcher.Events:
			if !ok {
				return
			}
			p.handleEvent(event)
		case err, ok := <-p.watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("文件监控错误", "error", err)
		}
	}
}

// handleEvent 处理文件系统事件
func (p *Protector) handleEvent(event fsnotify.Event) {
	var operation string
	var details string

	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		operation = "write"
		details = "文件被写入"
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		operation = "remove"
		details = "文件被删除"
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		operation = "rename"
		details = "文件被重命名"
	case event.Op&fsnotify.Create == fsnotify.Create:
		operation = "create"
		details = "文件被创建"
	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		operation = "chmod"
		details = "文件权限被修改"
	default:
		operation = "unknown"
		details = fmt.Sprintf("未知操作: %v", event.Op)
	}

	tamperEvent := TamperEvent{
		Path:      event.Name,
		Operation: operation,
		Timestamp: time.Now(),
		Details:   details,
	}

	// 发送事件(非阻塞)
	select {
	case p.eventCh <- tamperEvent:
		slog.Warn("检测到文件变动", "path", event.Name, "operation", operation, "details", details)
	default:
		slog.Warn("事件队列已满,丢弃事件", "path", event.Name)
	}
}

// periodicAttributeCheck 定期检查所有受保护目录的不可变属性
func (p *Protector) periodicAttributeCheck() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.checkTicker.C:
			p.checkAllAttributes()
		}
	}
}

// checkAllAttributes 检查所有受保护目录的属性
func (p *Protector) checkAllAttributes() {
	p.mu.RLock()
	paths := make([]string, 0, len(p.paths))
	for path := range p.paths {
		paths = append(paths, path)
	}
	p.mu.RUnlock()

	for _, path := range paths {
		p.checkAndRestoreImmutable(path)
	}
}

// checkAndRestoreImmutable 检查并恢复目录的不可变属性
func (p *Protector) checkAndRestoreImmutable(path string) {
	// 检查不可变属性
	hasImmutable, err := p.checkImmutable(path)
	if err != nil {
		slog.Warn("检查目录属性失败", "path", path, "error", err)
		return
	}

	// 如果不可变属性被移除
	if !hasImmutable {
		slog.Warn("检测到属性篡改", "path", path, "details", "不可变属性被移除")

		// 尝试恢复属性
		restored := false
		if err := p.setImmutable(path, true); err != nil {
			slog.Error("恢复目录不可变属性失败", "path", path, "error", err)
		} else {
			slog.Info("已自动恢复目录的不可变属性", "path", path)
			restored = true
		}

		// 发送告警
		alert := AttributeTamperAlert{
			Path:      path,
			Timestamp: time.Now(),
			Details:   "不可变属性被移除",
			Restored:  restored,
		}

		select {
		case p.alertCh <- alert:
			slog.Info("已发送属性篡改告警", "path", path)
		default:
			slog.Warn("告警队列已满,丢弃告警", "path", path)
		}
	}
}

// checkImmutable 使用 ioctl 检查目录是否具有不可变属性
func (p *Protector) checkImmutable(path string) (bool, error) {
	// 打开文件/目录
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	// 使用 chattr.go 中的 IsAttr 函数检查不可变属性
	return IsAttr(f, FS_IMMUTABLE_FL)
}

// setImmutable 设置或移除文件/目录的不可变属性
func (p *Protector) setImmutable(path string, immutable bool) error {
	// 打开文件/目录
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	// 使用 chattr.go 中的 SetAttr/UnsetAttr 函数
	if immutable {
		return SetAttr(f, FS_IMMUTABLE_FL)
	}
	return UnsetAttr(f, FS_IMMUTABLE_FL)
}

// filterFailed 过滤掉失败的项,返回成功的项
func filterFailed(all []string, failed []string) []string {
	if len(failed) == 0 {
		return all
	}

	failedMap := make(map[string]bool)
	for _, f := range failed {
		failedMap[f] = true
	}

	var success []string
	for _, item := range all {
		if !failedMap[item] {
			success = append(success, item)
		}
	}
	return success
}
