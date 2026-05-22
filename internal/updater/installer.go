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

	mountPoint := filepath.Join(os.TempDir(), "lan-file-share-dmg")
	os.RemoveAll(mountPoint)

	cmd := exec.Command("hdiutil", "attach", downloadPath, "-mountpoint", mountPoint, "-nobrowse", "-quiet")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount dmg: %s: %w", string(out), err)
	}
	defer exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run()

	entries, err := os.ReadDir(mountPoint)
	if err != nil {
		return fmt.Errorf("read mounted dmg: %w", err)
	}
	var newApp string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") && e.IsDir() {
			newApp = filepath.Join(mountPoint, e.Name())
			break
		}
	}
	if newApp == "" {
		return fmt.Errorf("no .app bundle found in dmg")
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
