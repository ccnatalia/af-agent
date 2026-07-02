package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func Path(inputPath string) (string, error) {
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
