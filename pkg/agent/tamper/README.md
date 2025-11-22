# 防篡改保护模块

## 功能概述

该模块为 Pika Agent 提供 Linux 文件系统防篡改保护功能。通过以下机制实现:

1. **文件属性保护**: 使用 `chattr +i` 命令将指定目录设置为不可变(immutable)属性
2. **实时监控**: 使用 `fsnotify` 库监控受保护目录的文件系统事件
3. **事件上报**: 将检测到的文件变动事件实时上报到服务端
4. **动态管理**: 支持动态增加、移除保护目录,自动计算差异并应用
5. **属性巡检**: 定期检查目录的不可变属性,检测并自动恢复被篡改的属性

## 系统要求

- **操作系统**: 仅支持 Linux 系统
- **权限**: 需要 root 权限才能执行 `chattr` 命令
- **依赖**: `github.com/fsnotify/fsnotify`

## 工作原理

### 动态目录管理

服务端每次发送的是**完整的目录列表**,Agent 会自动计算需要新增和移除的目录:

#### 场景示例

**第一次配置**: `/a /b /c`
```json
{
  "type": "tamper_protect",
  "data": {
    "paths": ["/a", "/b", "/c"]
  }
}
```
- Agent 操作: 新增保护 `/a /b /c`
- 响应: `added: ["/a", "/b", "/c"], removed: [], current: ["/a", "/b", "/c"]`

**第二次配置**: `/a /b` (移除 /c)
```json
{
  "type": "tamper_protect",
  "data": {
    "paths": ["/a", "/b"]
  }
}
```
- Agent 操作: 移除 `/c` 的保护 (执行 `chattr -i /c`)
- 响应: `added: [], removed: ["/c"], current: ["/a", "/b"]`

**第三次配置**: `/a /b /c /d` (新增 /c /d)
```json
{
  "type": "tamper_protect",
  "data": {
    "paths": ["/a", "/b", "/c", "/d"]
  }
}
```
- Agent 操作: 新增保护 `/c /d` (执行 `chattr +i /c` 和 `chattr +i /d`)
- 响应: `added: ["/c", "/d"], removed: [], current: ["/a", "/b", "/c", "/d"]`

**第四次配置**: `[]` (清空所有保护)
```json
{
  "type": "tamper_protect",
  "data": {
    "paths": []
  }
}
```
- Agent 操作: 移除所有目录的保护
- 响应: `added: [], removed: ["/a", "/b", "/c", "/d"], current: []`

### 核心流程

```
服务端发送完整列表
        ↓
Agent 计算差异
  • 新列表 - 当前列表 = 需要新增
  • 当前列表 - 新列表 = 需要移除
        ↓
执行操作
  • 对移除的目录: chattr -i + watcher.Remove()
  • 对新增的目录: chattr +i + watcher.Add()
        ↓
启动属性巡检 (首次有保护目录时)
  • 每 60 秒巡检一次
  • 检查所有目录的不可变属性
  • 发现属性被篡改时自动恢复并告警
        ↓
返回结果
  • 成功/失败状态
  • 新增的目录列表
  • 移除的目录列表
  • 当前所有保护的目录
```

### 属性巡检机制

为了防止其他用户或进程移除目录的不可变属性,系统会定期巡检所有受保护目录:

**巡检流程**:
1. 每 60 秒检查一次所有受保护目录
2. 使用 `lsattr -d` 命令检查不可变属性
3. 如果发现属性被移除:
   - 自动执行 `chattr +i` 恢复属性
   - 生成告警并上报服务端
   - 告警中包含是否成功恢复的状态

**告警示例**:
```json
{
  "type": "tamper_alert",
  "data": {
    "path": "/etc/nginx",
    "timestamp": 1700000000000,
    "details": "不可变属性被移除",
    "restored": true
  }
}
```

## 核心 API

### Protector.UpdatePaths()

```go
func (p *Protector) UpdatePaths(ctx context.Context, newPaths []string) (*UpdateResult, error)
```

**参数**:
- `newPaths`: 新的完整目录列表

**返回**:
- `UpdateResult`: 更新结果
  - `Added`: 本次新增保护的目录
  - `Removed`: 本次移除保护的目录
  - `Current`: 当前所有受保护的目录
- `error`: 错误信息(如果部分操作失败,仍会返回 UpdateResult)

