package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Envelope wraps a protocol message with its type for JSON serialization.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// EncodeMessage marshals msg into a JSON envelope, prepends a 4-byte
// big-endian length prefix, and writes the result to w.
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

	// Write 4-byte big-endian length prefix.
	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return fmt.Errorf("write length prefix: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	return nil
}

// DecodeMessage reads a length-prefixed JSON envelope from r and returns
// the decoded Message.
func DecodeMessage(r io.Reader) (Message, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("read length prefix: %w", err)
	}

	if length == 0 {
		return nil, errors.New("invalid message: zero length")
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read message data: %w", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	return decodePayload(env.Type, env.Payload)
}

// decodePayload dispatches to the correct concrete message type based on
// the type string and unmarshals the raw JSON payload.
func decodePayload(msgType string, payload json.RawMessage) (Message, error) {
	switch msgType {
	case TypeTransferRequest:
		var m TransferRequest
		if err := json.Unmarshal(payload, &m); err != nil {
			return nil, fmt.Errorf("unmarshal TransferRequest: %w", err)
		}
		return &m, nil
	case TypeTransferResponse:
		var m TransferResponse
		if err := json.Unmarshal(payload, &m); err != nil {
			return nil, fmt.Errorf("unmarshal TransferResponse: %w", err)
		}
		return &m, nil
	case TypeChunkData:
		var m ChunkData
		if err := json.Unmarshal(payload, &m); err != nil {
			return nil, fmt.Errorf("unmarshal ChunkData: %w", err)
		}
		return &m, nil
	case TypeProgressAck:
		var m ProgressAck
		if err := json.Unmarshal(payload, &m); err != nil {
			return nil, fmt.Errorf("unmarshal ProgressAck: %w", err)
		}
		return &m, nil
	case TypeTransferComplete:
		var m TransferComplete
		if err := json.Unmarshal(payload, &m); err != nil {
			return nil, fmt.Errorf("unmarshal TransferComplete: %w", err)
		}
		return &m, nil
	case TypeTransferVerify:
		var m TransferVerify
		if err := json.Unmarshal(payload, &m); err != nil {
			return nil, fmt.Errorf("unmarshal TransferVerify: %w", err)
		}
		return &m, nil
	case TypeTransferCancel:
		var m TransferCancel
		if err := json.Unmarshal(payload, &m); err != nil {
			return nil, fmt.Errorf("unmarshal TransferCancel: %w", err)
		}
		return &m, nil
	default:
		return nil, fmt.Errorf("unknown message type: %q", msgType)
	}
}

// EncodeToBytes is a convenience function that encodes a message into a
// []byte buffer using the length-prefixed JSON codec.
func EncodeToBytes(msg Message) ([]byte, error) {
	var buf bytes.Buffer
	if err := EncodeMessage(&buf, msg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
