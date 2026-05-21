package transfer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"local-file-share/internal/model"
)

func TestCalcFileMD5(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world, this is a test file for MD5")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	hash, err := calcFileMD5(tmpFile)
	if err != nil {
		t.Fatalf("calcFileMD5 returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("calcFileMD5 returned empty hash")
	}

	// Verify against known MD5
	expected := md5.Sum(content)
	expectedHex := hex.EncodeToString(expected[:])
	if hash != expectedHex {
		t.Errorf("MD5 mismatch: got %s, want %s", hash, expectedHex)
	}
}

func TestResolveSavePath(t *testing.T) {
	tmpDir := t.TempDir()

	// No conflict — returns direct path
	path := resolveSavePath(tmpDir, "file.txt")
	expected := filepath.Join(tmpDir, "file.txt")
	if path != expected {
		t.Errorf("no conflict: got %s, want %s", path, expected)
	}

	// Create the file so there is a conflict
	if err := os.WriteFile(expected, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to create conflicting file: %v", err)
	}

	path = resolveSavePath(tmpDir, "file.txt")
	expected = filepath.Join(tmpDir, "file(1).txt")
	if path != expected {
		t.Errorf("with conflict: got %s, want %s", path, expected)
	}

	// Create the (1) version too
	if err := os.WriteFile(expected, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to create conflicting file: %v", err)
	}

	path = resolveSavePath(tmpDir, "file.txt")
	expected = filepath.Join(tmpDir, "file(2).txt")
	if path != expected {
		t.Errorf("with double conflict: got %s, want %s", path, expected)
	}

	// No extension
	path = resolveSavePath(tmpDir, "README")
	if path != filepath.Join(tmpDir, "README") {
		// README doesn't exist yet so should be direct
		t.Errorf("no ext, no conflict: got %s", path)
	}
}

func TestEngineCreatesTask(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "payload.bin")
	content := []byte("test payload data for send task creation")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	engine := NewEngine("node-1", "TestNode")
	task := engine.CreateSendTask(tmpFile, "peer-1", "PeerNode", "127.0.0.1", 9000)
	if task == nil {
		t.Fatal("CreateSendTask returned nil")
	}

	if task.ID == "" {
		t.Error("task ID is empty")
	}
	if task.Type != model.TypeSend {
		t.Errorf("expected TypeSend, got %v", task.Type)
	}
	if task.State != model.StatePending {
		t.Errorf("expected StatePending, got %v", task.State)
	}
	if task.FileName != "payload.bin" {
		t.Errorf("expected filename payload.bin, got %s", task.FileName)
	}
	if task.FileSize != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), task.FileSize)
	}
	if task.FilePath != tmpFile {
		t.Errorf("expected path %s, got %s", tmpFile, task.FilePath)
	}
	if task.PeerID != "peer-1" {
		t.Errorf("expected peer-1, got %s", task.PeerID)
	}
	if task.PeerName != "PeerNode" {
		t.Errorf("expected PeerNode, got %s", task.PeerName)
	}
	if task.PeerIP != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %s", task.PeerIP)
	}
	if task.PeerPort != 9000 {
		t.Errorf("expected port 9000, got %d", task.PeerPort)
	}
	if task.FileMD5 == "" {
		t.Error("FileMD5 is empty")
	}
	expectedChunks := model.CalcChunks(int64(len(content)))
	if task.ChunksTotal != expectedChunks {
		t.Errorf("expected %d chunks, got %d", expectedChunks, task.ChunksTotal)
	}

	// Task should be stored in engine
	tasks := engine.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != task.ID {
		t.Error("stored task ID mismatch")
	}

	// Non-existent file returns nil
	badTask := engine.CreateSendTask("/nonexistent/path/file.txt", "peer-1", "Peer", "127.0.0.1", 9000)
	if badTask != nil {
		t.Error("expected nil for nonexistent file")
	}
}

