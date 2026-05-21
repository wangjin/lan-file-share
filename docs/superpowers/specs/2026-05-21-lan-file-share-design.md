# 局域网文件传输工具 — 设计文档

## 概述

一个基于 Go + Wails 的局域网文件传输工具，支持跨平台（macOS / Windows / Linux），具备设备自动发现、文件传输全生命周期控制（暂停/恢复/取消/断点续传）、传输队列管理和进度展示。

**定位：** 通用工具 — 个人多设备互传 + 小团队协作均适用。

## 技术栈

| 层级 | 技术选型 |
|------|---------|
| GUI 框架 | Wails v3 |
| 前端 | React + TypeScript（使用 frontend-design 技能美化） |
| 后端 | Go |
| 发现协议 | UDP 广播 |
| 传输协议 | 自研 TCP（Length-Prefixed JSON） |
| 加密 | 无（局域网可信环境） |
| 目标平台 | macOS / Windows / Linux |

## 架构

### 整体架构

```
┌─────────────────────────────────────────────┐
│              Wails Application              │
├──────────────────────┬──────────────────────┤
│    Go Backend        │   React Frontend     │
│                      │                      │
│  ┌──────────────┐    │  ┌────────────────┐  │
│  │  Discovery   │    │  │  Device List   │  │
│  │  (UDP)       │    │  │  Panel         │  │
│  └──────┬───────┘    │  └────────────────┘  │
│         │            │  ┌────────────────┐  │
│  ┌──────┴───────┐    │  │  File Browser  │  │
│  │  Transfer    │    │  │  & Send Panel  │  │
│  │  Engine      │    │  └────────────────┘  │
│  │  (TCP)       │    │  ┌────────────────┐  │
│  └──────┬───────┘    │  │  Transfer      │  │
│         │            │  │  Progress      │  │
│  ┌──────┴───────┐    │  │  Panel         │  │
│  │  Queue       │    │  └────────────────┘  │
│  │  Manager     │    │  ┌────────────────┐  │
│  └──────────────┘    │  │  Settings      │  │
│                      │  └────────────────┘  │
│  Wails Bindings ─────┤                      │
└──────────────────────┴──────────────────────┘
```

### 模块职责

- **Discovery** — UDP 广播发送/接收，维护在线设备列表，设备上下线事件通知
- **Transfer Engine** — TCP 文件传输协议实现，传输状态机管理
- **Queue Manager** — 本地传输队列，控制并发数，排队策略
- **React Frontend** — 设备发现、文件选择、传输控制、进度展示

### 数据流

前端通过 Wails 绑定调用 Go 方法 → Go 方法操作对应模块 → 模块状态变更通过 Wails 事件系统推送到前端刷新 UI。

## 发现模块（Discovery）

### 协议

- 每个 UDP 端口 `19876` 监听，每 3 秒广播一次心跳
- 启动时尝试端口 19876-19880，找到可用端口即使用

### 广播消息格式

```json
{
  "node_id": "uuid",
  "name": "Jin的MacBook",
  "ip": "192.168.1.100",
  "port": 19877,
  "os": "darwin",
  "timestamp": 1706000100
}
```

- `node_id` — 启动时生成的 UUID
- `name` — 设备名称（取 hostname，用户可修改）
- `ip` — 本机局域网 IP
- `port` — TCP 文件传输监听端口
- `os` — 操作系统类型
- `timestamp` — 发送时间戳

### 在线判定

- 收到广播 → 设备上线
- 超过 10 秒未收到 → 判定离线
- 退出时发送离开消息（尽力而为）

### 多网卡处理

枚举本机网卡，筛选私有 IP（10.x / 172.16-31.x / 192.168.x），向每个网段发送广播。

## 传输引擎（Transfer Engine）

### TCP 连接模型

发送方主动连接接收方（Discovery 广播的 port）。每次传输任务建立一个 TCP 连接，传输完毕断开。

### 协议消息格式

Length-Prefixed JSON：`[4字节长度][JSON消息体][文件数据流]`

