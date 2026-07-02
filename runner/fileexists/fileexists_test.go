package fileexists

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExecuteExistingFile(t *testing.T) {
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
	if !result.Exists {
		t.Fatal("exists = false, want true")
	}
	if !result.IsFile {
		t.Fatal("is_file = false, want true")
	}
	if result.IsDir {
		t.Fatal("is_dir = true, want false")
	}
	if result.Size != 5 {
		t.Fatalf("size = %d, want 5", result.Size)
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
	if result.Exists {
		t.Fatal("exists = true, want false")
	}
	if result.IsFile {
		t.Fatal("is_file = true, want false")
	}
	if result.IsDir {
		t.Fatal("is_dir = true, want false")
	}
	if result.Size != 0 {
		t.Fatalf("size = %d, want 0", result.Size)
	}
}

func TestExecuteExistingDirectory(t *testing.T) {
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

	got, err := Execute(payload)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := got.(Result)
	if !ok {
		t.Fatalf("result type = %T, want Result", got)
	}
	if !result.Exists {
		t.Fatal("exists = false, want true")
	}
	if result.IsFile {
		t.Fatal("is_file = true, want false")
	}
	if !result.IsDir {
		t.Fatal("is_dir = false, want true")
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

func TestExecuteDoesNotFollowSymlink(t *testing.T) {
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
	if !result.Exists {
		t.Fatal("exists = false, want true")
	}
	if result.IsFile {
		t.Fatal("is_file = true, want false")
	}
	if result.IsDir {
		t.Fatal("is_dir = true, want false")
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
