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
		prefix = "Nearfy-macos"
	case "windows":
		prefix = "Nearfy-windows"
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
	req.Header.Set("User-Agent", "nearfy-updater")

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
		return "Nearfy-macos.dmg"
	case "windows":
		return "Nearfy-windows-amd64.exe"
	default:
		return ""
	}
}
