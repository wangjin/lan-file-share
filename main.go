package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"lan-file-share/internal/updater"
)

var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed frontend/src/assets/images/logo.png
var iconData []byte

func main() {
	app := application.New(application.Options{
		Name:        "LAN File Share",
		Description: "LAN File Sharing Application",
		Icon:        iconData,
		Services: []application.Service{
			application.NewService(NewApp()),
			application.NewService(updater.NewService(version, "wangjin/lan-file-share")),
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
		Linux: application.LinuxWindow{
			Icon: iconData,
		},
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
