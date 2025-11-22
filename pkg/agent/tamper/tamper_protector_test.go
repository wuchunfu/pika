package tamper

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestNewProtector(t *testing.T) {
	p := NewProtector()
	if p == nil {
		t.Fatal("NewProtector() 返回 nil")
	}

	paths := p.GetProtectedPaths()
	if len(paths) != 0 {
		t.Errorf("新建的保护器不应该有受保护的路径，当前有 %d 个", len(paths))
	}
}

func TestUpdatePaths(t *testing.T) {
	// 此测试仅在 Linux 系统上运行
	if runtime.GOOS != "linux" {
		t.Skip("防篡改功能仅支持 Linux 系统")
	}

	// 检查是否有 root 权限
	if os.Geteuid() != 0 {
		t.Skip("此测试需要 root 权限才能运行 chattr 命令")
	}

	p := NewProtector()
	ctx := context.Background()

	// 创建测试目录
	testDir1 := "/tmp/tamper_test_1_" + time.Now().Format("20060102150405")
	testDir2 := "/tmp/tamper_test_2_" + time.Now().Format("20060102150405")
	testDir3 := "/tmp/tamper_test_3_" + time.Now().Format("20060102150405")
	testDir4 := "/tmp/tamper_test_4_" + time.Now().Format("20060102150405")

	if err := os.MkdirAll(testDir1, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir1)

	if err := os.MkdirAll(testDir2, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir2)

	if err := os.MkdirAll(testDir3, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir3)

	if err := os.MkdirAll(testDir4, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir4)

	defer p.StopAll()

	// 第一次更新: 添加 /a /b /c
	t.Run("第一次配置: /a /b /c", func(t *testing.T) {
		result, err := p.UpdatePaths(ctx, []string{testDir1, testDir2, testDir3})
		if err != nil {
			t.Fatalf("UpdatePaths() 失败: %v", err)
		}

		if len(result.Added) != 3 {
			t.Errorf("应该新增 3 个目录，实际新增 %d 个", len(result.Added))
		}

		if len(result.Removed) != 0 {
			t.Errorf("不应该移除任何目录，实际移除 %d 个", len(result.Removed))
		}

		if len(result.Current) != 3 {
			t.Errorf("当前应该保护 3 个目录，实际保护 %d 个", len(result.Current))
		}

		// 验证目录确实受保护
		if !p.IsProtected(testDir1) {
			t.Errorf("目录 %s 应该受保护", testDir1)
		}
	})

	// 第二次更新: 修改为 /a /b (移除 /c)
	t.Run("第二次配置: /a /b", func(t *testing.T) {
		result, err := p.UpdatePaths(ctx, []string{testDir1, testDir2})
		if err != nil {
			t.Fatalf("UpdatePaths() 失败: %v", err)
		}

		if len(result.Added) != 0 {
			t.Errorf("不应该新增任何目录，实际新增 %d 个", len(result.Added))
		}

		if len(result.Removed) != 1 {
			t.Errorf("应该移除 1 个目录，实际移除 %d 个", len(result.Removed))
		}

		if len(result.Current) != 2 {
			t.Errorf("当前应该保护 2 个目录，实际保护 %d 个", len(result.Current))
		}

		// 验证 /c 不再受保护
		if p.IsProtected(testDir3) {
			t.Errorf("目录 %s 不应该受保护", testDir3)
		}
	})

	// 第三次更新: 修改为 /a /b /c /d (新增 /c /d)
	t.Run("第三次配置: /a /b /c /d", func(t *testing.T) {
		result, err := p.UpdatePaths(ctx, []string{testDir1, testDir2, testDir3, testDir4})
		if err != nil {
			t.Fatalf("UpdatePaths() 失败: %v", err)
		}

		if len(result.Added) != 2 {
			t.Errorf("应该新增 2 个目录，实际新增 %d 个", len(result.Added))
		}

		if len(result.Removed) != 0 {
			t.Errorf("不应该移除任何目录，实际移除 %d 个", len(result.Removed))
		}

		if len(result.Current) != 4 {
			t.Errorf("当前应该保护 4 个目录，实际保护 %d 个", len(result.Current))
		}

		// 验证所有目录都受保护
		for _, dir := range []string{testDir1, testDir2, testDir3, testDir4} {
			if !p.IsProtected(dir) {
				t.Errorf("目录 %s 应该受保护", dir)
			}
		}
	})

	// 第四次更新: 清空列表
	t.Run("第四次配置: 清空", func(t *testing.T) {
		result, err := p.UpdatePaths(ctx, []string{})
		if err != nil {
			t.Fatalf("UpdatePaths() 失败: %v", err)
		}

		if len(result.Added) != 0 {
			t.Errorf("不应该新增任何目录，实际新增 %d 个", len(result.Added))
		}

		if len(result.Removed) != 4 {
			t.Errorf("应该移除 4 个目录，实际移除 %d 个", len(result.Removed))
		}

		if len(result.Current) != 0 {
			t.Errorf("当前应该保护 0 个目录，实际保护 %d 个", len(result.Current))
		}
	})

	// 第五次更新: 重复配置相同的列表
	t.Run("第五次配置: /a /b (重复)", func(t *testing.T) {
		// 先添加
		p.UpdatePaths(ctx, []string{testDir1, testDir2})

		// 再次添加相同的配置
		result, err := p.UpdatePaths(ctx, []string{testDir1, testDir2})
		if err != nil {
			t.Fatalf("UpdatePaths() 失败: %v", err)
		}

		if len(result.Added) != 0 {
			t.Errorf("重复配置不应该新增任何目录，实际新增 %d 个", len(result.Added))
		}

		if len(result.Removed) != 0 {
			t.Errorf("重复配置不应该移除任何目录，实际移除 %d 个", len(result.Removed))
		}

		if len(result.Current) != 2 {
			t.Errorf("当前应该保护 2 个目录，实际保护 %d 个", len(result.Current))
		}
	})
}

