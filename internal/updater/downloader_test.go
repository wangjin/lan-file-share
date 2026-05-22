package updater

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	content := make([]byte, 1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1024")
		w.Write(content)
	}))
	defer server.Close()

	var progressCalls []ProgressReport
	onProgress := func(p ProgressReport) {
		progressCalls = append(progressCalls, p)
	}

	destDir := t.TempDir()
	path, err := DownloadFile(server.URL+"/file", destDir, onProgress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if len(got) != 1024 {
		t.Errorf("file size = %d, want 1024", len(got))
	}

	if len(progressCalls) == 0 {
		t.Error("expected progress callbacks")
	}
	last := progressCalls[len(progressCalls)-1]
	if last.Percent != 100.0 {
		t.Errorf("last progress percent = %.1f, want 100.0", last.Percent)
	}
}

func TestDownloadFile_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	destDir := t.TempDir()
	_, err := DownloadFile(server.URL+"/file", destDir, nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestDownloadFile_VerifySize(t *testing.T) {
	content := []byte("short")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "5")
		w.Write(content)
	}))
	defer server.Close()

	destDir := t.TempDir()
	_, err := DownloadFile(server.URL+"/file", destDir, nil, WithExpectedSize(1000))
	if err == nil {
		t.Fatal("expected size mismatch error")
	}
}
