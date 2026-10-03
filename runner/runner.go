package runner

import (
	"encoding/json"

	"afagent/runner/deletefile"
	"afagent/runner/demo"
	"afagent/runner/downloadfile"
	"afagent/runner/downloadfilev2"
	"afagent/runner/fileexists"
	"afagent/runner/fullupdatefilemanifest"
	"afagent/runner/incrementalupdatefilemanifest"
	"afagent/runner/makefileexecutable"
	"afagent/runner/movefile"
	"afagent/runner/processexists"
	"afagent/runner/readfile"
	"afagent/runner/runstartupscript"
	"afagent/runner/terminateprocesses"
)

const TaskNameDemo = demo.Name
const TaskNameDeleteFile = deletefile.Name
const TaskNameDownloadFile = downloadfile.Name
const TaskNameDownloadFileV2 = downloadfilev2.Name
const TaskNameFileExists = fileexists.Name
const TaskNameFullUpdateFileManifest = fullupdatefilemanifest.Name
const TaskNameIncrementalUpdateFileManifest = incrementalupdatefilemanifest.Name
const TaskNameMakeFileExecutable = makefileexecutable.Name
const TaskNameMoveFile = movefile.Name
const TaskNameProcessExists = processexists.Name
const TaskNameReadFile = readfile.Name
const TaskNameRunStartupScript = runstartupscript.Name
const TaskNameTerminateProcesses = terminateprocesses.Name

type TaskRunner func(payload json.RawMessage) (any, error)

func Registry() map[string]TaskRunner {
	return map[string]TaskRunner{
		TaskNameDemo:                          demo.Execute,
		TaskNameDeleteFile:                    deletefile.Execute,
		TaskNameDownloadFile:                  downloadfile.Execute,
		TaskNameDownloadFileV2:                downloadfilev2.Execute,
		TaskNameFileExists:                    fileexists.Execute,
		TaskNameFullUpdateFileManifest:        fullupdatefilemanifest.Execute,
		TaskNameIncrementalUpdateFileManifest: incrementalupdatefilemanifest.Execute,
		TaskNameMakeFileExecutable:            makefileexecutable.Execute,
		TaskNameMoveFile:                      movefile.Execute,
		TaskNameProcessExists:                 processexists.Execute,
		TaskNameReadFile:                      readfile.Execute,
		TaskNameRunStartupScript:              runstartupscript.Execute,
		TaskNameTerminateProcesses:            terminateprocesses.Execute,
	}
}
