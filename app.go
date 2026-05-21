package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"lan-file-share/internal/discovery"
	"lan-file-share/internal/model"
	"lan-file-share/internal/queue"
	"lan-file-share/internal/transfer"
)

type App struct {
	ctx             context.Context
	discovery       *discovery.Service
	engine          *transfer.Engine
	queue           *queue.Manager
	pendingReceives sync.Map
}

func NewApp() *App {
	return &App{}
}

func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.ctx = ctx
	nodeName := model.GetHostname()

	svc := discovery.NewService(nodeName, 0)
	if err := svc.Start(); err != nil {
		return fmt.Errorf("discovery start failed: %w", err)
	}
	a.discovery = svc

	eng := transfer.NewEngine(svc.NodeID(), nodeName)
	eng.SetProgressCallback(func(task *model.TransferTask) {
		a.queue.UpdateProgress(task.ID, task.BytesTransferred, task.Speed)
		a.queue.UpdateState(task.ID, task.State)
	})
	eng.SetReceiveCallback(func(task *model.TransferTask) bool {
		a.queue.Add(task)
		ch := make(chan bool, 1)
		a.pendingReceives.Store(task.ID, ch)
		return <-ch
	})
	if err := eng.Start(); err != nil {
		svc.Stop()
		return fmt.Errorf("engine start failed: %w", err)
	}
	a.engine = eng
	svc.SetTCPPort(eng.TCPPort())

	svc.SetCallback(func(device *discovery.DeviceEntry, online bool) {
		application.Get().Event.Emit("device:changed", map[string]interface{}{
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
		application.Get().Event.Emit("task:changed", task)
	})
	a.queue = q

	return nil
}

func (a *App) ServiceShutdown() error {
	if a.discovery != nil {
		a.discovery.Stop()
	}
	if a.engine != nil {
		a.engine.Stop()
	}
	return nil
}
