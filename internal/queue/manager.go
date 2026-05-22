package queue

import (
	"fmt"
	"sync"

	"nearfy/internal/model"
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

func (m *Manager) SetCallback(cb StateChangeCallback) { m.callback = cb }

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
		if t.State == model.StatePending {
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

	// Separate pending tasks: collect target and the rest
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

	// Build reordered waiting list: insert target at newPosition
	reordered := make([]*model.TransferTask, 0, len(waiting)+1)
	reordered = append(reordered, waiting[:newPosition]...)
	reordered = append(reordered, target)
	reordered = append(reordered, waiting[newPosition:]...)

	// Rebuild tasks: non-pending tasks keep their position, pending tasks use reordered list
	var newTasks []*model.TransferTask
	pendIdx := 0
	for _, t := range m.tasks {
		if t.State != model.StatePending {
			newTasks = append(newTasks, t)
		} else {
			if pendIdx < len(reordered) {
				newTasks = append(newTasks, reordered[pendIdx])
				pendIdx++
			}
		}
	}
	m.tasks = newTasks
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
