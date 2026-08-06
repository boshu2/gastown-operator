package main

import (
	"encoding/json"
	"testing"
)

func TestStripTrailingAssistant(t *testing.T) {
	msgs := []map[string]any{
		{"role": "user", "content": "hi"},
		{"role": "assistant", "content": "prefill"},
	}
	got := stripTrailingAssistant(msgs)
	if len(got) != 1 || roleOf(got[0]) != "user" {
		t.Fatalf("expected user only, got %#v", got)
	}
}

func TestAnthropicToOpenAI_text(t *testing.T) {
	in := []byte(`{
		"model": "claude-sonnet-4",
		"max_tokens": 100,
		"system": "You are helpful",
		"messages": [
			{"role":"user","content":"hello"},
			{"role":"assistant","content":"drop me"}
		]
	}`)
	out, err := anthropicToOpenAI(in, "claude-opus-4-8", "u@pocketfm.com")
	if err != nil {
		t.Fatal(err)
	}
	var req map[string]any
	if err := json.Unmarshal(out, &req); err != nil {
		t.Fatal(err)
	}
	if req["model"] != "claude-opus-4-8" {
		t.Fatalf("model=%v", req["model"])
	}
	msgs, _ := req["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("expected system+user, got %d", len(msgs))
	}
}
