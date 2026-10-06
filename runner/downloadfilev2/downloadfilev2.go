package downloadfilev2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"afagent/runner/internal/workspace"
)

const Name = "download-file-v2"

const (
	maxDownloadFileBytes         int64 = 100 << 20 // 100 MB
	defaultConnectTimeout              = 10 * time.Second
	defaultTLSHandshakeTimeout         = 10 * time.Second
	defaultResponseHeaderTimeout       = 30 * time.Second
	defaultBodyIdleTimeout             = 2 * time.Minute
	defaultTotalTimeout                = 8 * time.Hour
	maxTotalTimeoutSeconds       int64 = 8 * 60 * 60 // 8 hours
)

var errBodyIdleTimeout = errors.New("response body idle timeout")

type Payload struct {
	URL        string         `json:"url"`
	TargetPath string         `json:"target_path"`
	Timeouts   TimeoutPayload `json:"timeouts,omitempty"`
}

type TimeoutPayload struct {
	ConnectSeconds        int64 `json:"connect_seconds,omitempty"`
	TLSHandshakeSeconds   int64 `json:"tls_handshake_seconds,omitempty"`
	ResponseHeaderSeconds int64 `json:"response_header_seconds,omitempty"`
	BodyIdleSeconds       int64 `json:"body_idle_seconds,omitempty"`
	TotalSeconds          int64 `json:"total_seconds,omitempty"`
}

type Result struct {
	URL         string `json:"url"`
	Path        string `json:"path"`
	Filename    string `json:"filename"`
	BackupPath  string `json:"backup_path,omitempty"`
	Bytes       int64  `json:"bytes"`
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type,omitempty"`
}

var activeTargets = struct {
	sync.Mutex
	paths map[string]struct{}
}{paths: make(map[string]struct{})}

type timeoutConfig struct {
	connect        time.Duration
	tlsHandshake   time.Duration
	responseHeader time.Duration
	bodyIdle       time.Duration
	total          time.Duration
}

