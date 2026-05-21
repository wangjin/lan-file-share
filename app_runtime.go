package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func runtimeOpenFileDialog() (string, error) {
	result, err := application.Get().Dialog.OpenFile().
		SetTitle("Select File to Send").
		PromptForSingleSelection()
	return result, err
}

func runtimeSaveFileDialog(defaultFilename string) (string, error) {
	result, err := application.Get().Dialog.SaveFile().
		SetMessage("Save File").
		SetFilename(defaultFilename).
		PromptForSingleSelection()
	return result, err
}
