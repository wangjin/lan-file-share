# 自动更新功能实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 LAN File Share 实现基于 GitHub Release 的自动更新，包含版本检测、自动下载替换、重启和回退机制。

**Architecture:** Go 后端 `internal/updater` 包负责版本比较、GitHub API 检测、下载和平台特定替换。Wails Service 桥接前后端，通过事件推送进度。前端使用 Toast + Modal 展示更新状态。

**Tech Stack:** Go 标准库 net/http、Wails v3 Service/Events、React + TypeScript

---

## File Structure

| File | Responsibility |
|------|---------------|
| `main.go` | 添加 `version` 变量，注册 UpdaterService |
| `internal/updater/version.go` | semver 解析和比较 |
| `internal/updater/version_test.go` | 版本解析/比较测试 |
| `internal/updater/checker.go` | GitHub Release API 请求 |
| `internal/updater/checker_test.go` | checker 测试（mock HTTP） |
| `internal/updater/downloader.go` | 文件下载 + 进度回调 |
| `internal/updater/downloader_test.go` | 下载测试（mock HTTP server） |
| `internal/updater/installer.go` | 平台特定替换和重启逻辑 |
| `internal/updater/service.go` | UpdaterService（Wails Service） |
| `internal/updater/service_test.go` | Service 集成测试 |
| `frontend/src/hooks/useUpdate.ts` | 更新状态管理 + 事件监听 |
| `frontend/src/components/UpdateToast.tsx` | 右下角 Toast 通知 |
| `frontend/src/components/UpdateModal.tsx` | 更新进度弹窗 |
| `frontend/src/App.tsx` | 集成更新组件 |
| `frontend/src/App.css` | 更新相关样式 |
| `Taskfile.yml` | 构建任务添加 ldflags |

---

### Task 1: 版本变量 + semver 解析比较

**Files:**
- Modify: `main.go`（添加 version 变量）
- Create: `internal/updater/version.go`
- Create: `internal/updater/version_test.go`

- [ ] **Step 1: 在 main.go 中添加 version 变量**

在 `package main` 声明区域、`import` 之前添加：

```go
var version = "dev"
```

修改位置：`main.go:8`（在 `import` 块之前）

- [ ] **Step 2: 编写 version_test.go 测试**

```go
package updater

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		major int
		minor int
		patch int
		ok    bool
	}{
		{"v1.2.3", 1, 2, 3, true},
		{"1.2.3", 1, 2, 3, true},
		{"v0.0.1", 0, 0, 1, true},
		{"v10.20.30", 10, 20, 30, true},
		{"dev", 0, 0, 0, false},
		{"v1", 0, 0, 0, false},
		{"v1.2", 0, 0, 0, false},
		{"", 0, 0, 0, false},
		{"abc", 0, 0, 0, false},
	}
	for _, tt := range tests {
		got, ok := ParseVersion(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseVersion(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok && got != (Version{tt.major, tt.minor, tt.patch}) {
			t.Errorf("ParseVersion(%q) = %+v, want {%d,%d,%d}", tt.input, got, tt.major, tt.minor, tt.patch)
		}
	}
}

func TestVersionGreaterThan(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"v2.0.0", "v1.9.9", true},
		{"v1.3.0", "v1.2.9", true},
		{"v1.2.4", "v1.2.3", true},
		{"v1.2.3", "v1.2.3", false},
		{"v1.2.2", "v1.2.3", false},
		{"v0.9.0", "v1.0.0", false},
	}
	for _, tt := range tests {
		a, _ := ParseVersion(tt.a)
		b, _ := ParseVersion(tt.b)
		if got := a.GreaterThan(b); got != tt.want {
			t.Errorf("ParseVersion(%q).GreaterThan(ParseVersion(%q)) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/updater/ -run TestParseVersion -v`
Expected: FAIL — `internal/updater` 包不存在

- [ ] **Step 4: 实现 version.go**

```go
package updater

import (
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(s string) (Version, bool) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return Version{}, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, false
	}
	return Version{Major: major, Minor: minor, Patch: patch}, true
}

func (v Version) GreaterThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	return v.Patch > other.Patch
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/updater/ -v`
Expected: PASS（全部测试通过）

- [ ] **Step 6: 提交**

```bash
git add main.go internal/updater/version.go internal/updater/version_test.go
git commit -m "feat(updater): add version variable and semver parsing"
```

---

### Task 2: GitHub Release 检测器

