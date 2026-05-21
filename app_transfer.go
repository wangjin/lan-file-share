package main

import (
	"fmt"
	"os"

	"local-file-share/internal/discovery"
	"local-file-share/internal/model"
)

func (a *App) SelectAndSend(peerID string) error {
	if a.engine == nil || a.discovery == nil {
		return fmt.Errorf("service not initialized")
	}

	filePath, err := runtimeOpenFileDialog(a.ctx)
	if err != nil || filePath == "" {
		return nil
	}

	devices := a.discovery.GetDevices()
	var peer *discovery.DeviceEntry
	for _, d := range devices {
		if d.NodeID == peerID {
			peer = d
			break
		}
	}
	if peer == nil {
		return fmt.Errorf("device not found: %s", peerID)
	}

	task := a.engine.CreateSendTask(filePath, peer.NodeID, peer.Name, peer.IP, peer.Port)
	if task == nil {
		return fmt.Errorf("failed to create send task")
	}
	a.queue.Add(task)

	go func() {
		if err := a.engine.SendFile(task.ID); err != nil {
			fmt.Fprintf(os.Stderr, "send failed: %v\n", err)
		}
	}()

	return nil
}

func (a *App) ChooseSavePath(fileName string) (string, error) {
	return runtimeSaveFileDialog(a.ctx, fileName)
}

func (a *App) RespondReceive(taskID string, accept bool, savePath string) error {
	if accept && savePath != "" {
		a.engine.SetTaskSavePath(taskID, savePath)
	}
	val, ok := a.pendingReceives.LoadAndDelete(taskID)
	if !ok {
		return fmt.Errorf("no pending receive for task: %s", taskID)
	}
	val.(chan bool) <- accept
	if !accept {
		a.queue.UpdateState(taskID, model.StateFailed)
	}
	return nil
}

func (a *App) CancelTask(taskID string) error {
	if err := a.queue.Cancel(taskID); err != nil {
		return err
	}
	a.engine.CancelTask(taskID)
	return nil
}

func (a *App) GetTasks() []*model.TransferTask {
	return a.queue.GetAll()
}
