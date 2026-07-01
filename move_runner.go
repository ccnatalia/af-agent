package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MoveFilePayload struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
}

type MoveFileResult struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	Bytes      int64  `json:"bytes"`
}

func executeMoveFileTask(payload json.RawMessage) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req MoveFilePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid move-file payload: %w", err)
	}

	req.SourcePath = strings.TrimSpace(req.SourcePath)
	req.TargetPath = strings.TrimSpace(req.TargetPath)
	if req.SourcePath == "" {
		return nil, errors.New("source_path is required")
	}
	if req.TargetPath == "" {
		return nil, errors.New("target_path is required")
	}

	sourcePath, err := workspacePath(req.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("source_path: %w", err)
	}

	targetPath, err := workspacePath(req.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("target_path: %w", err)
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("stat source file: %w", err)
	}
	if sourceInfo.IsDir() {
		return nil, errors.New("source_path must be a file")
	}

	if _, err := os.Stat(targetPath); err == nil {
		return nil, errors.New("target_path already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("stat target file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return nil, fmt.Errorf("create target dir: %w", err)
	}

	if err := os.Rename(sourcePath, targetPath); err != nil {
		return nil, fmt.Errorf("move file: %w", err)
	}

	return MoveFileResult{
		SourcePath: req.SourcePath,
		TargetPath: req.TargetPath,
		Bytes:      sourceInfo.Size(),
	}, nil
}

func workspacePath(inputPath string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	cleaned := filepath.Clean(inputPath)
	if filepath.IsAbs(cleaned) {
		return "", errors.New("absolute paths are not allowed")
	}

	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(wd, absPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path must stay inside workspace")
	}

	return absPath, nil
}
