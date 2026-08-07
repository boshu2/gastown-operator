package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type config struct {
	Port          string
	MadEyeBase    string
	MadEyeAPIKey  string
	MadEyeEmail   string
	ProxyKey      string
	UpstreamModel string
}

func loadConfig() (config, error) {
	host := strings.TrimSpace(os.Getenv("MADEYE_BASE_URL"))
	if host == "" {
		return config{}, fmt.Errorf("MADEYE_BASE_URL is required (set via K8s env on madeye-proxy pod)")
	}
	host = strings.TrimRight(host, "/")
	return config{
		Port:          envOr("PORT", "4000"),
		MadEyeBase:    host + "/v1",
		MadEyeAPIKey:  os.Getenv("MADEYE_API_KEY"),
		MadEyeEmail:   os.Getenv("MADEYE_USER_EMAIL"),
		ProxyKey:      os.Getenv("PROXY_MASTER_KEY"),
		UpstreamModel: envOr("MADEYE_UPSTREAM_MODEL", "claude-opus-4-8"),
	}, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type server struct {
	cfg    config
	client *http.Client
}

func newServer(cfg config) *server {
	return &server{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /health/liveliness", s.handleHealth)
	mux.HandleFunc("POST /v1/messages", s.handleMessages)
	return mux
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *server) authorize(r *http.Request) bool {
	if s.cfg.ProxyKey == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	return auth == "Bearer "+s.cfg.ProxyKey
}

func (s *server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(r) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if s.cfg.MadEyeAPIKey == "" {
		http.Error(w, `{"error":"MADEYE_API_KEY not configured"}`, http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		http.Error(w, `{"error":"read body"}`, http.StatusBadRequest)
		return
	}

	model := readModelFromAnthropic(body, s.cfg.UpstreamModel)
	openBody, err := anthropicToOpenAI(body, s.cfg.UpstreamModel, s.cfg.MadEyeEmail)
	if err != nil {
		log.Printf("convert request: %v body=%s", err, debugBody(body))
		http.Error(w, fmt.Sprintf(`{"error":"convert request: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	clientStream := isStreamRequest(body)

	upBody, status, err := s.fetchUpstream(r, openBody, body)
	if err != nil {
		log.Printf("upstream error: %v", err)
		http.Error(w, `{"error":"upstream unreachable"}`, http.StatusBadGateway)
		return
	}
	if status < 200 || status >= 300 {
		log.Printf("upstream HTTP %d", status)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(upBody)
		return
	}

	outBody, err := openAIToAnthropic(upBody, model)
	if err != nil {
		log.Printf("convert response: %v upstream=%s", err, debugBody(upBody))
		http.Error(w, fmt.Sprintf(`{"error":"convert response: %s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	if clientStream {
		if err := writeAnthropicAsSSE(w, outBody); err != nil {
			log.Printf("stream emit: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(outBody)
}

func (s *server) fetchUpstream(r *http.Request, openBody, anthropicBody []byte) ([]byte, int, error) {
	maxAttempts := 1
	if requestHasTools(anthropicBody) {
		maxAttempts = 3
	}

	var lastBody []byte
	var lastStatus int

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		upReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.cfg.MadEyeBase+"/chat/completions", bytes.NewReader(openBody))
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		upReq.Header.Set("Content-Type", "application/json")
		upReq.Header.Set("Authorization", "Bearer "+s.cfg.MadEyeAPIKey)
		if accept := r.Header.Get("Accept"); accept != "" {
			upReq.Header.Set("Accept", accept)
		}

		upResp, err := s.client.Do(upReq)
		if err != nil {
			return nil, http.StatusBadGateway, err
		}

		upBody, err := io.ReadAll(io.LimitReader(upResp.Body, 32<<20))
		upResp.Body.Close()
		if err != nil {
			return nil, http.StatusBadGateway, err
		}

		lastBody, lastStatus = upBody, upResp.StatusCode
		if upResp.StatusCode < 200 || upResp.StatusCode >= 300 {
			return upBody, upResp.StatusCode, nil
		}
		if !openAIMissingToolCalls(upBody) || attempt == maxAttempts {
			if openAIMissingToolCalls(upBody) {
				log.Printf("upstream still missing tool_calls after %d attempts", attempt)
			}
			return upBody, upResp.StatusCode, nil
		}
		log.Printf("upstream finish_reason=tool_calls without tool_calls (attempt %d/%d), retrying", attempt, maxAttempts)
	}

	return lastBody, lastStatus, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           newServer(cfg).routes(),
		ReadHeaderTimeout: 30 * time.Second,
	}
	log.Printf("madeye-proxy listening on :%s → %s/chat/completions", cfg.Port, cfg.MadEyeBase)
	log.Fatal(srv.ListenAndServe())
}
