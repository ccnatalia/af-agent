package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteDownloadFileTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("downloaded content"))
	}))
	defer server.Close()

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatal(err)
		}
	}()

	payload, err := json.Marshal(DownloadFilePayload{
		URL:      server.URL + "/files/example.txt",
		Filename: "example.txt",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := executeDownloadFileTask(payload)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := got.(DownloadFileResult)
	if !ok {
		t.Fatalf("result type = %T, want DownloadFileResult", got)
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
