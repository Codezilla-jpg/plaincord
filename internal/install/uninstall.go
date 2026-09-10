package install

import (
	"os"
	"path/filepath"
	"runtime"
)

var binaries = []string{"dis", "DiscordCli", "dc"}

func Uninstall(binDir, configDir string) ([]string, error) {
	var removed []string
	if binDir == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, err
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		binDir = filepath.Dir(exe)
	}
	names := append([]string{}, binaries...)
	if runtime.GOOS == "windows" {
		for i, n := range names {
			names[i] = n + ".exe"
		}
	}
	for _, name := range names {
		path := filepath.Join(binDir, name)
		info, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return removed, err
		}
		if info.IsDir() {
			continue
		}
		if err := os.Remove(path); err != nil {
			return removed, err
		}
		removed = append(removed, path)
	}
	if configDir != "" && filepath.Base(configDir) == "plaincord" {
		if _, err := os.Stat(configDir); err == nil {
			if err := os.RemoveAll(configDir); err != nil {
				return removed, err
			}
			removed = append(removed, configDir)
		}
	}
	return removed, nil
}
