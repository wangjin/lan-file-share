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

type progressPoint struct {
	bytes int64
	time  time.Time
}

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

	connMap   map[string]net.Conn
	connMutex sync.Mutex

	lastProgress map[string]progressPoint
	speedMutex   sync.Mutex
}

func NewEngine(localNodeID, localNodeName string) *Engine {
	return &Engine{
		localNodeID:  localNodeID,
		localNodeName: localNodeName,
		tasks:        make(map[string]*model.TransferTask),
		stopCh:       make(chan struct{}),
		connMap:      make(map[string]net.Conn),
		lastProgress: make(map[string]progressPoint),
	}
}

func (e *Engine) SetProgressCallback(cb ProgressCallback) { e.progressCb = cb }
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

func (e *Engine) TCPPort() int { return e.tcpPort }

func (e *Engine) SetTaskSavePath(taskID string, savePath string) {
	e.taskMutex.Lock()
	defer e.taskMutex.Unlock()
	if task, ok := e.tasks[taskID]; ok {
		task.SavePath = savePath
	}
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

	e.registerConn(taskID, conn)

	// Send transfer request
	req := &protocol.TransferRequest{
		FileName:   task.FileName,
		FileSize:   task.FileSize,
		FileMD5:    task.FileMD5,
		Chunks:     task.ChunksTotal,
		SenderID:   e.localNodeID,
		SenderName: e.localNodeName,
	}
	if err := protocol.EncodeMessage(conn, req); err != nil {
		e.updateState(task, model.StateFailed)
		return err
	}

	// Read response
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

	// Send file data
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
		e.taskMutex.RLock()
		cancelled := task.State == model.StateCancelled
		e.taskMutex.RUnlock()
		if cancelled {
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

		chunkMsg := &protocol.ChunkData{Sequence: chunk, Size: int64(n)}
		if err := protocol.EncodeMessage(conn, chunkMsg); err != nil {
			e.updateState(task, model.StateFailed)
			return err
		}
		if _, err := conn.Write(buf[:n]); err != nil {
			e.updateState(task, model.StateFailed)
			return err
		}

		e.taskMutex.Lock()
		task.BytesTransferred += int64(n)
		task.ChunksCompleted = chunk + 1
		snapshot := *task
		e.taskMutex.Unlock()
		e.notifyProgress(&snapshot)
	}

	// Send complete
	if err := protocol.EncodeMessage(conn, &protocol.TransferComplete{MD5: task.FileMD5}); err != nil {
		e.updateState(task, model.StateFailed)
		return err
	}

	// Verify
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

func (e *Engine) CancelTask(taskID string) error {
	e.closeTaskConn(taskID)

	e.taskMutex.Lock()
	task, ok := e.tasks[taskID]
	if !ok {
		e.taskMutex.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.IsTerminal() {
		e.taskMutex.Unlock()
		return fmt.Errorf("cannot cancel task in state %s", task.State)
	}
	task.State = model.StateCancelled
	now := time.Now()
	task.CompletedAt = &now
	snapshot := *task
	e.taskMutex.Unlock()

	e.notifyProgress(&snapshot)
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
		PeerID:      req.SenderID,
		PeerName:    req.SenderName,
		CreatedAt:   time.Now(),
	}
	e.taskMutex.Lock()
	e.tasks[task.ID] = task
	e.taskMutex.Unlock()

	accepted := false
	if e.receiveCb != nil {
		accepted = e.receiveCb(task)
	}
	protocol.EncodeMessage(conn, &protocol.TransferResponse{Accepted: accepted})
	if !accepted {
		e.updateState(task, model.StateFailed)
		return
	}

	e.registerConn(task.ID, conn)
	e.receiveFileData(conn, task)
}

func (e *Engine) receiveFileData(conn net.Conn, task *model.TransferTask) {
	e.updateState(task, model.StateTransferring)

	if task.SavePath == "" {
		saveDir := model.DefaultSaveDir()
		task.SavePath = resolveSavePath(saveDir, task.FileName)
	}
	tmpPath := task.SavePath + ".tmp"

	outFile, err := os.Create(tmpPath)
	if err != nil {
		e.updateState(task, model.StateFailed)
		return
	}
	defer outFile.Close()

	cleanup := func() {
		outFile.Close()
		os.Remove(tmpPath)
	}

	for {
		msg, err := protocol.DecodeMessage(conn)
		if err != nil {
			e.taskMutex.RLock()
			cancelled := task.State == model.StateCancelled
			e.taskMutex.RUnlock()
			cleanup()
			if !cancelled {
				e.updateState(task, model.StateFailed)
			}
			return
		}

		switch m := msg.(type) {
		case *protocol.ChunkData:
			data := make([]byte, m.Size)
			if _, err := io.ReadFull(conn, data); err != nil {
				e.taskMutex.RLock()
				cancelled := task.State == model.StateCancelled
				e.taskMutex.RUnlock()
				cleanup()
				if !cancelled {
					e.updateState(task, model.StateFailed)
				}
				return
			}
			if _, err := outFile.Write(data); err != nil {
				cleanup()
				e.updateState(task, model.StateFailed)
				return
			}
			e.taskMutex.Lock()
			task.BytesTransferred += m.Size
			task.ChunksCompleted = m.Sequence + 1
			snapshot := *task
			e.taskMutex.Unlock()
			e.notifyProgress(&snapshot)

		case *protocol.TransferComplete:
			outFile.Close()
			receivedMD5, _ := calcFileMD5(tmpPath)
			success := receivedMD5 == m.MD5
			if success {
				os.Rename(tmpPath, task.SavePath)
			} else {
				os.Remove(tmpPath)
			}
			protocol.EncodeMessage(conn, &protocol.TransferVerify{Success: success})
			if success {
				e.updateState(task, model.StateCompleted)
			} else {
				e.updateState(task, model.StateFailed)
			}
			return

		case *protocol.TransferCancel:
			cleanup()
			e.updateState(task, model.StateCancelled)
			return
		}
	}
}

func (e *Engine) updateState(task *model.TransferTask, state model.TransferState) {
	e.taskMutex.Lock()
	if task.IsTerminal() {
		e.taskMutex.Unlock()
		return
	}
	task.State = state
	if state == model.StateCompleted || state == model.StateFailed || state == model.StateCancelled {
		now := time.Now()
		task.CompletedAt = &now
	}
	snapshot := *task
	e.taskMutex.Unlock()
	e.notifyProgress(&snapshot)
}

func (e *Engine) notifyProgress(snapshot *model.TransferTask) {
	if e.progressCb == nil {
		return
	}
	// Calculate speed
	e.speedMutex.Lock()
	now := time.Now()
	if pt, ok := e.lastProgress[snapshot.ID]; ok {
		elapsed := now.Sub(pt.time).Seconds()
		if elapsed > 0 {
			snapshot.Speed = int64(float64(snapshot.BytesTransferred-pt.bytes) / elapsed)
		}
	}
	e.lastProgress[snapshot.ID] = progressPoint{bytes: snapshot.BytesTransferred, time: now}
	e.speedMutex.Unlock()

	go e.progressCb(snapshot)
}

func (e *Engine) registerConn(taskID string, conn net.Conn) {
	e.connMutex.Lock()
	e.connMap[taskID] = conn
	e.connMutex.Unlock()
}

func (e *Engine) closeTaskConn(taskID string) {
	e.connMutex.Lock()
	if conn, ok := e.connMap[taskID]; ok {
		conn.Close()
		delete(e.connMap, taskID)
	}
	e.connMutex.Unlock()
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