```
请求流程：
  Sender                          Receiver
    │                                │
    │── Connect ──────────────────→  │
    │── TransferRequest ──────────→  │  (文件名、大小、MD5、分块数)
    │←─ TransferResponse ─────────  │  (接受/拒绝)
    │── ChunkData × N ───────────→  │  (分块序号、分块大小、数据)
    │←─ ProgressAck (周期性) ────  │  (已接收字节、状态)
    │── TransferComplete ────────→  │
    │←─ TransferVerify ──────────  │  (MD5 校验结果)
```

### 传输状态机

```
                 ┌──────────┐
     创建传输 ──→ │  Pending  │
                 └────┬─────┘
                      │ 接受
                 ┌────▼─────┐
            ┌──→│Transfering│←──┐
            │   └────┬─────┘   │
            │  暂停   │         │ 恢复
            │   ┌─────▼────┐   │
            │   │  Paused   │───┘
            │   └──────────┘
            │        │ 取消
            │   ┌────▼─────┐
            │   │Cancelled  │
            │   └──────────┘
            │
       错误/取消
            │
       ┌────▼─────┐     ┌──────────┐
       │  Failed   │     │Completed │
       └──────────┘     └──────────┘
```

### 关键设计

- **分块传输** — 1MB 分块，支持断点续传（记录已完成分块序号）
- **流量控制** — 接收方 ProgressAck 反馈已处理字节数，发送方控制发送速率
- **暂停/恢复** — 停止发送新分块，保持 TCP 连接；恢复时从下一个未发送分块继续
- **取消** — 任一方可随时发送取消消息，双方关闭连接并清理临时文件
- **完整性校验** — 传输完成后 MD5 校验，失败可自动重试

## 队列管理（Queue Manager）

### 规则

- 最大同时传输数：2 个（可配置），超出入队等待
- 先进先出，用户可手动调整优先级

### 传输任务数据结构

```go
type TransferTask struct {
    ID           string
    Type         TransferType // Send / Receive
    State        TransferState
    FileName     string
    FileSize     int64
    FilePath     string
    PeerID       string
    PeerName     string
    Progress     float64 // 0.0 - 1.0
    Speed        int64   // bytes/s
    CompletedAt  *time.Time
    CreatedAt    time.Time
}
```

### 并发控制

- 发送前检查活跃发送数，未达上限直接开始，否则入队
- 接收时先询问用户，接受后检查活跃接收数
- 任务完成/取消/失败时自动启动队列中下一个任务

### 前端接口

- Wails 绑定：`GetQueue()` / `CancelTask(id)` / `PauseTask(id)` / `ResumeTask(id)` / `ReorderTask(id, position)`
- Wails 事件：`task:created` / `task:state_changed` / `task:progress` / `task:completed`

## 前端 UI

### 布局

左侧边栏 + 右侧内容区：
- **左侧栏** — 本机信息 + 在线设备列表（设备名、OS 类型、IP）
- **右侧内容区** — 选中设备后显示与该设备的传输列表
- **传输分区** — 传输中（进度条、速率、暂停/取消）、等待中、已完成
- **顶部操作栏** — 发送文件/文件夹按钮

### UI 特性

- 默认深色主题
- 接收文件时弹窗确认（显示文件信息和接受/拒绝按钮）
- 支持拖拽文件到设备头像发起传输
- 传输完成/收到文件时推送系统通知

实现时使用 frontend-design 技能进行 UI 美化。

## 错误处理

### 网络异常

- TCP 断开 → 传输标记 Failed，保留已接收分块，支持断点续传重试
- 目标设备离线 → 传输失败，设备从列表移除
- 端口占用 → 尝试下一端口（19876-19880），上限 5 次

### 文件异常

- 磁盘空间不足 → 接收前检查，提前拒绝
- 文件被占用 → 发送失败提示
- MD5 校验失败 → 保留临时文件供手动重试

### 并发冲突

- 同一文件同时发给同一设备 → 队列自动排队
- 接收文件名冲突 → 自动重命名（`file.txt` → `file(1).txt`）

### 平台兼容

- 接收保存位置：macOS `~/Downloads`、Windows `%USERPROFILE%\Downloads`、Linux `~/Downloads`
- 路径分隔符发送时统一 `/`，接收时转换为本机格式