**Files:**
- Create: `internal/updater/checker.go`
- Create: `internal/updater/checker_test.go`

- [ ] **Step 1: 编写 checker_test.go**

```go
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
				{"name": "lan-file-share-macos.tar.gz", "browser_download_url": "https://example.com/macos.tar.gz", "size": 12345},
				{"name": "lan-file-share-windows-amd64.exe", "browser_download_url": "https://example.com/win.exe", "size": 67890}
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
	if info.Assets[0].Name != "lan-file-share-macos.tar.gz" {
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
			{Name: "lan-file-share-macos.tar.gz", URL: "https://example.com/macos.tar.gz", Size: 12345},
			{Name: "lan-file-share-windows-amd64.exe", URL: "https://example.com/win.exe", Size: 67890},
		},
	}

	tests := []struct {
		goos   string
		name   string
		wantOK bool
	}{
		{"darwin", "lan-file-share-macos.tar.gz", true},
		{"windows", "lan-file-share-windows-amd64.exe", true},
		{"linux", "", false},
	}

	for _, tt := range tests {
		asset, ok := info.FindAsset(tt.goos)
		if ok != tt.wantOK {
			t.Errorf("FindAsset(%q) ok = %v, want %v", tt.goos, ok, tt.wantOK)
		}
		if ok && asset.Name != tt.name {
			t.Errorf("FindAsset(%q) name = %q, want %q", tt.goos, asset.Name, tt.name)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/updater/ -run TestCheckLatest -v`
Expected: FAIL — 函数未定义

- [ ] **Step 3: 实现 checker.go**

```go
package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

type ReleaseInfo struct {
	TagName string  `json:"tag_name"`
	Body    string  `json:"body"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

var defaultBaseURL = "https://api.github.com"

var httpClient = &http.Client{Timeout: 15 * time.Second}

func (r *ReleaseInfo) FindAsset(goos string) (*Asset, bool) {
	var prefix string
	switch goos {
	case "darwin":
		prefix = "lan-file-share-macos"
	case "windows":
		prefix = "lan-file-share-windows"
	default:
		return nil, false
	}
	for i := range r.Assets {
		if len(r.Assets[i].Name) >= len(prefix) && r.Assets[i].Name[:len(prefix)] == prefix {
			return &r.Assets[i], true
		}
	}
	return nil, false
}

func checkLatestRelease(baseURL, repo string) (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", baseURL, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "lan-file-share-updater")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var info ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &info, nil
}

func CheckForUpdate(currentVersion, repo string) (*ReleaseInfo, bool, error) {
	if currentVersion == "dev" {
		info, err := checkLatestRelease(defaultBaseURL, repo)
		if err != nil {
			return nil, false, err
		}
		return info, true, nil
	}
	current, ok := ParseVersion(currentVersion)
	if !ok {
		return nil, false, fmt.Errorf("invalid current version: %s", currentVersion)
	}
	info, err := checkLatestRelease(defaultBaseURL, repo)
	if err != nil {
		return nil, false, err
	}
	latest, ok := ParseVersion(info.TagName)
	if !ok {
		return nil, false, fmt.Errorf("invalid remote version: %s", info.TagName)
	}
	if latest.GreaterThan(current) {
		return info, true, nil
	}
	return nil, false, nil
}

