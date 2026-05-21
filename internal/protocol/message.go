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

// Message is the interface implemented by all protocol message types.
type Message interface {
	Type() string
}

type TransferRequest struct {
	FileName   string `json:"file_name"`
	FileSize   int64  `json:"file_size"`
	FileMD5    string `json:"file_md5"`
	Chunks     int    `json:"chunks"`
	SenderID   string `json:"sender_id"`
	SenderName string `json:"sender_name"`
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
