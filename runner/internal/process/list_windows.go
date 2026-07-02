//go:build windows

package process

import (
	"encoding/json"
	"os/exec"
)

type windowsInfo struct {
	ProcessID   int    `json:"ProcessId"`
	CommandLine string `json:"CommandLine"`
}

func List() ([]Info, error) {
	output, err := exec.Command(
		"powershell",
		"-NoProfile",
		"-Command",
		"Get-CimInstance Win32_Process | Select-Object ProcessId,CommandLine | ConvertTo-Json -Compress",
	).Output()
	if err != nil {
		return nil, err
	}

	var many []windowsInfo
	if err := json.Unmarshal(output, &many); err != nil {
		var one windowsInfo
		if err := json.Unmarshal(output, &one); err != nil {
			return nil, err
		}
		many = append(many, one)
	}

	processes := make([]Info, 0, len(many))
	for _, process := range many {
		if process.ProcessID <= 0 || process.CommandLine == "" {
			continue
		}

		processes = append(processes, Info{
			PID:     process.ProcessID,
			Command: process.CommandLine,
		})
	}

	return processes, nil
}
