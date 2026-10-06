package downloadfilev2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecuteDownloadsToTargetPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("downloaded content"))
	}))
	defer server.Close()

	withTempWorkingDir(t)

	got, err := Execute(marshal(t, Payload{
		URL:        server.URL + "/files/example.txt",
		TargetPath: "artifacts/releases/example.txt",
	}))
	if err != nil {
		t.Fatal(err)
	}

	result, ok := got.(Result)
	if !ok {
		t.Fatalf("result type = %T, want Result", got)
	}
	if result.Path != filepath.Join("artifacts", "releases", "example.txt") {
		t.Fatalf("path = %q", result.Path)
	}
	if result.Filename != "example.txt" {
		t.Fatalf("filename = %q, want example.txt", result.Filename)
	}
	if result.Bytes != int64(len("downloaded content")) {
		t.Fatalf("bytes = %d, want %d", result.Bytes, len("downloaded content"))
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", result.StatusCode, http.StatusOK)
	}
	if result.ContentType != "text/plain" {
		t.Fatalf("content type = %q, want text/plain", result.ContentType)
	}

	content, err := os.ReadFile(filepath.Join("artifacts", "releases", "example.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "downloaded content" {
		t.Fatalf("content = %q, want downloaded content", string(content))
	}
}

func TestExecuteRequiresURLAndTargetPath(t *testing.T) {
	tests := []struct {
		name    string
		payload Payload
		wantErr string
	}{
		{name: "url", payload: Payload{TargetPath: "file.txt"}, wantErr: "url is required"},
		{name: "target path", payload: Payload{URL: "https://example.com/file.txt"}, wantErr: "target_path is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Execute(marshal(t, tt.payload))
			assertErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestExecuteRejectsUnsafeTargetPaths(t *testing.T) {
	withTempWorkingDir(t)

	tests := []struct {
		name       string
		targetPath string
		wantErr    string
	}{
		{name: "absolute", targetPath: filepath.Join(string(filepath.Separator), "tmp", "file.txt"), wantErr: "absolute paths are not allowed"},
		{name: "outside workspace", targetPath: filepath.Join("..", "file.txt"), wantErr: "path must stay inside workspace"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Execute(marshal(t, Payload{
				URL:        "https://example.com/file.txt",
				TargetPath: tt.targetPath,
			}))
			assertErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestExecuteRejectsSymlinkInTargetPath(t *testing.T) {
	withTempWorkingDir(t)

	outside := t.TempDir()
	if err := os.Symlink(outside, "linked"); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	_, err := Execute(marshal(t, Payload{
		URL:        "https://example.com/file.txt",
		TargetPath: filepath.Join("linked", "file.txt"),
	}))
	assertErrorContains(t, err, "target_path must not contain symlinks")
}

func TestExecuteBacksUpExistingTarget(t *testing.T) {
	withTempWorkingDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("replacement"))
	}))
	defer server.Close()

	if err := os.WriteFile("existing.txt", []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := Execute(marshal(t, Payload{
		URL:        server.URL,
		TargetPath: "existing.txt",
	}))
	if err != nil {
		t.Fatal(err)
	}
	result := got.(Result)
	if !strings.HasPrefix(result.BackupPath, "existing.txt.") || !strings.HasSuffix(result.BackupPath, ".bak") {
		t.Fatalf("backup path = %q, want timestamped backup", result.BackupPath)
	}

	content, err := os.ReadFile("existing.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "replacement" {
		t.Fatalf("content = %q, want replacement", string(content))
	}
	backupContent, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(backupContent) != "original" {
		t.Fatalf("backup content = %q, want original", string(backupContent))
	}
}

func TestExecuteFailureKeepsBackupWithoutRestoringTarget(t *testing.T) {
	withTempWorkingDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	if err := os.WriteFile("existing.txt", []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Execute(marshal(t, Payload{URL: server.URL, TargetPath: "existing.txt"}))
	assertErrorContains(t, err, "unexpected status 500")
	assertErrorContains(t, err, "previous file preserved at existing.txt.")
	assertPathDoesNotExist(t, "existing.txt")

	backups, globErr := filepath.Glob("existing.txt.*.bak")
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(backups) != 1 {
		t.Fatalf("backups = %v, want exactly one", backups)
	}
	content, readErr := os.ReadFile(backups[0])
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != "original" {
		t.Fatalf("backup content = %q, want original", string(content))
	}
}

func TestBackupExistingTargetAvoidsBackupNameCollision(t *testing.T) {
	withTempWorkingDir(t)
	backupTime := time.Date(2026, time.October, 6, 15, 30, 12, 123456789, time.UTC)
	baseBackupPath := "existing.txt.20261006T153012.123456789Z.bak"
	if err := os.WriteFile("existing.txt", []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(baseBackupPath, []byte("older backup"), 0644); err != nil {
		t.Fatal(err)
	}

	backupPath, err := backupExistingTarget("existing.txt", backupTime)
	if err != nil {
		t.Fatal(err)
	}
	if backupPath != "existing.txt.20261006T153012.123456789Z-1.bak" {
		t.Fatalf("backup path = %q", backupPath)
	}
	content, err := os.ReadFile(baseBackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "older backup" {
		t.Fatalf("existing backup content = %q, want older backup", string(content))
	}
}

func TestExecuteRejectsNonRegularExistingTarget(t *testing.T) {
	withTempWorkingDir(t)
	if err := os.Mkdir("existing", 0755); err != nil {
		t.Fatal(err)
	}

	_, err := Execute(marshal(t, Payload{
		URL:        "https://example.com/file.txt",
		TargetPath: "existing",
	}))
	assertErrorContains(t, err, "existing target_path must be a regular file")
}

func TestExecuteRejectsConcurrentDownloadToSameTarget(t *testing.T) {
	withTempWorkingDir(t)
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-releaseRequest
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer server.Close()

	firstDone := make(chan error, 1)
	go func() {
		_, err := Execute(marshal(t, Payload{URL: server.URL, TargetPath: "same.txt"}))
		firstDone <- err
	}()
	<-requestStarted

	_, err := Execute(marshal(t, Payload{URL: server.URL, TargetPath: "same.txt"}))
	assertErrorContains(t, err, "target_path is already being downloaded")
	close(releaseRequest)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
}

func TestExecuteResponseHeaderTimeoutCleansTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()

	withTempWorkingDir(t)
	targetPath := "header-timeout.txt"
	_, err := Execute(marshal(t, Payload{
		URL:        server.URL,
		TargetPath: targetPath,
		Timeouts: TimeoutPayload{
			ResponseHeaderSeconds: 1,
		},
	}))
	assertErrorContains(t, err, "timeout awaiting response headers")
	assertPathDoesNotExist(t, targetPath)
}

func TestExecuteBodyIdleTimeoutCleansTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	withTempWorkingDir(t)
	targetPath := "body-idle-timeout.txt"
	_, err := Execute(marshal(t, Payload{
		URL:        server.URL,
		TargetPath: targetPath,
		Timeouts: TimeoutPayload{
			BodyIdleSeconds: 1,
			TotalSeconds:    5,
		},
	}))
	assertErrorContains(t, err, "response body idle timeout after 1s")
	assertPathDoesNotExist(t, targetPath)
}

func TestExecuteTotalTimeoutCleansTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, _ = w.Write([]byte("data"))
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}))
	defer server.Close()

	withTempWorkingDir(t)
	targetPath := "total-timeout.txt"
	_, err := Execute(marshal(t, Payload{
		URL:        server.URL,
		TargetPath: targetPath,
		Timeouts: TimeoutPayload{
			BodyIdleSeconds: 2,
			TotalSeconds:    1,
		},
	}))
	assertErrorContains(t, err, "total timeout after 1s")
	assertPathDoesNotExist(t, targetPath)
}

