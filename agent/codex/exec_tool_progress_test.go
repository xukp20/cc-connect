package codex

import (
	"context"
	"testing"

	"github.com/chenhg5/cc-connect/core"
)

func TestCodexMCPToolInput_FormatsSections(t *testing.T) {
	item := map[string]any{
		"type":   "mcp_tool_call",
		"server": "lean_steward",
		"tool":   "lint_check",
		"arguments": map[string]any{
			"active_node_path": "Demo.Node",
		},
	}

	got := codexMCPToolInput(item)
	want := "server>\nlean_steward\n\ntool>\nlint_check\n\narguments>\n{\"active_node_path\":\"Demo.Node\"}"
	if got != want {
		t.Fatalf("codexMCPToolInput() = %q, want %q", got, want)
	}
}

func TestCodexMCPToolResultBody_PrefersTextContent(t *testing.T) {
	item := map[string]any{
		"result": map[string]any{
			"content": []any{
				map[string]any{"type": "text", "text": "{\"success\":true}"},
			},
			"structured_content": map[string]any{"success": true},
		},
	}

	got := codexMCPToolResultBody(item)
	want := "result>\n{\"success\":true}\n\nstructured_content>\n{\"success\":true}"
	if got != want {
		t.Fatalf("codexMCPToolResultBody() = %q, want %q", got, want)
	}
}

func TestCodexSession_HandleItemStarted_MCPToolCallEmitsToolUse(t *testing.T) {
	cs := &codexSession{
		events: make(chan core.Event, 4),
		ctx:    context.Background(),
	}
	raw := map[string]any{
		"item": map[string]any{
			"type":   "mcp_tool_call",
			"server": "lean_steward",
			"tool":   "lint_check",
			"arguments": map[string]any{
				"active_node_path": "Demo.Node",
			},
		},
	}

	cs.handleItemStarted(raw)

	select {
	case evt := <-cs.events:
		if evt.Type != core.EventToolUse {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolUse)
		}
		if evt.ToolName != "MCP" {
			t.Fatalf("tool name = %q, want MCP", evt.ToolName)
		}
		if evt.ToolInput == "" || evt.ToolInput[:7] != "server>" {
			t.Fatalf("tool input = %q, want MCP input sections", evt.ToolInput)
		}
	default:
		t.Fatal("expected tool use event")
	}
}

func TestCodexSession_HandleItemCompleted_MCPToolCallEmitsToolResult(t *testing.T) {
	cs := &codexSession{
		events: make(chan core.Event, 4),
		ctx:    context.Background(),
	}
	raw := map[string]any{
		"item": map[string]any{
			"type":   "mcp_tool_call",
			"status": "completed",
			"server": "lean_steward",
			"tool":   "lint_check",
			"arguments": map[string]any{
				"active_node_path": "Demo.Node",
			},
			"result": map[string]any{
				"content": []any{
					map[string]any{"type": "text", "text": "{\"success\":true}"},
				},
				"structured_content": map[string]any{"success": true},
			},
			"error": nil,
		},
	}

	cs.handleItemCompleted(raw)

	select {
	case evt := <-cs.events:
		if evt.Type != core.EventToolResult {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolResult)
		}
		if evt.ToolName != "MCP" {
			t.Fatalf("tool name = %q, want MCP", evt.ToolName)
		}
		if evt.ToolStatus != "completed" {
			t.Fatalf("tool status = %q, want completed", evt.ToolStatus)
		}
		if evt.ToolSuccess == nil || !*evt.ToolSuccess {
			t.Fatalf("tool success = %#v, want true", evt.ToolSuccess)
		}
		if evt.ToolResult != "result>\n{\"success\":true}\n\nstructured_content>\n{\"success\":true}" {
			t.Fatalf("tool result = %q, want MCP result sections", evt.ToolResult)
		}
	default:
		t.Fatal("expected tool result event")
	}
}

func TestCodexSession_HandleItemCompleted_FileSearchEmitsToolResult(t *testing.T) {
	cs := &codexSession{
		events: make(chan core.Event, 4),
		ctx:    context.Background(),
	}
	raw := map[string]any{
		"item": map[string]any{
			"type":   "file_search",
			"status": "completed",
			"query":  "engine.go",
			"result": map[string]any{
				"content": []any{
					map[string]any{"type": "text", "text": "engine.go:42 matched"},
				},
			},
		},
	}

	cs.handleItemCompleted(raw)

	select {
	case evt := <-cs.events:
		if evt.Type != core.EventToolResult {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolResult)
		}
		if evt.ToolName != "FileSearch" {
			t.Fatalf("tool name = %q, want FileSearch", evt.ToolName)
		}
		if evt.ToolInput != "engine.go" {
			t.Fatalf("tool input = %q, want query", evt.ToolInput)
		}
		if evt.ToolResult != "engine.go:42 matched" {
			t.Fatalf("tool result = %q, want extracted text", evt.ToolResult)
		}
		if evt.ToolSuccess == nil || !*evt.ToolSuccess {
			t.Fatalf("tool success = %#v, want true", evt.ToolSuccess)
		}
	default:
		t.Fatal("expected tool result event")
	}
}
