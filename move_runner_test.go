package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteMoveFileTask(t *testing.T) {
	withTempWorkingDir(t)

	if err := os.MkdirAll("source", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("source", "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(MoveFilePayload{
		SourcePath: "source/a.txt",
		TargetPath: "target/b.txt",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := executeMoveFileTask(payload)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := got.(MoveFileResult)
	if !ok {
		t.Fatalf("result type = %T, want MoveFileResult", got)
	}
	if result.Bytes != 5 {
		t.Fatalf("bytes = %d, want 5", result.Bytes)
	}

	if _, err := os.Stat(filepath.Join("source", "a.txt")); !os.IsNotExist(err) {
		t.Fatalf("source still exists or stat failed unexpectedly: %v", err)
	}

	content, err := os.ReadFile(filepath.Join("target", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello" {
		t.Fatalf("content = %q, want hello", string(content))
	}
}

func TestExecuteMoveFileTaskRejectsOutsideWorkspace(t *testing.T) {
	withTempWorkingDir(t)

	payload, err := json.Marshal(MoveFilePayload{
		SourcePath: "../a.txt",
		TargetPath: "target/b.txt",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := executeMoveFileTask(payload); err == nil {
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
