# 局域网文件传输工具 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个基于 Go + Wails 的跨平台局域网文件传输工具，支持设备自动发现、传输全生命周期控制、队列管理和进度展示。

**Architecture:** 纯 P2P 架构，每个节点对等运行。UDP 广播发现设备，自研 TCP 协议（Length-Prefixed JSON + 分块传输）传输文件，本地队列管理并发控制。Wails 绑定桥接 Go 后端和 React 前端。

**Tech Stack:** Go 1.26 / Wails v2 / React + TypeScript / Vite / 自研 TCP 协议 / UDP 广播

---

## File Structure

```
local-file-share/
├── main.go                              # Wails 入口
├── app.go                               # Wails App 结构体，绑定方法
├── app_discovery.go                     # Discovery 相关绑定
├── app_transfer.go                      # Transfer 相关绑定
├── internal/
│   ├── model/
│   │   ├── device.go                    # Device 类型定义
│   │   └── transfer.go                  # TransferTask、TransferState 等类型
│   ├── discovery/
│   │   ├── discovery.go                 # Discovery 服务（广播/监听/设备追踪）
│   │   └── discovery_test.go
│   ├── protocol/
│   │   ├── message.go                   # 传输协议消息类型定义
│   │   ├── message_test.go
│   │   ├── codec.go                     # Length-Prefixed 编解码
│   │   └── codec_test.go
│   ├── transfer/
│   │   ├── state.go                     # 传输状态机
│   │   ├── state_test.go
│   │   ├── engine.go                    # 传输引擎（服务端 + 发送端）
│   │   └── engine_test.go
│   └── queue/
│       ├── manager.go                   # 队列管理器
│       └── manager_test.go
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── App.css
│       ├── components/
│       │   ├── Sidebar.tsx              # 左侧边栏（本机信息+设备列表）
│       │   ├── DeviceItem.tsx           # 设备列表项
│       │   ├── TopBar.tsx               # 右侧顶栏（设备信息+发送按钮）
│       │   ├── TransferPanel.tsx        # 传输列表面板
│       │   ├── TransferItem.tsx         # 单个传输项
│       │   └── ReceiveDialog.tsx        # 接收确认弹窗
│       └── hooks/
│           ├── useDevices.ts            # 设备列表 hook
│           └── useTransfers.ts          # 传输列表 hook
├── wails.json
├── go.mod
└── go.sum
```

---

### Task 1: Wails 项目脚手架

**Files:**
- Create: `main.go`, `app.go`, `wails.json`, `frontend/` 整个目录
- Create: `go.mod`, `go.sum`

- [ ] **Step 1: 用 Wails CLI 生成项目**

```bash
cd /Users/wangjin/GolandProjects
wails init -n local-file-share -t react-ts -d local-file-share
```

注意：由于项目目录已存在（有 .git 和 docs），需要先备份再初始化：

```bash
cd /Users/wangjin/GolandProjects/local-file-share
cp -r docs /tmp/lfs-docs-backup
cp .gitignore /tmp/lfs-gitignore-backup
rm -rf .git
cd /Users/wangjin/GolandProjects
rm -rf local-file-share
wails init -n local-file-share -t react-ts
cd local-file-share
git init
cp /tmp/lfs-gitignore-backup .gitignore
cp -r /tmp/lfs-docs-backup docs
rm -rf /tmp/lfs-docs-backup /tmp/lfs-gitignore-backup
echo ".superpowers/" >> .gitignore
git add .
git commit -m "chore: initialize Wails project with React-TS template"
```

- [ ] **Step 2: 验证项目构建**

```bash
cd /Users/wangjin/GolandProjects/local-file-share
wails build
```

Expected: 构建成功，在 `build/bin/` 下生成可执行文件。

---

### Task 2: 共享类型定义（internal/model）

**Files:**
- Create: `internal/model/device.go`
- Create: `internal/model/transfer.go`

- [ ] **Step 1: 编写 device 类型测试**

Create `internal/model/device_test.go`:

```go
package model

import (
	"os"
	"testing"
)

func TestDeviceCreation(t *testing.T) {
	d := Device{
		NodeID:  "test-id",
		Name:    "TestDevice",
		IP:      "192.168.1.100",
		Port:    19877,
		OS:      "darwin",
		Online:  true,
	}
	if d.NodeID != "test-id" {
		t.Errorf("expected NodeID test-id, got %s", d.NodeID)
	}
	if d.Port != 19877 {
		t.Errorf("expected Port 19877, got %d", d.Port)
	}
}

func TestDeviceDisplayOS(t *testing.T) {
	tests := []struct {
		os      string
		display string
	}{
		{"darwin", "macOS"},
		{"windows", "Windows"},
		{"linux", "Linux"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		d := Device{OS: tt.os}
		if got := d.DisplayOS(); got != tt.display {
			t.Errorf("DisplayOS(%s) = %s, want %s", tt.os, got, tt.display)
		}
	}
}

func TestDefaultSaveDir(t *testing.T) {
	dir := DefaultSaveDir()
	if dir == "" {
		t.Error("DefaultSaveDir should not be empty")
	}
	switch runtime := os.Getenv("GOOS"); runtime {
	case "darwin", "linux":
		if dir != filepath.Join(os.Getenv("HOME"), "Downloads") {
			t.Errorf("unexpected save dir: %s", dir)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /Users/wangjin/GolandProjects/local-file-share
go test ./internal/model/...
```

Expected: 编译失败，Device 类型未定义。

- [ ] **Step 3: 实现 device 类型**

Create `internal/model/device.go`:

```go
package model

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Device struct {
	NodeID  string `json:"node_id"`
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	OS      string `json:"os"`
	Online  bool   `json:"-"`
	LastSeen int64 `json:"-"`
}

func (d Device) DisplayOS() string {
	switch d.OS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return d.OS
	}
}

func DefaultSaveDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	default:
		return filepath.Join(os.Getenv("HOME"), "Downloads")
	}
}

func GetHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "Unknown"
	}
	return name
}

func GetOSName() string {
	return runtime.GOOS
}

func IsPrivateIP(ip string) bool {
	return len(ip) > 0 && (
		len(ip) >= 3 && ip[:3] == "10." ||
		len(ip) >= 8 && ip[:8] == "192.168." ||
		len(ip) >= 4 && ip[:4] == "172." && len(ip) >= 7)
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/model/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/model/
git commit -m "feat: add device model with OS display and platform helpers"
```

---

### Task 3: 传输任务类型定义（internal/model）

**Files:**
- Create: `internal/model/transfer.go`
- Create: `internal/model/transfer_test.go`

- [ ] **Step 1: 编写 transfer 类型测试**

Create `internal/model/transfer_test.go`:

```go
package model

import (
	"testing"
	"time"
)

func TestTransferStateString(t *testing.T) {
	tests := []struct {
		state TransferState
		str   string
	}{
		{StatePending, "pending"},
		{StateTransferring, "transferring"},
		{StatePaused, "paused"},
		{StateCompleted, "completed"},
		{StateFailed, "failed"},
		{StateCancelled, "cancelled"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.str {
			t.Errorf("TransferState(%d).String() = %s, want %s", tt.state, got, tt.str)
		}
	}
}

func TestTransferTypeString(t *testing.T) {
	if got := TypeSend.String(); got != "send" {
		t.Errorf("TypeSend.String() = %s, want send", got)
	}
	if got := TypeReceive.String(); got != "receive" {
		t.Errorf("TypeReceive.String() = %s, want receive", got)
	}
}

func TestTransferTaskIsTerminal(t *testing.T) {
	terminal := []TransferState{StateCompleted, StateFailed, StateCancelled}
	for _, s := range terminal {
		task := TransferTask{State: s}
		if !task.IsTerminal() {
			t.Errorf("state %s should be terminal", s)
		}
	}

	nonTerminal := []TransferState{StatePending, StateTransferring, StatePaused}
	for _, s := range nonTerminal {
		task := TransferTask{State: s}
		if task.IsTerminal() {
			t.Errorf("state %s should not be terminal", s)
		}
	}
}

func TestTransferTaskProgress(t *testing.T) {
	task := TransferTask{FileSize: 1000, BytesTransferred: 500}
	if got := task.Progress(); got != 0.5 {
		t.Errorf("Progress() = %f, want 0.5", got)
	}

	zeroTask := TransferTask{FileSize: 0}
	if got := zeroTask.Progress(); got != 0.0 {
		t.Errorf("Progress() with 0 size = %f, want 0.0", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/model/ -v -run TestTransfer
```

