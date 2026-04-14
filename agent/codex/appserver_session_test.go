package codex

import (
	"testing"

	"github.com/chenhg5/cc-connect/core"
)

func TestAppServerWebSearchInput_FallsBackToActionFields(t *testing.T) {
	item := map[string]any{
		"type":  "webSearch",
		"query": "",
		"action": map[string]any{
			"type":    "search",
			"query":   "OpenAI Codex app server documentation",
			"queries": []any{"OpenAI Codex app server documentation", "Codex app server initialize"},
		},
	}

	got := appServerWebSearchInput(item)
	want := "OpenAI Codex app server documentation"
	if got != want {
		t.Fatalf("appServerWebSearchInput() = %q, want %q", got, want)
	}
}

func TestAppServerWebSearchResultBody_Search(t *testing.T) {
	item := map[string]any{
		"type":  "webSearch",
		"query": "OpenAI Codex app server documentation",
		"action": map[string]any{
			"type":    "search",
			"query":   "OpenAI Codex app server documentation",
			"queries": []any{"OpenAI Codex app server documentation"},
		},
	}

	got := appServerWebSearchResultBody(item)
	want := "search>\nOpenAI Codex app server documentation"
	if got != want {
		t.Fatalf("appServerWebSearchResultBody() = %q, want %q", got, want)
	}
}

func TestAppServerWebSearchResultBody_OpenPage(t *testing.T) {
	item := map[string]any{
		"type":  "webSearch",
		"query": "OpenAI Codex app server documentation",
		"action": map[string]any{
			"type": "openPage",
			"url":  "https://developers.openai.com/codex/app-server",
		},
	}

	got := appServerWebSearchResultBody(item)
	want := "query>\nOpenAI Codex app server documentation\n\nopen_page>\nhttps://developers.openai.com/codex/app-server"
	if got != want {
		t.Fatalf("appServerWebSearchResultBody() = %q, want %q", got, want)
	}
}

func TestAppServerSession_HandleItemCompleted_WebSearchBackfillsInputAndFormatsResult(t *testing.T) {
	s := &appServerSession{
		events: make(chan core.Event, 4),
	}
	item := map[string]any{
		"type":  "webSearch",
		"query": "",
		"action": map[string]any{
			"type":    "search",
			"query":   "OpenAI Codex app server documentation",
			"queries": []any{"OpenAI Codex app server documentation"},
		},
	}

	s.handleItemCompleted(item)

	select {
	case evt := <-s.events:
		if evt.Type != core.EventToolResult {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolResult)
		}
		if evt.ToolName != "WebSearch" {
			t.Fatalf("tool name = %q, want WebSearch", evt.ToolName)
		}
		if evt.ToolInput != "OpenAI Codex app server documentation" {
			t.Fatalf("tool input = %q, want query backfill", evt.ToolInput)
		}
		if evt.ToolResult != "search>\nOpenAI Codex app server documentation" {
			t.Fatalf("tool result = %q, want formatted search body", evt.ToolResult)
		}
		if evt.ToolStatus != "completed" {
			t.Fatalf("tool status = %q, want completed", evt.ToolStatus)
		}
		if evt.ToolSuccess == nil || !*evt.ToolSuccess {
			t.Fatalf("tool success = %#v, want true", evt.ToolSuccess)
		}
	default:
		t.Fatal("expected tool result event")
	}
}

func TestAppServerSession_HandleItemStarted_FileChangeUsesReadablePatchSummary(t *testing.T) {
	s := &appServerSession{
		events: make(chan core.Event, 4),
	}
	item := map[string]any{
		"type": "fileChange",
		"changes": []any{
			map[string]any{
				"path": "/tmp/1.txt",
				"kind": map[string]any{"type": "add"},
				"diff": "hello\n",
			},
		},
	}

	s.handleItemStarted(item)

	select {
	case evt := <-s.events:
		if evt.Type != core.EventToolUse {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolUse)
		}
		if evt.ToolName != "Patch" {
			t.Fatalf("tool name = %q, want Patch", evt.ToolName)
		}
		if evt.ToolInput != "changes>\nA /tmp/1.txt" {
			t.Fatalf("tool input = %q, want readable patch summary", evt.ToolInput)
		}
	default:
		t.Fatal("expected tool use event")
	}
}

func TestAppServerSession_HandleItemCompleted_FileChangeEmitsToolResult(t *testing.T) {
	s := &appServerSession{
		events: make(chan core.Event, 4),
	}
	item := map[string]any{
		"type":   "fileChange",
		"status": "completed",
		"changes": []any{
			map[string]any{
				"path": "/tmp/1.txt",
				"kind": map[string]any{"type": "add"},
				"diff": "hello\n",
			},
		},
	}

	s.handleItemCompleted(item)

	select {
	case evt := <-s.events:
		if evt.Type != core.EventToolResult {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolResult)
		}
		if evt.ToolName != "Patch" {
			t.Fatalf("tool name = %q, want Patch", evt.ToolName)
		}
		if evt.ToolInput != "changes>\nA /tmp/1.txt" {
			t.Fatalf("tool input = %q, want readable patch summary", evt.ToolInput)
		}
		if evt.ToolResult != "" {
			t.Fatalf("tool result = %q, want empty success body", evt.ToolResult)
		}
		if evt.ToolStatus != "completed" {
			t.Fatalf("tool status = %q, want completed", evt.ToolStatus)
		}
		if evt.ToolSuccess == nil || !*evt.ToolSuccess {
			t.Fatalf("tool success = %#v, want true", evt.ToolSuccess)
		}
	default:
		t.Fatal("expected tool result event")
	}
}

