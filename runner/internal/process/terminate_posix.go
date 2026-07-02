//go:build !windows

package process

import (
	"os"
	"syscall"
)

func Terminate(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return process.Signal(syscall.SIGTERM)
}
