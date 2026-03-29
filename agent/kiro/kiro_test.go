package kiro

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhg5/cc-connect/agent/acp"
	"github.com/chenhg5/cc-connect/core"
)

func installFakeKiroCLI(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "kiro-cli")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake kiro-cli: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestNew_DefaultACPArgs(t *testing.T) {
	installFakeKiroCLI(t)

	a, err := New(map[string]any{"work_dir": "/tmp/project"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	agent := a.(*Agent)
	if got := agent.Name(); got != "kiro" {
		t.Fatalf("Name() = %q, want kiro", got)
	}
	if got := agent.CLIBinaryName(); got != "kiro-cli" {
		t.Fatalf("CLIBinaryName() = %q, want kiro-cli", got)
	}
	if got := agent.CLIDisplayName(); got != "Kiro CLI" {
		t.Fatalf("CLIDisplayName() = %q, want Kiro CLI", got)
	}
	if got := agent.base.GetWorkDir(); got != "/tmp/project" {
		t.Fatalf("work dir = %q, want /tmp/project", got)
	}
	if got := agent.base; got == nil {
		t.Fatal("expected wrapped ACP agent")
	}
}

func TestNew_ModeMapsToACPAgentFlag(t *testing.T) {
	installFakeKiroCLI(t)

	a, err := New(map[string]any{"mode": "spec-designer"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	agent := a.(*Agent)
	if got := agent.agentName; got != "spec-designer" {
		t.Fatalf("agentName = %q, want spec-designer", got)
	}
	if got := agent.base.CLIDisplayName(); got != "Kiro CLI" {
		t.Fatalf("wrapped display name = %q, want Kiro CLI", got)
	}
}

func TestNew_DefaultModeDoesNotSetACPAgentFlag(t *testing.T) {
	installFakeKiroCLI(t)

	a, err := New(map[string]any{"mode": "default"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	agent := a.(*Agent)
	if got := agent.agentName; got != "default" {
		t.Fatalf("agentName = %q, want default", got)
	}
}

func TestNew_ErrorWhenKiroCLIMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := New(nil)
	if err == nil {
		t.Fatal("expected missing kiro-cli error")
	}
	if !strings.Contains(err.Error(), "kiro-cli") {
		t.Fatalf("error = %q, want mention kiro-cli", err.Error())
	}
}

var _ core.Agent = (*Agent)(nil)
var _ core.AgentDoctorInfo = (*Agent)(nil)
var _ interface{ SetSessionEnv([]string) } = (*Agent)(nil)
var _ interface{ GetWorkDir() string } = (*Agent)(nil)
var _ = (*acp.Agent)(nil)