func TestEndToEndTransfer(t *testing.T) {
	// Create engine (acts as both sender and receiver)
	engine := NewEngine("node-1", "TestNode")

	// Auto-accept all incoming transfers
	engine.SetReceiveCallback(func(task *model.TransferTask) bool {
		return true
	})

	// Track progress updates
	var progressMu sync.Mutex
	progressStates := []model.TransferState{}
	engine.SetProgressCallback(func(task *model.TransferTask) {
		progressMu.Lock()
		progressStates = append(progressStates, task.State)
		progressMu.Unlock()
	})

	// Start the engine TCP listener
	if err := engine.Start(); err != nil {
		t.Fatalf("engine start failed: %v", err)
	}
	defer engine.Stop()

	// Create a temp file with known content
	tmpDir := t.TempDir()
	content := strings.Repeat("A", 2*1024*1024) // 2 MB to ensure > 1 chunk
	tmpFile := filepath.Join(tmpDir, "transfer_test.bin")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Create send task targeting self (127.0.0.1 + engine's TCP port)
	task := engine.CreateSendTask(tmpFile, "self", "Self", "127.0.0.1", engine.TCPPort())
	if task == nil {
		t.Fatal("CreateSendTask returned nil")
	}

	// Run the send in a goroutine and wait for completion
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.SendFile(task.ID)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("SendFile failed: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("transfer timed out")
	}

	// Verify the send task is completed
	if task.State != model.StateCompleted {
		t.Errorf("send task state = %v, want StateCompleted", task.State)
	}
	if task.CompletedAt == nil {
		t.Error("send task CompletedAt is nil")
	}

	// Find the receive task
	tasks := engine.GetTasks()
	var recvTask *model.TransferTask
	for _, t := range tasks {
		if t.Type == model.TypeReceive {
			recvTask = t
			break
		}
	}
	if recvTask == nil {
		t.Fatal("no receive task found")
	}
	if recvTask.State != model.StateCompleted {
		t.Errorf("receive task state = %v, want StateCompleted", recvTask.State)
	}
	if recvTask.SavePath == "" {
		t.Fatal("receive task SavePath is empty")
	}

	// Verify the received file exists and has correct content
	receivedData, err := os.ReadFile(recvTask.SavePath)
	if err != nil {
		t.Fatalf("failed to read received file: %v", err)
	}
	if string(receivedData) != content {
		t.Errorf("received content mismatch: got %d bytes, want %d bytes", len(receivedData), len(content))
	}

	// Verify MD5 of received file
	receivedMD5, err := calcFileMD5(recvTask.SavePath)
	if err != nil {
		t.Fatalf("failed to calc MD5 of received file: %v", err)
	}
	if receivedMD5 != task.FileMD5 {
		t.Errorf("MD5 mismatch: received %s, original %s", receivedMD5, task.FileMD5)
	}

	// Verify progress was reported
	progressMu.Lock()
	if len(progressStates) == 0 {
		t.Error("no progress callbacks were made")
	}
	progressMu.Unlock()
}

func TestEngineStartStop(t *testing.T) {
	engine := NewEngine("node-1", "TestNode")
	if err := engine.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if engine.TCPPort() == 0 {
		t.Error("TCPPort is 0 after Start")
	}

	// Verify the port is actually listening
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", engine.TCPPort()), time.Second)
	if err != nil {
		t.Fatalf("failed to connect to listener: %v", err)
	}
	conn.Close()

	engine.Stop()

	// After stop, connection should fail
	time.Sleep(100 * time.Millisecond)
	_, err = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", engine.TCPPort()), 500*time.Millisecond)
	if err == nil {
		t.Error("expected connection to fail after Stop")
	}
}

func TestEngineCancelTask(t *testing.T) {
	engine := NewEngine("node-1", "TestNode")

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "cancel_test.txt")
	os.WriteFile(tmpFile, []byte("cancel me"), 0644)

	task := engine.CreateSendTask(tmpFile, "peer-1", "Peer", "127.0.0.1", 9999)
	if task == nil {
		t.Fatal("CreateSendTask returned nil")
	}

	// Cancel the pending task
	if err := engine.CancelTask(task.ID); err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}
	if task.State != model.StateCancelled {
		t.Errorf("expected StateCancelled, got %v", task.State)
	}
	if task.CompletedAt == nil {
		t.Error("CompletedAt should be set after cancellation")
	}

	// Cannot cancel again (terminal state)
	if err := engine.CancelTask(task.ID); err == nil {
		t.Error("expected error when cancelling terminal task")
	}

	// Nonexistent task
	if err := engine.CancelTask("nonexistent"); err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestCalcFileMD5Empty(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(tmpFile, []byte{}, 0644)

	hash, err := calcFileMD5(tmpFile)
	if err != nil {
		t.Fatalf("calcFileMD5 on empty file returned error: %v", err)
	}

	// MD5 of empty data is well-known
	expected := "d41d8cd98f00b204e9800998ecf8427e"
	if hash != expected {
		t.Errorf("empty file MD5: got %s, want %s", hash, expected)
	}
}

func TestResolveSavePathNoExt(t *testing.T) {
	tmpDir := t.TempDir()

	// File without extension, with conflict
	path1 := filepath.Join(tmpDir, "Makefile")
	os.WriteFile(path1, []byte("all:"), 0644)

	result := resolveSavePath(tmpDir, "Makefile")
	if result != filepath.Join(tmpDir, "Makefile(1)") {
		t.Errorf("no-ext conflict: got %s", result)
	}
}

func TestGetTasksEmpty(t *testing.T) {
	engine := NewEngine("node-1", "TestNode")
	tasks := engine.GetTasks()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestSendFileNonexistentTask(t *testing.T) {
	engine := NewEngine("node-1", "TestNode")
	err := engine.SendFile("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
	if err.Error() != "task not found: nonexistent-id" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Test that a rejected transfer (receiveCb returns false) fails gracefully.
func TestRejectedTransfer(t *testing.T) {
	engine := NewEngine("node-1", "TestNode")

	// Reject all incoming transfers
	engine.SetReceiveCallback(func(task *model.TransferTask) bool {
		return false
	})

	if err := engine.Start(); err != nil {
		t.Fatalf("engine start failed: %v", err)
	}
	defer engine.Stop()

	// Create a small temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "reject.bin")
	content := []byte("this will be rejected")
	os.WriteFile(tmpFile, content, 0644)

	task := engine.CreateSendTask(tmpFile, "self", "Self", "127.0.0.1", engine.TCPPort())
	if task == nil {
		t.Fatal("CreateSendTask returned nil")
	}

	err := engine.SendFile(task.ID)
	if err == nil {
		t.Fatal("expected error for rejected transfer")
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Errorf("expected rejection error, got: %v", err)
	}

	// Send task should be in failed state
	if task.State != model.StateFailed {
		t.Errorf("send task state = %v, want StateFailed", task.State)
	}
}

// Benchmark calcFileMD5 to ensure it's reasonably fast.
func BenchmarkCalcFileMD5(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "bench.dat")
	content := make([]byte, 10*1024*1024) // 10 MB
	for i := range content {
		content[i] = byte(i % 256)
	}
	os.WriteFile(tmpFile, content, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := calcFileMD5(tmpFile); err != nil {
			b.Fatal(err)
		}
	}
}
