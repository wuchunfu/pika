package audit

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dushixiang/pika/internal/protocol"
)

// SSHChecker SSH 安全检查器
type SSHChecker struct {
	config   *Config
	cache    *ProcessCache
	evidence *EvidenceCollector
	executor *CommandExecutor
	strUtil  *StringUtils
}

// NewSSHChecker 创建 SSH 检查器
func NewSSHChecker(config *Config, cache *ProcessCache, evidence *EvidenceCollector, executor *CommandExecutor) *SSHChecker {
	return &SSHChecker{
		config:   config,
		cache:    cache,
		evidence: evidence,
		executor: executor,
		strUtil:  &StringUtils{},
	}
}

// Check 检查 SSH 后门和安全配置
func (sc *SSHChecker) Check() protocol.SecurityCheck {
	check := protocol.SecurityCheck{
		Category: "ssh_security",
		Status:   StatusPass,
		Message:  "SSH安全检查",
		Details:  []protocol.SecurityCheckSub{},
	}

	// 检查 SSH 服务是否运行
	if !sc.isSSHRunning() {
		check.Message = "SSH服务未运行"
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "ssh_status",
			Status:  StatusPass,
			Message: "SSH未运行",
		})
		return check
	}

	// 1. 检查 authorized_keys 文件内容
	suspiciousKeys := sc.checkAuthorizedKeys()
	if len(suspiciousKeys) > 0 {
		check.Status = StatusWarn
		for _, key := range suspiciousKeys {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "suspicious_authorized_key",
				Status:  StatusWarn,
				Message: key,
			})
		}
	}

	// 2. 检查 SSH 配置
	sshConfig := sc.readSSHConfig()

	// 检查是否允许 root 登录（仅记录，不告警）
	//permitRoot := sshConfig["permitrootlogin"]
	//if permitRoot != "" {
	//	check.Details = append(check.Details, protocol.SecurityCheckSub{
	//		Name:    "permit_root_login",
	//		Status:  StatusPass,
	//		Message: fmt.Sprintf("Root登录策略: %s", permitRoot),
	//	})
	//}

	// 检查空密码登录
	permitEmpty := sshConfig["permitemptypasswords"]
	if permitEmpty == "yes" {
		check.Status = StatusFail
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "permit_empty_passwords",
			Status:  StatusFail,
			Message: "允许空密码登录",
		})
	}

	// 检查密码认证
	passwordAuth := sshConfig["passwordauthentication"]
	if passwordAuth == "yes" {
		check.Status = StatusWarn
		check.Details = append(check.Details, protocol.SecurityCheckSub{
			Name:    "password_auth_enabled",
			Status:  StatusWarn,
			Message: "启用了密码认证(建议仅使用密钥认证)",
		})
	}

	// 3. 检查 SSH 后门进程
	suspiciousSSHD := sc.checkSuspiciousSSHD()
	if len(suspiciousSSHD) > 0 {
		check.Status = StatusFail
		for _, sshd := range suspiciousSSHD {
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "backdoor_sshd",
				Status:  StatusFail,
				Message: sshd,
			})
		}
	}

	// 4. 检查 SSH 二进制文件完整性
	for _, binPath := range sc.config.SSHConfig.BinaryPaths {
		info, err := os.Stat(binPath)
		if err != nil {
			continue
		}

		modTime := info.ModTime()
		age := time.Since(modTime)
		if age < time.Duration(sc.config.SSHConfig.RecentModifyDays)*24*time.Hour {
			fileHash := sc.evidence.hashCache.GetSHA256(binPath)
			check.Status = StatusWarn
			check.Details = append(check.Details, protocol.SecurityCheckSub{
				Name:    "ssh_binary_modified",
				Status:  StatusWarn,
				Message: fmt.Sprintf("SSH二进制文件最近被修改: %s (%d天前)", binPath, int(age.Hours()/24)),
				Evidence: &protocol.Evidence{
					FilePath:  binPath,
					FileHash:  fileHash,
					Timestamp: modTime.UnixMilli(),
					RiskLevel: "high",
				},
			})
		}
	}

	// 5. 检查 SSH 配置文件的软链接
	for _, configPath := range sc.config.SSHConfig.ConfigPaths {
		linkTarget, err := os.Readlink(configPath)
		if err == nil {
			// 是软链接，检查目标是否可疑
			if strings.Contains(linkTarget, "/tmp") || strings.Contains(linkTarget, "/dev/shm") {
				check.Status = StatusFail
				check.Details = append(check.Details, protocol.SecurityCheckSub{
					Name:    "ssh_config_symlink",
					Status:  StatusFail,
					Message: fmt.Sprintf("SSH配置文件是指向可疑位置的软链接: %s -> %s", configPath, linkTarget),
				})
			}
		}
	}

	// 更新总体消息
	switch check.Status {
	case StatusPass:
		check.Message = "SSH配置安全"
	case StatusWarn:
		check.Message = "SSH配置存在风险"
	case StatusFail:
		check.Message = "检测到SSH后门"
	}

	return check
}

// isSSHRunning 检查 SSH 服务是否运行
func (sc *SSHChecker) isSSHRunning() bool {
	procs, err := sc.cache.Get()
	if err != nil {
		return false
	}

	for _, p := range procs {
		name, err := p.Name()
		if err == nil && strings.Contains(name, "sshd") {
			return true
		}
	}

	return false
}