func TestStopAll(t *testing.T) {
	// 此测试仅在 Linux 系统上运行
	if runtime.GOOS != "linux" {
		t.Skip("防篡改功能仅支持 Linux 系统")
	}

	// 检查是否有 root 权限
	if os.Geteuid() != 0 {
		t.Skip("此测试需要 root 权限才能运行 chattr 命令")
	}

	p := NewProtector()

	// 创建测试目录
	testDir := "/tmp/tamper_test_stop_" + time.Now().Format("20060102150405")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir)

	ctx := context.Background()

	// 添加保护
	_, err := p.UpdatePaths(ctx, []string{testDir})
	if err != nil {
		t.Fatalf("UpdatePaths() 失败: %v", err)
	}

	// 验证目录受保护
	if !p.IsProtected(testDir) {
		t.Error("目录应该受保护")
	}

	// 停止所有保护
	err = p.StopAll()
	if err != nil {
		t.Fatalf("StopAll() 失败: %v", err)
	}

	// 验证目录不再受保护
	if p.IsProtected(testDir) {
		t.Error("目录不应该受保护")
	}

	paths := p.GetProtectedPaths()
	if len(paths) != 0 {
		t.Errorf("停止后不应该有受保护的路径，当前有 %d 个", len(paths))
	}
}

func TestProtectorEvents(t *testing.T) {
	// 此测试仅在 Linux 系统上运行
	if runtime.GOOS != "linux" {
		t.Skip("防篡改功能仅支持 Linux 系统")
	}

	// 检查是否有 root 权限
	if os.Geteuid() != 0 {
		t.Skip("此测试需要 root 权限才能运行 chattr 命令")
	}

	p := NewProtector()

	// 创建测试目录
	testDir := "/tmp/tamper_test_events_" + time.Now().Format("20060102150405")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir)

	ctx := context.Background()

	// 启动保护器
	_, err := p.UpdatePaths(ctx, []string{testDir})
	if err != nil {
		t.Fatalf("UpdatePaths() 失败: %v", err)
	}
	defer p.StopAll()

	// 获取事件通道
	eventCh := p.GetEvents()

	// 等待一段时间让 watcher 准备就绪
	time.Sleep(500 * time.Millisecond)

	// 尝试在受保护的目录中创建文件（这应该被阻止或触发事件）
	testFile := testDir + "/test.txt"
	_ = os.WriteFile(testFile, []byte("test"), 0644)

	// 等待事件
	select {
	case event := <-eventCh:
		t.Logf("收到防篡改事件: Path=%s, Operation=%s, Details=%s",
			event.Path, event.Operation, event.Details)
		// 验证事件字段
		if event.Path == "" {
			t.Error("事件路径不应为空")
		}
		if event.Operation == "" {
			t.Error("事件操作不应为空")
		}
	case <-time.After(3 * time.Second):
		t.Log("在超时时间内没有收到事件（这可能是正常的，因为 chattr +i 会阻止操作）")
	}
}

func TestTamperEventJSON(t *testing.T) {
	event := TamperEvent{
		Path:      "/tmp/test.txt",
		Operation: "write",
		Timestamp: time.Now(),
		Details:   "文件被写入",
	}

	// 测试 JSON 序列化
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 测试 JSON 反序列化
	var decoded TamperEvent
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// 验证字段
	if decoded.Path != event.Path {
		t.Errorf("Path = %s, want %s", decoded.Path, event.Path)
	}
	if decoded.Operation != event.Operation {
		t.Errorf("Operation = %s, want %s", decoded.Operation, event.Operation)
	}
	if decoded.Details != event.Details {
		t.Errorf("Details = %s, want %s", decoded.Details, event.Details)
	}
}

