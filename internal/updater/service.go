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
	ctx     context.Context
	version string
	repo    string
	baseURL string

	mu             sync.Mutex
	lastRelease    *ReleaseInfo
	downloadedPath string
	cancelDownload context.CancelFunc

	emitEvent func(event string, data map[string]any)
}

func NewService(version, repo string) *Service {
	return &Service{
		version: version,
		repo:    repo,
		baseURL: defaultBaseURL,
		emitEvent: func(event string, data map[string]any) {
			application.Get().Event.Emit(event, data)
		},
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
	baseURL := defaultBaseURL
	if s.baseURL != "" {
		baseURL = s.baseURL
	}

	currentVersion := s.version
	if currentVersion == "dev" {
		info, err := checkLatestRelease(baseURL, s.repo)
		if err != nil {
			s.emitEvent("update:error", map[string]any{"error": err.Error()})
			return nil, false, err
		}
		s.mu.Lock()
		s.lastRelease = info
		s.mu.Unlock()
		s.emitAvailable(info)
		return info, true, nil
	}

	current, ok := ParseVersion(currentVersion)
	if !ok {
		err := fmt.Errorf("invalid current version: %s", currentVersion)
		s.emitEvent("update:error", map[string]any{"error": err.Error()})
		return nil, false, err
	}

	info, err := checkLatestRelease(baseURL, s.repo)
	if err != nil {
		s.emitEvent("update:error", map[string]any{"error": err.Error()})
		return nil, false, err
	}

	latest, ok := ParseVersion(info.TagName)
	if !ok {
		err := fmt.Errorf("invalid remote version: %s", info.TagName)
		s.emitEvent("update:error", map[string]any{"error": err.Error()})
		return nil, false, err
	}

	if !latest.GreaterThan(current) {
		return nil, false, nil
	}

	s.mu.Lock()
	s.lastRelease = info
	s.mu.Unlock()
	s.emitAvailable(info)
	return info, true, nil
}

func (s *Service) emitAvailable(info *ReleaseInfo) {
	asset, ok := info.FindAsset(runtime.GOOS)
	assetURL := ""
	assetSize := int64(0)
	if ok {
		assetURL = asset.URL
		assetSize = asset.Size
	}
	s.emitEvent("update:available", map[string]any{
		"version":     info.TagName,
		"body":        info.Body,
		"downloadUrl": assetURL,
		"assetSize":   assetSize,
	})
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

	destDir := filepath.Join(os.TempDir(), "nearfy-update")
	go func() {
		defer cancel()
		path, err := DownloadFile(asset.URL, destDir, func(p ProgressReport) {
			s.emitEvent("update:progress", map[string]any{
				"downloaded": p.Downloaded,
				"total":      p.Total,
				"percent":    p.Percent,
				"speed":      p.Speed,
			})
		}, WithExpectedSize(asset.Size))

		if err != nil {
			select {
			case <-ctx.Done():
				s.emitEvent("update:error", map[string]any{"error": "download cancelled"})
			default:
				s.emitEvent("update:error", map[string]any{"error": err.Error()})
			}
			return
		}

		s.mu.Lock()
		s.downloadedPath = path
		s.mu.Unlock()

		s.emitEvent("update:downloaded", map[string]any{"path": path})
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
	application.Get().Browser.OpenURL(url)
}