Expected: 编译失败，类型未定义。

- [ ] **Step 3: 实现 transfer 类型**

Create `internal/model/transfer.go`:

```go
package model

import "time"

type TransferState int

const (
	StatePending TransferState = iota
	StateTransferring
	StatePaused
	StateCompleted
	StateFailed
	StateCancelled
)

func (s TransferState) String() string {
	names := []string{"pending", "transferring", "paused", "completed", "failed", "cancelled"}
	if int(s) < len(names) {
		return names[s]
	}
	return "unknown"
}

type TransferType int

const (
	TypeSend TransferType = iota
	TypeReceive
)

func (t TransferType) String() string {
	if t == TypeSend {
		return "send"
	}
	return "receive"
}

type TransferTask struct {
	ID               string        `json:"id"`
	Type             TransferType  `json:"type"`
	State            TransferState `json:"state"`
	FileName         string        `json:"file_name"`
	FileSize         int64         `json:"file_size"`
	FilePath         string        `json:"-"`
	SavePath         string        `json:"-"`
	PeerID           string        `json:"peer_id"`
	PeerName         string        `json:"peer_name"`
	PeerIP           string        `json:"-"`
	PeerPort         int           `json:"-"`
	BytesTransferred int64         `json:"bytes_transferred"`
	Speed            int64         `json:"speed"`
	FileMD5          string        `json:"file_md5"`
	ChunksTotal      int           `json:"chunks_total"`
	ChunksCompleted  int           `json:"chunks_completed"`
	CreatedAt        time.Time     `json:"created_at"`
	CompletedAt      *time.Time    `json:"completed_at"`
}

func (t *TransferTask) IsTerminal() bool {
	return t.State == StateCompleted || t.State == StateFailed || t.State == StateCancelled
}

func (t *TransferTask) Progress() float64 {
	if t.FileSize == 0 {
		return 0.0
	}
	return float64(t.BytesTransferred) / float64(t.FileSize)
}

const ChunkSize = 1024 * 1024 // 1MB

func CalcChunks(fileSize int64) int {
	chunks := fileSize / ChunkSize
	if fileSize%ChunkSize != 0 {
		chunks++
	}
	return int(chunks)
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/model/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/model/
git commit -m "feat: add transfer task types with state machine and progress"
```

---

### Task 4: 传输协议编解码（internal/protocol）

**Files:**
- Create: `internal/protocol/codec.go`
- Create: `internal/protocol/codec_test.go`

- [ ] **Step 1: 编写编解码测试**

Create `internal/protocol/codec_test.go`:

```go
package protocol

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeMessage(t *testing.T) {
	msg := &TransferRequest{
		FileName: "test.zip",
		FileSize: 1024000,
		FileMD5:  "abc123",
		Chunks:   1,
	}

	var buf bytes.Buffer
	err := EncodeMessage(&buf, msg)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	decoded, err := DecodeMessage(&buf)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	req, ok := decoded.(*TransferRequest)
	if !ok {
		t.Fatalf("expected *TransferRequest, got %T", decoded)
	}
	if req.FileName != "test.zip" {
		t.Errorf("FileName = %s, want test.zip", req.FileName)
	}
	if req.FileSize != 1024000 {
		t.Errorf("FileSize = %d, want 1024000", req.FileSize)
	}
	if req.FileMD5 != "abc123" {
		t.Errorf("FileMD5 = %s, want abc123", req.FileMD5)
	}
}

func TestDecodeEmptyBuffer(t *testing.T) {
	var buf bytes.Buffer
	_, err := DecodeMessage(&buf)
	if err == nil {
		t.Error("expected error for empty buffer")
	}
}

func TestEncodeDecodeAllMessageTypes(t *testing.T) {
	tests := []Message{
		&TransferRequest{FileName: "a.txt", FileSize: 100, FileMD5: "md5", Chunks: 1},
		&TransferResponse{Accepted: true},
		&TransferResponse{Accepted: false, Reason: "busy"},
		&ChunkData{Sequence: 5, Size: 1024},
		&ProgressAck{BytesReceived: 5000, State: "transferring"},
		&TransferComplete{MD5: "final_md5"},
		&TransferVerify{Success: true},
		&TransferCancel{Reason: "user cancelled"},
	}

	for _, msg := range tests {
		var buf bytes.Buffer
		if err := EncodeMessage(&buf, msg); err != nil {
			t.Fatalf("EncodeMessage(%T) failed: %v", msg, err)
		}
		decoded, err := DecodeMessage(&buf)
		if err != nil {
			t.Fatalf("DecodeMessage(%T) failed: %v", msg, err)
		}
		if msg.Type() != decoded.Type() {
			t.Errorf("type mismatch: encoded=%s decoded=%s", msg.Type(), decoded.Type())
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/protocol/ -v
```

Expected: 编译失败。

- [ ] **Step 3: 实现消息类型**

Create `internal/protocol/message.go`:

```go
package protocol

const (
	TypeTransferRequest  = "transfer_request"
	TypeTransferResponse = "transfer_response"
	TypeChunkData        = "chunk_data"
	TypeProgressAck      = "progress_ack"
	TypeTransferComplete = "transfer_complete"
	TypeTransferVerify   = "transfer_verify"
	TypeTransferCancel   = "transfer_cancel"
)

type Message interface {
	Type() string
}

type TransferRequest struct {
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	FileMD5  string `json:"file_md5"`
	Chunks   int    `json:"chunks"`
}

func (m *TransferRequest) Type() string { return TypeTransferRequest }

type TransferResponse struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

func (m *TransferResponse) Type() string { return TypeTransferResponse }

type ChunkData struct {
	Sequence int   `json:"sequence"`
	Size     int64 `json:"size"`
}

func (m *ChunkData) Type() string { return TypeChunkData }

type ProgressAck struct {
	BytesReceived int64  `json:"bytes_received"`
	State         string `json:"state"`
}

func (m *ProgressAck) Type() string { return TypeProgressAck }

type TransferComplete struct {
	MD5 string `json:"md5"`
}

func (m *TransferComplete) Type() string { return TypeTransferComplete }

type TransferVerify struct {
	Success bool `json:"success"`
}

func (m *TransferVerify) Type() string { return TypeTransferVerify }

type TransferCancel struct {
	Reason string `json:"reason"`
}

func (m *TransferCancel) Type() string { return TypeTransferCancel }
```

- [ ] **Step 4: 实现编解码器**

Create `internal/protocol/codec.go`:

```go
package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func EncodeMessage(w io.Writer, msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	env := Envelope{
		Type:    msg.Type(),
		Payload: payload,
	}

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	length := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return fmt.Errorf("write length: %w", err)
	}

	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	return nil
}

func DecodeMessage(r io.Reader) (Message, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	var env Envelope
	if err := json.Unmarshal(buf, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	return decodePayload(env.Type, env.Payload)
}

func decodePayload(msgType string, payload json.RawMessage) (Message, error) {
	var msg Message
	switch msgType {
	case TypeTransferRequest:
		msg = &TransferRequest{}
	case TypeTransferResponse:
		msg = &TransferResponse{}
	case TypeChunkData:
		msg = &ChunkData{}
	case TypeProgressAck:
		msg = &ProgressAck{}
	case TypeTransferComplete:
		msg = &TransferComplete{}
	case TypeTransferVerify:
		msg = &TransferVerify{}
	case TypeTransferCancel:
		msg = &TransferCancel{}
	default:
		return nil, fmt.Errorf("unknown message type: %s", msgType)
	}

	if err := json.Unmarshal(payload, msg); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", msgType, err)
	}

	return msg, nil
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
go test ./internal/protocol/ -v
```

Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/protocol/
git commit -m "feat: add transfer protocol messages and length-prefixed codec"
```

---

### Task 5: 传输状态机（internal/transfer）

**Files:**
- Create: `internal/transfer/state.go`
- Create: `internal/transfer/state_test.go`

- [ ] **Step 1: 编写状态机测试**

Create `internal/transfer/state_test.go`:

```go
package transfer

