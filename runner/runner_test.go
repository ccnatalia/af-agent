package runner

import "testing"

func TestRegistryIncludesRunners(t *testing.T) {
	registry := Registry()

	for _, name := range []string{
		TaskNameDemo,
		TaskNameDeleteFile,
		TaskNameDownloadFile,
		TaskNameDownloadFileV2,
		TaskNameFileExists,
		TaskNameFullUpdateFileManifest,
		TaskNameIncrementalUpdateFileManifest,
		TaskNameMakeFileExecutable,
		TaskNameMoveFile,
		TaskNameProcessExists,
		TaskNameReadFile,
		TaskNameRunStartupScript,
		TaskNameTerminateProcesses,
	} {
		if _, ok := registry[name]; !ok {
			t.Fatalf("task runner %q is not registered", name)
		}
	}
}
