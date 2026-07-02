package runner

import (
	"encoding/json"

	"afagent/runner/demo"
	"afagent/runner/downloadfile"
	"afagent/runner/makefileexecutable"
	"afagent/runner/movefile"
	"afagent/runner/processexists"
	"afagent/runner/terminateprocesses"
)

const TaskNameDemo = demo.Name
const TaskNameDownloadFile = downloadfile.Name
const TaskNameMakeFileExecutable = makefileexecutable.Name
const TaskNameMoveFile = movefile.Name
const TaskNameProcessExists = processexists.Name
const TaskNameTerminateProcesses = terminateprocesses.Name

type TaskRunner func(payload json.RawMessage) (any, error)

func Registry() map[string]TaskRunner {
	return map[string]TaskRunner{
		TaskNameDemo:               demo.Execute,
		TaskNameDownloadFile:       downloadfile.Execute,
		TaskNameMakeFileExecutable: makefileexecutable.Execute,
		TaskNameMoveFile:           movefile.Execute,
		TaskNameProcessExists:      processexists.Execute,
		TaskNameTerminateProcesses: terminateprocesses.Execute,
	}
}
