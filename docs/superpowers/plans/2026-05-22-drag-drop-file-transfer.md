# 拖拽文件传输功能 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 选中在线设备后，拖拽文件/文件夹到右侧区域实现文件传输，文件夹自动 zip 打包。

**Architecture:** 后端新增 `SendPaths` 方法和 `zipDir` 工具函数处理批量路径；前端使用 HTML5 Drag and Drop API 获取文件路径，通过 Wails 绑定调用后端。

**Tech Stack:** Go (后端)、React/TypeScript (前端)、Wails v3 (绑定自动生成)

---

### Task 1: 后端 zipDir 工具函数

**Files:**
- Create: `app_zip.go`
- Create: `app_zip_test.go`

- [ ] **Step 1: 编写 zipDir 测试**

```go
// app_zip_test.go
package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestZipDir(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("hello"), 0644)
	sub := filepath.Join(srcDir, "sub")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "b.txt"), []byte("world"), 0644)

	zipPath, err := zipDir(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(zipPath)

	if filepath.Ext(zipPath) != ".zip" {
		t.Fatalf("expected .zip extension, got %s", zipPath)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	if !names["a.txt"] {
		t.Error("missing a.txt in zip")
	}
	if !names["sub/b.txt"] {
		t.Error("missing sub/b.txt in zip")
	}
}

func TestZipDirOverwrites(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("v1"), 0644)

	zipPath1, _ := zipDir(srcDir)

	os.WriteFile(filepath.Join(srcDir, "f.txt"), []byte("v2"), 0644)
	zipPath2, _ := zipDir(srcDir)

	if zipPath1 != zipPath2 {
		t.Fatalf("zip paths should be identical: %s vs %s", zipPath1, zipPath2)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /Users/wangjin/GolandProjects/local-file-share && go test -run TestZipDir -v .`
Expected: FAIL — `zipDir` 未定义

- [ ] **Step 3: 实现 zipDir**

```go
// app_zip.go
package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func zipDir(srcDir string) (string, error) {
	zipPath := filepath.Join(os.TempDir(), filepath.Base(srcDir)+".zip")
	outFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(relPath))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		return "", err
	}
	return zipPath, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /Users/wangjin/GolandProjects/local-file-share && go test -run TestZipDir -v .`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add app_zip.go app_zip_test.go
git commit -m "feat: add zipDir utility for folder compression"
```

---

### Task 2: 后端 SendPaths 方法

**Files:**
- Modify: `app_transfer.go`

- [ ] **Step 1: 在 `app_transfer.go` 末尾添加 SendPaths 方法**

在 `app_transfer.go` 文件末尾（`GetTasks` 方法之后）添加：

```go
func (a *App) SendPaths(peerID string, paths []string) error {
	if a.engine == nil || a.discovery == nil {
		return fmt.Errorf("service not initialized")
	}

	devices := a.discovery.GetDevices()
	var peer *discovery.DeviceEntry
	for _, d := range devices {
		if d.NodeID == peerID {
			peer = d
			break
		}
	}
	if peer == nil {
		return fmt.Errorf("device not found: %s", peerID)
	}

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}

		filePath := p
		if info.IsDir() {
			zp, err := zipDir(p)
			if err != nil {
				continue
			}
			filePath = zp
		}

		task := a.engine.CreateSendTask(filePath, peer.NodeID, peer.Name, peer.IP, peer.Port)
		if task == nil {
			continue
		}
		a.queue.Add(task)

		go func(taskID string) {
			if err := a.engine.SendFile(taskID); err != nil {
				fmt.Fprintf(os.Stderr, "send failed: %v\n", err)
			}
		}(task.ID)
	}

	return nil
}
```

注意：`app_transfer.go` 已有 `"fmt"` 和 `"os"` 导入，无需修改 import。

- [ ] **Step 2: 验证编译**

Run: `cd /Users/wangjin/GolandProjects/local-file-share && go build ./...`
Expected: 编译成功

- [ ] **Step 3: 提交**

```bash
git add app_transfer.go
git commit -m "feat: add SendPaths method for batch file/folder sending"
```

---

### Task 3: 重新生成 Wails 绑定

**Files:**
- Modify: `frontend/bindings/lan-file-share/app.ts` (自动生成)

- [ ] **Step 1: 生成绑定**

Run: `cd /Users/wangjin/GolandProjects/local-file-share && wails3 generate bindings`
Expected: `frontend/bindings/lan-file-share/app.ts` 中新增 `SendPaths` 函数

- [ ] **Step 2: 确认绑定文件包含 SendPaths**

Run: `grep -n "SendPaths" frontend/bindings/lan-file-share/app.ts`
Expected: 显示 `SendPaths` 函数声明

- [ ] **Step 3: 提交**

```bash
git add frontend/bindings/
git commit -m "chore: regenerate Wails bindings for SendPaths"
```

---

### Task 4: 前端 useDragDrop hook

**Files:**
- Create: `frontend/src/hooks/useDragDrop.ts`

- [ ] **Step 1: 创建 useDragDrop hook**

```typescript
// frontend/src/hooks/useDragDrop.ts
import { useState, useCallback } from 'react';

