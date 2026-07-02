package main

import (
	"os"
	"testing"
)

func TestConfigureWorkspaceFromEnvUsesAFAGENTWorkspace(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get original working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	workspaceDir := t.TempDir()
	t.Setenv("AF_AGENT_WORKSPACE", workspaceDir)

	if err := configureWorkspaceFromEnv(); err != nil {
		t.Fatalf("configureWorkspaceFromEnv returned error: %v", err)
	}

	got, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	wantInfo, err := os.Stat(workspaceDir)
	if err != nil {
		t.Fatal(err)
	}

	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("working directory = %q, want %q", got, workspaceDir)
	}
}

func TestConfigureWorkspaceFromEnvLeavesWorkingDirWhenUnset(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get original working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	t.Setenv("AF_AGENT_WORKSPACE", "")

	if err := configureWorkspaceFromEnv(); err != nil {
		t.Fatalf("configureWorkspaceFromEnv returned error: %v", err)
	}

	got, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if got != originalWd {
		t.Fatalf("working directory = %q, want %q", got, originalWd)
	}
}