import (
	"testing"

	"local-file-share/internal/model"
)

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		from model.TransferState
		to   model.TransferState
	}{
		{model.StatePending, model.StateTransferring},
		{model.StatePending, model.StateCancelled},
		{model.StateTransferring, model.StatePaused},
		{model.StateTransferring, model.StateCompleted},
		{model.StateTransferring, model.StateFailed},
		{model.StateTransferring, model.StateCancelled},
		{model.StatePaused, model.StateTransferring},
		{model.StatePaused, model.StateCancelled},
	}
	for _, tt := range tests {
		sm := NewStateMachine(tt.from)
		err := sm.Transition(tt.to)
		if err != nil {
			t.Errorf("Transition(%s → %s) should succeed, got error: %v", tt.from, tt.to, err)
		}
		if sm.Current() != tt.to {
			t.Errorf("current state = %s, want %s", sm.Current(), tt.to)
		}
	}
}

func TestInvalidTransitions(t *testing.T) {
	tests := []struct {
		from model.TransferState
		to   model.TransferState
	}{
		{model.StateCompleted, model.StateTransferring},
		{model.StateFailed, model.StateTransferring},
		{model.StateCancelled, model.StatePaused},
		{model.StatePending, model.StateCompleted},
		{model.StatePending, model.StatePaused},
	}
	for _, tt := range tests {
		sm := NewStateMachine(tt.from)
		err := sm.Transition(tt.to)
		if err == nil {
			t.Errorf("Transition(%s → %s) should fail", tt.from, tt.to)
		}
	}
}

func TestStateCallback(t *testing.T) {
	sm := NewStateMachine(model.StatePending)
	var called model.TransferState
	sm.OnChange(func(from, to model.TransferState) {
		called = to
	})
	_ = sm.Transition(model.StateTransferring)
	if called != model.StateTransferring {
		t.Errorf("callback received %s, want %s", called, model.StateTransferring)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/transfer/ -v
```

Expected: 编译失败。

- [ ] **Step 3: 实现状态机**

Create `internal/transfer/state.go`:

```go
package transfer

import (
	"fmt"

	"local-file-share/internal/model"
)

type StateCallback func(from, to model.TransferState)

type StateMachine struct {
	current  model.TransferState
	callback StateCallback
}

var validTransitions = map[model.TransferState][]model.TransferState{
	model.StatePending:      {model.StateTransferring, model.StateCancelled},
	model.StateTransferring: {model.StatePaused, model.StateCompleted, model.StateFailed, model.StateCancelled},
	model.StatePaused:       {model.StateTransferring, model.StateCancelled},
	model.StateCompleted:    {},
	model.StateFailed:       {},
	model.StateCancelled:    {},
}

func NewStateMachine(initial model.TransferState) *StateMachine {
	return &StateMachine{current: initial}
}

func (sm *StateMachine) Current() model.TransferState {
	return sm.current
}

func (sm *StateMachine) OnChange(cb StateCallback) {
	sm.callback = cb
}

func (sm *StateMachine) Transition(to model.TransferState) error {
	allowed, ok := validTransitions[sm.current]
	if !ok {
		return fmt.Errorf("unknown state: %s", sm.current)
	}
	for _, s := range allowed {
		if s == to {
			from := sm.current
			sm.current = to
			if sm.callback != nil {
				sm.callback(from, to)
			}
			return nil
		}
	}
	return fmt.Errorf("invalid transition: %s → %s", sm.current, to)
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/transfer/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/transfer/
git commit -m "feat: add transfer state machine with validation and callbacks"
```

---

### Task 6: 发现服务（internal/discovery）

**Files:**
- Create: `internal/discovery/discovery.go`
- Create: `internal/discovery/discovery_test.go`

- [ ] **Step 1: 编写发现服务测试**

Create `internal/discovery/discovery_test.go`:

```go
package discovery

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestBroadcastMessageSerialize(t *testing.T) {
	msg := BroadcastMessage{
		NodeID:   "test-id",
		Name:     "TestDevice",
		IP:       "192.168.1.100",
		Port:     19877,
		OS:       "darwin",
		Timestamp: time.Now().Unix(),
		Leave:    false,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded BroadcastMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.NodeID != msg.NodeID {
		t.Errorf("NodeID mismatch: %s != %s", decoded.NodeID, msg.NodeID)
	}
	if decoded.Port != msg.Port {
		t.Errorf("Port mismatch: %d != %d", decoded.Port, msg.Port)
	}
}

func TestDiscoveryAddRemoveDevice(t *testing.T) {
	svc := &Service{
		devices:   make(map[string]*DeviceEntry),
		onlineTTL: 10 * time.Second,
	}

	svc.updateDevice("node-1", "Device1", "192.168.1.100", 19877, "darwin")
	if len(svc.devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(svc.devices))
	}

	d := svc.devices["node-1"]
	if d.Name != "Device1" {
		t.Errorf("Name = %s, want Device1", d.Name)
	}
	if !d.Online {
		t.Error("device should be online")
	}
}

func TestDiscoveryDeviceExpiry(t *testing.T) {
	svc := &Service{
		devices:   make(map[string]*DeviceEntry),
		onlineTTL: 100 * time.Millisecond,
	}

	svc.updateDevice("node-1", "Device1", "192.168.1.100", 19877, "darwin")
	time.Sleep(150 * time.Millisecond)
	svc.removeStale()

	if len(svc.devices) != 0 {
		t.Errorf("device should be removed after TTL, got %d", len(svc.devices))
	}
}

func TestDiscoveryLeaveMessage(t *testing.T) {
	svc := &Service{
		devices:   make(map[string]*DeviceEntry),
		onlineTTL: 10 * time.Second,
	}

	svc.updateDevice("node-1", "Device1", "192.168.1.100", 19877, "darwin")
	svc.removeDevice("node-1")

	if len(svc.devices) != 0 {
		t.Errorf("device should be removed, got %d", len(svc.devices))
	}
}

func TestGetPrivateIPs(t *testing.T) {
	ips := getPrivateIPs()
	for _, ip := range ips {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			t.Errorf("invalid IP: %s", ip)
		}
		if !parsed.IsPrivate() {
			t.Errorf("IP %s is not private", ip)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/discovery/ -v
```

Expected: 编译失败。

- [ ] **Step 3: 实现发现服务**

Create `internal/discovery/discovery.go`:

```go
package discovery

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultPort     = 19876
	MaxPortAttempts = 5
	BroadcastInterval = 3 * time.Second
	OnlineTTL        = 10 * time.Second
)

type BroadcastMessage struct {
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	OS       string `json:"os"`
	Timestamp int64 `json:"timestamp"`
	Leave    bool   `json:"leave,omitempty"`
}

type DeviceEntry struct {
	NodeID   string
	Name     string
	IP       string
	Port     int
	OS       string
	Online   bool
	LastSeen time.Time
}

type DeviceChangeCallback func(device *DeviceEntry, online bool)

type Service struct {
	nodeID    string
	nodeName  string
	tcpPort   int
	udpPort   int
	devices   map[string]*DeviceEntry
	mu        sync.RWMutex
	onlineTTL time.Duration
	callback  DeviceChangeCallback
	stopCh    chan struct{}
}

func NewService(nodeName string, tcpPort int) *Service {
	return &Service{
		nodeID:    uuid.New().String(),
		nodeName:  nodeName,
		tcpPort:   tcpPort,
		devices:   make(map[string]*DeviceEntry),
		onlineTTL: OnlineTTL,
		stopCh:    make(chan struct{}),
	}
}

func (s *Service) SetCallback(cb DeviceChangeCallback) {
	s.callback = cb
}

func (s *Service) NodeID() string {
	return s.nodeID
}

func (s *Service) Start() error {
	port := DefaultPort
	var conn *net.UDPConn
	var err error
	for i := 0; i < MaxPortAttempts; i++ {
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
		conn, err = net.ListenUDP("udp", addr)
		if err == nil {
			s.udpPort = port
			break
		}
		port++
	}
	if err != nil {
		return fmt.Errorf("no available UDP port in range %d-%d: %w", DefaultPort, port-1, err)
	}

	go s.listen(conn)
	go s.broadcastLoop()
	go s.cleanupLoop()

	return nil
}

func (s *Service) Stop() {
	close(s.stopCh)
	s.sendLeave()
}

func (s *Service) GetDevices() []*DeviceEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DeviceEntry, 0, len(s.devices))
	for _, d := range s.devices {
		result = append(result, d)
	}
	return result
}

func (s *Service) listen(conn *net.UDPConn) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-s.stopCh:
			conn.Close()
			return
		default:
		}
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}
		s.handleMessage(buf[:n])
	}
}