export function useDragDrop(onDrop: (paths: string[]) => void, enabled: boolean) {
  const [isDragging, setIsDragging] = useState(false);
  let dragCounter = 0;

  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!e.dataTransfer.types.includes('Files')) return;
    dragCounter++;
    setIsDragging(true);
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounter--;
    if (dragCounter === 0) {
      setIsDragging(false);
    }
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounter = 0;
    setIsDragging(false);

    if (!enabled) return;

    const paths: string[] = [];
    const files = e.dataTransfer.files;
    for (let i = 0; i < files.length; i++) {
      const path = (files[i] as any).path as string | undefined;
      if (path) {
        paths.push(path);
      }
    }
    if (paths.length > 0) {
      onDrop(paths);
    }
  }, [enabled, onDrop]);

  return {
    isDragging,
    handlers: {
      onDragEnter: handleDragEnter,
      onDragOver: handleDragOver,
      onDragLeave: handleDragLeave,
      onDrop: handleDrop,
    },
  };
}
```

- [ ] **Step 2: 确认 TypeScript 编译无误**

Run: `cd /Users/wangjin/GolandProjects/local-file-share/frontend && npx tsc --noEmit --skipLibCheck`
Expected: 无错误

- [ ] **Step 3: 提交**

```bash
git add frontend/src/hooks/useDragDrop.ts
git commit -m "feat: add useDragDrop hook for file drag-and-drop"
```

---

### Task 5: TransferPanel 拖放区域 + CSS

**Files:**
- Modify: `frontend/src/components/TransferPanel.tsx`
- Modify: `frontend/src/App.css`

- [ ] **Step 1: 修改 TransferPanel 组件**

将 `frontend/src/components/TransferPanel.tsx` 替换为：

```tsx
import React from 'react';
import { TransferTask, taskStateName } from '../hooks/useTransfers';
import { TransferItem } from './TransferItem';

interface Props {
  tasks: TransferTask[];
  peerId: string | null;
  deviceName: string | undefined;
  onCancel: (id: string) => void;
  onRespond: (id: string, accept: boolean) => void;
  isDragging: boolean;
  dropHandlers: {
    onDragEnter: (e: React.DragEvent) => void;
    onDragOver: (e: React.DragEvent) => void;
    onDragLeave: (e: React.DragEvent) => void;
    onDrop: (e: React.DragEvent) => void;
  };
}

export const TransferPanel: React.FC<Props> = ({
  tasks, peerId, deviceName, onCancel, onRespond, isDragging, dropHandlers,
}) => {
  const filtered = peerId ? tasks.filter(t => t.peer_id === peerId) : tasks;
  const active = filtered.filter(t => taskStateName(t.state) === 'transferring');
  const waiting = filtered.filter(t => taskStateName(t.state) === 'pending');
  const done = filtered.filter(t => ['completed', 'failed', 'cancelled'].includes(taskStateName(t.state)));

  return (
    <div className="transfer-panel" {...dropHandlers}>
      {active.length > 0 && (
        <div className="section">
          <div className="section-title">传输中 ({active.length})</div>
          {active.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} onRespond={onRespond} />)}
        </div>
      )}
      {waiting.length > 0 && (
        <div className="section">
          <div className="section-title">等待中 ({waiting.length})</div>
          {waiting.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} onRespond={onRespond} />)}
        </div>
      )}
      {done.length > 0 && (
        <div className="section">
          <div className="section-title">已完成 ({done.length})</div>
          {done.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} onRespond={onRespond} />)}
        </div>
      )}
      {!active.length && !waiting.length && !done.length && !isDragging && (
        <div className="empty">暂无传输任务</div>
      )}
      {isDragging && (
        <div className={`dropzone-overlay ${peerId ? 'active' : 'disabled'}`}>
          {peerId ? (
            <span>释放以发送到 {deviceName}</span>
          ) : (
            <span>请先选择一个设备</span>
          )}
        </div>
      )}
    </div>
  );
};
```

- [ ] **Step 2: 在 App.css 末尾添加拖放样式**

在 `frontend/src/App.css` 文件末尾追加：

```css
/* ---------- Drop Zone Overlay ---------- */

