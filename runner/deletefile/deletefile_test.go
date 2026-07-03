package deletefile

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
	targetPath := filepath.Join("downloads", "myfile")
	if err := os.WriteFile(targetPath, []byte("hello"), 0644); err != nil {
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
	if !result.Deleted {
		t.Fatal("deleted = false, want true")
	}
	if result.Bytes != 5 {
		t.Fatalf("bytes = %d, want 5", result.Bytes)
	}

	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("file still exists or stat failed unexpectedly: %v", err)
	}
}

func TestExecuteRejectsOutsideWorkspace(t *testing.T) {
	withTempWorkingDir(t)

	payload, err := json.Marshal(Payload{
		Path: "../myfile",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(payload); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecuteRequiresPath(t *testing.T) {
	withTempWorkingDir(t)

	payload, err := json.Marshal(Payload{
		Path: " ",
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

	if err := os.Mkdir("downloads", 0755); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		Path: "downloads",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(payload); err == nil {
		t.Fatal("expected error, got nil")
	}

	if _, err := os.Stat("downloads"); err != nil {
		t.Fatalf("directory was removed or stat failed unexpectedly: %v", err)
	}
}

func TestExecuteMissingFile(t *testing.T) {
	withTempWorkingDir(t)

	payload, err := json.Marshal(Payload{
		Path: "downloads/missing",
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
	if result.Path != "downloads/missing" {
		t.Fatalf("path = %q, want downloads/missing", result.Path)
	}
	if result.Deleted {
		t.Fatal("deleted = true, want false")
	}
	if result.Bytes != 0 {
		t.Fatalf("bytes = %d, want 0", result.Bytes)
	}
}

func TestExecuteDeletesSymlinkOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on windows")
	}

	withTempWorkingDir(t)

	if err := os.WriteFile("target", []byte("hello"), 0644); err != nil {
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

	got, err := Execute(payload)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := got.(Result)
	if !ok {
		t.Fatalf("result type = %T, want Result", got)
	}
	if !result.Deleted {
		t.Fatal("deleted = false, want true")
	}

	if _, err := os.Lstat("link"); !os.IsNotExist(err) {
		t.Fatalf("link still exists or stat failed unexpectedly: %v", err)
	}
	content, err := os.ReadFile("target")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello" {
		t.Fatalf("target content = %q, want hello", string(content))
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
