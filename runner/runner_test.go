package runner

import "testing"

func TestRegistryIncludesRunners(t *testing.T) {
	registry := Registry()

	for _, name := range []string{
		TaskNameDemo,
		TaskNameDownloadFile,
		TaskNameMakeFileExecutable,
		TaskNameMoveFile,
		TaskNameProcessExists,
		TaskNameTerminateProcesses,
	} {
		if _, ok := registry[name]; !ok {
			t.Fatalf("task runner %q is not registered", name)
		}
	}
}
