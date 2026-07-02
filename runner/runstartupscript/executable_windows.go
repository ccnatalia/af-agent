//go:build windows

package runstartupscript

import "os"

func isExecutable(info os.FileInfo) bool {
	return info.Mode().IsRegular()
}
