package runstartupscript

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"afagent/runner/internal/workspace"
)

const Name = "run-startup-script"

const defaultTimeoutSeconds = 30
const maxTimeoutSeconds = 300
const maxOutputBytes = 64 << 10

type Payload struct {
	Path           string   `json:"path"`
	Args           []string `json:"args,omitempty"`
	WorkingDir     string   `json:"working_dir,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
}

type Result struct {
	Path       string   `json:"path"`
	Args       []string `json:"args"`
	WorkingDir string   `json:"working_dir"`
	ExitCode   int      `json:"exit_code"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	DurationMS int64    `json:"duration_ms"`
}

func Execute(payload json.RawMessage) (any, error) {
	return executeWithCommand(payload, runCommand)
}

type commandRunner func(scriptPath string, args []string, workingDir string, timeout time.Duration) (commandResult, error)

type commandResult struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	DurationMS int64
}

func executeWithCommand(payload json.RawMessage, run commandRunner) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid run-startup-script payload: %w", err)
	}

	normalized, scriptPath, workingDir, timeout, err := normalize(req)
	if err != nil {
		return nil, err
	}

	commandResult, err := run(scriptPath, normalized.Args, workingDir, timeout)
	if err != nil {
		return nil, err
	}

	return Result{
		Path:       normalized.Path,
		Args:       normalized.Args,
		WorkingDir: normalized.WorkingDir,
		ExitCode:   commandResult.ExitCode,
		Stdout:     commandResult.Stdout,
		Stderr:     commandResult.Stderr,
		DurationMS: commandResult.DurationMS,
	}, nil
}

func normalize(req Payload) (Payload, string, string, time.Duration, error) {
	req.Path = strings.TrimSpace(req.Path)
	req.WorkingDir = strings.TrimSpace(req.WorkingDir)
	if req.WorkingDir == "" {
		req.WorkingDir = "."
	}
	if req.Args == nil {
		req.Args = []string{}
	}

	if req.Path == "" {
		return Payload{}, "", "", 0, errors.New("path is required")
	}
	if err := validateArgs(req.Args); err != nil {
		return Payload{}, "", "", 0, err
	}

	commandPath, err := resolveCommandPath(req.Path)
	if err != nil {
		return Payload{}, "", "", 0, err
	}

	workingDir, err := workspace.Path(req.WorkingDir)
	if err != nil {
		return Payload{}, "", "", 0, fmt.Errorf("working_dir: %w", err)
	}
	if err := validateWorkingDir(workingDir); err != nil {
		return Payload{}, "", "", 0, err
	}

	timeout, err := normalizeTimeout(req.TimeoutSeconds)
	if err != nil {
		return Payload{}, "", "", 0, err
	}

	return req, commandPath, workingDir, timeout, nil
}

func resolveCommandPath(path string) (string, error) {
	workspacePath, err := workspace.Path(path)
	if err != nil {
		return "", fmt.Errorf("path: %w", err)
	}

	if err := validateScript(workspacePath); err == nil {
		return workspacePath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if strings.ContainsRune(path, os.PathSeparator) {
		return "", fmt.Errorf("stat path: %w", os.ErrNotExist)
	}

	commandPath, err := exec.LookPath(path)
	if err != nil {
		return "", fmt.Errorf("look up command: %w", err)
	}

	return commandPath, nil
}

func validateArgs(args []string) error {
	for _, arg := range args {
		if strings.ContainsAny(arg, "\x00\r\n") {
			return errors.New("args must not contain control characters")
		}
	}

	return nil
}

func validateScript(scriptPath string) error {
	info, err := os.Lstat(scriptPath)
	if err != nil {
		return fmt.Errorf("stat path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("path must not be a symlink")
	}
	if !info.Mode().IsRegular() {
		return errors.New("path must be a file")
	}
	if !isExecutable(info) {
		return errors.New("path must be executable")
	}

	return nil
}

func validateWorkingDir(workingDir string) error {
	info, err := os.Stat(workingDir)
	if err != nil {
		return fmt.Errorf("stat working_dir: %w", err)
	}
	if !info.IsDir() {
		return errors.New("working_dir must be a directory")
	}

	return nil
}

func normalizeTimeout(timeoutSeconds int) (time.Duration, error) {
	if timeoutSeconds == 0 {
		timeoutSeconds = defaultTimeoutSeconds
	}
	if timeoutSeconds < 0 {
		return 0, errors.New("timeout_seconds must be positive")
	}
	if timeoutSeconds > maxTimeoutSeconds {
		return 0, fmt.Errorf("timeout_seconds must be at most %d", maxTimeoutSeconds)
	}

	return time.Duration(timeoutSeconds) * time.Second, nil
}

func runCommand(scriptPath string, args []string, workingDir string, timeout time.Duration) (commandResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startedAt := time.Now()
	cmd := exec.CommandContext(ctx, scriptPath, args...)
	cmd.Dir = workingDir

	stdoutFile, err := os.CreateTemp("", "af-startup-stdout-*.log")
	if err != nil {
		return commandResult{}, fmt.Errorf("create stdout file: %w", err)
	}
	defer os.Remove(stdoutFile.Name())
	defer stdoutFile.Close()

	stderrFile, err := os.CreateTemp("", "af-startup-stderr-*.log")
	if err != nil {
		return commandResult{}, fmt.Errorf("create stderr file: %w", err)
	}
	defer os.Remove(stderrFile.Name())
	defer stderrFile.Close()

	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	err = cmd.Run()
	durationMS := time.Since(startedAt).Milliseconds()
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	stdout, readStdoutErr := readLimitedFile(stdoutFile)
	stderr, readStderrErr := readLimitedFile(stderrFile)
	if readStdoutErr != nil {
		return commandResult{}, fmt.Errorf("read stdout: %w", readStdoutErr)
	}
	if readStderrErr != nil {
		return commandResult{}, fmt.Errorf("read stderr: %w", readStderrErr)
	}

	result := commandResult{
		ExitCode:   exitCode,
		Stdout:     stdout,
		Stderr:     stderr,
		DurationMS: durationMS,
	}

	if ctx.Err() == context.DeadlineExceeded {
		return commandResult{}, fmt.Errorf("startup script timed out after %s", timeout)
	}
	if err != nil {
		return commandResult{}, fmt.Errorf("startup script failed with exit code %d: %w", result.ExitCode, err)
	}

	return result, nil
}

func readLimitedFile(file *os.File) (string, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}

	content, err := io.ReadAll(io.LimitReader(file, maxOutputBytes))
	if err != nil {
		return "", err
	}

	return string(content), nil
}