**特性**:
- **幂等性**: 多次使用相同参数调用,结果一致
- **原子性**: 单个目录的添加/移除操作失败不影响其他目录
- **智能差异**: 自动计算需要变更的目录,避免重复操作

### Protector.StopAll()

```go
func (p *Protector) StopAll() error
```

停止所有防篡改保护,移除所有目录的不可变属性,关闭文件监控器。

### Protector.GetProtectedPaths()

```go
func (p *Protector) GetProtectedPaths() []string
```

获取当前所有受保护的目录列表。

### Protector.IsProtected()

```go
func (p *Protector) IsProtected(path string) bool
```

检查指定路径是否受保护。

## Protocol 定义

### 消息类型

```go
MessageTypeTamperProtect MessageType = "tamper_protect" // 配置防篡改保护
MessageTypeTamperEvent   MessageType = "tamper_event"   // 文件变动事件
MessageTypeTamperAlert   MessageType = "tamper_alert"   // 属性篡改告警
```

### 数据结构

#### 配置请求
```go
type TamperProtectConfig struct {
    Paths []string // 要保护的目录列表(完整列表)
}
```

#### 配置响应
```go
type TamperProtectResponse struct {
    Success bool     // 是否成功
    Message string   // 响应消息
    Paths   []string // 当前保护的目录列表
    Added   []string // 本次新增的目录
    Removed []string // 本次移除的目录
    Error   string   // 错误信息
}
```

#### 事件数据
```go
type TamperEventData struct {
    Path      string // 被修改的路径
    Operation string // 操作类型: write, remove, rename, chmod, create
    Timestamp int64  // 事件时间(毫秒)
    Details   string // 详细信息
}
```

#### 属性篡改告警数据
```go
type TamperAlertData struct {
    Path      string // 被篡改的路径
    Timestamp int64  // 检测时间(毫秒)
    Details   string // 详细信息(如: "不可变属性被移除")
    Restored  bool   // 是否已自动恢复
}
```

## Agent 集成

在 `pkg/agent/service/agent.go` 中的集成:

```go
// 初始化
agent.tamperProtector = tamper.NewProtector()

// 接收配置消息
case protocol.MessageTypeTamperProtect:
    go a.handleTamperProtect(msg.Data)

// 处理配置
func (a *Agent) handleTamperProtect(data json.RawMessage) {
    var config protocol.TamperProtectConfig
    // ...解析配置

    if len(config.Paths) == 0 {
        // 清空所有保护
        a.tamperProtector.StopAll()
    } else {
        // 更新保护目录
        result, err := a.tamperProtector.UpdatePaths(ctx, config.Paths)
        // ...发送响应
    }
}

// 事件监控循环
func (a *Agent) tamperEventLoop(ctx context.Context, conn *safeConn, done chan struct{}) {
    eventCh := a.tamperProtector.GetEvents()
    for event := range eventCh {
        // 上报事件到服务端
        sendTamperEvent(event)
    }
}
```

## 使用示例

### 服务端: 第一次配置

```go
config := protocol.TamperProtectConfig{
    Paths: []string{"/etc/nginx", "/var/www"},
}

// 发送到 Agent
sendMessage(protocol.MessageTypeTamperProtect, config)

// 期望响应
// {
//   "success": true,
//   "message": "防篡改保护已更新: 新增 2 个, 移除 0 个, 当前保护 2 个目录",
//   "paths": ["/etc/nginx", "/var/www"],
//   "added": ["/etc/nginx", "/var/www"],
//   "removed": []
// }
```

### 服务端: 修改配置(移除一个目录)

```go
config := protocol.TamperProtectConfig{
    Paths: []string{"/etc/nginx"}, // 只保留一个
}

sendMessage(protocol.MessageTypeTamperProtect, config)

// 期望响应
// {
//   "success": true,
//   "message": "防篡改保护已更新: 新增 0 个, 移除 1 个, 当前保护 1 个目录",
//   "paths": ["/etc/nginx"],
//   "added": [],
//   "removed": ["/var/www"]
// }
```

### 服务端: 添加新目录

```go
config := protocol.TamperProtectConfig{
    Paths: []string{"/etc/nginx", "/etc/ssh", "/opt/app"},
}

sendMessage(protocol.MessageTypeTamperProtect, config)

// 期望响应
// {
//   "success": true,
//   "message": "防篡改保护已更新: 新增 2 个, 移除 0 个, 当前保护 3 个目录",
//   "paths": ["/etc/nginx", "/etc/ssh", "/opt/app"],
//   "added": ["/etc/ssh", "/opt/app"],
//   "removed": []
// }
```

