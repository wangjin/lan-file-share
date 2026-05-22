package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "LAN File Share",
		Description: "LAN File Sharing Application",
		Services: []application.Service{
			application.NewService(NewApp()),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:           "LAN File Share",
		Width:           1024,
		Height:          680,
		DevToolsEnabled: true,
		EnableFileDrop:  true,
	})

	win.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		application.Get().Event.Emit("files-dropped", map[string]any{
			"files": files,
		})
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