func (s *Service) handleMessage(data []byte) {
	var msg BroadcastMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	if msg.NodeID == s.nodeID {
		return
	}
	if msg.Leave {
		s.removeDevice(msg.NodeID)
		return
	}
	s.updateDevice(msg.NodeID, msg.Name, msg.IP, msg.Port, msg.OS)
}

func (s *Service) updateDevice(nodeID, name, ip string, port int, osName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	isNew := false
	d, ok := s.devices[nodeID]
	if !ok {
		d = &DeviceEntry{NodeID: nodeID}
		s.devices[nodeID] = d
		isNew = true
	}
	d.Name = name
	d.IP = ip
	d.Port = port
	d.OS = osName
	d.Online = true
	d.LastSeen = time.Now()

	if s.callback != nil {
		go s.callback(d, isNew)
	}
}

func (s *Service) removeDevice(nodeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[nodeID]
	if !ok {
		return
	}
	delete(s.devices, nodeID)
	d.Online = false
	if s.callback != nil {
		go s.callback(d, false)
	}
}

func (s *Service) removeStale() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, d := range s.devices {
		if now.Sub(d.LastSeen) > s.onlineTTL {
			d.Online = false
			delete(s.devices, id)
			if s.callback != nil {
				go s.callback(d, false)
			}
		}
	}
}

func (s *Service) broadcastLoop() {
	ticker := time.NewTicker(BroadcastInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.broadcast()
		}
	}
}

func (s *Service) broadcast() {
	ips := getPrivateIPs()
	if len(ips) == 0 {
		return
	}

	msg := BroadcastMessage{
		NodeID:    s.nodeID,
		Name:      s.nodeName,
		Port:      s.tcpPort,
		OS:        runtime.GOOS,
		Timestamp: time.Now().Unix(),
	}

	for _, localIP := range ips {
		msg.IP = localIP
		data, _ := json.Marshal(msg)
		broadcastIP := getBroadcastIP(localIP)
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastIP, s.udpPort))
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			continue
		}
		conn.Write(data)
		conn.Close()
	}
}

func (s *Service) sendLeave() {
	msg := BroadcastMessage{
		NodeID: s.nodeID,
		Leave:  true,
	}
	ips := getPrivateIPs()
	for _, localIP := range ips {
		data, _ := json.Marshal(msg)
		broadcastIP := getBroadcastIP(localIP)
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastIP, s.udpPort))
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			continue
		}
		conn.Write(data)
		conn.Close()
	}
}

func (s *Service) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.removeStale()
		}
	}
}

func getPrivateIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	var result []string
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.IsPrivate() {
			if ipNet.IP.To4() != nil {
				result = append(result, ipNet.IP.String())
			}
		}
	}
	return result
}

func getBroadcastIP(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "255.255.255.255"
	}
	ip4 := parsed.To4()
	if ip4 == nil {
		return "255.255.255.255"
	}
	return fmt.Sprintf("%d.%d.%d.255", ip4[0], ip4[1], ip4[2])
}
```

- [ ] **Step 4: 安装 uuid 依赖并运行测试**

```bash
cd /Users/wangjin/GolandProjects/local-file-share
go get github.com/google/uuid
go test ./internal/discovery/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/discovery/ go.mod go.sum
git commit -m "feat: add UDP broadcast discovery service with device tracking"
```

---

### Task 7: 传输引擎（internal/transfer/engine）

**Files:**
- Create: `internal/transfer/engine.go`
- Create: `internal/transfer/engine_test.go`

- [ ] **Step 1: 编写传输引擎测试**

Create `internal/transfer/engine_test.go`:

```go
package transfer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalcFileMD5(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	md5, err := calcFileMD5(tmpFile)
	if err != nil {
		t.Fatalf("calcFileMD5 failed: %v", err)
	}
	if md5 == "" {
		t.Error("md5 should not be empty")
	}
}

func TestResolveSavePath(t *testing.T) {
	dir := t.TempDir()

	path := resolveSavePath(dir, "test.txt")
	expected := filepath.Join(dir, "test.txt")
	if path != expected {
		t.Errorf("resolveSavePath = %s, want %s", path, expected)
	}

	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("x"), 0644)
	path = resolveSavePath(dir, "test.txt")
	expected = filepath.Join(dir, "test(1).txt")
	if path != expected {
		t.Errorf("resolveSavePath conflict = %s, want %s", path, expected)
	}
}

func TestEngineCreatesTask(t *testing.T) {
	eng := NewEngine("node-1", "TestDevice")
	tmpFile := filepath.Join(t.TempDir(), "send.bin")
	os.WriteFile(tmpFile, make([]byte, 100), 0644)

	task := eng.CreateSendTask(tmpFile, "node-2", "Peer", "192.168.1.100", 19877)
	if task == nil {
		t.Fatal("CreateSendTask returned nil")
	}
	if task.FileName != "send.bin" {
		t.Errorf("FileName = %s, want send.bin", task.FileName)
	}
	if task.PeerID != "node-2" {
		t.Errorf("PeerID = %s, want node-2", task.PeerID)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/transfer/ -v
```

Expected: 编译失败。

- [ ] **Step 3: 实现传输引擎**

Create `internal/transfer/engine.go`:

```go
package transfer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"local-file-share/internal/model"
	"local-file-share/internal/protocol"

	"github.com/google/uuid"
)

type ProgressCallback func(task *model.TransferTask)

type Engine struct {
	localNodeID   string
	localNodeName string
	tasks         map[string]*model.TransferTask
	taskMutex     sync.RWMutex
	tcpListener   net.Listener
	tcpPort       int
	progressCb    ProgressCallback
	receiveCb     func(task *model.TransferTask) bool
	stopCh        chan struct{}
}

func NewEngine(localNodeID, localNodeName string) *Engine {
	return &Engine{
		localNodeID:   localNodeID,
		localNodeName: localNodeName,
		tasks:         make(map[string]*model.TransferTask),
		stopCh:        make(chan struct{}),
	}
}

func (e *Engine) SetProgressCallback(cb ProgressCallback) {
	e.progressCb = cb
}

func (e *Engine) SetReceiveCallback(cb func(task *model.TransferTask) bool) {
	e.receiveCb = cb
}

func (e *Engine) Start() error {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("listen tcp: %w", err)
	}
	e.tcpListener = listener
	e.tcpPort = listener.Addr().(*net.TCPAddr).Port
	go e.acceptConnections()
	return nil
}

func (e *Engine) TCPPort() int {
	return e.tcpPort
}

func (e *Engine) Stop() {
	close(e.stopCh)
	if e.tcpListener != nil {
		e.tcpListener.Close()
	}
}

func (e *Engine) GetTasks() []*model.TransferTask {
	e.taskMutex.RLock()
	defer e.taskMutex.RUnlock()
	result := make([]*model.TransferTask, 0, len(e.tasks))
	for _, t := range e.tasks {
		result = append(result, t)
	}
	return result
}

func (e *Engine) CreateSendTask(filePath, peerID, peerName, peerIP string, peerPort int) *model.TransferTask {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil
	}

	md5Hash, _ := calcFileMD5(filePath)

	task := &model.TransferTask{
		ID:          uuid.New().String(),
		Type:        model.TypeSend,
		State:       model.StatePending,
		FileName:    filepath.Base(filePath),
		FileSize:    info.Size(),
		FilePath:    filePath,
		PeerID:      peerID,
		PeerName:    peerName,
		PeerIP:      peerIP,
		PeerPort:    peerPort,
		FileMD5:     md5Hash,
		ChunksTotal: model.CalcChunks(info.Size()),
		CreatedAt:   time.Now(),
	}

	e.taskMutex.Lock()
	e.tasks[task.ID] = task
	e.taskMutex.Unlock()

	return task
}

