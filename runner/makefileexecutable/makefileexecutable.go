package makefileexecutable

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"afagent/runner/internal/workspace"
)

const Name = "make-file-executable"

type Payload struct {
	Path string `json:"path"`
}

type Result struct {
	Path string `json:"path"`
}

func Execute(payload json.RawMessage) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid make-file-executable payload: %w", err)
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
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("path must not be a symlink")
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("path must be a file")
	}

	if err := makeExecutable(absPath, info); err != nil {
		return nil, fmt.Errorf("make file executable: %w", err)
	}

	return Result{
		Path: req.Path,
	}, nil
}
