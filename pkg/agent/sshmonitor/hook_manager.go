package sshmonitor

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const (
	PAMConfigFile  = "/etc/pam.d/sshd"
	PAMConfigLine  = "session optional pam_exec.so /usr/local/bin/pika_ssh_hook.sh"
	HookScriptPath = "/usr/local/bin/pika_ssh_hook.sh"
)

// HookManager PAM Hook 管理器
type HookManager struct{}

// NewHookManager 创建管理器
func NewHookManager() *HookManager {
	return &HookManager{}
}

// Install 安装 PAM Hook
func (h *HookManager) Install() error {
	// 检查是否为 root 用户
	if os.Geteuid() != 0 {
		return fmt.Errorf("安装 PAM Hook 需要 root 权限")
	}

	// 检查是否已安装（幂等性）
	if h.isInstalled() {
		slog.Info("PAM Hook 已安装，跳过")
		return nil
	}

	// 部署 Hook 脚本
	if err := h.deployScript(); err != nil {
		return fmt.Errorf("部署脚本失败: %w", err)
	}

	// 修改 PAM 配置
	if err := h.modifyPAMConfig(true); err != nil {
		// 回滚脚本部署
		os.Remove(HookScriptPath)
		return fmt.Errorf("修改 PAM 配置失败: %w", err)
	}

	slog.Info("PAM Hook 安装成功")
	return nil
}

// Uninstall 卸载 PAM Hook
func (h *HookManager) Uninstall() error {
	// 移除 PAM 配置
	if err := h.modifyPAMConfig(false); err != nil {
		slog.Warn("移除 PAM 配置失败", "error", err)
	}

	// 删除脚本文件
	if err := os.Remove(HookScriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除脚本失败: %w", err)
	}

	slog.Info("PAM Hook 卸载成功")
	return nil
}

// isInstalled 检查是否已安装
func (h *HookManager) isInstalled() bool {
	// 检查 PAM 配置文件是否包含我们的配置行
	f, err := os.Open(PAMConfigFile)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == PAMConfigLine {
			return true
		}
	}

	return false
}

// deployScript 部署 Hook 脚本
func (h *HookManager) deployScript() error {
	// 脚本内容（嵌入式）
	scriptContent := `#!/bin/bash
# Pika SSH Login Hook
# 由 Pika Agent 自动安装

LOG_FILE="/var/log/pika/ssh_login.log"
LOG_DIR="/var/log/pika"

# 确保日志目录存在
mkdir -p "$LOG_DIR"

# 获取登录信息
TIMESTAMP=$(date +%s)000  # 毫秒时间戳
USERNAME="$PAM_USER"
IP="${PAM_RHOST:-localhost}"
TTY="${PAM_TTY:-unknown}"
SESSION_ID="$$"

# 判断登录状态（通过 PAM_TYPE）
if [ "$PAM_TYPE" = "open_session" ]; then
    STATUS="success"
else
    STATUS="unknown"
fi

# 尝试获取端口号（从 SSH 连接信息）
PORT=$(echo "$SSH_CONNECTION" | awk '{print $2}')

# 尝试获取认证方式（简化处理）
METHOD="password"

# 构建 JSON 日志
JSON_LOG=$(cat <<EOF
{"timestamp":$TIMESTAMP,"username":"$USERNAME","ip":"$IP","port":"$PORT","status":"$STATUS","method":"$METHOD","tty":"$TTY","sessionId":"$SESSION_ID"}
EOF
)

# 追加到日志文件
echo "$JSON_LOG" >> "$LOG_FILE"

exit 0
`

	// 写入脚本文件
	if err := os.WriteFile(HookScriptPath, []byte(scriptContent), 0755); err != nil {
		return err
	}

	slog.Info("PAM Hook 脚本部署成功", "path", HookScriptPath)
	return nil
}

// modifyPAMConfig 修改 PAM 配置
// add: true=添加配置, false=移除配置
func (h *HookManager) modifyPAMConfig(add bool) error {
	// 读取现有配置
	f, err := os.Open(PAMConfigFile)
	if err != nil {
		return err
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	f.Close()

	if err := scanner.Err(); err != nil {
		return err
	}

	// 修改配置
	var newLines []string
	found := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 跳过我们的配置行（移除模式）或检测是否已存在（添加模式）
		if trimmed == PAMConfigLine {
			found = true
			if add {
				newLines = append(newLines, line) // 保留
			}
			// 移除模式：不添加到 newLines
			continue
		}

		newLines = append(newLines, line)
	}

	// 如果是添加模式且未找到，则添加到文件末尾
	if add && !found {
		newLines = append(newLines, PAMConfigLine)
	}

	// 备份原配置
	backupPath := PAMConfigFile + ".bak"
	if err := os.Rename(PAMConfigFile, backupPath); err != nil {
		return err
	}

	// 写入新配置
	outF, err := os.Create(PAMConfigFile)
	if err != nil {
		// 恢复备份
		os.Rename(backupPath, PAMConfigFile)
		return err
	}
	defer outF.Close()

	writer := bufio.NewWriter(outF)
	for _, line := range newLines {
		writer.WriteString(line + "\n")
	}
	if err := writer.Flush(); err != nil {
		// 恢复备份
		outF.Close()
		os.Rename(backupPath, PAMConfigFile)
		return err
	}

	// 设置正确的权限
	os.Chmod(PAMConfigFile, 0644)

	slog.Info("PAM 配置修改成功", "add", add)
	return nil
}
