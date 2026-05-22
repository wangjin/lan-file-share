package updater

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v2.0.0",
			"body": "Bug fixes and improvements",
			"assets": [
				{"name": "Nearfy-macos.dmg", "browser_download_url": "https://example.com/macos.tar.gz", "size": 12345},
				{"name": "Nearfy-windows-amd64.exe", "browser_download_url": "https://example.com/win.exe", "size": 67890}
			]
		}`))
	}))
	defer server.Close()

	info, err := checkLatestRelease(server.URL, "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TagName != "v2.0.0" {
		t.Errorf("TagName = %q, want v2.0.0", info.TagName)
	}
	if info.Body != "Bug fixes and improvements" {
		t.Errorf("Body = %q, want release notes", info.Body)
	}
	if len(info.Assets) != 2 {
		t.Fatalf("len(Assets) = %d, want 2", len(info.Assets))
	}
	if info.Assets[0].Name != "Nearfy-macos.dmg" {
		t.Errorf("Asset[0].Name = %q", info.Assets[0].Name)
	}
	if info.Assets[0].Size != 12345 {
		t.Errorf("Asset[0].Size = %d, want 12345", info.Assets[0].Size)
	}
}

func TestCheckLatestRelease_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := checkLatestRelease(server.URL, "owner/repo")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestFindAsset(t *testing.T) {
	info := &ReleaseInfo{
		Assets: []Asset{
			{Name: "Nearfy-macos.dmg", URL: "https://example.com/macos.tar.gz", Size: 12345},
			{Name: "Nearfy-windows-amd64.exe", URL: "https://example.com/win.exe", Size: 67890},
		},
	}

	tests := []struct {
		goos     string
		wantName string
		wantOK   bool
	}{
		{"darwin", "Nearfy-macos.dmg", true},
		{"windows", "Nearfy-windows-amd64.exe", true},
		{"linux", "", false},
	}

	for _, tt := range tests {
		asset, ok := info.FindAsset(tt.goos)
		if ok != tt.wantOK {
			t.Errorf("FindAsset(%q) ok = %v, want %v", tt.goos, ok, tt.wantOK)
		}
		if ok && asset.Name != tt.wantName {
			t.Errorf("FindAsset(%q) name = %q, want %q", tt.goos, asset.Name, tt.wantName)
		}
	}
}

func TestCheckForUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v2.0.0",
			"body": "New release",
			"assets": []
		}`))
	}))
	defer server.Close()

	origBaseURL := defaultBaseURL
	defaultBaseURL = server.URL
	defer func() { defaultBaseURL = origBaseURL }()

	_, hasUpdate, err := CheckForUpdate("v1.0.0", "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasUpdate {
		t.Error("expected update available")
	}

	_, hasUpdate, err = CheckForUpdate("v2.0.0", "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasUpdate {
		t.Error("expected no update for same version")
	}
}
