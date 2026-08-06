package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

func postGenAxisWebhook(cfg config, eventType, requestID string, result map[string]any, errMsg string) {
	payload := map[string]any{
		"type":       eventType,
		"request_id": requestID,
		"result":     result,
		"error":      errMsg,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook: marshal error: %v", err)
		return
	}

	log.Printf("webhook: [promo-api→GenAxis] POST %s type=%s request_id=%s body=%s", cfg.WebhookURL, eventType, requestID, string(body))

	if cfg.WebhookURL == "" {
		log.Printf("webhook: skipped (no URL configured)")
		return
	}
	if cfg.GenAxisAPIKey == "" {
		log.Printf("webhook: skipped (GENAXIS_API_KEY not set)")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, strings.NewReader(string(body)))
	if err != nil {
		log.Printf("webhook: request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", cfg.GenAxisAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("webhook: post error: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("webhook: POST %s -> HTTP %d", cfg.WebhookURL, resp.StatusCode)
}
