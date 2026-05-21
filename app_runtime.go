package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func runtimeEventsEmit(ctx context.Context, eventName string, data ...interface{}) {
	runtime.EventsEmit(ctx, eventName, data...)
}

func runtimeOpenFileDialog(ctx context.Context) (string, error) {
	return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select File to Send",
	})
}

func runtimeSaveFileDialog(ctx context.Context, defaultFilename string) (string, error) {
	return runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Save File",
		DefaultFilename: defaultFilename,
	})
}
