package incrementalupdatefilemanifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExecute(t *testing.T) {
	withTempWorkingDir(t)
	if err := os.MkdirAll("files", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("files", "example.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(Payload{
		RootDir:      "files",
		RelativePath: "example.txt",
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
	if result.RelativePath != "example.txt" {
		t.Fatalf("relative path = %q", result.RelativePath)
	}

	content, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "5D41402ABC4B2A76B9719D911017C592,example.txt\n" {
		t.Fatalf("content = %q", string(content))
	}
}

func withTempWorkingDir(t *testing.T) {
	t.Helper()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatal(err)
		}
	})
}