func TestAppServerSession_HandleItemCompleted_MCPToolCallEmitsStructuredResult(t *testing.T) {
	s := &appServerSession{
		events: make(chan core.Event, 4),
	}
	item := map[string]any{
		"type":   "mcpToolCall",
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
			"structuredContent": map[string]any{"success": true},
		},
	}

	s.handleItemCompleted(item)

	select {
	case evt := <-s.events:
		if evt.Type != core.EventToolResult {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolResult)
		}
		if evt.ToolName != "MCP" {
			t.Fatalf("tool name = %q, want MCP", evt.ToolName)
		}
		if evt.ToolInput == "" || evt.ToolInput[:7] != "server>" {
			t.Fatalf("tool input = %q, want MCP input sections", evt.ToolInput)
		}
		want := "result>\n{\"success\":true}\n\nstructured_content>\n{\"success\":true}"
		if evt.ToolResult != want {
			t.Fatalf("tool result = %q, want %q", evt.ToolResult, want)
		}
		if evt.ToolSuccess == nil || !*evt.ToolSuccess {
			t.Fatalf("tool success = %#v, want true", evt.ToolSuccess)
		}
	default:
		t.Fatal("expected tool result event")
	}
}

func TestAppServerSession_HandleItemCompleted_CollabAgentToolCallEmitsResult(t *testing.T) {
	s := &appServerSession{
		events: make(chan core.Event, 4),
	}
	item := map[string]any{
		"type":              "collabAgentToolCall",
		"status":            "completed",
		"tool":              "spawnAgent",
		"prompt":            "Investigate the parser",
		"model":             "gpt-5.4-mini",
		"receiverThreadIds": []any{"thread-2"},
		"agentsStates": map[string]any{
			"thread-2": map[string]any{
				"status":  "running",
				"message": "agent booted",
			},
		},
	}

	s.handleItemCompleted(item)

	select {
	case evt := <-s.events:
		if evt.Type != core.EventToolResult {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolResult)
		}
		if evt.ToolName != "CollabAgent" {
			t.Fatalf("tool name = %q, want CollabAgent", evt.ToolName)
		}
		if evt.ToolInput == "" || evt.ToolInput[:5] != "tool>" {
			t.Fatalf("tool input = %q, want collab input sections", evt.ToolInput)
		}
		if evt.ToolResult != "agents>\nthread-2: running (agent booted)" {
			t.Fatalf("tool result = %q, want agent state summary", evt.ToolResult)
		}
		if evt.ToolSuccess == nil || !*evt.ToolSuccess {
			t.Fatalf("tool success = %#v, want true", evt.ToolSuccess)
		}
	default:
		t.Fatal("expected tool result event")
	}
}

func TestAppServerSession_HandleItemCompleted_ImageGenerationEmitsResult(t *testing.T) {
	s := &appServerSession{
		events: make(chan core.Event, 4),
	}
	item := map[string]any{
		"type":          "imageGeneration",
		"status":        "completed",
		"revisedPrompt": "draw a sunrise",
		"result":        "https://example.com/generated.png",
		"savedPath":     "/tmp/generated.png",
	}

	s.handleItemCompleted(item)

	select {
	case evt := <-s.events:
		if evt.Type != core.EventToolResult {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolResult)
		}
		if evt.ToolName != "ImageGeneration" {
			t.Fatalf("tool name = %q, want ImageGeneration", evt.ToolName)
		}
		if evt.ToolInput != "draw a sunrise" {
			t.Fatalf("tool input = %q, want revised prompt", evt.ToolInput)
		}
		want := "result>\nhttps://example.com/generated.png\n\nsaved_path>\n/tmp/generated.png"
		if evt.ToolResult != want {
			t.Fatalf("tool result = %q, want %q", evt.ToolResult, want)
		}
	default:
		t.Fatal("expected tool result event")
	}
}

func TestAppServerSession_HandleItemCompleted_ImageViewEmitsResult(t *testing.T) {
	s := &appServerSession{
		events: make(chan core.Event, 4),
	}
	item := map[string]any{
		"type": "imageView",
		"path": "/tmp/example.png",
	}

	s.handleItemCompleted(item)

	select {
	case evt := <-s.events:
		if evt.Type != core.EventToolResult {
			t.Fatalf("event type = %q, want %q", evt.Type, core.EventToolResult)
		}
		if evt.ToolName != "ImageView" {
			t.Fatalf("tool name = %q, want ImageView", evt.ToolName)
		}
		if evt.ToolInput != "path>\n/tmp/example.png" {
			t.Fatalf("tool input = %q, want image path section", evt.ToolInput)
		}
		if evt.ToolSuccess == nil || !*evt.ToolSuccess {
			t.Fatalf("tool success = %#v, want true", evt.ToolSuccess)
		}
	default:
		t.Fatal("expected tool result event")
	}
}
