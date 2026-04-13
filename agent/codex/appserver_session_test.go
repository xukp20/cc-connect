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
