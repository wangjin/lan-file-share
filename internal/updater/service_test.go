package updater

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestService(version string, server *httptest.Server) *Service {
	svc := &Service{
		version:   version,
		repo:      "owner/repo",
		baseURL:   server.URL,
		emitEvent: func(string, map[string]any) {},
	}
	return svc
}

func TestService_CheckUpdate_NewAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v99.0.0",
			"body": "Test release",
			"assets": [
				{"name": "lan-file-share-macos.tar.gz", "browser_download_url": "http://example.com/macos.tar.gz", "size": 100}
			]
		}`))
	}))
	defer server.Close()

	svc := newTestService("v1.0.0", server)

	info, hasUpdate, err := svc.CheckUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasUpdate {
		t.Error("expected update available")
	}
	if info.TagName != "v99.0.0" {
		t.Errorf("TagName = %q, want v99.0.0", info.TagName)
	}
}

func TestService_CheckUpdate_UpToDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v1.0.0",
			"body": "",
			"assets": []
		}`))
	}))
	defer server.Close()

	svc := newTestService("v1.0.0", server)

	_, hasUpdate, err := svc.CheckUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasUpdate {
		t.Error("expected no update")
	}
}

func TestService_CheckUpdate_DevVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v0.0.1",
			"body": "",
			"assets": []
		}`))
	}))
	defer server.Close()

	svc := newTestService("dev", server)

	_, hasUpdate, err := svc.CheckUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasUpdate {
		t.Error("dev version should always show update available")
	}
}

func TestService_GetVersion(t *testing.T) {
	svc := &Service{version: "v1.2.3", emitEvent: func(string, map[string]any) {}}
	if v := svc.GetVersion(); v != "v1.2.3" {
		t.Errorf("GetVersion() = %q, want v1.2.3", v)
	}
}
