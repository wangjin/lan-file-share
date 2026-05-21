package model
import (
	
	"os"
	"path/filepath"
	"runtime"
)
type Device struct {
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	OS       string `json:"os"`
	Online   bool   `json:"-"`
	LastSeen int64  `json:"-"`
}
func (d Device) DisplayOS() string {
	switch d.OS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return d.OS
	}
}
func DefaultSaveDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	default:
		return filepath.Join(os.Getenv("HOME"), "Downloads")
	}
}
func GetHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "Unknown"
	}
	return name
}
func GetOSName() string {
	return runtime.GOOS
}
