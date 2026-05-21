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
