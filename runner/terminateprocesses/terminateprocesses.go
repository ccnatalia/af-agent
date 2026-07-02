package terminateprocesses

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	processlib "afagent/runner/internal/process"
)

const Name = "terminate-processes"

type Payload struct {
	Keyword string `json:"keyword"`
}

type Result struct {
	Keyword   string              `json:"keyword"`
	Count     int                 `json:"count"`
	Processes []TerminatedProcess `json:"processes"`
}

type TerminatedProcess struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
}

func Execute(payload json.RawMessage) (any, error) {
	return executeWithDeps(payload, os.Getpid(), processlib.List, processlib.Terminate)
}

func executeWithDeps(
	payload json.RawMessage,
	currentPID int,
	list func() ([]processlib.Info, error),
	terminate func(pid int) error,
) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid terminate-processes payload: %w", err)
	}

	keyword, err := processlib.NormalizeKeyword(req.Keyword)
	if err != nil {
		return nil, err
	}

	processes, err := terminateMatching(keyword, currentPID, list, terminate)
	if err != nil {
		return nil, err
	}

	return Result{
		Keyword:   keyword,
		Count:     len(processes),
		Processes: processes,
	}, nil
}

func terminateMatching(
	keyword string,
	currentPID int,
	list func() ([]processlib.Info, error),
	terminate func(pid int) error,
) ([]TerminatedProcess, error) {
	matches, err := processlib.FindMatching(keyword, currentPID, list)
	if err != nil {
		return nil, err
	}

	var terminated []TerminatedProcess
	for _, process := range matches {
		if err := terminate(process.PID); err != nil {
			return nil, fmt.Errorf("terminate process %d: %w", process.PID, err)
		}

		terminated = append(terminated, TerminatedProcess{
			PID:     process.PID,
			Command: process.Command,
		})
	}

	return terminated, nil
}