### 服务端: 清空所有保护

```go
config := protocol.TamperProtectConfig{
    Paths: []string{}, // 空列表
}

sendMessage(protocol.MessageTypeTamperProtect, config)

// 期望响应
// {
//   "success": true,
//   "message": "已停止所有防篡改保护",
//   "paths": [],
//   "added": [],
//   "removed": ["/etc/nginx", "/etc/ssh", "/opt/app"]
// }
```

### 服务端: 接收文件变动事件

```go
// 监听文件变动事件
case protocol.MessageTypeTamperEvent:
    var event protocol.TamperEventData
    json.Unmarshal(msg.Data, &event)

    // 处理防篡改事件
    log.Printf("检测到文件篡改: %s - %s at %d",
        event.Path, event.Operation, event.Timestamp)

    // 可以进行告警、记录日志等操作
    sendAlert(event)
```

### 服务端: 接收属性篡改告警

```go
// 监听属性篡改告警
case protocol.MessageTypeTamperAlert:
    var alert protocol.TamperAlertData
    json.Unmarshal(msg.Data, &alert)

    // 处理属性篡改告警
    status := "未恢复"
    if alert.Restored {
        status = "已自动恢复"
    }

    log.Printf("⚠️ 检测到属性篡改: %s - %s (%s)",
        alert.Path, alert.Details, status)

    // 发送高优先级告警
    sendCriticalAlert(alert)

    // 如果未能自动恢复,触发人工介入流程
    if !alert.Restored {
        notifyAdministrator(alert)
    }
```

## 测试

### 运行测试

```bash
# 运行所有测试(在非 Linux 系统上部分测试会跳过)
go test -v ./pkg/agent/tamper/

# 在 Linux 系统上以 root 权限运行完整测试
sudo go test -v ./pkg/agent/tamper/
```

### 测试场景

测试覆盖了完整的动态管理场景:

1. ✅ **第一次配置 `/a /b /c`**: 验证新增 3 个目录
2. ✅ **第二次配置 `/a /b`**: 验证移除 1 个目录
3. ✅ **第三次配置 `/a /b /c /d`**: 验证新增 2 个目录
4. ✅ **第四次配置 `[]`**: 验证移除所有目录
5. ✅ **重复配置**: 验证幂等性(无变化时不执行操作)
6. ✅ **事件监控**: 验证文件变动事件上报
7. ✅ **JSON 序列化**: 验证数据结构正确性

## 设计优势

### 1. 简化服务端逻辑

服务端无需维护状态,每次只需发送完整的目录列表,差异计算由 Agent 完成:

```
❌ 复杂的方式(需要服务端计算差异):
  - 服务端需要知道 Agent 当前保护的目录
  - 需要对比计算差异
  - 发送增删指令

✅ 简单的方式(Agent 计算差异):
  - 服务端只需发送完整列表
  - Agent 自动计算并应用差异
  - 响应中包含完整的变更信息
```

### 2. 幂等性保证

相同的配置多次发送不会产生副作用:

```go
// 第一次
UpdatePaths(["/a", "/b"]) → added: ["/a", "/b"]

// 第二次(相同配置)
UpdatePaths(["/a", "/b"]) → added: [], removed: [] (无操作)
```

### 3. 容错性

部分目录操作失败不影响其他目录:

```go
result, err := UpdatePaths(["/valid", "/invalid", "/valid2"])
// 即使 /invalid 失败,/valid 和 /valid2 仍然会被保护
// result.Added = ["/valid", "/valid2"]
// err 会包含失败信息
```

### 4. 清晰的状态反馈

响应中包含完整的变更信息,便于服务端验证:

```json
{
  "success": true,
  "paths": ["/a", "/b", "/c"],    // 当前状态
  "added": ["/c"],                 // 本次新增
  "removed": ["/d"]                // 本次移除
}
```

## 注意事项

1. **权限要求**: 必须以 root 权限运行 Agent,否则 `chattr` 命令会失败
2. **操作系统限制**: 仅支持 Linux 系统,在其他系统上会返回错误
3. **文件系统支持**: 目标文件系统必须支持 chattr 属性(如 ext2/ext3/ext4/xfs)
4. **性能影响**:
   - `chattr +i` 会阻止所有写操作,包括合法的系统更新
   - 文件监控会消耗一定的系统资源
