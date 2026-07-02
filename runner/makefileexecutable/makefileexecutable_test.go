package makefileexecutable

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExecute(t *testing.T) {
	withTempWorkingDir(t)

	if err := os.MkdirAll("downloads", 0755); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join("downloads", "tool")
	if err := os.WriteFile(targetPath, []byte("binary"), 0644); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		Path: targetPath,
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
	if result.Path != targetPath {
		t.Fatalf("path = %q, want %q", result.Path, targetPath)
	}

	if runtime.GOOS == "windows" {
		return
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if gotMode, wantMode := info.Mode().Perm(), os.FileMode(0744); gotMode != wantMode {
		t.Fatalf("mode = %v, want %v", gotMode, wantMode)
	}
}

func TestExecuteRejectsOutsideWorkspace(t *testing.T) {
	withTempWorkingDir(t)

	payload, err := json.Marshal(Payload{
		Path: "../tool",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(payload); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecuteRejectsDirectory(t *testing.T) {
	withTempWorkingDir(t)

	if err := os.Mkdir("tool", 0755); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		Path: "tool",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(payload); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecuteRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on windows")
	}

	withTempWorkingDir(t)

	if err := os.WriteFile("target", []byte("binary"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", "link"); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		Path: "link",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(payload); err == nil {
		t.Fatal("expected error, got nil")
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