func Execute(payload json.RawMessage) (_ any, returnErr error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid download-file-v2 payload: %w", err)
	}

	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		return nil, errors.New("url is required")
	}

	parsedURL, err := url.ParseRequestURI(req.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("url must use http or https")
	}

	req.TargetPath = strings.TrimSpace(req.TargetPath)
	if req.TargetPath == "" {
		return nil, errors.New("target_path is required")
	}

	targetPath, err := workspace.Path(req.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("target_path: %w", err)
	}
	if err := ensureSafeTargetParent(targetPath); err != nil {
		return nil, fmt.Errorf("target_path: %w", err)
	}

	timeouts, err := resolveTimeouts(req.Timeouts)
	if err != nil {
		return nil, err
	}

	if !reserveTarget(targetPath) {
		return nil, errors.New("target_path is already being downloaded")
	}
	defer releaseTarget(targetPath)

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return nil, fmt.Errorf("create target dir: %w", err)
	}
	if err := ensureSafeTargetParent(targetPath); err != nil {
		return nil, fmt.Errorf("target_path: %w", err)
	}

	backupPath, err := backupExistingTarget(targetPath, time.Now())
	if err != nil {
		return nil, err
	}
	backupResultPath := ""
	if backupPath != "" {
		backupResultPath = backupPath
		if relativePath, err := workspaceRelativePath(backupPath); err == nil {
			backupResultPath = relativePath
		}
		defer func() {
			if returnErr != nil {
				returnErr = fmt.Errorf("%w; previous file preserved at %s", returnErr, backupResultPath)
			}
		}()
	}

	targetFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(err, os.ErrExist) {
		return nil, errors.New("target_path already exists")
	}
	if err != nil {
		return nil, fmt.Errorf("create target file: %w", err)
	}

	completed := false
	defer func() {
		_ = targetFile.Close()
		if !completed {
			_ = os.Remove(targetPath)
		}
	}()

	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{
		Timeout:   timeouts.connect,
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = dialer.DialContext
	transport.TLSHandshakeTimeout = timeouts.tlsHandshake
	transport.ResponseHeaderTimeout = timeouts.responseHeader
	defer transport.CloseIdleConnections()

	totalCtx, cancelTotal := context.WithTimeout(context.Background(), timeouts.total)
	defer cancelTotal()
	requestCtx, cancelRequest := context.WithCancelCause(totalCtx)
	defer cancelRequest(nil)

	httpReq, err := http.NewRequestWithContext(requestCtx, http.MethodGet, req.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Do(httpReq)
	if err != nil {
		if errors.Is(totalCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("download file: total timeout after %s", timeouts.total)
		}
		return nil, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download file: unexpected status %d", resp.StatusCode)
	}
	if resp.ContentLength > maxDownloadFileBytes {
		return nil, fmt.Errorf("download file exceeds %d bytes", maxDownloadFileBytes)
	}

	idleTimer := time.AfterFunc(timeouts.bodyIdle, func() {
		cancelRequest(errBodyIdleTimeout)
	})
	defer idleTimer.Stop()

	body := &progressReader{
		reader:  resp.Body,
		timer:   idleTimer,
		timeout: timeouts.bodyIdle,
	}
	written, err := io.Copy(targetFile, io.LimitReader(body, maxDownloadFileBytes+1))
	if err != nil {
		switch {
		case errors.Is(context.Cause(requestCtx), errBodyIdleTimeout):
			return nil, fmt.Errorf("download file: response body idle timeout after %s", timeouts.bodyIdle)
		case errors.Is(totalCtx.Err(), context.DeadlineExceeded):
			return nil, fmt.Errorf("download file: total timeout after %s", timeouts.total)
		default:
			return nil, fmt.Errorf("write download file: %w", err)
		}
	}
	if written > maxDownloadFileBytes {
		return nil, fmt.Errorf("download file exceeds %d bytes", maxDownloadFileBytes)
	}

	if err := targetFile.Close(); err != nil {
		return nil, fmt.Errorf("close download file: %w", err)
	}
	completed = true

	return Result{
		URL:         req.URL,
		Path:        filepath.Clean(req.TargetPath),
		Filename:    filepath.Base(targetPath),
		BackupPath:  backupResultPath,
		Bytes:       written,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func reserveTarget(targetPath string) bool {
	activeTargets.Lock()
	defer activeTargets.Unlock()

	if _, exists := activeTargets.paths[targetPath]; exists {
		return false
	}
	activeTargets.paths[targetPath] = struct{}{}
	return true
}

func releaseTarget(targetPath string) {
	activeTargets.Lock()
	delete(activeTargets.paths, targetPath)
	activeTargets.Unlock()
}

func backupExistingTarget(targetPath string, backupTime time.Time) (string, error) {
	info, err := os.Lstat(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("stat existing target file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("existing target_path must be a regular file")
	}

	timestamp := backupTime.UTC().Format("20060102T150405.000000000Z")
	for sequence := 0; ; sequence++ {
		suffix := timestamp
		if sequence > 0 {
			suffix = fmt.Sprintf("%s-%d", timestamp, sequence)
		}
		backupPath := fmt.Sprintf("%s.%s.bak", targetPath, suffix)

		if _, err := os.Lstat(backupPath); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("check backup path: %w", err)
		}

		if err := os.Rename(targetPath, backupPath); err != nil {
			return "", fmt.Errorf("backup existing target file: %w", err)
		}
		return backupPath, nil
	}
}

func workspaceRelativePath(path string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	relativePath, err := filepath.Rel(wd, path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(relativePath), nil
}

func resolveTimeouts(payload TimeoutPayload) (timeoutConfig, error) {
	values := []struct {
		name         string
		seconds      int64
		defaultValue time.Duration
	}{
		{name: "connect_seconds", seconds: payload.ConnectSeconds, defaultValue: defaultConnectTimeout},
		{name: "tls_handshake_seconds", seconds: payload.TLSHandshakeSeconds, defaultValue: defaultTLSHandshakeTimeout},
		{name: "response_header_seconds", seconds: payload.ResponseHeaderSeconds, defaultValue: defaultResponseHeaderTimeout},
		{name: "body_idle_seconds", seconds: payload.BodyIdleSeconds, defaultValue: defaultBodyIdleTimeout},
		{name: "total_seconds", seconds: payload.TotalSeconds, defaultValue: defaultTotalTimeout},
	}

	resolved := make([]time.Duration, len(values))
	for i, value := range values {
		if value.seconds < 0 {
			return timeoutConfig{}, fmt.Errorf("timeouts.%s must not be negative", value.name)
		}
		if value.seconds > maxTotalTimeoutSeconds {
			return timeoutConfig{}, fmt.Errorf("timeouts.%s must not exceed %d", value.name, maxTotalTimeoutSeconds)
		}
		if value.seconds == 0 {
			resolved[i] = value.defaultValue
		} else {
			resolved[i] = time.Duration(value.seconds) * time.Second
		}
	}

	return timeoutConfig{
		connect:        resolved[0],
		tlsHandshake:   resolved[1],
		responseHeader: resolved[2],
		bodyIdle:       resolved[3],
		total:          resolved[4],
	}, nil
}

func ensureSafeTargetParent(targetPath string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	relParent, err := filepath.Rel(wd, filepath.Dir(targetPath))
	if err != nil {
		return fmt.Errorf("resolve parent directory: %w", err)
	}

	currentPath := wd
	for _, part := range strings.Split(relParent, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		currentPath = filepath.Join(currentPath, part)
		info, err := os.Lstat(currentPath)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stat parent directory: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("target_path must not contain symlinks")
		}
		if !info.IsDir() {
			return errors.New("target_path parent must be a directory")
		}
	}

	return nil
}

type progressReader struct {
	reader  io.Reader
	timer   *time.Timer
	timeout time.Duration
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.timer.Reset(r.timeout)
	}
	return n, err
}
