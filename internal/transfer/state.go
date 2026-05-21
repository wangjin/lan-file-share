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
