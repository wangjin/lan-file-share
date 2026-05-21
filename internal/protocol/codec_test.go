package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
)

func TestEncodeDecodeMessage(t *testing.T) {
	original := &TransferRequest{
		FileName: "test.pdf",
		FileSize: 1024 * 1024,
		FileMD5:  "d41d8cd98f00b204e9800998ecf8427e",
		Chunks:   10,
	}

	var buf bytes.Buffer
	if err := EncodeMessage(&buf, original); err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	decoded, err := DecodeMessage(&buf)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	got, ok := decoded.(*TransferRequest)
	if !ok {
		t.Fatalf("expected *TransferRequest, got %T", decoded)
	}

	if got.FileName != original.FileName {
		t.Errorf("FileName: got %q, want %q", got.FileName, original.FileName)
	}
	if got.FileSize != original.FileSize {
		t.Errorf("FileSize: got %d, want %d", got.FileSize, original.FileSize)
	}
	if got.FileMD5 != original.FileMD5 {
		t.Errorf("FileMD5: got %q, want %q", got.FileMD5, original.FileMD5)
	}
	if got.Chunks != original.Chunks {
		t.Errorf("Chunks: got %d, want %d", got.Chunks, original.Chunks)
	}
	if got.Type() != TypeTransferRequest {
		t.Errorf("Type(): got %q, want %q", got.Type(), TypeTransferRequest)
	}
}

func TestDecodeEmptyBuffer(t *testing.T) {
	buf := bytes.Buffer{}
	_, err := DecodeMessage(&buf)
	if err == nil {
		t.Fatal("expected error from empty buffer, got nil")
	}
}

func TestEncodeDecodeAllMessageTypes(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		wantErr bool
	}{
		{
			name: "TransferRequest",
			msg: &TransferRequest{
				FileName: "photo.jpg",
				FileSize: 2048000,
				FileMD5:  "abc123",
				Chunks:   5,
			},
		},
		{
			name: "TransferResponse accepted",
			msg: &TransferResponse{
				Accepted: true,
			},
		},
		{
			name: "TransferResponse rejected",
			msg: &TransferResponse{
				Accepted: false,
				Reason:   "disk full",
			},
		},
		{
			name: "ChunkData",
			msg: &ChunkData{
				Sequence: 3,
				Size:     65536,
			},
		},
		{
			name: "ProgressAck",
			msg: &ProgressAck{
				BytesReceived: 131072,
				State:         "receiving",
			},
		},
		{
			name: "TransferComplete",
			msg: &TransferComplete{
				MD5: "finalmd5hash",
			},
		},
		{
			name: "TransferVerify",
			msg: &TransferVerify{
				Success: true,
			},
		},
		{
			name: "TransferCancel",
			msg: &TransferCancel{
				Reason: "user cancelled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := EncodeMessage(&buf, tt.msg); err != nil {
				t.Fatalf("EncodeMessage failed: %v", err)
			}
			encoded := buf.Bytes()

			decoded, err := DecodeMessage(&buf)
			if err != nil {
				t.Fatalf("DecodeMessage failed: %v", err)
			}

			if decoded.Type() != tt.msg.Type() {
				t.Errorf("type mismatch: got %q, want %q", decoded.Type(), tt.msg.Type())
			}

			// Verify the roundtrip by re-encoding and comparing bytes.
			var buf2 bytes.Buffer
			if err := EncodeMessage(&buf2, decoded); err != nil {
				t.Fatalf("re-encode failed: %v", err)
			}

			if !bytes.Equal(encoded, buf2.Bytes()) {
				t.Errorf("roundtrip bytes differ:\n  encoded: %x\n  re-encoded: %x", encoded, buf2.Bytes())
			}
		})
	}
}

func TestEncodeToBytes(t *testing.T) {
	msg := &TransferCancel{Reason: "timeout"}
	data, err := EncodeToBytes(msg)
	if err != nil {
		t.Fatalf("EncodeToBytes failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("EncodeToBytes returned empty slice")
	}

	buf := bytes.NewReader(data)
	decoded, err := DecodeMessage(buf)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	got, ok := decoded.(*TransferCancel)
	if !ok {
		t.Fatalf("expected *TransferCancel, got %T", decoded)
	}
	if got.Reason != msg.Reason {
		t.Errorf("Reason: got %q, want %q", got.Reason, msg.Reason)
	}
}

func TestDecodeUnknownType(t *testing.T) {
	env := Envelope{
		Type:    "unknown_type",
		Payload: []byte(`{}`),
	}

	var buf bytes.Buffer
	data, _ := json.Marshal(env)
	writeLengthPrefix(&buf, uint32(len(data)))
	buf.Write(data)

	_, err := DecodeMessage(&buf)
	if err == nil {
		t.Fatal("expected error for unknown message type, got nil")
	}
	if !strings.Contains(err.Error(), "unknown message type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecodeZeroLength(t *testing.T) {
	var buf bytes.Buffer
	writeLengthPrefix(&buf, 0)

	_, err := DecodeMessage(&buf)
	if err == nil {
		t.Fatal("expected error for zero-length message, got nil")
	}
}

// Helper to write a raw length prefix without using EncodeMessage.
func writeLengthPrefix(buf *bytes.Buffer, length uint32) {
	_ = binary.Write(buf, binary.BigEndian, length)
}