5. **事件队列**: 事件通道缓冲区大小为 100,如果处理不及时可能丢失事件
6. **并发安全**: 使用了读写锁保护共享状态,支持并发操作
7. **watcher 生命周期**: 首次调用 UpdatePaths 时创建,调用 StopAll 时销毁

## 安全建议

1. **谨慎选择保护路径**: 不要保护系统关键目录(如 `/`, `/usr`, `/bin`),可能导致系统无法正常运行
2. **定期审查**: 定期检查防篡改事件,及时发现异常行为
3. **配合其他安全措施**: 防篡改保护应作为纵深防御的一部分,配合防火墙、入侵检测等
4. **备份重要文件**: 在启用保护前备份重要配置文件
5. **测试环境验证**: 先在测试环境验证,确认不影响业务后再部署到生产环境

## 故障排查

### chattr 命令失败

```
错误: 执行 chattr 失败: permission denied
解决: 确保以 root 权限运行 Agent
```

### 不支持的文件系统

```
错误: 执行 chattr 失败: Inappropriate ioctl for device
解决: 检查目标目录所在的文件系统是否支持 chattr 属性
```

### 事件未上报

```
问题: 配置了防篡改保护,但没有收到事件
排查:
1. 检查 Agent 日志,确认保护已启用
2. 确认文件确实发生了变动
3. 检查事件队列是否已满
4. 检查网络连接是否正常
```

### 部分目录操作失败

```
响应: {
  "success": false,
  "error": "部分操作失败: 添加失败 1 个, 移除失败 0 个",
  "paths": ["/valid"],
  "added": ["/valid"],
  "removed": []
}

处理:
1. 检查失败目录的权限
2. 检查失败目录是否存在
3. 检查文件系统类型
4. 查看 Agent 日志获取详细错误
```

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                         服务端                               │
│                                                              │
│  管理员配置: ["/etc/nginx", "/var/www", "/opt/app"]          │
│                          ↓                                   │
│  发送完整列表(TamperProtectConfig)                           │
└────────────────────────┬────────────────────────────────────┘
                         │ WebSocket
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                       Agent                                  │
│                                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │            Protector.UpdatePaths()                 │     │
│  │                                                     │     │
│  │  1. 计算差异:                                       │     │
│  │     • 新列表 - 当前 = 需要新增                      │     │
│  │     • 当前 - 新列表 = 需要移除                      │     │
│  │                                                     │     │
│  │  2. 执行操作:                                       │     │
│  │     移除: chattr -i + watcher.Remove()            │     │
│  │     新增: chattr +i + watcher.Add()               │     │
│  │                                                     │     │
│  │  3. 返回结果:                                       │     │
│  │     { added: [...], removed: [...], current: [...] }   │     │
│  └────────────────────────────────────────────────────┘     │
│                         ↓                                    │
│  ┌────────────────────────────────────────────────────┐     │
│  │         文件监控(fsnotify.Watcher)                  │     │
│  │                                                     │     │
│  │  监控所有受保护的目录                               │     │
│  │    ↓                                               │     │
│  │  检测到文件变动                                     │     │
│  │    ↓                                               │     │
│  │  生成 TamperEvent                                  │     │
│  │    ↓                                               │     │
│  │  通过 channel 发送                                  │     │
│  └────────────────────────────────────────────────────┘     │
│                         ↓                                    │
│  上报事件到服务端(TamperEventData)                           │
└────────────────────────┬────────────────────────────────────┘
                         │ WebSocket
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                      服务端                                  │
│                                                              │
│  接收事件 → 告警 → 日志 → 审计                              │
└─────────────────────────────────────────────────────────────┘
```

## 未来改进

- [ ] 支持递归监控子目录
- [ ] 支持白名单机制(允许特定进程修改)
- [ ] 增加事件过滤器(过滤掉不重要的事件)
- [ ] 支持更多文件系统属性(如 append-only)
- [ ] 增加性能监控指标
- [ ] 支持 Windows 平台(使用不同的实现机制)
- [ ] 支持配置持久化(Agent 重启后恢复保护状态)
- [ ] 支持按文件粒度保护(不仅仅是目录)
