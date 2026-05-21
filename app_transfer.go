package main

import (
	"context"
	"fmt"
	"os"

	"local-file-share/internal/discovery"
	"local-file-share/internal/model"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// runtimeOpenFileDialog opens a native file picker dialog and returns the selected file path.
func runtimeOpenFileDialog(ctx context.Context) (string, error) {
	return wailsRuntime.OpenFileDialog(ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select File to Send",
	})
}

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

func (a *App) CancelTask(taskID string) error {
	return a.queue.Cancel(taskID)
}

func (a *App) GetTasks() []*model.TransferTask {
	return a.queue.GetAll()
}

func (a *App) AcceptReceive(taskID string) {
	a.engine.SetReceiveCallback(func(task *model.TransferTask) bool {
		return true
	})
}
