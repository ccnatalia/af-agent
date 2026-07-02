package fileexists

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"afagent/runner/internal/workspace"
)

const Name = "file-exists"

type Payload struct {
	Path string `json:"path"`
}

type Result struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	IsFile bool   `json:"is_file"`
	IsDir  bool   `json:"is_dir"`
	Size   int64  `json:"size"`
}

func Execute(payload json.RawMessage) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid file-exists payload: %w", err)
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
		return nil, fmt.Errorf("stat path: %w", err)
	}

	return Result{
		Path:   req.Path,
		Exists: true,
		IsFile: info.Mode().IsRegular(),
		IsDir:  info.IsDir(),
		Size:   info.Size(),
	}, nil
}
