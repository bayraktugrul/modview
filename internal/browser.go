package internal

import (
	"fmt"
	"os/exec"
	"runtime"
)

func OpenInBrowser(path string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{path}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", path}
	case "darwin":
		cmd = "open"
		args = []string{path}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}
