package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntegration_messages_nonStream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("upstream path=%s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Fatal("missing bearer auth to MadEye")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		msgs, _ := body["messages"].([]any)
		if len(msgs) < 2 {
			t.Fatalf("expected system+user messages, got %d", len(msgs))
		}
		if body["model"] != "claude-opus-4-8" {
			t.Fatalf("model=%v", body["model"])
		}
		meta, _ := body["metadata"].(map[string]any)
		if meta["user_email"] != "test@pocketfm.com" {
			t.Fatalf("metadata=%v", meta)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-test",
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"role":    "assistant",
						"content": "PROXY_OK",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 2,
				"total_tokens":      12,
			},
		})
	}))
	defer upstream.Close()

	srv := newServer(config{
		Port:          "0",
		MadEyeBase:    upstream.URL,
		MadEyeAPIKey:  "madeye-test-key",
		MadEyeEmail:   "test@pocketfm.com",
		ProxyKey:      "proxy-test-key",
		UpstreamModel: "claude-opus-4-8",
	})
	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	reqBody := `{
		"model": "claude-sonnet-4",
		"max_tokens": 32,
		"system": "You are helpful",
		"messages": [
			{"role":"user","content":"ping"},
			{"role":"assistant","content":"prefill-drop"}
		]
	}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/messages", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer proxy-test-key")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, b)
	}

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["type"] != "message" {
		t.Fatalf("type=%v", out["type"])
	}
	content, _ := out["content"].([]any)
	if len(content) == 0 {
		t.Fatal("empty content")
	}
	block, _ := content[0].(map[string]any)
	if block["text"] != "PROXY_OK" {
		t.Fatalf("text=%v", block["text"])
	}
}

func TestIntegration_messages_stream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["stream"] == true {
			t.Fatal("upstream must receive non-stream request")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-test",
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"role":    "assistant",
						"content": "PROXY_OK",
					},
					"finish_reason": "stop",
				},
			},
		})
	}))
	defer upstream.Close()

	srv := newServer(config{
		MadEyeBase:    upstream.URL,
		MadEyeAPIKey:  "madeye-test-key",
		MadEyeEmail:   "test@pocketfm.com",
		ProxyKey:      "proxy-test-key",
		UpstreamModel: "claude-opus-4-8",
	})
	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	reqBody := `{"model":"claude-opus-4-8","max_tokens":32,"stream":true,"messages":[{"role":"user","content":"ping"}]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer proxy-test-key")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "event: content_block_delta") {
		t.Fatalf("missing anthropic deltas: %s", text)
	}
	if !strings.Contains(text, "PROXY_OK") {
		t.Fatalf("missing streamed text: %s", text)
	}
}

func TestIntegration_unauthorized(t *testing.T) {
	srv := newServer(config{ProxyKey: "secret"})
	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/messages", strings.NewReader(`{"messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
