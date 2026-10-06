package filemanifest

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const Filename = "file-md5.txt"

const temporaryFilePrefix = ".file-md5-"

type record struct {
	Path string
	MD5  string
}

type manifestLock struct {
	mutex      sync.Mutex
	references int
}

var manifestLocks = struct {
	sync.Mutex
	byRootDir map[string]*manifestLock
}{byRootDir: make(map[string]*manifestLock)}

func FullUpdate(rootDir string) error {
	rootDir, err := validateRootDir(rootDir)
	if err != nil {
		return err
	}
	unlock := lockManifest(rootDir)
	defer unlock()

	manifestPath := filepath.Join(rootDir, Filename)
	records := make([]record, 0)
	err = filepath.WalkDir(rootDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == rootDir || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if path == manifestPath || isTemporaryManifest(rootDir, path) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat file %q: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		nativeRelativePath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("resolve relative path %q: %w", path, err)
		}
		relativePath := filepath.ToSlash(nativeRelativePath)
		if err := validateRelativePath(relativePath); err != nil {
			return err
		}

		fileMD5, err := hashFile(path)
		if err != nil {
			return err
		}
		records = append(records, record{Path: relativePath, MD5: fileMD5})
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk root directory: %w", err)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Path < records[j].Path
	})

	return replaceManifest(rootDir, func(writer *bufio.Writer) error {
		for _, item := range records {
			if err := writeRecord(writer, item); err != nil {
				return err
			}
		}
		return nil
	})
}