// checkAuthorizedKeys 检查 authorized_keys 文件内容
func (sc *SSHChecker) checkAuthorizedKeys() []string {
	var suspicious []string

	// 获取所有用户目录
	userDirs := sc.getAllUserDirectories()

	for _, userDir := range userDirs {
		keyPath := filepath.Join(userDir.Path, ".ssh", "authorized_keys")

		info, err := os.Stat(keyPath)
		if err != nil {
			continue
		}

		// 检查文件大小
		if info.Size() > sc.config.SSHConfig.MaxAuthorizedKeysSize {
			suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 的 authorized_keys 文件过大 (%.2f MB)，可能是异常",
				userDir.Username, float64(info.Size())/1024/1024))
		}

		// 检查最近修改
		if time.Since(info.ModTime()) < time.Duration(sc.config.SSHConfig.AuthKeysRecentModifyDays)*24*time.Hour {
			suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 的 authorized_keys 最近被修改: %s (%d天前)",
				userDir.Username, info.ModTime().Format("2006-01-02 15:04:05"), int(time.Since(info.ModTime()).Hours()/24)))
		}

		// 检查文件内容
		f, err := os.Open(keyPath)
		if err != nil {
			continue
		}

		reader := io.LimitReader(f, sc.config.PerformanceConfig.AuthKeysReadLimitKB*1024)
		content, err := io.ReadAll(reader)
		f.Close()
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		keyCount := 0

		for lineNum, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// 检查可疑的公钥选项
			if strings.Contains(line, "command=") {
				suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 的公钥包含command选项(行%d): %s...",
					userDir.Username, lineNum+1, sc.strUtil.Truncate(line, 60)))
			}

			if strings.Contains(line, "from=") {
				if strings.Contains(line, `from="*"`) || strings.Contains(line, "from=*") {
					suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 的公钥允许任意来源(from=*): 行%d", userDir.Username, lineNum+1))
				}
			}

			// 统计公钥数量
			if strings.HasPrefix(line, "ssh-rsa") || strings.HasPrefix(line, "ssh-ed25519") || strings.HasPrefix(line, "ecdsa-") {
				keyCount++
			}
		}

		if keyCount > sc.config.SSHConfig.MaxKeysCount {
			suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 拥有大量公钥(%d个), 建议审查", userDir.Username, keyCount))
		}

		// 检查文件权限
		if info.Mode().Perm()&0022 != 0 {
			suspicious = append(suspicious, fmt.Sprintf("用户 '%s' 的 authorized_keys 权限过高: %o (Group/Other不可写)", userDir.Username, info.Mode().Perm()))
		}
	}

	return suspicious
}

// readSSHConfig 读取 SSH 配置
func (sc *SSHChecker) readSSHConfig() map[string]string {
	config := make(map[string]string)

	// 优先使用 sshd -T
	output, err := sc.executor.Execute("sshd", "-T")
	if err == nil {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				config[strings.ToLower(parts[0])] = parts[1]
			}
		}
		return config
	}

	// 备用：读配置文件
	for _, path := range sc.config.SSHConfig.ConfigPaths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.Fields(line)
			if len(parts) >= 2 {
				config[strings.ToLower(parts[0])] = parts[1]
			}
		}
		break
	}

	return config
}

// checkSuspiciousSSHD 检查可疑的 sshd 进程
func (sc *SSHChecker) checkSuspiciousSSHD() []string {
	var suspicious []string

	procs, err := sc.cache.Get()
	if err != nil {
		return suspicious
	}

	sshdCount := 0
	for _, p := range procs {
		name, _ := p.Name()
		if strings.Contains(name, "sshd") {
			sshdCount++
			exe, _ := p.Exe()

			// 检查 sshd 路径是否正常
			if exe != "" && !strings.HasPrefix(exe, "/usr/sbin/sshd") {
				suspicious = append(suspicious, fmt.Sprintf("异常sshd路径: %s (PID: %d)", exe, p.Pid))
			}
		}
	}

	// 如果有多个 sshd 守护进程可能异常
	if sshdCount > 10 {
		suspicious = append(suspicious, fmt.Sprintf("sshd进程数异常: %d", sshdCount))
	}

	return suspicious
}

// UserDirectory 用户目录信息
type UserDirectory struct {
	Username string
	Path     string
	Source   string
}

// getAllUserDirectories 获取所有用户目录
func (sc *SSHChecker) getAllUserDirectories() []UserDirectory {
	var userDirs []UserDirectory
	seenPaths := make(map[string]bool)

	// 1. 从 /etc/passwd 获取用户
	users := sc.getAllUsers()
	for _, user := range users {
		if user.Home != "" && !seenPaths[user.Home] {
			userDirs = append(userDirs, UserDirectory{
				Username: user.Username,
				Path:     user.Home,
				Source:   "passwd",
			})
			seenPaths[user.Home] = true
		}
	}

	// 2. 扫描 /home 目录，发现 LDAP/AD 等外部认证用户
	entries, err := os.ReadDir("/home")
	if err != nil {
		return userDirs
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		homePath := filepath.Join("/home", entry.Name())

		if seenPaths[homePath] {
			continue
		}

		// 检查是否有 .ssh 目录
		sshPath := filepath.Join(homePath, ".ssh")
		if stat, err := os.Stat(sshPath); err == nil && stat.IsDir() {
			userDirs = append(userDirs, UserDirectory{
				Username: entry.Name() + " (LDAP/AD)",
				Path:     homePath,
				Source:   "home_scan",
			})
			seenPaths[homePath] = true
		}
	}

	return userDirs
}

// UserInfo 用户信息
type UserInfo struct {
	Username string
	UID      string
	Home     string
	Shell    string
}

// getAllUsers 获取所有用户
func (sc *SSHChecker) getAllUsers() []UserInfo {
	var users []UserInfo

	file, err := os.Open("/etc/passwd")
	if err != nil {
		return users
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 7 {
			users = append(users, UserInfo{
				Username: parts[0],
				UID:      parts[2],
				Home:     parts[5],
				Shell:    parts[6],
			})
		}
	}

	return users
}
