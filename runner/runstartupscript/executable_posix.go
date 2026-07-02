//go:build !windows

package runstartupscript

import "os"

func isExecutable(info os.FileInfo) bool {
	return info.Mode().Perm()&0111 != 0
}
