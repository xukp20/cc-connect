package codex

import (
	"encoding/json"
	"strings"
)

func codexPatchChangesSummary(raw any) string {
	changes, ok := raw.([]any)
	if !ok || len(changes) == 0 {
		return ""
	}

	lines := make([]string, 0, len(changes))
	for _, entry := range changes {
		change, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if line := codexPatchChangeLine(change); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "changes>\n" + strings.Join(lines, "\n")
}

func codexPatchChangeLine(change map[string]any) string {
	path, _ := change["path"].(string)
	path = strings.TrimSpace(path)

	kindType, movePath := codexPatchKind(change["kind"])
	movePath = strings.TrimSpace(movePath)

	switch kindType {
	case "add":
		if path == "" {
			return ""
		}
		return "A " + path
	case "delete":
		if path == "" {
			return ""
		}
		return "D " + path
	case "update":
		if path == "" {
			path = movePath
		}
		if path == "" {
			return ""
		}
		if movePath != "" && movePath != path {
			return "R " + path + " -> " + movePath
		}
		return "M " + path
	default:
		if path != "" {
			return strings.ToUpper(kindType) + " " + path
		}
	}

	blob, err := json.Marshal(change)
	if err != nil {
		return ""
	}
	return string(blob)
}

func codexPatchKind(raw any) (kindType string, movePath string) {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v), ""
	case map[string]any:
		kindType, _ = v["type"].(string)
		movePath, _ = v["move_path"].(string)
		if strings.TrimSpace(movePath) == "" {
			movePath, _ = v["movePath"].(string)
		}
		return strings.TrimSpace(kindType), strings.TrimSpace(movePath)
	default:
		return "", ""
	}
}

func codexPatchResultBody(item map[string]any) string {
	if item == nil {
		return ""
	}
	if msg, _ := item["message"].(string); strings.TrimSpace(msg) != "" {
		return "error>\n" + strings.TrimSpace(msg)
	}
	if errText := strings.TrimSpace(appServerJSON(item["error"])); errText != "" && errText != "null" {
		return "error>\n" + errText
	}
	return ""
}