.transfer-panel {
  position: relative;
}

.dropzone-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  z-index: 10;
  pointer-events: none;
  animation: dropzone-fade-in 0.15s ease-out;
}

.dropzone-overlay.active {
  background: rgba(91, 141, 239, 0.08);
  border: 2px dashed #5b8def;
  color: #5b8def;
}

.dropzone-overlay.disabled {
  background: rgba(139, 148, 158, 0.06);
  border: 2px dashed #30363d;
  color: #8b949e;
}

@keyframes dropzone-fade-in {
  from { opacity: 0; }
  to   { opacity: 1; }
}
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/components/TransferPanel.tsx frontend/src/App.css
git commit -m "feat: add drop zone overlay to TransferPanel"
```

---

### Task 6: App.tsx 集成

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: 修改 App.tsx，集成 useDragDrop 和 SendPaths**

将 `frontend/src/App.tsx` 替换为：

```tsx
import React, { useState } from 'react';
import { useDevices } from './hooks/useDevices';
import { useTransfers } from './hooks/useTransfers';
import { useDragDrop } from './hooks/useDragDrop';
import { SendPaths } from '../bindings/lan-file-share/app';
import { Sidebar } from './components/Sidebar';
import { TopBar } from './components/TopBar';
import { TransferPanel } from './components/TransferPanel';
import './App.css';

function App() {
  const { devices, localInfo } = useDevices();
  const { tasks, sendFile, respondReceive, cancelTask } = useTransfers();
  const [selectedPeerId, setSelectedPeerId] = useState<string | null>(null);
  const selectedDevice = devices.find(d => d.node_id === selectedPeerId);

  const handleDrop = async (paths: string[]) => {
    if (!selectedPeerId) return;
    await SendPaths(selectedPeerId, paths);
  };

  const { isDragging, handlers: dropHandlers } = useDragDrop(handleDrop, !!selectedPeerId);

  return (
    <div className="app">
      <Sidebar
        devices={devices}
        localInfo={localInfo}
        selectedPeerId={selectedPeerId}
        onSelectDevice={setSelectedPeerId}
      />
      <div className="main">
        <TopBar
          device={selectedDevice}
          onSendFile={() => selectedPeerId && sendFile(selectedPeerId)}
        />
        <TransferPanel
          tasks={tasks}
          peerId={selectedPeerId}
          deviceName={selectedDevice?.name}
          onCancel={cancelTask}
          onRespond={respondReceive}
          isDragging={isDragging}
          dropHandlers={dropHandlers}
        />
      </div>
    </div>
  );
}

export default App;
```

- [ ] **Step 2: 确认 TypeScript 编译无误**

Run: `cd /Users/wangjin/GolandProjects/local-file-share/frontend && npx tsc --noEmit --skipLibCheck`
Expected: 无错误

- [ ] **Step 3: 提交**

```bash
git add frontend/src/App.tsx
git commit -m "feat: integrate drag-drop file transfer in App"
```

---

### Task 7: 构建验证

- [ ] **Step 1: 完整构建**

Run: `cd /Users/wangjin/GolandProjects/local-file-share && go build ./...`
Expected: 编译成功

- [ ] **Step 2: 运行所有 Go 测试**

Run: `cd /Users/wangjin/GolandProjects/local-file-share && go test ./...`
Expected: 全部 PASS

- [ ] **Step 3: 启动应用手动验证**

Run: `cd /Users/wangjin/GolandProjects/local-file-share && wails3 dev`

验证项：
1. 选中左侧设备后，拖拽文件到右侧区域 → 应创建传输任务
2. 拖拽文件夹到右侧 → 应创建 zip 文件传输任务
3. 未选中设备时拖入 → 应显示"请先选择一个设备"
4. 拖入时显示蓝色虚线覆盖层提示
