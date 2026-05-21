package main

import (
	"context"
	"fmt"
	"os"

	"local-file-share/internal/discovery"
	"local-file-share/internal/model"
	"local-file-share/internal/queue"
	"local-file-share/internal/transfer"
)

type App struct {
	ctx       context.Context
	discovery *discovery.Service
	engine    *transfer.Engine
	queue     *queue.Manager
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	nodeName := model.GetHostname()

	svc := discovery.NewService(nodeName, 0)
	if err := svc.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "discovery start failed: %v\n", err)
		return
	}
	a.discovery = svc

	eng := transfer.NewEngine(svc.NodeID(), nodeName)
	eng.SetProgressCallback(func(task *model.TransferTask) {
		a.queue.UpdateProgress(task.ID, task.BytesTransferred, task.Speed)
		if task.IsTerminal() {
			a.queue.UpdateState(task.ID, task.State)
		}
	})
	if err := eng.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "engine start failed: %v\n", err)
		return
	}
	a.engine = eng

	svc.SetCallback(func(device *discovery.DeviceEntry, online bool) {
		runtimeEventsEmit(a.ctx, "device:changed", map[string]interface{}{
			"node_id": device.NodeID,
			"name":    device.Name,
			"ip":      device.IP,
			"port":    device.Port,
			"os":      device.OS,
			"online":  online,
		})
	})

	q := queue.NewManager(2)
	q.SetCallback(func(task *model.TransferTask) {
		runtimeEventsEmit(a.ctx, "task:changed", task)
	})
	a.queue = q
}

func (a *App) Shutdown(ctx context.Context) {
	if a.discovery != nil {
		a.discovery.Stop()
	}
	if a.engine != nil {
		a.engine.Stop()
	}
}
