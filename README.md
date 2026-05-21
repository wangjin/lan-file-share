# LAN File Share

局域网文件传输工具，基于 Wails v3 构建，支持 macOS、Windows 和 Linux。

## 功能

- **自动发现设备** — UDP 广播自动发现同一局域网内的在线设备
- **文件传输** — 点对点 TCP 直传，支持大文件分块传输
- **接收确认** — 接收方可选择保存路径后再接收
- **进度显示** — 实时显示传输进度和速率
- **MD5 校验** — 传输完成后自动校验文件完整性
- **取消传输** — 发送方和接收方均可随时取消，自动清理临时文件

## 技术栈

- **后端:** Go + Wails v3
- **前端:** React + TypeScript + Vite
- **协议:** UDP 广播发现 + TCP 文件传输，自定义二进制协议（长度前缀 + JSON 信封）

## 开发

### 前置依赖

- Go 1.25+
- Node.js 22+
- [Wails v3 CLI](https://wails.io/) (`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`)
- [Task](https://taskfile.dev/) (可选，用于 Taskfile 构建)

### 运行开发模式

```bash
wails3 dev
# 或
task dev
```

### 构建

```bash
# 当前平台
task build

# 指定平台
task build:darwin:arm64
task build:windows:amd64
task build:linux:amd64

# 全平台
task build:all
```

## 项目结构

```
├── app.go                          # 应用初始化、服务编排
├── app_transfer.go                 # 文件传输相关前端调用入口
├── app_discovery.go                # 设备发现相关前端调用入口
├── app_runtime.go                  # Wails 运行时封装（对话框、事件）
├── main.go                         # 程序入口
├── internal/
│   ├── discovery/discovery.go      # UDP 广播设备发现服务
│   ├── transfer/engine.go         # TCP 文件传输引擎
│   ├── protocol/                   # 传输协议编解码
│   ├── queue/manager.go           # 传输任务队列管理
│   └── model/                      # 数据模型
├── frontend/                       # React 前端
│   ├── bindings/                   # 自动生成的 Wails 绑定
│   └── src/
│       ├── hooks/                  # 状态管理 hooks
│       └── components/             # UI 组件
├── Taskfile.yml                    # Taskfile 多平台构建
├── BUILD.bazel / MODULE.bazel      # Bazel 构建配置
└── .github/workflows/build.yml    # CI 构建
```
