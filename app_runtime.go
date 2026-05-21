package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// runtimeEventsEmit wraps runtime.EventsEmit to emit events to the frontend.
func runtimeEventsEmit(ctx context.Context, eventName string, data ...interface{}) {
	runtime.EventsEmit(ctx, eventName, data...)
}
