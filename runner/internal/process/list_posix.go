//go:build !windows

package process

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

func List() ([]Info, error) {
	output, err := exec.Command("ps", "-eo", "pid=,args=").Output()
	if err != nil {
		return nil, err
	}

	var processes []Info
	for _, line := range bytes.Split(output, []byte{'\n'}) {
		text := strings.TrimSpace(string(line))
		fields := strings.Fields(text)
		if len(fields) < 2 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		processes = append(processes, Info{
			PID:     pid,
			Command: strings.TrimSpace(strings.TrimPrefix(text, fields[0])),
		})
	}

	return processes, nil
}
