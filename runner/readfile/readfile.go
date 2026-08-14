package readfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"afagent/runner/internal/workspace"
)

const Name = "read-file"
const MaxReadBytes int64 = 65536 // 64K

type Payload struct {
	Path     string `json:"path"`
	MaxBytes int64  `json:"max_bytes,omitempty"`
}

type Result struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Bytes     int64  `json:"bytes"`
	Size      int64  `json:"size"`
	Truncated bool   `json:"truncated"`
}

func Execute(payload json.RawMessage) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid read-file payload: %w", err)
	}

	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		return nil, errors.New("path is required")
	}
	if req.MaxBytes < 0 {
		return nil, errors.New("max_bytes must not be negative")
	}
	if req.MaxBytes == 0 {
		req.MaxBytes = MaxReadBytes
	}
	if req.MaxBytes > MaxReadBytes {
		return nil, fmt.Errorf("max_bytes must not exceed %d", MaxReadBytes)
	}

	absPath, err := workspace.Path(req.Path)
	if err != nil {
		return nil, fmt.Errorf("path: %w", err)
	}
	if err := rejectSymlinkPath(absPath); err != nil {
		return nil, err
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

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, req.MaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	truncated := int64(len(content)) > req.MaxBytes
	if truncated {
		content = content[:req.MaxBytes]
	}

	return Result{
		Path:      req.Path,
		Content:   string(content),
		Bytes:     int64(len(content)),
		Size:      info.Size(),
		Truncated: truncated,
	}, nil
}

func rejectSymlinkPath(absPath string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	relPath, err := filepath.Rel(wd, absPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	currentPath := wd
	for _, part := range strings.Split(relPath, string(filepath.Separator)) {
		currentPath = filepath.Join(currentPath, part)
		info, err := os.Lstat(currentPath)
		if err != nil {
			return fmt.Errorf("stat file: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("path must not contain symlinks")
		}
	}

	return nil
}
