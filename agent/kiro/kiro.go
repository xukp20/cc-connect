package kiro

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/chenhg5/cc-connect/agent/acp"
	"github.com/chenhg5/cc-connect/core"
)

func init() {
	core.RegisterAgent("kiro", New)
}

// Agent adapts Kiro CLI through the generic ACP transport.
type Agent struct {
	base      *acp.Agent
	agentName string
}

func New(opts map[string]any) (core.Agent, error) {
	if _, err := exec.LookPath("kiro-cli"); err != nil {
		return nil, fmt.Errorf("kiro: 'kiro-cli' not found in PATH, install from https://kiro.dev/docs/cli/installation/")
	}

	merged := make(map[string]any, len(opts)+4)
	for k, v := range opts {
		merged[k] = v
	}
	merged["command"] = "kiro-cli"
	merged["display_name"] = "Kiro CLI"

	args := []string{"acp"}
	agentName, _ := opts["mode"].(string)
	agentName = strings.TrimSpace(agentName)
	if agentName == "" {
		if explicit, _ := opts["agent"].(string); strings.TrimSpace(explicit) != "" {
			agentName = strings.TrimSpace(explicit)
		}
	}
	if agentName != "" && !strings.EqualFold(agentName, "default") {
		args = append(args, "--agent", agentName)
	}
	merged["args"] = args

	base, err := acp.New(merged)
	if err != nil {
		return nil, err
	}
	return &Agent{
		base:      base.(*acp.Agent),
		agentName: agentName,
	}, nil
}

func (a *Agent) Name() string { return "kiro" }

func (a *Agent) SetWorkDir(dir string) { a.base.SetWorkDir(dir) }

func (a *Agent) GetWorkDir() string { return a.base.GetWorkDir() }

func (a *Agent) SetSessionEnv(env []string) { a.base.SetSessionEnv(env) }

func (a *Agent) StartSession(ctx context.Context, sessionID string) (core.AgentSession, error) {
	return a.base.StartSession(ctx, sessionID)
}

func (a *Agent) ListSessions(ctx context.Context) ([]core.AgentSessionInfo, error) {
	return a.base.ListSessions(ctx)
}

func (a *Agent) Stop() error { return a.base.Stop() }

func (a *Agent) CLIBinaryName() string { return "kiro-cli" }

func (a *Agent) CLIDisplayName() string { return "Kiro CLI" }
