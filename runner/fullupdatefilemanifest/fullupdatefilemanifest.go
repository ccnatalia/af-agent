package fullupdatefilemanifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"afagent/runner/filemanifest"
	"afagent/runner/internal/workspace"
)

const Name = "full-update-file-manifest"

type Payload struct {
	RootDir string `json:"root_dir"`
}

type Result struct {
	RootDir      string `json:"root_dir"`
	ManifestPath string `json:"manifest_path"`
}

func Execute(payload json.RawMessage) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid full-update-file-manifest payload: %w", err)
	}

	req.RootDir = strings.TrimSpace(req.RootDir)
	if req.RootDir == "" {
		return nil, errors.New("root_dir is required")
	}

	rootDir, err := workspace.Path(req.RootDir)
	if err != nil {
		return nil, fmt.Errorf("root_dir: %w", err)
	}
	if err := filemanifest.FullUpdate(rootDir); err != nil {
		return nil, err
	}

	return Result{
		RootDir:      filepath.Clean(req.RootDir),
		ManifestPath: filepath.Join(filepath.Clean(req.RootDir), filemanifest.Filename),
	}, nil
}