func PlatformAssetName() string {
	switch runtime.GOOS {
	case "darwin":
		return "lan-file-share-macos.tar.gz"
	case "windows":
		return "lan-file-share-windows-amd64.exe"
	default:
		return ""
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/updater/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/updater/checker.go internal/updater/checker_test.go
git commit -m "feat(updater): add GitHub release checker"
```

---

### Task 3: 文件下载器

**Files:**
- Create: `internal/updater/downloader.go`
- Create: `internal/updater/downloader_test.go`

- [ ] **Step 1: 编写 downloader_test.go**

```go
package updater

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		w.Header().Set("Content-Length", "1000")
		w.Write(content)
	}))
	defer server.Close()

	destDir := t.TempDir()
	_, err := DownloadFile(server.URL+"/file", destDir, nil, WithExpectedSize(1000))
	if err == nil {
		t.Fatal("expected size mismatch error")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/updater/ -run TestDownload -v`
Expected: FAIL — 函数未定义

- [ ] **Step 3: 实现 downloader.go**

```go
package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ProgressReport struct {
	Downloaded int64
	Total      int64
	Percent    float64
	Speed      float64
}

type ProgressCallback func(p ProgressReport)

type downloadOption struct {
	expectedSize int64
}

type DownloadOption func(*downloadOption)

func WithExpectedSize(size int64) DownloadOption {
	return func(o *downloadOption) { o.expectedSize = size }
}

func DownloadFile(url, destDir string, onProgress ProgressCallback, opts ...DownloadOption) (string, error) {
	opt := downloadOption{}
	for _, o := range opts {
		o(&opt)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "lan-file-share-updater")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	filename := filepath.Base(url)
	destPath := filepath.Join(destDir, filename)
	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	total := resp.ContentLength
	var downloaded int64
	start := time.Now()
	buf := make([]byte, 32*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			written, writeErr := f.Write(buf[:n])
			if writeErr != nil {
				os.Remove(destPath)
				return "", fmt.Errorf("write file: %w", writeErr)
			}
			downloaded += int64(written)
			if onProgress != nil && total > 0 {
				elapsed := time.Since(start).Seconds()
				speed := float64(0)
				if elapsed > 0 {
					speed = float64(downloaded) / elapsed
				}
				onProgress(ProgressReport{
					Downloaded: downloaded,
					Total:      total,
					Percent:    float64(downloaded) / float64(total) * 100,
					Speed:      speed,
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			os.Remove(destPath)
			return "", fmt.Errorf("read response: %w", readErr)
		}
	}

	if opt.expectedSize > 0 && downloaded != opt.expectedSize {
		os.Remove(destPath)
		return "", fmt.Errorf("size mismatch: downloaded %d, expected %d", downloaded, opt.expectedSize)
	}

	return destPath, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/updater/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/updater/downloader.go internal/updater/downloader_test.go
git commit -m "feat(updater): add file downloader with progress"
```

---

### Task 4: 平台特定安装器

**Files:**
- Create: `internal/updater/installer.go`

- [ ] **Step 1: 实现 installer.go**

```go
package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func findAppBundle() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return "", fmt.Errorf("abs path: %w", err)
	}
	dir := filepath.Dir(abs)
	for dir != "/" && dir != "." {
		if strings.HasSuffix(dir, ".app") {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("not running inside a .app bundle (exe: %s)", abs)
}

func installMacOS(downloadPath string) error {
	bundlePath, err := findAppBundle()
	if err != nil {
		return err
	}

	tmpExtract := filepath.Join(os.TempDir(), "lan-file-share-update")
	os.RemoveAll(tmpExtract)
	if err := os.MkdirAll(tmpExtract, 0755); err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	cmd := exec.Command("tar", "-xzf", downloadPath, "-C", tmpExtract)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract: %s: %w", string(out), err)
	}

	entries, err := os.ReadDir(tmpExtract)
	if err != nil {
		return fmt.Errorf("read extracted dir: %w", err)
	}
	var newApp string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") && e.IsDir() {
			newApp = filepath.Join(tmpExtract, e.Name())
			break
		}
	}
	if newApp == "" {
		return fmt.Errorf("no .app bundle found in archive")
	}

	cmd = exec.Command("cp", "-R", newApp+"/", bundlePath+"/")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("copy: %s: %w", string(out), err)
	}

	cmd = exec.Command("codesign", "--force", "--deep", "--sign", "-", bundlePath)
	cmd.CombinedOutput()

	return nil
}

func installWindows(downloadPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}

	bat := fmt.Sprintf(`@echo off
:wait
tasklist /fi "pid eq %d" 2>nul | find "%d" >nul
if %%errorlevel%%==0 (
    timeout /t 1 /nobreak >nul
    goto wait
)
copy /y "%s" "%s"
start "" "%s"
del "%%~f0"
`, os.Getpid(), os.Getpid(), downloadPath, abs, abs)

	batPath := filepath.Join(os.TempDir(), "lan-file-share-update.bat")
	if err := os.WriteFile(batPath, []byte(bat), 0644); err != nil {
		return fmt.Errorf("write bat: %w", err)
	}

	cmd := exec.Command("cmd", "/c", "start", "", "/b", batPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start updater: %w", err)
	}

	return nil
}

