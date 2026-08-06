package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func mapModel(name string, upstream string) string {
	if upstream != "" {
		return upstream
	}
	return "claude-opus-4-8"
}

func anthropicToOpenAI(body []byte, upstreamModel, userEmail string) ([]byte, error) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	msgs, _ := req["messages"].([]any)
	clean := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		clean = append(clean, mm)
	}
	clean = stripTrailingAssistant(clean)

	outMsgs := make([]map[string]any, 0, len(clean)+1)
	if sys := req["system"]; sys != nil {
		if text := contentToString(sys); text != "" {
			outMsgs = append(outMsgs, map[string]any{
				"role":    "system",
				"content": text,
			})
		}
	}

	for _, msg := range clean {
		converted, err := convertAnthropicMessage(msg)
		if err != nil {
			return nil, err
		}
		outMsgs = append(outMsgs, converted...)
	}

	out := map[string]any{
		"model":    mapModel(asString(req["model"]), upstreamModel),
		"messages": outMsgs,
	}
	if v, ok := req["max_tokens"]; ok {
		out["max_tokens"] = v
	}
	if v, ok := req["temperature"]; ok {
		out["temperature"] = v
	}
	if v, ok := req["top_p"]; ok {
		out["top_p"] = v
	}
	// Never stream upstream: the streaming proxy path does not forward tool_calls yet.
	// Streaming clients get a synthetic SSE sequence from the full response instead.
	if tools := req["tools"]; tools != nil {
		if ot, err := convertTools(tools); err == nil && len(ot) > 0 {
			out["tools"] = ot
		}
	}
	if tc := req["tool_choice"]; tc != nil {
		if otc, err := convertToolChoice(tc); err == nil {
			out["tool_choice"] = otc
		}
	}
	if userEmail != "" {
		out["metadata"] = map[string]any{"user_email": userEmail}
	}

	return json.Marshal(out)
}

func convertAnthropicMessage(msg map[string]any) ([]map[string]any, error) {
	role := roleOf(msg)
	content := msg["content"]

	switch role {
	case "user":
		if blocks, ok := content.([]any); ok {
			var toolResults []map[string]any
			var textParts []string
			for _, b := range blocks {
				block, ok := b.(map[string]any)
				if !ok {
					continue
				}
				switch asString(block["type"]) {
				case "tool_result":
					toolResults = append(toolResults, map[string]any{
						"role":         "tool",
						"tool_call_id": asString(block["tool_use_id"]),
						"content":      contentToString(block["content"]),
					})
				case "text":
					if t := asString(block["text"]); t != "" {
						textParts = append(textParts, t)
					}
				default:
					if t := contentToString(block); t != "" {
						textParts = append(textParts, t)
					}
				}
			}
			out := make([]map[string]any, 0, 1+len(toolResults))
			if len(textParts) > 0 {
				out = append(out, map[string]any{"role": "user", "content": strings.Join(textParts, "\n")})
			}
			for _, tr := range toolResults {
				out = append(out, tr)
			}
			if len(out) == 0 {
				out = append(out, map[string]any{"role": "user", "content": ""})
			}
			return out, nil
		}
		return []map[string]any{{"role": "user", "content": contentToString(content)}}, nil

	case "assistant":
		text := ""
		var toolCalls []map[string]any
		if blocks, ok := content.([]any); ok {
			for _, b := range blocks {
				block, ok := b.(map[string]any)
				if !ok {
					continue
				}
				switch asString(block["type"]) {
				case "text":
					text += asString(block["text"])
				case "tool_use":
					args, _ := json.Marshal(block["input"])
					toolCalls = append(toolCalls, map[string]any{
						"id":   asString(block["id"]),
						"type": "function",
						"function": map[string]any{
							"name":      asString(block["name"]),
							"arguments": string(args),
						},
					})
				}
			}
		} else {
			text = contentToString(content)
		}
		out := map[string]any{"role": "assistant"}
		if text != "" {
			out["content"] = text
		}
		if len(toolCalls) > 0 {
			out["tool_calls"] = toolCalls
		}
		return []map[string]any{out}, nil

	default:
		return nil, fmt.Errorf("unsupported role %q", role)
	}
}

func convertTools(tools any) ([]map[string]any, error) {
	arr, ok := tools.([]any)
	if !ok {
		return nil, fmt.Errorf("tools not array")
	}
	out := make([]map[string]any, 0, len(arr))
	for _, t := range arr {
		tool, ok := t.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        asString(tool["name"]),
				"description": asString(tool["description"]),
				"parameters":  tool["input_schema"],
			},
		})
	}
	return out, nil
}

func convertToolChoice(tc any) (any, error) {
	switch v := tc.(type) {
	case string:
		if v == "auto" {
			return "auto", nil
		}
		if v == "any" {
			return "required", nil
		}
		return v, nil
	case map[string]any:
		if asString(v["type"]) == "tool" {
			name := asString(v["name"])
			return map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": name,
				},
			}, nil
		}
	}
	return tc, nil
}

func contentToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		var parts []string
		for _, item := range x {
			if block, ok := item.(map[string]any); ok {
				if asString(block["type"]) == "text" {
					parts = append(parts, asString(block["text"]))
				}
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		if asString(x["type"]) == "text" {
			return asString(x["text"])
		}
	}
	return ""
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
