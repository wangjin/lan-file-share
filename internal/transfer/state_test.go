package transfer

import (
	"testing"

	"local-file-share/internal/model"
)

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from model.TransferState
		to   model.TransferState
	}{
		{"Pending -> Transferring", model.StatePending, model.StateTransferring},
		{"Pending -> Cancelled", model.StatePending, model.StateCancelled},
		{"Transferring -> Paused", model.StateTransferring, model.StatePaused},
		{"Transferring -> Completed", model.StateTransferring, model.StateCompleted},
		{"Transferring -> Failed", model.StateTransferring, model.StateFailed},
		{"Transferring -> Cancelled", model.StateTransferring, model.StateCancelled},
		{"Paused -> Transferring", model.StatePaused, model.StateTransferring},
		{"Paused -> Cancelled", model.StatePaused, model.StateCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine(tt.from)
			if err := sm.Transition(tt.to); err != nil {
				t.Errorf("Transition(%s → %s) failed: %v", tt.from, tt.to, err)
			}
			if sm.Current() != tt.to {
				t.Errorf("Current() = %s, want %s", sm.Current(), tt.to)
			}
		})
	}
}

func TestInvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from model.TransferState
		to   model.TransferState
	}{
		{"Completed -> Transferring", model.StateCompleted, model.StateTransferring},
		{"Failed -> Transferring", model.StateFailed, model.StateTransferring},
		{"Cancelled -> Paused", model.StateCancelled, model.StatePaused},
		{"Pending -> Completed (skip)", model.StatePending, model.StateCompleted},
		{"Pending -> Paused (skip)", model.StatePending, model.StatePaused},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine(tt.from)
			err := sm.Transition(tt.to)
			if err == nil {
				t.Errorf("Transition(%s → %s) should have failed", tt.from, tt.to)
			}
			if sm.Current() != tt.from {
				t.Errorf("Current() = %s, want %s (unchanged)", sm.Current(), tt.from)
			}
		})
	}
}

func TestCallbackFires(t *testing.T) {
	sm := NewStateMachine(model.StatePending)

	var capturedFrom, capturedTo model.TransferState
	called := false

	sm.OnChange(func(from, to model.TransferState) {
		capturedFrom = from
		capturedTo = to
		called = true
	})

	if err := sm.Transition(model.StateTransferring); err != nil {
		t.Fatalf("Transition failed: %v", err)
	}

	if !called {
		t.Error("callback was not called")
	}
	if capturedFrom != model.StatePending {
		t.Errorf("callback from = %s, want %s", capturedFrom, model.StatePending)
	}
	if capturedTo != model.StateTransferring {
		t.Errorf("callback to = %s, want %s", capturedTo, model.StateTransferring)
	}
}

func TestCallbackNotFiredOnInvalidTransition(t *testing.T) {
	sm := NewStateMachine(model.StateCompleted)

	called := false
	sm.OnChange(func(from, to model.TransferState) {
		called = true
	})

	err := sm.Transition(model.StateTransferring)
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
	if called {
		t.Error("callback should not have been called for invalid transition")
	}
}

func TestNewStateMachineInitial(t *testing.T) {
	sm := NewStateMachine(model.StatePaused)
	if sm.Current() != model.StatePaused {
		t.Errorf("Current() = %s, want %s", sm.Current(), model.StatePaused)
	}
}

func TestSequentialTransitions(t *testing.T) {
	sm := NewStateMachine(model.StatePending)

	steps := []model.TransferState{
		model.StateTransferring,
		model.StatePaused,
		model.StateTransferring,
		model.StateCompleted,
	}

	for i, to := range steps {
		if err := sm.Transition(to); err != nil {
			t.Errorf("step %d: Transition to %s failed: %v", i, to, err)
		}
		if sm.Current() != to {
			t.Errorf("step %d: Current() = %s, want %s", i, sm.Current(), to)
		}
	}
}

func TestTransitionErrorMessage(t *testing.T) {
	sm := NewStateMachine(model.StateCompleted)
	err := sm.Transition(model.StateTransferring)
	if err == nil {
		t.Fatal("expected error")
	}

	expectedMsg := "invalid transition: completed → transferring"
	if err.Error() != expectedMsg {
		t.Errorf("error message = %q, want %q", err.Error(), expectedMsg)
	}
}
