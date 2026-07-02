//go:build windows

package process

import (
	"os/exec"
	"strconv"
)

func Terminate(pid int) error {
	return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T").Run()
}
