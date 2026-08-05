package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type apiClient struct {
	base   string
	client *http.Client
}

func newAPIClient(baseURL string) *apiClient {
	return &apiClient{
		base:   strings.TrimRight(baseURL, "/"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type artifact struct {
	Key      string `json:"key"`
	FilePath string `json:"file_path"`
	FileURL  string `json:"file_url"`
	S3URI    string `json:"s3_uri"`
	S3Key    string `json:"s3_key"`
}

type uploadResult struct {
	JobName   string     `json:"job_name"`
	RequestID string     `json:"request_id"`
	Artifacts []artifact `json:"artifacts"`
}

type webhookInput struct {
	RequestID string     `json:"request_id"`
	Artifacts []artifact `json:"artifacts"`
}

func (c *apiClient) upload(jobName string, body any) (uploadResult, error) {
	var out uploadResult
	err := c.postJSON(fmt.Sprintf("/v1/promo/jobs/%s/upload", jobName), body, &out)
	return out, err
}

func (c *apiClient) webhook(jobName string, body webhookInput) error {
	var discard map[string]any
	return c.postJSON(fmt.Sprintf("/v1/promo/jobs/%s/webhook", jobName), body, &discard)
}

func (c *apiClient) postJSON(path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("post %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
