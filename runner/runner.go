package runner

import (
	"encoding/json"

	"afagent/runner/deletefile"
	"afagent/runner/demo"
	"afagent/runner/downloadfile"
	"afagent/runner/fileexists"
	"afagent/runner/makefileexecutable"
	"afagent/runner/movefile"
	"afagent/runner/processexists"
	"afagent/runner/runstartupscript"
	"afagent/runner/terminateprocesses"
)

const TaskNameDemo = demo.Name
const TaskNameDeleteFile = deletefile.Name
const TaskNameDownloadFile = downloadfile.Name
const TaskNameFileExists = fileexists.Name
const TaskNameMakeFileExecutable = makefileexecutable.Name
const TaskNameMoveFile = movefile.Name
const TaskNameProcessExists = processexists.Name
const TaskNameRunStartupScript = runstartupscript.Name
const TaskNameTerminateProcesses = terminateprocesses.Name

type TaskRunner func(payload json.RawMessage) (any, error)

func Registry() map[string]TaskRunner {
	return map[string]TaskRunner{
		TaskNameDemo:               demo.Execute,
		TaskNameDeleteFile:         deletefile.Execute,
		TaskNameDownloadFile:       downloadfile.Execute,
		TaskNameFileExists:         fileexists.Execute,
		TaskNameMakeFileExecutable: makefileexecutable.Execute,
		TaskNameMoveFile:           movefile.Execute,
		TaskNameProcessExists:      processexists.Execute,
		TaskNameRunStartupScript:   runstartupscript.Execute,
		TaskNameTerminateProcesses: terminateprocesses.Execute,
	}
}