func (e *Engine) SendFile(taskID string) error {
	e.taskMutex.RLock()
	task, ok := e.tasks[taskID]
	e.taskMutex.RUnlock()
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", task.PeerIP, task.PeerPort), 10*time.Second)
	if err != nil {
		e.updateState(task, model.StateFailed)
		return fmt.Errorf("connect to peer: %w", err)
	}
	defer conn.Close()

	if err := e.sendTransferRequest(conn, task); err != nil {
		e.updateState(task, model.StateFailed)
		return err
	}

	resp, err := protocol.DecodeMessage(conn)
	if err != nil {
		e.updateState(task, model.StateFailed)
		return fmt.Errorf("read response: %w", err)
	}

	response, ok := resp.(*protocol.TransferResponse)
	if !ok || !response.Accepted {
		e.updateState(task, model.StateFailed)
		return fmt.Errorf("transfer rejected: %s", response.Reason)
	}

	return e.sendFileData(conn, task)
}

func (e *Engine) CancelTask(taskID string) error {
	e.taskMutex.Lock()
	defer e.taskMutex.Unlock()
	task, ok := e.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.IsTerminal() {
		return fmt.Errorf("cannot cancel task in state %s", task.State)
	}
	task.State = model.StateCancelled
	now := time.Now()
	task.CompletedAt = &now
	e.notifyProgress(task)
	return nil
}

func (e *Engine) acceptConnections() {
	for {
		select {
		case <-e.stopCh:
			return
		default:
		}
		conn, err := e.tcpListener.Accept()
		if err != nil {
			return
		}
		go e.handleConnection(conn)
	}
}

func (e *Engine) handleConnection(conn net.Conn) {
	defer conn.Close()

	msg, err := protocol.DecodeMessage(conn)
	if err != nil {
		return
	}

	req, ok := msg.(*protocol.TransferRequest)
	if !ok {
		return
	}

	task := &model.TransferTask{
		ID:          uuid.New().String(),
		Type:        model.TypeReceive,
		State:       model.StatePending,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		FileMD5:     req.FileMD5,
		ChunksTotal: req.Chunks,
		CreatedAt:   time.Now(),
	}

	e.taskMutex.Lock()
	e.tasks[task.ID] = task
	e.taskMutex.Unlock()

	accepted := false
	if e.receiveCb != nil {
		accepted = e.receiveCb(task)
	}

	resp := &protocol.TransferResponse{Accepted: accepted}
	protocol.EncodeMessage(conn, resp)

	if !accepted {
		e.updateState(task, model.StateFailed)
		return
	}

	e.receiveFileData(conn, task)
}

func (e *Engine) sendTransferRequest(conn net.Conn, task *model.TransferTask) error {
	req := &protocol.TransferRequest{
		FileName: task.FileName,
		FileSize: task.FileSize,
		FileMD5:  task.FileMD5,
		Chunks:   task.ChunksTotal,
	}
	return protocol.EncodeMessage(conn, req)
}

func (e *Engine) sendFileData(conn net.Conn, task *model.TransferTask) error {
	e.updateState(task, model.StateTransferring)

	file, err := os.Open(task.FilePath)
	if err != nil {
		e.updateState(task, model.StateFailed)
		return err
	}
	defer file.Close()

	buf := make([]byte, model.ChunkSize)
	for chunk := 0; chunk < task.ChunksTotal; chunk++ {
		select {
		case <-e.stopCh:
			e.updateState(task, model.StateCancelled)
			return nil
		default:
		}

		if task.State == model.StateCancelled {
			protocol.EncodeMessage(conn, &protocol.TransferCancel{Reason: "cancelled"})
			return nil
		}

		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			e.updateState(task, model.StateFailed)
			return err
		}
		if n == 0 {
			break
		}

		chunkMsg := &protocol.ChunkData{
			Sequence: chunk,
			Size:     int64(n),
		}
		if err := protocol.EncodeMessage(conn, chunkMsg); err != nil {
			e.updateState(task, model.StateFailed)
			return err
		}
		if _, err := conn.Write(buf[:n]); err != nil {
			e.updateState(task, model.StateFailed)
			return err
		}

		task.BytesTransferred += int64(n)
		task.ChunksCompleted = chunk + 1
		e.notifyProgress(task)
	}

	completeMsg := &protocol.TransferComplete{MD5: task.FileMD5}
	if err := protocol.EncodeMessage(conn, completeMsg); err != nil {
		e.updateState(task, model.StateFailed)
		return err
	}

	verifyMsg, err := protocol.DecodeMessage(conn)
	if err != nil {
		e.updateState(task, model.StateFailed)
		return err
	}
	if verify, ok := verifyMsg.(*protocol.TransferVerify); ok && verify.Success {
		e.updateState(task, model.StateCompleted)
	} else {
		e.updateState(task, model.StateFailed)
	}
	return nil
}

func (e *Engine) receiveFileData(conn net.Conn, task *model.TransferTask) {
	e.updateState(task, model.StateTransferring)

	saveDir := model.DefaultSaveDir()
	savePath := resolveSavePath(saveDir, task.FileName)
	task.SavePath = savePath

	tmpPath := savePath + ".tmp"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		e.updateState(task, model.StateFailed)
		return
	}
	defer outFile.Close()

	for {
		msg, err := protocol.DecodeMessage(conn)
		if err != nil {
			e.updateState(task, model.StateFailed)
			return
		}

		switch m := msg.(type) {
		case *protocol.ChunkData:
			data := make([]byte, m.Size)
			if _, err := io.ReadFull(conn, data); err != nil {
				e.updateState(task, model.StateFailed)
				return
			}
			if _, err := outFile.Write(data); err != nil {
				e.updateState(task, model.StateFailed)
				return
			}
			task.BytesTransferred += m.Size
			task.ChunksCompleted = m.Sequence + 1
			e.notifyProgress(task)

		case *protocol.TransferComplete:
			outFile.Close()
			receivedMD5, _ := calcFileMD5(tmpPath)
			success := receivedMD5 == m.MD5
			if success {
				os.Rename(tmpPath, savePath)
			}
			protocol.EncodeMessage(conn, &protocol.TransferVerify{Success: success})
			if success {
				e.updateState(task, model.StateCompleted)
			} else {
				e.updateState(task, model.StateFailed)
			}
			return

		case *protocol.TransferCancel:
			outFile.Close()
			os.Remove(tmpPath)
			e.updateState(task, model.StateCancelled)
			return
		}
	}
}

func (e *Engine) updateState(task *model.TransferTask, state model.TransferState) {
	e.taskMutex.Lock()
	task.State = state
	if state == model.StateCompleted || state == model.StateFailed || state == model.StateCancelled {
		now := time.Now()
		task.CompletedAt = &now
	}
	e.taskMutex.Unlock()
	e.notifyProgress(task)
}

func (e *Engine) notifyProgress(task *model.TransferTask) {
	if e.progressCb != nil {
		go e.progressCb(task)
	}
}

func calcFileMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func resolveSavePath(dir, filename string) string {
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	for i := 1; ; i++ {
		ext := filepath.Ext(filename)
		name := filename[:len(filename)-len(ext)]
		path = filepath.Join(dir, fmt.Sprintf("%s(%d)%s", name, i, ext))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/transfer/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/transfer/
git commit -m "feat: add transfer engine with TCP send/receive and MD5 verification"
```

---

### Task 8: 队列管理器（internal/queue）

**Files:**
- Create: `internal/queue/manager.go`
- Create: `internal/queue/manager_test.go`

- [ ] **Step 1: 编写队列管理器测试**

Create `internal/queue/manager_test.go`:

```go
package queue

import (
	"testing"

	"local-file-share/internal/model"
)

func TestAddTask(t *testing.T) {
	q := NewManager(2)
	task := &model.TransferTask{ID: "t1", State: model.StatePending}
	q.Add(task)

	tasks := q.GetAll()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "t1" {
		t.Errorf("task ID = %s, want t1", tasks[0].ID)
	}
}

func TestMaxConcurrent(t *testing.T) {
	q := NewManager(2)

	t1 := &model.TransferTask{ID: "t1", State: model.StatePending}
	t2 := &model.TransferTask{ID: "t2", State: model.StatePending}
	t3 := &model.TransferTask{ID: "t3", State: model.StatePending}

	q.Add(t1)
	q.Add(t2)
	q.Add(t3)

	active := q.ActiveCount()
	if active != 2 {
		t.Errorf("ActiveCount = %d, want 2", active)
	}

	waiting := q.WaitingCount()
	if waiting != 1 {
		t.Errorf("WaitingCount = %d, want 1", waiting)
	}
}

func TestCompleteFreesSlot(t *testing.T) {
	q := NewManager(1)

	t1 := &model.TransferTask{ID: "t1", State: model.StatePending}
	t2 := &model.TransferTask{ID: "t2", State: model.StatePending}
	q.Add(t1)
	q.Add(t2)

	if q.ActiveCount() != 1 {
		t.Errorf("ActiveCount = %d, want 1", q.ActiveCount())
	}

	q.UpdateState("t1", model.StateCompleted)

	if q.ActiveCount() != 1 {
		t.Errorf("ActiveCount after complete = %d, want 1 (t2 auto-started)", q.ActiveCount())
	}

	if q.WaitingCount() != 0 {
		t.Errorf("WaitingCount after complete = %d, want 0", q.WaitingCount())
	}
}

func TestCancelTask(t *testing.T) {
	q := NewManager(2)
	t1 := &model.TransferTask{ID: "t1", State: model.StatePending}
	q.Add(t1)

	err := q.Cancel("t1")
	if err != nil {
		t.Errorf("Cancel failed: %v", err)
	}

	task := q.Get("t1")
	if task.State != model.StateCancelled {
		t.Errorf("state = %s, want cancelled", task.State)
	}
}

func TestReorderTask(t *testing.T) {
	q := NewManager(1)

	t1 := &model.TransferTask{ID: "t1", State: model.StateTransferring}
	t2 := &model.TransferTask{ID: "t2", State: model.StatePending}
	t3 := &model.TransferTask{ID: "t3", State: model.StatePending}
	q.Add(t1)
	q.Add(t2)
	q.Add(t3)

	q.Reorder("t3", 0)

	waiting := q.GetWaiting()
	if waiting[0].ID != "t3" {
		t.Errorf("first waiting task = %s, want t3", waiting[0].ID)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/queue/ -v
```

Expected: 编译失败。

- [ ] **Step 3: 实现队列管理器**

Create `internal/queue/manager.go`:

```go
package queue

import (
	"fmt"
	"sync"

	"local-file-share/internal/model"
)

type StateChangeCallback func(task *model.TransferTask)

type Manager struct {
	maxConcurrent int
	tasks         []*model.TransferTask
	mu            sync.RWMutex
	callback      StateChangeCallback
}

func NewManager(maxConcurrent int) *Manager {
	return &Manager{
		maxConcurrent: maxConcurrent,
		tasks:         make([]*model.TransferTask, 0),
	}
}

func (m *Manager) SetCallback(cb StateChangeCallback) {
	m.callback = cb
}

func (m *Manager) Add(task *model.TransferTask) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = append(m.tasks, task)
	m.processQueue()
}

func (m *Manager) Get(id string) *model.TransferTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (m *Manager) GetAll() []*model.TransferTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*model.TransferTask, len(m.tasks))
	copy(result, m.tasks)
	return result
}

func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, t := range m.tasks {
		if t.State == model.StateTransferring {
			count++
		}
	}
	return count
}

func (m *Manager) WaitingCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, t := range m.tasks {
		if t.State == model.StatePending && !t.IsTerminal() {
			count++
		}
	}
	return count
}

func (m *Manager) GetWaiting() []*model.TransferTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*model.TransferTask
	for _, t := range m.tasks {
		if t.State == model.StatePending {
			result = append(result, t)
		}
	}
	return result
}

func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		if t.ID == id {
			if t.IsTerminal() {
				return fmt.Errorf("cannot cancel task in state %s", t.State)
			}
			t.State = model.StateCancelled
			m.notify(t)
			m.processQueue()
			return nil
		}
	}
	return fmt.Errorf("task not found: %s", id)
}

func (m *Manager) UpdateState(id string, state model.TransferState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		if t.ID == id {
			t.State = state
			m.notify(t)
			break
		}
	}
	m.processQueue()
}

func (m *Manager) Reorder(id string, newPosition int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var waiting []*model.TransferTask
	var target *model.TransferTask
	for _, t := range m.tasks {
		if t.State == model.StatePending {
			if t.ID == id {
				target = t
			} else {
				waiting = append(waiting, t)
			}
		}
	}
	if target == nil {
		return
	}

	if newPosition < 0 {
		newPosition = 0
	}
	if newPosition > len(waiting) {
		newPosition = len(waiting)
	}

	waiting = append(waiting[:newPosition], append([]*model.TransferTask{target}, waiting[newPosition:]...)...)

	idx := 0
	for _, t := range m.tasks {
		if t.State == model.StatePending {
			t.ID = waiting[idx].ID
			idx++
		}
	}
}

func (m *Manager) UpdateProgress(id string, bytesTransferred int64, speed int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tasks {
		if t.ID == id {
			t.BytesTransferred = bytesTransferred
			t.Speed = speed
			m.notify(t)
			return
		}
	}
}

func (m *Manager) processQueue() {
	active := 0
	for _, t := range m.tasks {
		if t.State == model.StateTransferring {
			active++
		}
	}
	if active >= m.maxConcurrent {
		return
	}
	for _, t := range m.tasks {
		if t.State == model.StatePending {
			m.notify(t)
			return
		}
	}
}