func TestExecuteRejectsOversizedContentLength(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(maxDownloadFileBytes+1))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	withTempWorkingDir(t)
	targetPath := "oversized.bin"
	_, err := Execute(marshal(t, Payload{
		URL:        server.URL,
		TargetPath: targetPath,
	}))
	assertErrorContains(t, err, "download file exceeds")
	assertPathDoesNotExist(t, targetPath)
}

func TestResolveTimeouts(t *testing.T) {
	got, err := resolveTimeouts(TimeoutPayload{})
	if err != nil {
		t.Fatal(err)
	}
	if got.connect != defaultConnectTimeout ||
		got.tlsHandshake != defaultTLSHandshakeTimeout ||
		got.responseHeader != defaultResponseHeaderTimeout ||
		got.bodyIdle != defaultBodyIdleTimeout ||
		got.total != defaultTotalTimeout {
		t.Fatalf("default timeouts = %+v", got)
	}

	got, err = resolveTimeouts(TimeoutPayload{
		ConnectSeconds:        1,
		TLSHandshakeSeconds:   2,
		ResponseHeaderSeconds: 3,
		BodyIdleSeconds:       4,
		TotalSeconds:          5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.connect != time.Second ||
		got.tlsHandshake != 2*time.Second ||
		got.responseHeader != 3*time.Second ||
		got.bodyIdle != 4*time.Second ||
		got.total != 5*time.Second {
		t.Fatalf("custom timeouts = %+v", got)
	}
}

func TestResolveTimeoutsRejectsInvalidValues(t *testing.T) {
	_, err := resolveTimeouts(TimeoutPayload{ConnectSeconds: -1})
	assertErrorContains(t, err, "timeouts.connect_seconds must not be negative")

	_, err = resolveTimeouts(TimeoutPayload{TotalSeconds: maxTotalTimeoutSeconds + 1})
	assertErrorContains(t, err, "timeouts.total_seconds must not exceed 28800")
}

func marshal(t *testing.T, payload Payload) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return data
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

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want substring %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err, want)
	}
}

func assertPathDoesNotExist(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stat %q error = %v, want not exist", path, err)
	}
}
