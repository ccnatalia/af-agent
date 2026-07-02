package processexists

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	processlib "afagent/runner/internal/process"
)

func TestExecuteReturnsMatches(t *testing.T) {
	payload, err := json.Marshal(Payload{
		Keyword: "worker",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := executeWithList(
		payload,
		100,
		func() ([]processlib.Info, error) {
			return []processlib.Info{
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

	result, ok := got.(Result)
	if !ok {
		t.Fatalf("result type = %T, want Result", got)
	}
	want := []processlib.Matched{
		{PID: 101, Command: "sidecar worker --queue default"},
		{PID: 103, Command: "batch-worker --once"},
	}
	if !reflect.DeepEqual(result.Processes, want) {
		t.Fatalf("matched processes = %#v, want %#v", result.Processes, want)
	}
	if !result.Exists {
		t.Fatal("exists = false, want true")
	}
	if result.Count != 2 {
		t.Fatalf("count = %d, want 2", result.Count)
	}
}

func TestExecuteReturnsStableEmptyProcesses(t *testing.T) {
	payload, err := json.Marshal(Payload{
		Keyword: "missing",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := executeWithList(
		payload,
		100,
		func() ([]processlib.Info, error) {
			return []processlib.Info{{PID: 101, Command: "worker"}}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := got.(Result)
	if !ok {
		t.Fatalf("result type = %T, want Result", got)
	}
	if result.Exists {
		t.Fatal("exists = true, want false")
	}
	if result.Count != 0 {
		t.Fatalf("count = %d, want 0", result.Count)
	}
	if result.Processes == nil {
		t.Fatal("processes must be a non-nil empty slice")
	}
	if len(result.Processes) != 0 {
		t.Fatalf("processes = %#v, want empty", result.Processes)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"processes":[]`) {
		t.Fatalf("encoded result = %s, want processes empty array", encoded)
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