func (m *Manager) notify(task *model.TransferTask) {
	if m.callback != nil {
		go m.callback(task)
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/queue/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/queue/
git commit -m "feat: add transfer queue manager with concurrency control"
```

---

### Task 9: Wails 绑定层（app.go）

**Files:**
- Create: `app_discovery.go`
- Create: `app_transfer.go`
- Modify: `app.go`

- [ ] **Step 1: 重写 app.go 作为主绑定结构**

Replace `app.go` with:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"local-file-share/internal/discovery"
	"local-file-share/internal/model"
	"local-file-share/internal/queue"
	"local-file-share/internal/transfer"
)

type App struct {
	ctx       context.Context
	discovery *discovery.Service
	engine    *transfer.Engine
	queue     *queue.Manager
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	nodeName := model.GetHostname()

	svc := discovery.NewService(nodeName, 0)
	if err := svc.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "discovery start failed: %v\n", err)
		return
	}
	a.discovery = svc

	eng := transfer.NewEngine(svc.NodeID(), nodeName)
	eng.SetProgressCallback(func(task *model.TransferTask) {
		a.queue.UpdateProgress(task.ID, task.BytesTransferred, task.Speed)
	})
	if err := eng.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "engine start failed: %v\n", err)
		return
	}
	a.engine = eng

	svc.SetCallback(func(device *discovery.DeviceEntry, online bool) {
		a.EventsEmit(a.ctx, "device:changed", map[string]interface{}{
			"node_id": device.NodeID,
			"name":    device.Name,
			"ip":      device.IP,
			"port":    device.Port,
			"os":      device.OS,
			"online":  online,
		})
	})

	q := queue.NewManager(2)
	q.SetCallback(func(task *model.TransferTask) {
		a.EventsEmit(a.ctx, "task:changed", task)
	})
	a.queue = q
}

func (a *App) Shutdown(ctx context.Context) {
	if a.discovery != nil {
		a.discovery.Stop()
	}
	if a.engine != nil {
		a.engine.Stop()
	}
}
```

- [ ] **Step 2: 创建 discovery 绑定**

Create `app_discovery.go`:

```go
package main

import "local-file-share/internal/model"

func (a *App) GetDevices() []map[string]interface{} {
	if a.discovery == nil {
		return nil
	}
	devices := a.discovery.GetDevices()
	result := make([]map[string]interface{}, 0, len(devices))
	for _, d := range devices {
		result = append(result, map[string]interface{}{
			"node_id": d.NodeID,
			"name":    d.Name,
			"ip":      d.IP,
			"port":    d.Port,
			"os":      d.OS,
			"online":  d.Online,
		})
	}
	return result
}

func (a *App) GetLocalInfo() map[string]interface{} {
	if a.discovery == nil {
		return nil
	}
	return map[string]interface{}{
		"node_id": a.discovery.NodeID(),
		"name":    model.GetHostname(),
		"os":      model.GetOSName(),
	}
}

func (a *App) SetNodeName(name string) {
	if a.discovery == nil {
		return
	}
	// Discovery service broadcasts name each interval
}
```

- [ ] **Step 3: 创建 transfer 绑定**

Create `app_transfer.go`:

```go
package main

import (
	"local-file-share/internal/model"
)

func (a *App) SelectAndSend(peerID string) error {
	if a.engine == nil || a.discovery == nil {
		return nil
	}

	filePath, err := a.runtime.FileDialogOpen(a.ctx, map[string]string{})
	if err != nil || filePath == "" {
		return nil
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
		return nil
	}

	task := a.engine.CreateSendTask(filePath, peer.NodeID, peer.Name, peer.IP, peer.Port)
	if task == nil {
		return nil
	}
	a.queue.Add(task)

	go func() {
		if err := a.engine.SendFile(task.ID); err != nil {
			fmt.Fprintf(os.Stderr, "send failed: %v\n", err)
		}
	}()

	return nil
}

func (a *App) CancelTask(taskID string) error {
	return a.queue.Cancel(taskID)
}

func (a *App) GetTasks() []*model.TransferTask {
	return a.queue.GetAll()
}

func (a *App) AcceptReceive(taskID string) {
	a.engine.SetReceiveCallback(func(task *model.TransferTask) bool {
		return true
	})
}
```

注意：`app_transfer.go` 中的 `SelectAndSend` 使用 Wails runtime 的文件对话框。`AcceptReceive` 的回调逻辑需要在前端 ReceiveDialog 中配合完成。实际实现时需要通过 Wails 事件机制实现异步确认：收到请求 → 发送事件到前端 → 用户确认 → 调用 AcceptReceive。

- [ ] **Step 4: 运行构建验证编译**

```bash
cd /Users/wangjin/GolandProjects/local-file-share
go build ./...
```

Expected: 编译通过（可能有未使用 import 警告，后续 Task 10 修正）。

- [ ] **Step 5: 修正 main.go**

确保 `main.go` 正确引用 App：

```go
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Local File Share",
		Width:  1024,
		Height: 680,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
```

- [ ] **Step 6: 提交**

```bash
git add app.go app_discovery.go app_transfer.go main.go
git commit -m "feat: add Wails bindings for discovery, transfer, and queue"
```

---

### Task 10: 前端基础框架与布局

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.css`
- Create: `frontend/src/hooks/useDevices.ts`
- Create: `frontend/src/hooks/useTransfers.ts`
- Create: `frontend/src/components/Sidebar.tsx`
- Create: `frontend/src/components/DeviceItem.tsx`
- Create: `frontend/src/components/TopBar.tsx`
- Create: `frontend/src/components/TransferPanel.tsx`
- Create: `frontend/src/components/TransferItem.tsx`
- Create: `frontend/src/components/ReceiveDialog.tsx`

此 Task 的前端代码在实现时使用 frontend-design 技能进行美化，以下为功能骨架和组件接口。

- [ ] **Step 1: 创建 hooks**

Create `frontend/src/hooks/useDevices.ts`:

```typescript
import { useState, useEffect } from 'react';

export interface Device {
  node_id: string;
  name: string;
  ip: string;
  port: number;
  os: string;
  online: boolean;
}

export function useDevices() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [localInfo, setLocalInfo] = useState<{ node_id: string; name: string; os: string } | null>(null);

  useEffect(() => {
    window['go']['main']['App']['GetLocalInfo']().then(setLocalInfo);
    window['go']['main']['App']['GetDevices']().then(setDevices);

    const unlisten = window['runtime']['EventsOn']('device:changed', (_: any, data: Device) => {
      setDevices(prev => {
        const idx = prev.findIndex(d => d.node_id === data.node_id);
        if (data.online) {
          if (idx >= 0) {
            const next = [...prev];
            next[idx] = data;
            return next;
          }
          return [...prev, data];
        }
        return prev.filter(d => d.node_id !== data.node_id);
      });
    });

    return () => { unlisten.then(fn => fn()); };
  }, []);

  return { devices, localInfo };
}
```

Create `frontend/src/hooks/useTransfers.ts`:

```typescript
import { useState, useEffect } from 'react';

export interface TransferTask {
  id: string;
  type: 'send' | 'receive';
  state: 'pending' | 'transferring' | 'paused' | 'completed' | 'failed' | 'cancelled';
  file_name: string;
  file_size: number;
  peer_id: string;
  peer_name: string;
  bytes_transferred: number;
  speed: number;
  created_at: string;
  completed_at?: string;
}

export function useTransfers() {
  const [tasks, setTasks] = useState<TransferTask[]>([]);

  useEffect(() => {
    window['go']['main']['App']['GetTasks']().then(setTasks);

    const unlisten = window['runtime']['EventsOn']('task:changed', (_: any, task: TransferTask) => {
      setTasks(prev => {
        const idx = prev.findIndex(t => t.id === task.id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = task;
          return next;
        }
        return [...prev, task];
      });
    });

    return () => { unlisten.then(fn => fn()); };
  }, []);

  const sendFile = async (peerId: string) => {
    await window['go']['main']['App']['SelectAndSend'](peerId);
  };

  const cancelTask = async (taskId: string) => {
    await window['go']['main']['App']['CancelTask'](taskId);
  };

  return { tasks, sendFile, cancelTask };
}
```

- [ ] **Step 2: 创建组件骨架**

Create `frontend/src/components/Sidebar.tsx`:

```tsx
import React from 'react';
import { Device } from '../hooks/useDevices';
import { DeviceItem } from './DeviceItem';

interface Props {
  devices: Device[];
  localInfo: { node_id: string; name: string; os: string } | null;
  selectedPeerId: string | null;
  onSelectDevice: (nodeId: string) => void;
}

export const Sidebar: React.FC<Props> = ({ devices, localInfo, selectedPeerId, onSelectDevice }) => (
  <div className="sidebar">
    <div className="local-info">
      <div className="avatar">{localInfo?.name?.[0] || '?'}</div>
      <div className="info">
        <div className="name">{localInfo?.name}</div>
        <div className="status">在线</div>
      </div>
    </div>
    <div className="device-section">
      <div className="section-title">在线设备 ({devices.length})</div>
      {devices.map(d => (
        <DeviceItem
          key={d.node_id}
          device={d}
          selected={d.node_id === selectedPeerId}
          onClick={() => onSelectDevice(d.node_id)}
        />
      ))}
    </div>
  </div>
);
```

Create `frontend/src/components/DeviceItem.tsx`:

```tsx
import React from 'react';
import { Device } from '../hooks/useDevices';

interface Props {
  device: Device;
  selected: boolean;
  onClick: () => void;
}

const osIcons: Record<string, string> = { darwin: '🍎', windows: '🪟', linux: '🐧' };

export const DeviceItem: React.FC<Props> = ({ device, selected, onClick }) => (
  <div className={`device-item ${selected ? 'selected' : ''}`} onClick={onClick}>
    <div className="device-avatar">{device.name[0]}</div>
    <div className="device-info">
      <div className="device-name">{device.name}</div>
      <div className="device-meta">{osIcons[device.os] || '💻'} {device.os} · {device.ip}</div>
    </div>
  </div>
);
```

Create `frontend/src/components/TopBar.tsx`:

```tsx
import React from 'react';
import { Device } from '../hooks/useDevices';

interface Props {
  device: Device | undefined;
  onSendFile: () => void;
  onSendFolder: () => void;
}

export const TopBar: React.FC<Props> = ({ device, onSendFile, onSendFolder }) => {
  if (!device) return <div className="topbar">选择一个设备开始传输</div>;
  return (
    <div className="topbar">
      <div className="peer-info">
        <span className="peer-name">{device.name}</span>
        <span className="peer-ip">{device.ip}</span>
      </div>
      <div className="actions">
        <button className="btn-primary" onClick={onSendFile}>发送文件</button>
        <button className="btn-secondary" onClick={onSendFolder}>发送文件夹</button>
      </div>
    </div>
  );
};
```

Create `frontend/src/components/TransferItem.tsx`:

```tsx
import React from 'react';
import { TransferTask } from '../hooks/useTransfers';

interface Props {
  task: TransferTask;
  onCancel: (id: string) => void;
}

export const TransferItem: React.FC<Props> = ({ task, onCancel }) => {
  const progress = task.file_size > 0
    ? (task.bytes_transferred / task.file_size * 100).toFixed(1)
    : 0;
  const speedMB = (task.speed / 1024 / 1024).toFixed(1);
  const sizeMB = (task.file_size / 1024 / 1024).toFixed(1);
  const transferredMB = (task.bytes_transferred / 1024 / 1024).toFixed(1);

  return (
    <div className={`transfer-item state-${task.state}`}>
      <div className="transfer-header">
        <span className="direction">{task.type === 'send' ? '↑' : '↓'}</span>
        <span className="filename">{task.file_name}</span>
        {(task.state === 'transferring' || task.state === 'pending') && (
          <button className="btn-icon btn-danger" onClick={() => onCancel(task.id)}>✕</button>
        )}
      </div>
      <div className="transfer-meta">
        {task.state === 'transferring' && `${transferredMB} MB / ${sizeMB} MB · ${speedMB} MB/s`}
        {task.state === 'pending' && `${sizeMB} MB · 等待传输...`}
        {task.state === 'completed' && `${sizeMB} MB · 完成`}
        {task.state === 'failed' && `${sizeMB} MB · 失败`}
        {task.state === 'cancelled' && `${sizeMB} MB · 已取消`}
      </div>
      {task.state === 'transferring' && (
        <div className="progress-bar">
          <div className="progress-fill" style={{ width: `${progress}%` }} />
        </div>
      )}
    </div>
  );
};
```

Create `frontend/src/components/TransferPanel.tsx`:

```tsx
import React from 'react';
import { TransferTask } from '../hooks/useTransfers';
import { TransferItem } from './TransferItem';

interface Props {
  tasks: TransferTask[];
  peerId: string | null;
  onCancel: (id: string) => void;
}

export const TransferPanel: React.FC<Props> = ({ tasks, peerId, onCancel }) => {
  const filtered = peerId ? tasks.filter(t => t.peer_id === peerId) : tasks;
  const active = filtered.filter(t => t.state === 'transferring');
  const waiting = filtered.filter(t => t.state === 'pending');
  const done = filtered.filter(t => ['completed', 'failed', 'cancelled'].includes(t.state));

  return (
    <div className="transfer-panel">
      {active.length > 0 && (
        <div className="section">
          <div className="section-title">传输中 ({active.length})</div>
          {active.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} />)}
        </div>
      )}
      {waiting.length > 0 && (
        <div className="section">
          <div className="section-title">等待中 ({waiting.length})</div>
          {waiting.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} />)}
        </div>
      )}
      {done.length > 0 && (
        <div className="section">
          <div className="section-title">已完成 ({done.length})</div>
          {done.map(t => <TransferItem key={t.id} task={t} onCancel={onCancel} />)}
        </div>
      )}
      {!active.length && !waiting.length && !done.length && (
        <div className="empty">暂无传输任务</div>
      )}
    </div>
  );
};
```

Create `frontend/src/components/ReceiveDialog.tsx`:

```tsx
import React from 'react';
import { TransferTask } from '../hooks/useTransfers';