func TestUpdateResultJSON(t *testing.T) {
	result := UpdateResult{
		Added:   []string{"/tmp/a", "/tmp/b"},
		Removed: []string{"/tmp/c"},
		Current: []string{"/tmp/a", "/tmp/b", "/tmp/d"},
	}

	// 测试 JSON 序列化
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 测试 JSON 反序列化
	var decoded UpdateResult
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// 验证字段
	if len(decoded.Added) != len(result.Added) {
		t.Errorf("Added length = %d, want %d", len(decoded.Added), len(result.Added))
	}
	if len(decoded.Removed) != len(result.Removed) {
		t.Errorf("Removed length = %d, want %d", len(decoded.Removed), len(result.Removed))
	}
	if len(decoded.Current) != len(result.Current) {
		t.Errorf("Current length = %d, want %d", len(decoded.Current), len(result.Current))
	}
}

func TestAttributeTamperAlertJSON(t *testing.T) {
	alert := AttributeTamperAlert{
		Path:      "/tmp/test",
		Timestamp: time.Now(),
		Details:   "不可变属性被移除",
		Restored:  true,
	}

	// 测试 JSON 序列化
	data, err := json.Marshal(alert)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 测试 JSON 反序列化
	var decoded AttributeTamperAlert
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// 验证字段
	if decoded.Path != alert.Path {
		t.Errorf("Path = %s, want %s", decoded.Path, alert.Path)
	}
	if decoded.Details != alert.Details {
		t.Errorf("Details = %s, want %s", decoded.Details, alert.Details)
	}
	if decoded.Restored != alert.Restored {
		t.Errorf("Restored = %v, want %v", decoded.Restored, alert.Restored)
	}
}

func TestCheckImmutable(t *testing.T) {
	// 此测试仅在 Linux 系统上运行
	if runtime.GOOS != "linux" {
		t.Skip("防篡改功能仅支持 Linux 系统")
	}

	// 检查是否有 root 权限
	if os.Geteuid() != 0 {
		t.Skip("此测试需要 root 权限才能运行")
	}

	p := NewProtector()

	// 创建测试目录
	testDir := "/tmp/tamper_test_check_" + time.Now().Format("20060102150405")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 设置不可变属性
	if err := p.setImmutable(testDir, true); err != nil {
		t.Fatalf("设置不可变属性失败: %v", err)
	}
	defer p.setImmutable(testDir, false)

	// 检查属性
	hasImmutable, err := p.checkImmutable(testDir)
	if err != nil {
		t.Fatalf("检查不可变属性失败: %v", err)
	}
	if !hasImmutable {
		t.Error("目录应该具有不可变属性")
	}

	// 移除属性
	if err := p.setImmutable(testDir, false); err != nil {
		t.Fatalf("移除不可变属性失败: %v", err)
	}

	// 再次检查
	hasImmutable, err = p.checkImmutable(testDir)
	if err != nil {
		t.Fatalf("检查不可变属性失败: %v", err)
	}
	if hasImmutable {
		t.Error("目录不应该具有不可变属性")
	}
}

func TestChmodEventDetectsAttributeTamper(t *testing.T) {
	// 此测试仅在 Linux 系统上运行
	if runtime.GOOS != "linux" {
		t.Skip("防篡改功能仅支持 Linux 系统")
	}

	// 检查是否有 root 权限
	if os.Geteuid() != 0 {
		t.Skip("此测试需要 root 权限才能运行")
	}

	p := NewProtector()

	// 创建测试目录
	testDir := "/tmp/tamper_test_chmod_" + time.Now().Format("20060102150405")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir)

	ctx := context.Background()

	// 添加保护
	_, err := p.UpdatePaths(ctx, []string{testDir})
	if err != nil {
		t.Fatalf("UpdatePaths() 失败: %v", err)
	}
	defer p.StopAll()

	// 获取告警通道
	alertCh := p.GetAlerts()

	// 等待一段时间让监控器准备就绪
	time.Sleep(1 * time.Second)

	// 模拟外部篡改: 使用 chattr 命令移除不可变属性
	// 这会触发 fsnotify.Chmod 事件
	t.Log("模拟外部篡改: 使用 chattr -i 移除不可变属性")
	cmd := exec.Command("chattr", "-i", testDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("移除不可变属性失败: %v", err)
	}

	// 等待告警 (fsnotify.Chmod 事件应该会触发检查)
	select {
	case alert := <-alertCh:
		t.Logf("收到属性篡改告警: Path=%s, Details=%s, Restored=%v",
			alert.Path, alert.Details, alert.Restored)

		// 验证告警字段
		if alert.Path != testDir {
			t.Errorf("告警路径 = %s, want %s", alert.Path, testDir)
		}
		if alert.Details != "不可变属性被移除" {
			t.Errorf("告警详情 = %s, want '不可变属性被移除'", alert.Details)
		}
		if !alert.Restored {
			t.Error("属性应该被自动恢复")
		}

		// 验证属性确实被恢复
		hasImmutable, err := p.checkImmutable(testDir)
		if err != nil {
			t.Fatalf("检查不可变属性失败: %v", err)
		}
		if !hasImmutable {
			t.Error("不可变属性应该被自动恢复")
		}

	case <-time.After(5 * time.Second):
		t.Fatal("超时未收到属性篡改告警")
	}
}
