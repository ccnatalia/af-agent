package terminateprocesses

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	processlib "afagent/runner/internal/process"
)

func TestTerminateMatchingTerminatesMatches(t *testing.T) {
	var terminatedPIDs []int

	got, err := terminateMatching(
		"worker",
		100,
		func() ([]processlib.Info, error) {
			return []processlib.Info{
				{PID: 100, Command: "af-agent worker"},
				{PID: 101, Command: "sidecar worker --queue default"},
				{PID: 102, Command: "unrelated process"},
				{PID: 103, Command: "batch-worker --once"},
			}, nil
		},
		func(pid int) error {
			terminatedPIDs = append(terminatedPIDs, pid)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if want := []int{101, 103}; !reflect.DeepEqual(terminatedPIDs, want) {
		t.Fatalf("terminated pids = %v, want %v", terminatedPIDs, want)
	}
	if len(got) != 2 {
		t.Fatalf("terminated processes = %d, want 2", len(got))
	}
}

func TestTerminateMatchingReturnsTerminateError(t *testing.T) {
	_, err := terminateMatching(
		"worker",
		100,
		func() ([]processlib.Info, error) {
			return []processlib.Info{{PID: 101, Command: "worker"}}, nil
		},
		func(pid int) error {
			return errors.New("permission denied")
		},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExecuteRejectsShortKeyword(t *testing.T) {
	payload, err := json.Marshal(Payload{
		Keyword: "sh",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(payload); err == nil {
		t.Fatal("expected error, got nil")
	}
}