interface Props {
  task: TransferTask | null;
  onAccept: () => void;
  onReject: () => void;
}

const sizeMB = (bytes: number) => (bytes / 1024 / 1024).toFixed(1);

export const ReceiveDialog: React.FC<Props> = ({ task, onAccept, onReject }) => {
  if (!task) return null;
  return (
    <div className="dialog-overlay">
      <div className="dialog">
        <h3>收到文件</h3>
        <div className="dialog-info">
          <p><strong>来自：</strong>{task.peer_name}</p>
          <p><strong>文件：</strong>{task.file_name}</p>
          <p><strong>大小：</strong>{sizeMB(task.file_size)} MB</p>
        </div>
        <div className="dialog-actions">
          <button className="btn-primary" onClick={onAccept}>接收</button>
          <button className="btn-secondary" onClick={onReject}>拒绝</button>
        </div>
      </div>
    </div>
  );
};
```

- [ ] **Step 3: 编写 App.tsx 主入口**

Replace `frontend/src/App.tsx`:

```tsx
import React, { useState } from 'react';
import { useDevices } from './hooks/useDevices';
import { useTransfers } from './hooks/useTransfers';
import { Sidebar } from './components/Sidebar';
import { TopBar } from './components/TopBar';
import { TransferPanel } from './components/TransferPanel';
import { ReceiveDialog } from './components/ReceiveDialog';
import './App.css';

function App() {
  const { devices, localInfo } = useDevices();
  const { tasks, sendFile, cancelTask } = useTransfers();
  const [selectedPeerId, setSelectedPeerId] = useState<string | null>(null);
  const [pendingReceive, setPendingReceive] = useState<TransferTask | null>(null);

  const selectedDevice = devices.find(d => d.node_id === selectedPeerId);

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
          onSendFolder={() => selectedPeerId && sendFile(selectedPeerId)}
        />
        <TransferPanel
          tasks={tasks}
          peerId={selectedPeerId}
          onCancel={cancelTask}
        />
      </div>
      <ReceiveDialog
        task={pendingReceive}
        onAccept={() => {
          if (pendingReceive) {
            window['go']['main']['App']['AcceptReceive'](pendingReceive.id);
          }
          setPendingReceive(null);
        }}
        onReject={() => setPendingReceive(null)}
      />
    </div>
  );
}

export default App;
```

- [ ] **Step 4: 前端构建验证**

```bash
cd /Users/wangjin/GolandProjects/local-file-share/frontend
npm install
npm run build
```

Expected: 构建成功。

- [ ] **Step 5: 提交**

```bash
git add frontend/src/ app.go app_discovery.go app_transfer.go main.go
git commit -m "feat: add React frontend components and Wails binding integration"
```

---

### Task 11: 前端 UI 美化

**Files:**
- Modify: `frontend/src/App.css`

此 Task 使用 **frontend-design 技能** 进行美化，将骨架 CSS 替换为高设计品质的深色主题样式。

- [ ] **Step 1: 调用 frontend-design 技能美化 CSS**

使用 frontend-design 技能，基于设计文档中的线框图设计，为所有组件编写深色主题样式。关键要求：
- 深色背景（`#1a1a2e` 系列）
- 主题色（`#e94560` 作为强调色）
- 进度条渐变（`#e94560` → `#5b8def`）
- 圆角卡片式传输项
- 平滑动画过渡
- 拖拽区域高亮反馈

- [ ] **Step 2: 验证构建**

```bash
cd /Users/wangjin/GolandProjects/local-file-share/frontend
npm run build
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/App.css
git commit -m "style: apply polished dark theme UI via frontend-design skill"
```

---

### Task 12: 端到端构建与集成测试

**Files:**
- Modify: 可能修正编译错误

- [ ] **Step 1: 完整构建**

```bash
cd /Users/wangjin/GolandProjects/local-file-share
wails build
```

Expected: 构建成功，在 `build/bin/` 下生成可执行文件。

- [ ] **Step 2: 运行所有 Go 测试**

```bash
go test ./internal/... -v
```

Expected: 所有测试 PASS。

- [ ] **Step 3: 手动集成验证**

```bash
# 启动应用
wails dev
```

验证项：
1. 应用启动后能看到本机信息
2. 第二个实例启动后能看到对方设备
3. 选择文件发送后进度条正常显示
4. 暂停/取消功能正常
5. MD5 校验通过后显示完成

- [ ] **Step 4: 最终提交**

```bash
git add .
git commit -m "chore: integration build and test verification"
```

---

## Self-Review Checklist

**Spec coverage:**
- [x] UDP 广播发现 — Task 6
- [x] 广播消息格式 — Task 6
- [x] 在线判定（10秒TTL） — Task 6
- [x] 多网卡处理 — Task 6
- [x] Length-Prefixed JSON 协议 — Task 4
- [x] 分块传输（1MB） — Task 7
- [x] 流量控制（ProgressAck） — Task 7
- [x] 暂停/恢复/取消 — Task 7
- [x] MD5 校验 — Task 7
- [x] 状态机 — Task 5
- [x] 队列管理（最大并发 2） — Task 8
- [x] 优先级调整 — Task 8
- [x] 接收弹窗确认 — Task 10
- [x] 拖拽发送 — Task 11 (frontend-design)
- [x] 深色主题 — Task 11
- [x] 系统通知 — Task 11
- [x] 跨平台保存目录 — Task 2
- [x] 文件名冲突自动重命名 — Task 7
- [x] 错误处理（网络/文件异常） — Task 7

**Placeholder scan:** 无 TBD/TODO 占位符。

**Type consistency:** 检查所有方法名和类型引用，确保前后一致。
