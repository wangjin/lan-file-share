package model

import "testing"

func TestTransferStateString(t *testing.T) {
	tests := []struct {
		state TransferState
		want  string
	}{
		{StatePending, "pending"},
		{StateTransferring, "transferring"},
		{StatePaused, "paused"},
		{StateCompleted, "completed"},
		{StateFailed, "failed"},
		{StateCancelled, "cancelled"},
		{TransferState(99), "unknown"},
	}
	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("TransferState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestTransferTypeString(t *testing.T) {
	if got := TypeSend.String(); got != "send" {
		t.Errorf("TypeSend.String() = %q, want %q", got, "send")
	}
	if got := TypeReceive.String(); got != "receive" {
		t.Errorf("TypeReceive.String() = %q, want %q", got, "receive")
	}
}

func TestIsTerminal(t *testing.T) {
	terminalStates := []TransferState{StateCompleted, StateFailed, StateCancelled}
	for _, s := range terminalStates {
		task := &TransferTask{State: s}
		if !task.IsTerminal() {
			t.Errorf("IsTerminal() = false for state %s, want true", s)
		}
	}

	nonTerminalStates := []TransferState{StatePending, StateTransferring, StatePaused}
	for _, s := range nonTerminalStates {
		task := &TransferTask{State: s}
		if task.IsTerminal() {
			t.Errorf("IsTerminal() = true for state %s, want false", s)
		}
	}
}

func TestProgress(t *testing.T) {
	tests := []struct {
		name            string
		fileSize        int64
		bytesTransferred int64
		want            float64
	}{
		{"zero file size", 0, 0, 0.0},
		{"half progress", 1000, 500, 0.5},
		{"full progress", 1000, 1000, 1.0},
		{"no progress", 1000, 0, 0.0},
		{"partial progress", 200, 50, 0.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &TransferTask{
				FileSize:         tt.fileSize,
				BytesTransferred: tt.bytesTransferred,
			}
			got := task.Progress()
			if got != tt.want {
				t.Errorf("Progress() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestCalcChunks(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
		want     int
	}{
		{"exact division", 2 * 1024 * 1024, 2},
		{"non-exact division", 1024*1024 + 1, 2},
		{"less than one chunk", 512, 1},
		{"zero size", 0, 0},
		{"one byte under", 3*1024*1024 - 1, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcChunks(tt.fileSize)
			if got != tt.want {
				t.Errorf("CalcChunks(%d) = %d, want %d", tt.fileSize, got, tt.want)
			}
		})
	}
}
