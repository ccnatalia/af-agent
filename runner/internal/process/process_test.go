package process

import (
	"reflect"
	"testing"
)

func TestFindMatching(t *testing.T) {
	got, err := FindMatching(
		"worker",
		100,
		func() ([]Info, error) {
			return []Info{
				{PID: 100, Command: "af-agent worker"},
				{PID: 101, Command: "sidecar worker --queue default"},
				{PID: 102, Command: "unrelated process"},
				{PID: 103, Command: "batch-worker --once"},
			}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	want := []Matched{
		{PID: 101, Command: "sidecar worker --queue default"},
		{PID: 103, Command: "batch-worker --once"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("matched processes = %#v, want %#v", got, want)
	}
}

func TestListProcesses(t *testing.T) {
	processes, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(processes) == 0 {
		t.Fatal("expected at least one process")
	}

	for _, process := range processes {
		t.Logf("process.Info{PID:%d, Command:%q}", process.PID, process.Command)
	}
}