func InstallUpdate(downloadPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return installMacOS(downloadPath)
	case "windows":
		return installWindows(downloadPath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func Restart() error {
	switch runtime.GOOS {
	case "darwin":
		bundle, err := findAppBundle()
		if err != nil {
			return err
		}
		if err := exec.Command("open", bundle).Start(); err != nil {
			return fmt.Errorf("restart: %w", err)
		}
	case "windows":
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if err := exec.Command("cmd", "/c", "start", "", exe).Start(); err != nil {
			return fmt.Errorf("restart: %w", err)
		}
	default:
		return fmt.Errorf("unsupported platform")
	}
	os.Exit(0)
	return nil
}
```

- [ ] **Step 2: 提交**

```bash
git add internal/updater/installer.go
git commit -m "feat(updater): add platform-specific installer and restart"
```

---

### Task 5: UpdaterService（Wails Service）

**Files:**
- Create: `internal/updater/service.go`
- Create: `internal/updater/service_test.go`

- [ ] **Step 1: 编写 service_test.go**

```go
package updater

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	svc := &Service{
		version:     "v1.0.0",
		repo:        "owner/repo",
		baseURL:     server.URL,
		ignoreCheck: true,
	}

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

	svc := &Service{
		version:     "v1.0.0",
		repo:        "owner/repo",
		baseURL:     server.URL,
		ignoreCheck: true,
	}

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

	svc := &Service{
		version:     "dev",
		repo:        "owner/repo",
		baseURL:     server.URL,
		ignoreCheck: true,
	}

	_, hasUpdate, err := svc.CheckUpdate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasUpdate {
		t.Error("dev version should always show update available")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/updater/ -run TestService -v`
Expected: FAIL — Service 类型未定义

- [ ] **Step 3: 实现 service.go**

```go
package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	ctx         context.Context
	version     string
	repo        string
	baseURL     string
	ignoreCheck bool

	mu             sync.Mutex
	lastRelease    *ReleaseInfo
	downloadedPath string
	cancelDownload context.CancelFunc
}

func NewService(version, repo string) *Service {
	return &Service{
		version: version,
		repo:    repo,
		baseURL: defaultBaseURL,
	}
}

func (s *Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.ctx = ctx
	go func() {
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}
		s.CheckUpdate()

		ticker := time.NewTicker(4 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.CheckUpdate()
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.mu.Lock()
	if s.cancelDownload != nil {
		s.cancelDownload()
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) CheckUpdate() (*ReleaseInfo, bool, error) {
	info, hasUpdate, err := CheckForUpdate(s.version, s.repo)
	if err != nil {
		application.Get().Event.Emit("update:error", map[string]any{
			"error": err.Error(),
		})
		return nil, false, err
	}
	if hasUpdate {
		s.mu.Lock()
		s.lastRelease = info
		s.mu.Unlock()

		asset, ok := info.FindAsset(runtime.GOOS)
		assetURL := ""
		assetSize := int64(0)
		if ok {
			assetURL = asset.URL
			assetSize = asset.Size
		}

		application.Get().Event.Emit("update:available", map[string]any{
			"version":     info.TagName,
			"body":        info.Body,
			"downloadUrl": assetURL,
			"assetSize":   assetSize,
		})
	}
	return info, hasUpdate, nil
}

func (s *Service) StartDownload() error {
	s.mu.Lock()
	release := s.lastRelease
	s.mu.Unlock()

	if release == nil {
		return fmt.Errorf("no release info available")
	}

	asset, ok := release.FindAsset(runtime.GOOS)
	if !ok {
		return fmt.Errorf("no asset found for platform %s", runtime.GOOS)
	}

	ctx, cancel := context.WithCancel(s.ctx)
	s.mu.Lock()
	s.cancelDownload = cancel
	s.mu.Unlock()

	destDir := filepath.Join(os.TempDir(), "lan-file-share-update")
	go func() {
		defer cancel()
		path, err := DownloadFile(asset.URL, destDir, func(p ProgressReport) {
			application.Get().Event.Emit("update:progress", map[string]any{
				"downloaded": p.Downloaded,
				"total":      p.Total,
				"percent":    p.Percent,
				"speed":      p.Speed,
			})
		}, WithExpectedSize(asset.Size))

		if err != nil {
			select {
			case <-ctx.Done():
				application.Get().Event.Emit("update:error", map[string]any{
					"error": "download cancelled",
				})
			default:
				application.Get().Event.Emit("update:error", map[string]any{
					"error": err.Error(),
				})
			}
			return
		}

		s.mu.Lock()
		s.downloadedPath = path
		s.mu.Unlock()

		application.Get().Event.Emit("update:downloaded", map[string]any{
			"path": path,
		})
	}()

	return nil
}

func (s *Service) CancelDownload() {
	s.mu.Lock()
	if s.cancelDownload != nil {
		s.cancelDownload()
		s.cancelDownload = nil
	}
	s.mu.Unlock()
}

func (s *Service) InstallAndRestart() error {
	s.mu.Lock()
	path := s.downloadedPath
	s.mu.Unlock()

	if path == "" {
		return fmt.Errorf("no downloaded update")
	}

	if err := InstallUpdate(path); err != nil {
		return err
	}

	return Restart()
}

func (s *Service) GetVersion() string {
	return s.version
}

func (s *Service) OpenReleasePage() {
	url := fmt.Sprintf("https://github.com/%s/releases/latest", s.repo)
	application.Get().BrowserOpenURL(url)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/updater/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/updater/service.go internal/updater/service_test.go
git commit -m "feat(updater): add UpdaterService with Wails service interface"
```

---

### Task 6: 集成到 main.go + 更新 Taskfile.yml

**Files:**
- Modify: `main.go`
- Modify: `Taskfile.yml`

- [ ] **Step 1: 修改 main.go 注册 UpdaterService**

更新 import 和 Services 列表：

```go
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"lan-file-share/internal/updater"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed frontend/src/assets/images/logo.png
var iconData []byte

var version = "dev"

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
```

- [ ] **Step 2: 更新 Taskfile.yml 构建任务添加 ldflags**

在 `Taskfile.yml` 的 vars 部分添加 VERSION 变量，并修改所有 `go build` 命令：

vars 部分：

```yaml
vars:
  APP_NAME: lan-file-share
  FRONTEND_DIR: frontend
  BUILD_DIR: build/bin
  BINDINGS_DIR: frontend/bindings
  VERSION:
    sh: git describe --tags --always 2>/dev/null || echo "dev"
```

每个 `go build` 命令改为添加 `-ldflags`，以 `build` 任务为例：

```yaml
  build:
    desc: Build binary for current platform
    deps: [frontend:build]
    cmds:
      - go build -ldflags "-X main.version={{.VERSION}}" -o {{.BUILD_DIR}}/{{.APP_NAME}} .
```

需要修改的任务及其 go build 行（全部加 `-ldflags "-X main.version={{.VERSION}}"`）：

- `build:darwin:amd64`: `GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version={{.VERSION}}" -o {{.BUILD_DIR}}/{{.APP_NAME}}-darwin-amd64 .`
- `build:darwin:arm64`: `GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version={{.VERSION}}" -o {{.BUILD_DIR}}/{{.APP_NAME}}-darwin-arm64 .`
- `build:darwin`: `go build -ldflags "-X main.version={{.VERSION}}" -o {{.BUILD_DIR}}/{{.APP_NAME}} .`
- `build:windows:amd64`: `GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version={{.VERSION}}" -o {{.BUILD_DIR}}/{{.APP_NAME}}-windows-amd64.exe .`
- `build:linux:amd64`: `GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version={{.VERSION}}" -o {{.BUILD_DIR}}/{{.APP_NAME}}-linux-amd64 .`
- `build:linux:arm64`: `GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version={{.VERSION}}" -o {{.BUILD_DIR}}/{{.APP_NAME}}-linux-arm64 .`
- `build`: `go build -ldflags "-X main.version={{.VERSION}}" -o {{.BUILD_DIR}}/{{.APP_NAME}} .`

- [ ] **Step 3: 验证构建**

Run: `task build`
Expected: 成功构建，二进制文件在 `build/bin/lan-file-share`

- [ ] **Step 4: 提交**

```bash
git add main.go Taskfile.yml
git commit -m "feat(updater): integrate updater service and add ldflags to build"
```

---

### Task 7: 前端 useUpdate Hook

**Files:**
- Create: `frontend/src/hooks/useUpdate.ts`

- [ ] **Step 1: 实现 useUpdate.ts**

```typescript
import { useState, useEffect, useCallback } from 'react';
import { Events } from '@wailsio/runtime';

export interface UpdateInfo {
  version: string;
  body: string;
  downloadUrl: string;
  assetSize: number;
}

export interface DownloadProgress {
  downloaded: number;
  total: number;
  percent: number;
  speed: number;
}

export type UpdateStatus = 'idle' | 'available' | 'downloading' | 'downloaded' | 'error';

interface UpdateState {
  status: UpdateStatus;
  info: UpdateInfo | null;
  progress: DownloadProgress | null;
  error: string | null;
}

export function useUpdate() {
  const [state, setState] = useState<UpdateState>({
    status: 'idle',
    info: null,
    progress: null,
    error: null,
  });
  const [ignoredVersion, setIgnoredVersion] = useState<string | null>(() => {
    return localStorage.getItem('ignoredUpdateVersion');
  });

  useEffect(() => {
    const off1 = Events.On('update:available', (ev: any) => {
      const data = ev.data as UpdateInfo;
      if (data.version === ignoredVersion) return;
      setState({ status: 'available', info: data, progress: null, error: null });
    });

    const off2 = Events.On('update:progress', (ev: any) => {
      setState(prev => ({
        ...prev,
        status: 'downloading',
        progress: ev.data as DownloadProgress,
      }));
    });

    const off3 = Events.On('update:downloaded', () => {
      setState(prev => ({ ...prev, status: 'downloaded', progress: null }));
    });

    const off4 = Events.On('update:error', (ev: any) => {
      setState(prev => ({
        ...prev,
        status: 'error',
        error: (ev.data as { error: string }).error,
      }));
    });

    return () => {
      off1();
      off2();
      off3();
      off4();
    };
  }, [ignoredVersion]);

  const ignoreUpdate = useCallback(() => {
    if (state.info) {
      const v = state.info.version;
      setIgnoredVersion(v);
      localStorage.setItem('ignoredUpdateVersion', v);
      setState({ status: 'idle', info: null, progress: null, error: null });
    }
  }, [state.info]);

  return {
    ...state,
    ignoredVersion,
    ignoreUpdate,
  };
}
```

- [ ] **Step 2: 提交**

```bash
git add frontend/src/hooks/useUpdate.ts
git commit -m "feat(updater): add useUpdate hook for frontend state management"
```

---

### Task 8: UpdateToast 组件

**Files:**
- Create: `frontend/src/components/UpdateToast.tsx`

- [ ] **Step 1: 实现 UpdateToast.tsx**

```tsx
import React from 'react';
import { UpdateInfo } from '../hooks/useUpdate';

interface Props {
  info: UpdateInfo;
  onViewDetails: () => void;
  onDismiss: () => void;
}

export const UpdateToast: React.FC<Props> = ({ info, onViewDetails, onDismiss }) => (
  <div className="update-toast">
    <div className="update-toast-content">
      <div className="update-toast-title">发现新版本 {info.version}</div>
      <div className="update-toast-actions">
        <button className="update-toast-btn primary" onClick={onViewDetails}>查看详情</button>
        <button className="update-toast-btn" onClick={onDismiss}>忽略</button>
      </div>
    </div>
  </div>
);
```

- [ ] **Step 2: 添加 Toast 样式到 App.css**

在 `frontend/src/App.css` 末尾追加：

```css
/* ---------- Update Toast ---------- */
.update-toast {
  position: fixed;
  bottom: 20px;
  right: 20px;
  background: #1c2333;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 14px 18px;
  z-index: 1000;
  animation: toast-slide-in 0.3s ease-out;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
}

.update-toast-title {
  font-size: 14px;
  color: #e6edf3;
  margin-bottom: 10px;
}

.update-toast-actions {
  display: flex;
  gap: 8px;
}

.update-toast-btn {
  padding: 5px 14px;
  border: 1px solid #30363d;
  border-radius: 4px;
  background: transparent;
  color: #8b949e;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s;
}

.update-toast-btn:hover {
  border-color: #5b8def;
  color: #5b8def;
}

.update-toast-btn.primary {
  background: #5b8def;
  border-color: #5b8def;
  color: #fff;
}

.update-toast-btn.primary:hover {
  background: #4a7de0;
}

@keyframes toast-slide-in {
  from { transform: translateY(20px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/components/UpdateToast.tsx frontend/src/App.css
git commit -m "feat(updater): add UpdateToast notification component"
```

---

### Task 9: UpdateModal 组件

**Files:**
- Create: `frontend/src/components/UpdateModal.tsx`

- [ ] **Step 1: 实现 UpdateModal.tsx**

```tsx
import React from 'react';
import { UpdateInfo, DownloadProgress, UpdateStatus } from '../hooks/useUpdate';

interface Props {
  visible: boolean;
  status: UpdateStatus;
  info: UpdateInfo | null;
  progress: DownloadProgress | null;
  error: string | null;
  onStartDownload: () => void;
  onCancel: () => void;
  onInstall: () => void;
  onManualDownload: () => void;
  onClose: () => void;
}

function formatSpeed(bytesPerSec: number): string {
  if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(0)} B/s`;
  if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`;
  return `${(bytesPerSec / (1024 * 1024)).toFixed(1)} MB/s`;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export const UpdateModal: React.FC<Props> = ({
  visible, status, info, progress, error,
  onStartDownload, onCancel, onInstall, onManualDownload, onClose,
}) => {
  if (!visible || !info) return null;

  return (
    <div className="update-modal-overlay" onClick={status === 'downloading' ? undefined : onClose}>
      <div className="update-modal" onClick={e => e.stopPropagation()}>
        <div className="update-modal-header">
          <h3>更新到 {info.version}</h3>
          {status !== 'downloading' && (
            <button className="update-modal-close" onClick={onClose}>✕</button>
          )}
        </div>

        {info.body && (
          <div className="update-modal-body">
            {info.body.split('\n').map((line, i) => (
              <p key={i}>{line}</p>
            ))}
          </div>
        )}

        {status === 'downloading' && progress && (
          <div className="update-modal-progress">
            <div className="progress-bar">
              <div className="progress-bar-fill" style={{ width: `${progress.percent}%` }} />
            </div>
            <div className="progress-info">
              <span>{progress.percent.toFixed(1)}%</span>
              <span>{formatSpeed(progress.speed)}</span>
              <span>{formatSize(progress.downloaded)} / {formatSize(progress.total)}</span>
            </div>
          </div>
        )}

        {status === 'error' && error && (
          <div className="update-modal-error">
            下载失败：{error}
          </div>
        )}

        <div className="update-modal-actions">
          {status === 'available' && (
            <>
              <button className="btn primary" onClick={onStartDownload}>立即更新</button>
              <button className="btn" onClick={onManualDownload}>手动下载</button>
              <button className="btn" onClick={onClose}>关闭</button>
            </>
          )}
          {status === 'downloading' && (
            <button className="btn" onClick={onCancel}>取消</button>
          )}
          {status === 'downloaded' && (
            <button className="btn primary" onClick={onInstall}>重启并安装</button>
          )}
          {status === 'error' && (
            <button className="btn primary" onClick={onManualDownload}>手动下载</button>
          )}
        </div>
      </div>
    </div>
  );
};
```

- [ ] **Step 2: 添加 Modal 样式到 App.css**

追加到 `frontend/src/App.css` 末尾：

```css
/* ---------- Update Modal ---------- */
.update-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.update-modal {
  background: #1c2333;
  border: 1px solid #30363d;
  border-radius: 10px;
  width: 420px;
  max-height: 80vh;
  overflow-y: auto;
  box-shadow: 0 8px 40px rgba(0, 0, 0, 0.5);
}

.update-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #30363d;
}

.update-modal-header h3 {
  margin: 0;
  font-size: 16px;
  color: #e6edf3;
}

.update-modal-close {
  background: none;
  border: none;
  color: #8b949e;
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
}

.update-modal-close:hover {
  color: #e6edf3;
}

.update-modal-body {
  padding: 16px 20px;
  color: #8b949e;
  font-size: 13px;
  line-height: 1.6;
}

.update-modal-body p {
  margin: 0 0 4px;
}

.update-modal-progress {
  padding: 16px 20px;
}

.progress-bar {
  height: 6px;
  background: #30363d;
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 8px;
}

.progress-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #e94560, #5b8def);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #8b949e;
}

.update-modal-error {
  padding: 12px 20px;
  color: #e94560;
  font-size: 13px;
}

.update-modal-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid #30363d;
}

.update-modal-actions .btn {
  padding: 7px 18px;
  border: 1px solid #30363d;
  border-radius: 6px;
  background: transparent;
  color: #e6edf3;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s;
}

.update-modal-actions .btn:hover {
  border-color: #5b8def;
}

.update-modal-actions .btn.primary {
  background: #5b8def;
  border-color: #5b8def;
  color: #fff;
}

.update-modal-actions .btn.primary:hover {
  background: #4a7de0;
}
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/components/UpdateModal.tsx frontend/src/App.css
git commit -m "feat(updater): add UpdateModal with progress and states"
```

---

### Task 10: 集成到 App.tsx

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: 重新生成 TypeScript 绑定**

Run: `task generate`
Expected: 生成 `frontend/bindings/lan-file-share/updater/` 目录

- [ ] **Step 2: 更新 App.tsx 集成更新组件**

```tsx
import { useState, useCallback } from 'react';
import { useDevices } from './hooks/useDevices';
import { useTransfers } from './hooks/useTransfers';
import { useDragDrop } from './hooks/useDragDrop';
import { useUpdate } from './hooks/useUpdate';
import { SendPaths } from '../bindings/lan-file-share/app.js';
import {
  StartDownload,
  CancelDownload,
  InstallAndRestart,
  OpenReleasePage,
} from '../bindings/lan-file-share/updater/Service.js';
import { Sidebar } from './components/Sidebar';
import { TopBar } from './components/TopBar';
import { TransferPanel } from './components/TransferPanel';
import { UpdateToast } from './components/UpdateToast';
import { UpdateModal } from './components/UpdateModal';
import './App.css';

function App() {
  const { devices, localInfo } = useDevices();
  const { tasks, sendFile, respondReceive, cancelTask } = useTransfers();
  const { status, info, progress, error, ignoreUpdate } = useUpdate();
  const [selectedPeerId, setSelectedPeerId] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);
  const selectedDevice = devices.find(d => d.node_id === selectedPeerId);

  const handleDrop = async (paths: string[]) => {
    if (!selectedPeerId) return;
    await SendPaths(selectedPeerId, paths);
  };

  const handleViewDetails = useCallback(() => {
    setShowModal(true);
  }, []);

  const handleStartDownload = useCallback(async () => {
    await StartDownload();
  }, []);

  const handleCancel = useCallback(async () => {
    await CancelDownload();
    setShowModal(false);
  }, []);

  const handleInstall = useCallback(async () => {
    await InstallAndRestart();
  }, []);

  const handleManualDownload = useCallback(async () => {
    setShowModal(false);
    await OpenReleasePage();
  }, []);

  const handleDismissToast = useCallback(() => {
    ignoreUpdate();
  }, [ignoreUpdate]);

  const handleDismissModal = useCallback(() => {
    if (status !== 'downloading') {
      setShowModal(false);
    }
  }, [status]);

  const { isDragging, handlers: dragHandlers } = useDragDrop(handleDrop);

  return (
    <div className="app">
      <Sidebar
        devices={devices}
        localInfo={localInfo}
        selectedPeerId={selectedPeerId}
        onSelectDevice={setSelectedPeerId}
      />
      <div className="main">
        <TopBar
          device={selectedDevice}
          onSendFile={() => selectedPeerId && sendFile(selectedPeerId)}
        />
        <TransferPanel
          tasks={tasks}
          peerId={selectedPeerId}
          deviceName={selectedDevice?.name}
          onCancel={cancelTask}
          onRespond={respondReceive}
          isDragging={isDragging}
          dragHandlers={dragHandlers}
        />
      </div>
      {status === 'available' && !showModal && info && (
        <UpdateToast info={info} onViewDetails={handleViewDetails} onDismiss={handleDismissToast} />
      )}
      <UpdateModal
        visible={showModal}
        status={status}
        info={info}
        progress={progress}
        error={error}
        onStartDownload={handleStartDownload}
        onCancel={handleCancel}
        onInstall={handleInstall}
        onManualDownload={handleManualDownload}
        onClose={handleDismissModal}
      />
    </div>
  );
}

export default App;
```

**注意：** 绑定导入路径取决于 Task 6 之后 `task generate` 的实际输出路径。如果绑定生成到其他路径，需相应调整 import。检查 `frontend/bindings/` 目录确认实际路径。

- [ ] **Step 3: 验证前端构建**

Run: `cd frontend && npm run build`
Expected: 编译成功，无 TypeScript 错误

- [ ] **Step 4: 提交**

```bash
git add frontend/src/App.tsx
git commit -m "feat(updater): integrate update UI into App"
```

---

### Task 11: 端到端验证

**Files:** 无新文件

- [ ] **Step 1: 运行完整测试套件**

Run: `go test ./... -v`
Expected: 全部通过

- [ ] **Step 2: 构建并运行应用**

Run: `task build && ./build/bin/lan-file-share`
Expected: 应用正常启动，5 秒后控制台无 panic（版本为 "dev" 时会检测到 v1.0.0 有更新，弹出 Toast）

- [ ] **Step 3: 验证 Toast 和 Modal 交互**

手动操作：
1. 等待 Toast 出现在右下角
2. 点击"查看详情"打开 Modal
3. 点击"手动下载"验证浏览器打开 GitHub Release 页面
4. 关闭应用

- [ ] **Step 4: 最终提交**

```bash
git add -A
git commit -m "feat: complete auto-update feature"
```
