# 自动更新功能设计

## 概述

为 LAN File Share 桌面应用实现基于 GitHub Release 的自动更新功能。检测到新版本后自动下载并替换当前安装，下载失败时回退为手动下载。

## 版本管理

### 编译时注入

Go 代码中定义包级变量：

```go
var version = "dev"
```

构建时通过 ldflags 注入实际版本号：

```bash
go build -ldflags "-X main.version=v1.2.3" -o build/bin/lan-file-share .
```

CI Release 工作流中从 git tag 提取版本号注入。

### 版本比较

- 去掉 `v` 前缀后按 semver 规则比较（major.minor.patch）
- `dev` 版本视为最低版本，始终提示更新
- 手写轻量 semver 比较函数，不引入外部依赖

### GitHub Release 检测

- 请求 `https://api.github.com/repos/wangjin/lan-file-share/releases/latest`
- 从 `tag_name` 提取最新版本号
- 从 `assets` 中按平台匹配下载文件：
  - macOS: `lan-file-share-macos.tar.gz`
  - Windows: `lan-file-share-windows-amd64.exe`
- 从 `body` 提取 Release Notes 用于前端展示

## 更新检测

### 检测时机

- 启动后延迟 5 秒首次检测
- 之后每 4 小时定时检测
- 提供 `CheckUpdate()` 方法供前端手动触发

### 下载流程

1. 下载到系统临时目录子目录中
2. 通过 Wails 事件实时推送下载进度（字节数、总量、速度、百分比）
3. 下载前检查磁盘剩余空间
4. 下载完成后校验文件大小（与 API 返回的 `size` 字段对比）

### 平台匹配

运行时通过 `runtime.GOOS` + `runtime.GOARCH` 确定，匹配对应 release asset 文件名。

### 错误处理

- 网络错误、下载失败时回退为打开浏览器到 Release 页面
- 所有错误通过事件推送到前端展示

## 替换与重启

### macOS 流程

1. 解压 `lan-file-share-macos.tar.gz` 到临时目录
2. 获取当前 .app bundle 路径（`os.Executable()` → 向上查找 `.app` 目录）
3. `cp -R` 将新 .app 内容覆盖到旧路径
4. 执行 `codesign --force --deep --sign - <bundlePath>` 重签名
5. `exec.Command("open", bundlePath).Start()` 后退出当前进程

### Windows 流程

1. 获取当前 exe 路径（`os.Executable()`）
2. 生成临时 bat 脚本：等待旧进程退出 → 复制新 exe → 启动新 exe → 删除 bat 自身
3. 启动 bat 脚本后立即退出当前进程

### 安全措施

- 替换前验证下载文件完整性（文件大小校验）
- 替换失败时回退为打开浏览器手动下载
- macOS 保留 codesign

## 前端 UI

### Toast 通知

- 位于右下角弹出
- 显示版本号和简短说明
- 按钮："查看详情"和"忽略"
- 点击"忽略"后记录忽略的版本号到 localStorage，同版本不再重复提示

### 更新进度弹窗

点击"查看详情"后弹出小型 Modal：

- 上方：当前版本 → 新版本号，Release Notes 摘要
- 中间：下载进度条 + 百分比 + 速度 + 剩余时间
- 底部按钮状态流转：
  - 初始："立即更新"、"手动下载"、"关闭"
  - 下载中："取消"
  - 下载完成："重启并安装"
  - 下载失败：错误信息 + "手动下载"

### 服务注册

- `UpdaterService` 注册为 Wails Service，绑定暴露方法给前端
- 前端通过 Wails 事件监听：
  - `update:available` — 检测到新版本
  - `update:progress` — 下载进度更新
  - `update:downloaded` — 下载完成
  - `update:error` — 发生错误

## 文件结构

```
internal/updater/
  updater.go       — UpdaterService 主结构，Wails Service 接口实现
  checker.go       — GitHub API 调用，版本比较
  downloader.go    — 文件下载，进度回调
  installer.go     — 平台感知的替换与重启逻辑（build tag 或内部 switch）
  updater_test.go  — 单元测试
frontend/src/components/
  UpdateToast.tsx  — 右下角 Toast 通知
  UpdateModal.tsx  — 更新进度弹窗
frontend/src/hooks/
  useUpdate.ts     — 更新相关状态管理，事件监听
```

## 构建变更

- `Taskfile.yml`：所有构建任务添加 `-ldflags` 注入版本号，版本号从 `git describe --tags --always` 获取
- `.github/workflows/release.yml`：确保 Release 流程使用注入的版本号
