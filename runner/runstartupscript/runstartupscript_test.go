package runstartupscript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestExecuteRunsStartupScriptWithWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script execution is POSIX-specific")
	}

	withTempWorkingDir(t)

	if err := os.MkdirAll(filepath.Join("services", "foo", "logs"), 0755); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join("services", "foo", "start.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nprintf started > logs/result.txt\n"), 0755); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		Path:       scriptPath,
		Args:       []string{"--port", "9000"},
		WorkingDir: filepath.Join("services", "foo"),
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := Execute(payload)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := got.(Result)
	if !ok {
		t.Fatalf("result type = %T, want Result", got)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
	if result.WorkingDir != filepath.Join("services", "foo") {
		t.Fatalf("working_dir = %q", result.WorkingDir)
	}

	content, err := os.ReadFile(filepath.Join("services", "foo", "logs", "result.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "started" {
		t.Fatalf("content = %q, want started", string(content))
	}
}

func TestExecuteDoesNotWaitForBackgroundProcessOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script execution is POSIX-specific")
	}

	withTempWorkingDir(t)

	if err := os.WriteFile("start.sh", []byte("#!/bin/sh\n(sleep 2; printf late) &\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		Path:           "start.sh",
		TimeoutSeconds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	startedAt := time.Now()
	if _, err := Execute(payload); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(startedAt); elapsed >= 1500*time.Millisecond {
		t.Fatalf("elapsed = %s, expected runner not to wait for background process", elapsed)
	}
}

func TestExecuteRunsCommandFromPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("whoami command lookup is POSIX-specific")
	}

	withTempWorkingDir(t)

	payload, err := json.Marshal(Payload{
		Path: "whoami",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := Execute(payload)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := got.(Result)
	if !ok {
		t.Fatalf("result type = %T, want Result", got)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
	if result.Stdout == "" {
		t.Fatal("stdout is empty")
	}
}

func TestExecuteRejectsOutsideWorkspace(t *testing.T) {
	withTempWorkingDir(t)

	payload, err := json.Marshal(Payload{
		Path: "../start.sh",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(payload); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecuteRejectsNonExecutableScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows executable checks differ")
	}

	withTempWorkingDir(t)

	if err := os.WriteFile("start.sh", []byte("#!/bin/sh\nexit 0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		Path: "start.sh",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(payload); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecuteRejectsWorkingDirOutsideWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script execution is POSIX-specific")
	}

	withTempWorkingDir(t)

	if err := os.WriteFile("start.sh", []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		Path:       "start.sh",
		WorkingDir: "..",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(payload); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNormalizeRejectsTooLargeTimeout(t *testing.T) {
	withTempWorkingDir(t)

	if err := os.WriteFile("start.sh", []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	_, _, _, _, err := normalize(Payload{
		Path:           "start.sh",
		TimeoutSeconds: maxTimeoutSeconds + 1,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecuteWithCommandPassesNormalizedInputs(t *testing.T) {
	withTempWorkingDir(t)

	if err := os.Mkdir("service", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("service", "start.sh"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		Path:       filepath.Join("service", "start.sh"),
		Args:       []string{"one"},
		WorkingDir: "service",
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotPath string
	var gotArgs []string
	var gotWorkingDir string
	got, err := executeWithCommand(payload, func(scriptPath string, args []string, workingDir string, timeout time.Duration) (commandResult, error) {
		gotPath = scriptPath
		gotArgs = args
		gotWorkingDir = workingDir
		return commandResult{
			ExitCode:   0,
			DurationMS: 1,
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := got.(Result); !ok {
		t.Fatalf("result type = %T, want Result", got)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != filepath.Join(wd, "service", "start.sh") {
		t.Fatalf("script path = %q", gotPath)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "one" {
		t.Fatalf("args = %v", gotArgs)
	}
	if gotWorkingDir != filepath.Join(wd, "service") {
		t.Fatalf("working dir = %q", gotWorkingDir)
	}
}

func withTempWorkingDir(t *testing.T) {
	t.Helper()

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatal(err)
		}
	})
}
