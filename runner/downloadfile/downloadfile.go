package downloadfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const Name = "download-file"
const downloadDir = "downloads"
const maxDownloadFileBytes int64 = 100 << 20

type Payload struct {
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
}

type Result struct {
	URL         string `json:"url"`
	Path        string `json:"path"`
	Filename    string `json:"filename"`
	Bytes       int64  `json:"bytes"`
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type,omitempty"`
}

func Execute(payload json.RawMessage) (any, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req Payload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid download-file payload: %w", err)
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

	filename, err := downloadFilename(parsedURL, req.Filename)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("create download dir: %w", err)
	}

	targetPath, targetFile, err := createDownloadFile(downloadDir, filename)
	if err != nil {
		return nil, err
	}
	defer targetFile.Close()

	client := &http.Client{
		Timeout: 1 * time.Minute,
	}

	resp, err := client.Get(req.URL)
	if err != nil {
		os.Remove(targetPath)
		return nil, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		os.Remove(targetPath)
		return nil, fmt.Errorf("download file: unexpected status %d", resp.StatusCode)
	}

	written, err := io.Copy(targetFile, io.LimitReader(resp.Body, maxDownloadFileBytes+1))
	if err != nil {
		os.Remove(targetPath)
		return nil, fmt.Errorf("write download file: %w", err)
	}

	if written > maxDownloadFileBytes {
		os.Remove(targetPath)
		return nil, fmt.Errorf("download file exceeds %d bytes", maxDownloadFileBytes)
	}

	return Result{
		URL:         req.URL,
		Path:        targetPath,
		Filename:    filepath.Base(targetPath),
		Bytes:       written,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func downloadFilename(parsedURL *url.URL, requestedFilename string) (string, error) {
	filename := strings.TrimSpace(requestedFilename)
	if filename == "" {
		filename = path.Base(parsedURL.Path)
	}
	if filename == "." || filename == "/" || filename == "" {
		filename = fmt.Sprintf("download-%d", time.Now().UnixNano())
	}

	filename = filepath.Base(filename)
	if filename == "." || filename == string(filepath.Separator) || filename == "" {
		return "", errors.New("filename is invalid")
	}

	return filename, nil
}

func createDownloadFile(dir string, filename string) (string, *os.File, error) {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	for i := 0; i < 100; i++ {
		candidate := filename
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d%s", base, time.Now().UnixNano(), ext)
		}

		targetPath := filepath.Join(dir, candidate)
		file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			return targetPath, file, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", nil, fmt.Errorf("create download file: %w", err)
		}
	}

	return "", nil, errors.New("create download file: too many filename collisions")
}
