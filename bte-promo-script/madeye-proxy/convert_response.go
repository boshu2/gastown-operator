package main

import (
	"encoding/json"
	"fmt"
)

func openAIToAnthropic(body []byte, model string) ([]byte, error) {
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	choices, _ := resp["choices"].([]any)
	if len(choices) == 0 {
		return nil, fmt.Errorf("openai response missing choices")
	}
	choice, _ := choices[0].(map[string]any)
	message, _ := choice["message"].(map[string]any)

	content := make([]map[string]any, 0, 4)
	if text, _ := message["content"].(string); text != "" {
		content = append(content, map[string]any{"type": "text", "text": text})
	}
	if tcs, ok := message["tool_calls"].([]any); ok {
		for _, tc := range tcs {
			call, _ := tc.(map[string]any)
			fn, _ := call["function"].(map[string]any)
			var input any
			_ = json.Unmarshal([]byte(asString(fn["arguments"])), &input)
			if input == nil {
				input = map[string]any{}
			}
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    asString(call["id"]),
				"name":  asString(fn["name"]),
				"input": input,
			})
		}
	}

	stopReason := "end_turn"
	if fr := asString(choice["finish_reason"]); fr == "length" {
		stopReason = "max_tokens"
	} else if fr == "tool_calls" {
		stopReason = "tool_use"
	}

	usage := map[string]any{"input_tokens": 0, "output_tokens": 0}
	if u, ok := resp["usage"].(map[string]any); ok {
		usage["input_tokens"] = u["prompt_tokens"]
		usage["output_tokens"] = u["completion_tokens"]
	}

	out := map[string]any{
		"id":           asString(resp["id"]),
		"type":         "message",
		"role":         "assistant",
		"model":        model,
		"content":      content,
		"stop_reason":  stopReason,
		"stop_sequence": nil,
		"usage":        usage,
	}
	if out["id"] == "" {
		out["id"] = "msg_madeye_proxy"
	}
	return json.Marshal(out)
}
