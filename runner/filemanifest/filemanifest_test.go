package filemanifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
		"5D41402ABC4B2A76B9719D911017C592,nested/a.txt",
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
	if err := IncrementalUpdate(rootDir, "nested/c.txt"); err != nil {
		t.Fatal(err)
	}

	want := strings.Join([]string{
		"5D41402ABC4B2A76B9719D911017C592,a.txt",
		"8977DFAC2F8E04CB96E66882235F5ABA,b.txt",
		"D41D8CD98F00B204E9800998ECF8427E,nested/c.txt",
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

func TestIncrementalUpdateSerializesConcurrentUpdates(t *testing.T) {
	rootDir := t.TempDir()
	const fileCount = 32
	for i := 0; i < fileCount; i++ {
		name := fmt.Sprintf("file-%02d.txt", i)
		writeTestFile(t, filepath.Join(rootDir, name), fmt.Sprintf("content-%02d", i))
	}

	start := make(chan struct{})
	errors := make(chan error, fileCount)
	var workers sync.WaitGroup
	for i := 0; i < fileCount; i++ {
		name := fmt.Sprintf("file-%02d.txt", i)
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			errors <- IncrementalUpdate(rootDir, name)
		}()
	}
	close(start)
	workers.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}

	var want strings.Builder
	for i := 0; i < fileCount; i++ {
		name := fmt.Sprintf("file-%02d.txt", i)
		fileMD5, err := hashFile(filepath.Join(rootDir, name))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&want, "%s,%s\n", fileMD5, name)
	}
	assertFileContent(t, filepath.Join(rootDir, Filename), want.String())
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

	for _, relativePath := range []string{"", ".", "../file.txt", `nested\file.txt`, Filename} {
		t.Run(relativePath, func(t *testing.T) {
			if err := IncrementalUpdate(rootDir, relativePath); err == nil {
				t.Fatalf("IncrementalUpdate(%q) returned nil error", relativePath)
			}
		})
	}
}

func TestFullUpdateTemporaryManifestPathRules(t *testing.T) {
	rootDir := t.TempDir()
	writeTestFile(t, filepath.Join(rootDir, temporaryFilePrefix+"stale.tmp"), "temporary")
	writeTestFile(t, filepath.Join(rootDir, temporaryFilePrefix+".tmp"), "temporary")
	writeTestFile(t, filepath.Join(rootDir, temporaryFilePrefix+"notes"), "hello")
	writeTestFile(t, filepath.Join(rootDir, temporaryFilePrefix+"stale.tmp.bak"), "world")
	writeTestFile(t, filepath.Join(rootDir, "ordinary.tmp"), "changed")
	writeTestFile(t, filepath.Join(rootDir, temporaryFilePrefix+"cache", "item.txt"), "")

	if err := FullUpdate(rootDir); err != nil {
		t.Fatal(err)
	}

	want := strings.Join([]string{
		"D41D8CD98F00B204E9800998ECF8427E,.file-md5-cache/item.txt",
		"5D41402ABC4B2A76B9719D911017C592,.file-md5-notes",
		"7D793037A0760186574B0282F2F435E7,.file-md5-stale.tmp.bak",
		"8977DFAC2F8E04CB96E66882235F5ABA,ordinary.tmp",
		"",
	}, "\n")
	assertFileContent(t, filepath.Join(rootDir, Filename), want)
}

func TestIncrementalUpdateTemporaryManifestPathRules(t *testing.T) {
	rootDir := t.TempDir()
	manifestPath := filepath.Join(rootDir, Filename)
	reservedPath := temporaryFilePrefix + "stale.tmp"
	writeTestFile(t, filepath.Join(rootDir, "a.txt"), "hello")
	writeTestFile(t, filepath.Join(rootDir, reservedPath), "temporary")
	if err := FullUpdate(rootDir); err != nil {
		t.Fatal(err)
	}
	originalManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := IncrementalUpdate(rootDir, reservedPath); err == nil {
		t.Fatalf("IncrementalUpdate(%q) returned nil error", reservedPath)
	}
	assertFileContent(t, manifestPath, string(originalManifest))

	allowedPaths := []string{
		temporaryFilePrefix + "notes",
		temporaryFilePrefix + "cache/item.tmp",
	}
	for _, relativePath := range allowedPaths {
		writeTestFile(t, filepath.Join(rootDir, filepath.FromSlash(relativePath)), "world")
		if err := IncrementalUpdate(rootDir, relativePath); err != nil {
			t.Fatalf("IncrementalUpdate(%q): %v", relativePath, err)
		}
	}
	want := strings.Join([]string{
		"7D793037A0760186574B0282F2F435E7,.file-md5-cache/item.tmp",
		"7D793037A0760186574B0282F2F435E7,.file-md5-notes",
		"5D41402ABC4B2A76B9719D911017C592,a.txt",
		"",
	}, "\n")
	assertFileContent(t, manifestPath, want)
}

func TestIsTemporaryManifestName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: ".file-md5-a.tmp", want: true},
		{name: ".file-md5-.tmp", want: true},
		{name: ".file-md5-a", want: false},
		{name: ".file-md5-a.tmp.bak", want: false},
		{name: "file-md5-a.tmp", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isTemporaryManifestName(test.name); got != test.want {
				t.Fatalf("isTemporaryManifestName(%q) = %t, want %t", test.name, got, test.want)
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
