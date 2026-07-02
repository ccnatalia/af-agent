//go:build !windows

package makefileexecutable

import "os"

func makeExecutable(absPath string, info os.FileInfo) error {
	return os.Chmod(absPath, info.Mode().Perm()|0100)
}
