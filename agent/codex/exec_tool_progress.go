package codex

import (
	"strings"
)

func codexExecToolName(itemType string) string {
	switch itemType {
	case "mcp_tool_call", "mcp_tool":
		return "MCP"
	default:
		return codexToolNames[itemType]
	}
}

func codexExecToolInput(item map[string]any) string {
	itemType, _ := item["type"].(string)
	switch itemType {
	case "web_search":
		return codexWebSearchInput(item)
	case "mcp_tool_call", "mcp_tool":
		return codexMCPToolInput(item)
	default:
		return codexExtractToolInput(item)
	}
}

func codexMCPToolInput(item map[string]any) string {
	server, _ := item["server"].(string)
	tool, _ := item["tool"].(string)
	argsText := appServerJSON(item["arguments"])

	var sections []string
	appendSection := func(header string, body string) {
		header = strings.TrimSpace(header)
		body = strings.TrimSpace(body)
		if header == "" || body == "" {
			return
		}
		sections = append(sections, header+">\n"+body)
	}

	appendSection("server", server)
	appendSection("tool", tool)
	appendSection("arguments", argsText)
	return strings.Join(sections, "\n\n")
}

func codexMCPToolResultBody(item map[string]any) string {
	if errText := codexErrorText(item["error"]); errText != "" {
		return "error>\n" + errText
	}
	if resultText := codexMCPResultText(item["result"]); resultText != "" {
		return resultText
	}
	return ""
}

func codexMCPResultText(raw any) string {
	result, ok := raw.(map[string]any)
	if !ok || result == nil {
		return appServerJSON(raw)
	}

	var sections []string
	appendSection := func(header string, body string) {
		header = strings.TrimSpace(header)
		body = strings.TrimSpace(body)
		if header == "" || body == "" {
			return
		}
		sections = append(sections, header+">\n"+body)
	}

	appendSection("result", codexContentText(result["content"]))
	appendSection("structured_content", appServerJSON(result["structured_content"]))
	if len(sections) > 0 {
		return strings.Join(sections, "\n\n")
	}
	return appServerJSON(raw)
}

func codexGenericToolResultBody(item map[string]any) string {
	if errText := codexErrorText(item["error"]); errText != "" {
		return "error>\n" + errText
	}
	for _, key := range []string{"output", "result", "content"} {
		if text := codexToolFieldText(item[key]); text != "" {
			return text
		}
	}
	if msg, _ := item["message"].(string); strings.TrimSpace(msg) != "" {
		return "error>\n" + strings.TrimSpace(msg)
	}
	return ""
}

func codexToolFieldText(raw any) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if text := codexContentText(v["content"]); text != "" {
			return text
		}
		if text, _ := v["text"].(string); strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
		return appServerJSON(v)
	case []any:
		if text := codexContentText(v); text != "" {
			return text
		}
		return appServerJSON(v)
	default:
		return appServerJSON(v)
	}
}

func codexContentText(raw any) string {
	items, ok := raw.([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, entry := range items {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if text, _ := m["text"].(string); strings.TrimSpace(text) != "" {
			parts = append(parts, strings.TrimSpace(text))
		}
	}
	return strings.Join(parts, "\n")
}

func codexErrorText(raw any) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if msg, _ := v["message"].(string); strings.TrimSpace(msg) != "" {
			return strings.TrimSpace(msg)
		}
		return appServerJSON(v)
	default:
		return appServerJSON(v)
	}
}
