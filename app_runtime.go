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
