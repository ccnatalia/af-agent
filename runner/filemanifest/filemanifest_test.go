package filemanifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFullUpdate(t *testing.T) {
	rootDir := t.TempDir()
	writeTestFile(t, filepath.Join(rootDir, "z.txt"), "world")
	writeTestFile(t, filepath.Join(rootDir, "nested", "a.txt"), "hello")
	writeTestFile(t, filepath.Join(rootDir, Filename), "stale manifest")
	writeTestFile(t, filepath.Join(rootDir, temporaryFilePrefix+"stale.tmp"), "stale temporary file")

	if err := FullUpdate(rootDir); err != nil {
		t.Fatal(err)
	}

	want := strings.Join([]string{
		"5D41402ABC4B2A76B9719D911017C592," + filepath.Join("nested", "a.txt"),
		"7D793037A0760186574B0282F2F435E7,z.txt",
		"",
	}, "\n")
	assertFileContent(t, filepath.Join(rootDir, Filename), want)

	if err := FullUpdate(rootDir); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(rootDir, Filename), want)
}

func TestFullUpdateRejectsNonDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	writeTestFile(t, path, "content")

	if err := FullUpdate(path); err == nil || !strings.Contains(err.Error(), "must be a directory") {
		t.Fatalf("error = %v, want directory error", err)
	}
}

func TestIncrementalUpdateReplacesAndAddsRecords(t *testing.T) {
	rootDir := t.TempDir()
	writeTestFile(t, filepath.Join(rootDir, "a.txt"), "hello")
	writeTestFile(t, filepath.Join(rootDir, "b.txt"), "world")
	if err := FullUpdate(rootDir); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(rootDir, "b.txt"), "changed")
	if err := IncrementalUpdate(rootDir, "b.txt"); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(rootDir, "nested", "c.txt"), "")
	if err := IncrementalUpdate(rootDir, filepath.Join("nested", "c.txt")); err != nil {
		t.Fatal(err)
	}

	want := strings.Join([]string{
		"5D41402ABC4B2A76B9719D911017C592,a.txt",
		"8977DFAC2F8E04CB96E66882235F5ABA,b.txt",
		"D41D8CD98F00B204E9800998ECF8427E," + filepath.Join("nested", "c.txt"),
		"",
	}, "\n")
	assertFileContent(t, filepath.Join(rootDir, Filename), want)
}

func TestIncrementalUpdateCreatesManifest(t *testing.T) {
	rootDir := t.TempDir()
	writeTestFile(t, filepath.Join(rootDir, "file.txt"), "hello")

	if err := IncrementalUpdate(rootDir, "file.txt"); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(rootDir, Filename), "5D41402ABC4B2A76B9719D911017C592,file.txt\n")
}

func TestIncrementalUpdatePreservesMalformedManifest(t *testing.T) {
	rootDir := t.TempDir()
	manifestPath := filepath.Join(rootDir, Filename)
	writeTestFile(t, manifestPath, "invalid manifest\n")
	writeTestFile(t, filepath.Join(rootDir, "file.txt"), "hello")

	err := IncrementalUpdate(rootDir, "file.txt")
	if err == nil || !strings.Contains(err.Error(), "parse manifest line 1") {
		t.Fatalf("error = %v, want parse error", err)
	}
	assertFileContent(t, manifestPath, "invalid manifest\n")
}

func TestIncrementalUpdateRejectsUnsafePath(t *testing.T) {
	rootDir := t.TempDir()

	for _, relativePath := range []string{"", ".", filepath.Join("..", "file.txt"), Filename} {
		t.Run(relativePath, func(t *testing.T) {
			if err := IncrementalUpdate(rootDir, relativePath); err == nil {
				t.Fatalf("IncrementalUpdate(%q) returned nil error", relativePath)
			}
		})
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("content = %q, want %q", string(content), want)
	}
}
