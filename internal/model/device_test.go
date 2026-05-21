package model

import (
	"testing"
)

func TestDeviceCreation(t *testing.T) {
	d := Device{
		NodeID: "abc-123",
		Name:   "TestDevice",
		IP:     "192.168.1.100",
		Port:   8080,
		OS:     "darwin",
		Online: true,
		LastSeen: 1700000000,
	}

	if d.NodeID != "abc-123" {
		t.Errorf("expected NodeID 'abc-123', got '%s'", d.NodeID)
	}
	if d.Name != "TestDevice" {
		t.Errorf("expected Name 'TestDevice', got '%s'", d.Name)
	}
	if d.IP != "192.168.1.100" {
		t.Errorf("expected IP '192.168.1.100', got '%s'", d.IP)
	}
	if d.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", d.Port)
	}
	if d.OS != "darwin" {
		t.Errorf("expected OS 'darwin', got '%s'", d.OS)
	}
	if d.Online != true {
		t.Errorf("expected Online true, got %v", d.Online)
	}
	if d.LastSeen != 1700000000 {
		t.Errorf("expected LastSeen 1700000000, got %d", d.LastSeen)
	}
}

func TestDisplayOS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"darwin", "macOS"},
		{"windows", "Windows"},
		{"linux", "Linux"},
		{"unknown", "unknown"},
		{"freebsd", "freebsd"},
	}

	for _, tt := range tests {
		d := Device{OS: tt.input}
		got := d.DisplayOS()
		if got != tt.expected {
			t.Errorf("DisplayOS() with OS=%q: expected %q, got %q", tt.input, tt.expected, got)
		}
	}
}

func TestDefaultSaveDir(t *testing.T) {
	dir := DefaultSaveDir()
	if dir == "" {
		t.Error("DefaultSaveDir() returned empty string")
	}
}

func TestGetHostname(t *testing.T) {
	name := GetHostname()
	if name == "" {
		t.Error("GetHostname() returned empty string")
	}
}

func TestGetOSName(t *testing.T) {
	osName := GetOSName()
	if osName == "" {
		t.Error("GetOSName() returned empty string")
	}
}
