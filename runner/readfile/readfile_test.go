package readfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExecuteReadsFile(t *testing.T) {
	withTempWorkingDir(t)

	if err := os.WriteFile("message.txt", []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result := execute(t, Payload{Path: "message.txt"})
	if result.Path != "message.txt" {
		t.Fatalf("path = %q, want message.txt", result.Path)
	}
	if result.Content != "hello" {
		t.Fatalf("content = %q, want hello", result.Content)
	}
	if result.Bytes != 5 {
		t.Fatalf("bytes = %d, want 5", result.Bytes)
	}
	if result.Size != 5 {
		t.Fatalf("size = %d, want 5", result.Size)
	}
	if result.Truncated {
		t.Fatal("truncated = true, want false")
	}
}

func TestExecuteLimitsReturnedBytes(t *testing.T) {
	withTempWorkingDir(t)

	if err := os.WriteFile("message.txt", []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result := execute(t, Payload{Path: "message.txt", MaxBytes: 3})
	if result.Content != "hel" {
		t.Fatalf("content = %q, want hel", result.Content)
	}
	if result.Bytes != 3 {
		t.Fatalf("bytes = %d, want 3", result.Bytes)
	}
	if result.Size != 5 {
		t.Fatalf("size = %d, want 5", result.Size)
	}
	if !result.Truncated {
		t.Fatal("truncated = false, want true")
	}
}

func TestExecuteDoesNotMarkExactLimitAsTruncated(t *testing.T) {
	withTempWorkingDir(t)

	if err := os.WriteFile("message.txt", []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result := execute(t, Payload{Path: "message.txt", MaxBytes: 5})
	if result.Content != "hello" {
		t.Fatalf("content = %q, want hello", result.Content)
	}
	if result.Truncated {
		t.Fatal("truncated = true, want false")
	}
}

func TestExecuteUsesDefaultLimit(t *testing.T) {
	withTempWorkingDir(t)

	content := strings.Repeat("a", int(MaxReadBytes+1))
	if err := os.WriteFile("large.txt", []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := execute(t, Payload{Path: "large.txt"})
	if result.Bytes != MaxReadBytes {
		t.Fatalf("bytes = %d, want %d", result.Bytes, MaxReadBytes)
	}
	if !result.Truncated {
		t.Fatal("truncated = false, want true")
	}
}

func TestExecuteRejectsInvalidRequests(t *testing.T) {
	withTempWorkingDir(t)

	if err := os.WriteFile("message.txt", []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		payload json.RawMessage
	}{
		{name: "missing payload"},
		{name: "invalid payload", payload: json.RawMessage(`{`)},
		{name: "missing path", payload: marshal(t, Payload{Path: " "})},
		{name: "negative limit", payload: marshal(t, Payload{Path: "message.txt", MaxBytes: -1})},
		{name: "limit above maximum", payload: marshal(t, Payload{Path: "message.txt", MaxBytes: MaxReadBytes + 1})},
		{name: "outside workspace", payload: marshal(t, Payload{Path: filepath.Join("..", "message.txt")})},
		{name: "directory", payload: marshal(t, Payload{Path: "."})},
		{name: "missing file", payload: marshal(t, Payload{Path: "missing.txt"})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Execute(tt.payload); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestExecuteRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on windows")
	}

	withTempWorkingDir(t)
	if err := os.WriteFile("target.txt", []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", "link.txt"); err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(marshal(t, Payload{Path: "link.txt"})); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecuteRejectsSymlinkDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on windows")
	}

	withTempWorkingDir(t)
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, "linked-dir"); err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(marshal(t, Payload{Path: filepath.Join("linked-dir", "secret.txt")})); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func execute(t *testing.T, req Payload) Result {
	t.Helper()

	got, err := Execute(marshal(t, req))
	if err != nil {
		t.Fatal(err)
	}
	result, ok := got.(Result)
	if !ok {
		t.Fatalf("result type = %T, want Result", got)
	}
	return result
}

func marshal(t *testing.T, req Payload) json.RawMessage {
	t.Helper()

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	return payload
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