func IncrementalUpdate(rootDir string, relativePath string) error {
	rootDir, err := validateRootDir(rootDir)
	if err != nil {
		return err
	}

	relativePath = path.Clean(relativePath)
	if err := validateRelativePath(relativePath); err != nil {
		return err
	}
	if relativePath == Filename || (!strings.Contains(relativePath, "/") && isTemporaryManifestName(relativePath)) {
		return errors.New("relative path must not reference the manifest")
	}
	unlock := lockManifest(rootDir)
	defer unlock()

	filePath := filepath.Join(rootDir, filepath.FromSlash(relativePath))
	if err := rejectSymlinkPath(rootDir, relativePath); err != nil {
		return err
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("stat file %q: %w", relativePath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("path %q must be a regular file", relativePath)
	}

	fileMD5, err := hashFile(filePath)
	if err != nil {
		return err
	}
	updated := record{Path: relativePath, MD5: fileMD5}

	return replaceManifest(rootDir, func(writer *bufio.Writer) error {
		manifestPath := filepath.Join(rootDir, Filename)
		manifest, err := os.Open(manifestPath)
		if errors.Is(err, os.ErrNotExist) {
			return writeRecord(writer, updated)
		}
		if err != nil {
			return fmt.Errorf("open manifest: %w", err)
		}
		defer manifest.Close()

		scanner := bufio.NewScanner(manifest)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		inserted := false
		previousPath := ""
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			current, err := parseRecord(scanner.Text())
			if err != nil {
				return fmt.Errorf("parse manifest line %d: %w", lineNumber, err)
			}
			if lineNumber > 1 && current.Path <= previousPath {
				return fmt.Errorf("manifest line %d: paths must be strictly sorted", lineNumber)
			}
			previousPath = current.Path

			if !inserted && updated.Path < current.Path {
				if err := writeRecord(writer, updated); err != nil {
					return err
				}
				inserted = true
			}
			if current.Path == updated.Path {
				if !inserted {
					if err := writeRecord(writer, updated); err != nil {
						return err
					}
					inserted = true
				}
				continue
			}
			if err := writeRecord(writer, current); err != nil {
				return err
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read manifest: %w", err)
		}
		if !inserted {
			return writeRecord(writer, updated)
		}
		return nil
	})
}

func lockManifest(rootDir string) func() {
	manifestLocks.Lock()
	lock := manifestLocks.byRootDir[rootDir]
	if lock == nil {
		lock = &manifestLock{}
		manifestLocks.byRootDir[rootDir] = lock
	}
	lock.references++
	manifestLocks.Unlock()

	lock.mutex.Lock()
	return func() {
		lock.mutex.Unlock()

		manifestLocks.Lock()
		lock.references--
		if lock.references == 0 {
			delete(manifestLocks.byRootDir, rootDir)
		}
		manifestLocks.Unlock()
	}
}

func validateRootDir(rootDir string) (string, error) {
	if rootDir == "" {
		return "", errors.New("root directory is required")
	}
	absolutePath, err := filepath.Abs(rootDir)
	if err != nil {
		return "", fmt.Errorf("resolve root directory: %w", err)
	}
	info, err := os.Lstat(absolutePath)
	if err != nil {
		return "", fmt.Errorf("stat root directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("root directory must not be a symlink")
	}
	if !info.IsDir() {
		return "", errors.New("root directory must be a directory")
	}
	return absolutePath, nil
}

func validateRelativePath(relativePath string) error {
	if relativePath == "" || relativePath == "." {
		return errors.New("relative path is required")
	}
	if path.IsAbs(relativePath) {
		return errors.New("path must be relative to root directory")
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, "../") {
		return errors.New("path must stay inside root directory")
	}
	if strings.Contains(relativePath, "\\") {
		return errors.New("path must use forward slashes as directory separators")
	}
	if strings.ContainsAny(relativePath, "\r\n") {
		return errors.New("path must not contain a line break")
	}
	return nil
}

func rejectSymlinkPath(rootDir string, relativePath string) error {
	currentPath := rootDir
	for _, part := range strings.Split(relativePath, "/") {
		currentPath = filepath.Join(currentPath, part)
		info, err := os.Lstat(currentPath)
		if err != nil {
			return fmt.Errorf("stat file %q: %w", relativePath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path %q must not contain symlinks", relativePath)
		}
	}
	return nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file %q: %w", path, err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash file %q: %w", path, err)
	}
	return strings.ToUpper(hex.EncodeToString(hash.Sum(nil))), nil
}

func replaceManifest(rootDir string, write func(*bufio.Writer) error) error {
	temporaryFile, err := os.CreateTemp(rootDir, temporaryFilePrefix+"*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary manifest: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temporaryFile.Close()
		}
		_ = os.Remove(temporaryPath)
	}()

	writer := bufio.NewWriter(temporaryFile)
	if err := write(writer); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush temporary manifest: %w", err)
	}
	if err := temporaryFile.Sync(); err != nil {
		return fmt.Errorf("sync temporary manifest: %w", err)
	}
	if err := temporaryFile.Chmod(0644); err != nil {
		return fmt.Errorf("set manifest permissions: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		return fmt.Errorf("close temporary manifest: %w", err)
	}
	closed = true

	manifestPath := filepath.Join(rootDir, Filename)
	if err := os.Rename(temporaryPath, manifestPath); err != nil {
		return fmt.Errorf("replace manifest: %w", err)
	}
	return nil
}

func parseRecord(line string) (record, error) {
	if len(line) < 34 || line[32] != ',' {
		return record{}, errors.New("record must use MD5,path format")
	}
	item := record{MD5: line[:32], Path: line[33:]}
	if !isUpperMD5(item.MD5) {
		return record{}, errors.New("MD5 must be 32 uppercase hexadecimal characters")
	}
	if err := validateRelativePath(item.Path); err != nil {
		return record{}, err
	}
	return item, nil
}

func isUpperMD5(value string) bool {
	if len(value) != md5.Size*2 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

func writeRecord(writer *bufio.Writer, item record) error {
	if _, err := fmt.Fprintf(writer, "%s,%s\n", item.MD5, item.Path); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func isTemporaryManifest(rootDir string, path string) bool {
	return filepath.Dir(path) == rootDir && isTemporaryManifestName(filepath.Base(path))
}

func isTemporaryManifestName(name string) bool {
	return strings.HasPrefix(name, temporaryFilePrefix) && strings.HasSuffix(name, ".tmp")
}
