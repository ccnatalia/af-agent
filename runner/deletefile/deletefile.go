package deletefile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"afagent/runner/internal/workspace"
)

const Name = "delete-file"

type Payload struct {
	Path string `json:"path"`
}

type Result struct {
	Path    string `json:"path"`
	Deleted bool   `json:"deleted"`
	Bytes   int64  `json:"bytes"`
}

func Execute(payload json.RawMessage) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid delete-file payload: %w", err)
	}

	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		return nil, errors.New("path is required")
	}

	absPath, err := workspace.Path(req.Path)
	if err != nil {
		return nil, fmt.Errorf("path: %w", err)
	}

	info, err := os.Lstat(absPath)
	if errors.Is(err, os.ErrNotExist) {
		return Result{
			Path: req.Path,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if info.IsDir() {
		return nil, errors.New("path must be a file")
	}

	if err := os.Remove(absPath); err != nil {
		return nil, fmt.Errorf("delete file: %w", err)
	}

	return Result{
		Path:    req.Path,
		Deleted: true,
		Bytes:   info.Size(),
	}, nil
}
