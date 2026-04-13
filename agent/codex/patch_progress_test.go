package codex

import "testing"

func TestCodexPatchChangesSummary_FormatsReadableLines(t *testing.T) {
	raw := []any{
		map[string]any{
			"path": "/tmp/new.txt",
			"kind": map[string]any{"type": "add"},
			"diff": "hello\n",
		},
		map[string]any{
			"path": "agent/codex/session.go",
			"kind": map[string]any{"type": "update", "move_path": "agent/codex/session_v2.go"},
			"diff": "@@\n-old\n+new\n",
		},
		map[string]any{
			"path": "old.txt",
			"kind": "delete",
		},
	}

	got := codexPatchChangesSummary(raw)
	want := "changes>\nA /tmp/new.txt\nR agent/codex/session.go -> agent/codex/session_v2.go\nD old.txt"
	if got != want {
		t.Fatalf("codexPatchChangesSummary() = %q, want %q", got, want)
	}
}

func TestCodexPatchResultBody_FormatsErrorMessage(t *testing.T) {
	item := map[string]any{"message": "patch failed: invalid hunk"}

	got := codexPatchResultBody(item)
	want := "error>\npatch failed: invalid hunk"
	if got != want {
		t.Fatalf("codexPatchResultBody() = %q, want %q", got, want)
	}
}
