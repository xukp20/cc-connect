package codex

import "strings"

func appServerCollabToolInput(item map[string]any) string {
	var sections []string
	appendSection := func(header string, lines ...string) {
		header = strings.TrimSpace(header)
		if header == "" {
			return
		}
		var trimmed []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				trimmed = append(trimmed, line)
			}
		}
		if len(trimmed) == 0 {
			return
		}
		sections = append(sections, header+">\n"+strings.Join(trimmed, "\n"))
	}

	if tool, _ := item["tool"].(string); tool != "" {
		appendSection("tool", tool)
	}
	if prompt, _ := item["prompt"].(string); prompt != "" {
		appendSection("prompt", prompt)
	}
	appendSection("receivers", appServerStringList(item["receiverThreadIds"])...)
	if model, _ := item["model"].(string); model != "" {
		appendSection("model", model)
	}
	if effort, _ := item["reasoningEffort"].(string); effort != "" {
		appendSection("reasoning_effort", effort)
	}
	return strings.Join(sections, "\n\n")
}

func appServerCollabToolResultBody(item map[string]any) string {
	states, _ := item["agentsStates"].(map[string]any)
	if len(states) == 0 {
		return ""
	}
	var lines []string
	for agentID, raw := range states {
		state, _ := raw.(map[string]any)
		status, _ := state["status"].(string)
		line := strings.TrimSpace(agentID)
		if strings.TrimSpace(status) != "" {
			line += ": " + strings.TrimSpace(status)
		}
		if msg, _ := state["message"].(string); strings.TrimSpace(msg) != "" {
			line += " (" + strings.TrimSpace(msg) + ")"
		}
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "agents>\n" + strings.Join(lines, "\n")
}

func appServerImageGenerationInput(item map[string]any) string {
	if prompt, _ := item["revisedPrompt"].(string); strings.TrimSpace(prompt) != "" {
		return strings.TrimSpace(prompt)
	}
	return ""
}

func appServerImageGenerationResultBody(item map[string]any) string {
	var sections []string
	appendSection := func(header string, body string) {
		header = strings.TrimSpace(header)
		body = strings.TrimSpace(body)
		if header == "" || body == "" {
			return
		}
		sections = append(sections, header+">\n"+body)
	}

	if result, _ := item["result"].(string); strings.TrimSpace(result) != "" {
		appendSection("result", result)
	}
	if path, _ := item["savedPath"].(string); strings.TrimSpace(path) != "" {
		appendSection("saved_path", path)
	}
	return strings.Join(sections, "\n\n")
}

func appServerImageViewInput(item map[string]any) string {
	if path, _ := item["path"].(string); strings.TrimSpace(path) != "" {
		return "path>\n" + strings.TrimSpace(path)
	}
	return ""
}

func appServerStringList(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, entry := range items {
		s, _ := entry.(string)
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
