package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// writeAnthropicAsSSE converts a complete Anthropic messages JSON response into
// Anthropic SSE events (including tool_use blocks) for streaming clients.
func writeAnthropicAsSSE(w http.ResponseWriter, anthropicBody []byte) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	var msg map[string]any
	if err := json.Unmarshal(anthropicBody, &msg); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	msgID := asString(msg["id"])
	if msgID == "" {
		msgID = "msg_madeye_proxy"
	}
	model := asString(msg["model"])
	stopReason := asString(msg["stop_reason"])
	if stopReason == "" {
		stopReason = "end_turn"
	}

	writeSSE(w, flusher, "message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            msgID,
			"type":          "message",
			"role":          "assistant",
			"model":         model,
			"content":       []any{},
			"stop_reason":   nil,
			"usage":         msg["usage"],
		},
	})

	blocks, _ := msg["content"].([]any)
	for i, block := range blocks {
		b, ok := block.(map[string]any)
		if !ok {
			continue
		}
		switch asString(b["type"]) {
		case "text":
			text := asString(b["text"])
			writeSSE(w, flusher, "content_block_start", map[string]any{
				"type":  "content_block_start",
				"index": i,
				"content_block": map[string]any{
					"type": "text",
					"text": "",
				},
			})
			if text != "" {
				writeSSE(w, flusher, "content_block_delta", map[string]any{
					"type":  "content_block_delta",
					"index": i,
					"delta": map[string]any{
						"type": "text_delta",
						"text": text,
					},
				})
			}
			writeSSE(w, flusher, "content_block_stop", map[string]any{
				"type":  "content_block_stop",
				"index": i,
			})
		case "tool_use":
			input := b["input"]
			if input == nil {
				input = map[string]any{}
			}
			writeSSE(w, flusher, "content_block_start", map[string]any{
				"type":  "content_block_start",
				"index": i,
				"content_block": map[string]any{
					"type":  "tool_use",
					"id":    asString(b["id"]),
					"name":  asString(b["name"]),
					"input": map[string]any{},
				},
			})
			if raw, err := json.Marshal(input); err == nil && len(raw) > 2 {
				writeSSE(w, flusher, "content_block_delta", map[string]any{
					"type":  "content_block_delta",
					"index": i,
					"delta": map[string]any{
						"type":         "input_json_delta",
						"partial_json": string(raw),
					},
				})
			}
			writeSSE(w, flusher, "content_block_stop", map[string]any{
				"type":  "content_block_stop",
				"index": i,
			})
		}
	}

	writeSSE(w, flusher, "message_delta", map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": msg["usage"],
	})
	writeSSE(w, flusher, "message_stop", map[string]any{
		"type": "message_stop",
	})
	return nil
}

func proxyStream(w http.ResponseWriter, upstreamResp *http.Response, model string) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(upstreamResp.StatusCode)

	msgID := "msg_madeye_proxy"
	writeSSE(w, flusher, "message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":    msgID,
			"type":  "message",
			"role":  "assistant",
			"model": model,
			"content": []any{},
			"stop_reason": nil,
			"usage": map[string]any{"input_tokens": 0, "output_tokens": 0},
		},
	})
	writeSSE(w, flusher, "content_block_start", map[string]any{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]any{
			"type": "text",
			"text": "",
		},
	})

	scanner := bufio.NewScanner(upstreamResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	stopReason := "end_turn"

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk map[string]any
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		choices, _ := chunk["choices"].([]any)
		if len(choices) == 0 {
			continue
		}
		choice, _ := choices[0].(map[string]any)
		delta, _ := choice["delta"].(map[string]any)
		if text, _ := delta["content"].(string); text != "" {
			writeSSE(w, flusher, "content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]any{
					"type": "text_delta",
					"text": text,
				},
			})
		}
		if fr := asString(choice["finish_reason"]); fr != "" {
			if fr == "length" {
				stopReason = "max_tokens"
			} else if fr == "tool_calls" {
				stopReason = "tool_use"
			}
		}
	}

	writeSSE(w, flusher, "content_block_stop", map[string]any{
		"type":  "content_block_stop",
		"index": 0,
	})
	writeSSE(w, flusher, "message_delta", map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]any{"output_tokens": 0},
	})
	writeSSE(w, flusher, "message_stop", map[string]any{
		"type": "message_stop",
	})
	return scanner.Err()
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, data map[string]any) {
	raw, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", raw)
	flusher.Flush()
}

func readModelFromAnthropic(body []byte, fallback string) string {
	var req map[string]any
	if json.Unmarshal(body, &req) == nil {
		if m := asString(req["model"]); m != "" {
			return mapModel(m, fallback)
		}
	}
	return mapModel("", fallback)
}

func isStreamRequest(body []byte) bool {
	var req map[string]any
	if json.Unmarshal(body, &req) != nil {
		return false
	}
	stream, _ := req["stream"].(bool)
	return stream
}

func copyUpstreamError(w http.ResponseWriter, resp *http.Response) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func debugBody(body []byte) string {
	if len(body) > 512 {
		return string(body[:512]) + "…"
	}
	return string(body)
}

func cloneBody(body []byte) *bytes.Reader {
	return bytes.NewReader(body)
}
