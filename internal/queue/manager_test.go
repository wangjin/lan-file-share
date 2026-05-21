package queue

import (
	"testing"

	"local-file-share/internal/model"
)

func TestAddTask(t *testing.T) {
	m := NewManager(2)
	task := &model.TransferTask{
		ID:    "task-1",
		State: model.StatePending,
	}
	m.Add(task)

	all := m.GetAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 task, got %d", len(all))
	}
	if all[0].ID != "task-1" {
		t.Errorf("expected task ID 'task-1', got '%s'", all[0].ID)
	}

	got := m.Get("task-1")
	if got == nil {
		t.Fatal("expected to find task-1, got nil")
	}
	if got.ID != "task-1" {
		t.Errorf("expected task ID 'task-1', got '%s'", got.ID)
	}
}

func TestMaxConcurrent(t *testing.T) {
	m := NewManager(2)

	for i := 0; i < 3; i++ {
		m.Add(&model.TransferTask{
			ID:    string(rune('a' + i)),
			State: model.StatePending,
		})
	}

	wc := m.WaitingCount()
	if wc != 3 {
		t.Errorf("expected 3 waiting tasks, got %d", wc)
	}
}

func TestCompleteFreesSlot(t *testing.T) {
	m := NewManager(1)

	done := make(chan struct{}, 1)
	m.SetCallback(func(task *model.TransferTask) {
		if task.State == model.StatePending {
			go func() {
				m.UpdateState(task.ID, model.StateTransferring)
				done <- struct{}{}
			}()
		}
	})

	t1 := &model.TransferTask{ID: "t1", State: model.StateTransferring}
	t2 := &model.TransferTask{ID: "t2", State: model.StatePending}
	m.Add(t1)
	m.Add(t2)

	// t1 is Transferring, maxConcurrent=1, so t2 stays Pending
	if m.WaitingCount() != 1 {
		t.Errorf("expected 1 waiting task before completion, got %d", m.WaitingCount())
	}

	// Complete t1 — processQueue will find t2 (Pending) and notify callback,
	// which sets t2 to Transferring
	m.UpdateState("t1", model.StateCompleted)

	// Wait for the async callback to change t2's state
	<-done

	wc := m.WaitingCount()
	if wc != 0 {
		t.Errorf("expected 0 waiting tasks after completion, got %d", wc)
	}
}

func TestCancelTask(t *testing.T) {
	m := NewManager(2)
	task := &model.TransferTask{
		ID:    "task-1",
		State: model.StatePending,
	}
	m.Add(task)

	err := m.Cancel("task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := m.Get("task-1")
	if got == nil {
		t.Fatal("task not found")
	}
	if got.State != model.StateCancelled {
		t.Errorf("expected state Cancelled, got %s", got.State)
	}

	// Cancelling a terminal task should fail
	err = m.Cancel("task-1")
	if err == nil {
		t.Error("expected error when cancelling a terminal task")
	}
}

func TestReorderTask(t *testing.T) {
	m := NewManager(3)

	t1 := &model.TransferTask{ID: "t1", State: model.StateTransferring}
	t2 := &model.TransferTask{ID: "t2", State: model.StatePending}
	t3 := &model.TransferTask{ID: "t3", State: model.StatePending}
	m.Add(t1)
	m.Add(t2)
	m.Add(t3)

	// Before reorder: waiting order should be [t2, t3]
	waiting := m.GetWaiting()
	if len(waiting) != 2 {
		t.Fatalf("expected 2 waiting tasks, got %d", len(waiting))
	}
	if waiting[0].ID != "t2" || waiting[1].ID != "t3" {
		t.Errorf("expected waiting order [t2, t3], got [%s, %s]", waiting[0].ID, waiting[1].ID)
	}

	// Reorder t3 to position 0
	m.Reorder("t3", 0)

	waiting = m.GetWaiting()
	if len(waiting) != 2 {
		t.Fatalf("expected 2 waiting tasks after reorder, got %d", len(waiting))
	}
	if waiting[0].ID != "t3" {
		t.Errorf("expected t3 to be first in waiting list, got %s", waiting[0].ID)
	}
	if waiting[1].ID != "t2" {
		t.Errorf("expected t2 to be second in waiting list, got %s", waiting[1].ID)
	}
}
