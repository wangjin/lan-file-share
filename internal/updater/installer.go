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

	exec.Command("codesign", "--force", "--deep", "--sign", "-", bundlePath).CombinedOutput()

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

	pid := os.Getpid()
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
`, pid, pid, downloadPath, abs, abs)

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
