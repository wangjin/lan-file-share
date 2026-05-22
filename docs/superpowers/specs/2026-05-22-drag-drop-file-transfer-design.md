# 拖拽文件传输功能设计

## 目标

选中左侧在线设备后，支持将文件或文件夹拖拽到右侧主区域发起传输。

## 行为

- 拖入文件：逐个创建传输任务，直接发送
- 拖入文件夹：自动 zip 打包后作为单文件传输，接收端收到 `.zip`
- 拖入多个项目（文件+文件夹混合）：逐个处理
- 未选中设备时拖入：显示提示，不执行传输

## 后端

### 新增方法 `SendPaths(peerID string, paths []string) error`

1. 校验 peerID 对应设备存在
2. 遍历 paths，对每个路径：
   - 文件：调用 `engine.CreateSendTask` 创建任务
   - 目录：调用 `zipDir` 打包后，用 zip 路径创建任务
3. 每个任务入队并发送（与 `SelectAndSend` 逻辑一致）

### 新增工具函数 `zipDir(srcDir string) (string, error)`

- 将目录递归打包为 `os.TempDir()/{dirName}.zip`
- 若同名 zip 已存在则覆盖
- 返回 zip 文件完整路径

## 前端

### TransferPanel 增加拖放区域

- 在 `TransferPanel` 外层监听 `dragenter/dragover/dragleave/drop`
- 拖入时根据是否选中设备显示不同覆盖层：
  - 已选中：虚线边框 + "释放以发送到 {设备名}"
  - 未选中：灰色覆盖 + "请先选择一个设备"
- drop 时从 `event.dataTransfer.files` 提取 `file.path`（Wails webview 提供），调用 `SendPaths(peerID, paths)`

### 新增 hook `useDragDrop`

- 管理 `isDragging` / `canDrop` 状态
- 返回事件处理器和拖放状态

## Wails 绑定

`SendPaths` 作为公开方法自动生成 TS 绑定，前端直接调用。

## 不做的事

- 不支持接收端自动解压 zip（保持简单）
- 不做拖放进度提示（每个文件独立任务，用现有 UI 展示）
