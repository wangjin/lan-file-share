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
	req.Header.Set("User-Agent", "nearfy-updater")

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
