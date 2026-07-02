package processexists

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	processlib "afagent/runner/internal/process"
)

const Name = "process-exists"

type Payload struct {
	Keyword string `json:"keyword"`
}

type Result struct {
	Keyword   string               `json:"keyword"`
	Exists    bool                 `json:"exists"`
	Count     int                  `json:"count"`
	Processes []processlib.Matched `json:"processes"`
}

func Execute(payload json.RawMessage) (any, error) {
	return executeWithList(payload, os.Getpid(), processlib.List)
}

func executeWithList(
	payload json.RawMessage,
	currentPID int,
	list func() ([]processlib.Info, error),
) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid process-exists payload: %w", err)
	}

	keyword, err := processlib.NormalizeKeyword(req.Keyword)
	if err != nil {
		return nil, err
	}

	processes, err := processlib.FindMatching(keyword, currentPID, list)
	if err != nil {
		return nil, err
	}
	if processes == nil {
		processes = []processlib.Matched{}
	}

	return Result{
		Keyword:   keyword,
		Exists:    len(processes) > 0,
		Count:     len(processes),
		Processes: processes,
	}, nil
}
