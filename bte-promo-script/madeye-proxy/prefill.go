package main

import "strings"

func roleOf(msg map[string]any) string {
	r, _ := msg["role"].(string)
	return strings.ToLower(r)
}

// stripTrailingAssistant removes trailing assistant turns (Claude Code prefills).
func stripTrailingAssistant(messages []map[string]any) []map[string]any {
	if len(messages) == 0 {
		return messages
	}
	out := append([]map[string]any(nil), messages...)
	for len(out) > 0 && roleOf(out[len(out)-1]) == "assistant" {
		out = out[:len(out)-1]
	}
	return out
}
