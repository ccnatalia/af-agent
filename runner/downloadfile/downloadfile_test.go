package downloadfile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestExecute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("downloaded content"))
	}))
	defer server.Close()

	withTempWorkingDir(t)

	payload, err := json.Marshal(Payload{
		URL:      server.URL + "/files/example.txt",
		Filename: "example.txt",
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

	if result.Filename != "example.txt" {
		t.Fatalf("filename = %q, want example.txt", result.Filename)
	}
	if result.Bytes != int64(len("downloaded content")) {
		t.Fatalf("bytes = %d, want %d", result.Bytes, len("downloaded content"))
	}

	content, err := os.ReadFile(filepath.Join("downloads", "example.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "downloaded content" {
		t.Fatalf("content = %q, want downloaded content", string(content))
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
