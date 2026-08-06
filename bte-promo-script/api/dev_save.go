package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type devSaveRequest struct {
	JobName  string `json:"job_name"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// saveDevScript POSTs the W4 script body to a host listener (Docker Desktop Mac).
// LOCAL TEST ONLY — hostPath mounts do not sync writes back to macOS from K8s pods.
func saveDevScript(jobName string, files []filePayload) {
	if DefaultDevOutputHostPath == "" {
		return
	}
	url := strings.TrimSpace(os.Getenv("PROMO_DEV_SAVE_URL"))
	if url == "" {
		url = DefaultDevSaveURL
	}
	if url == "" {
		return
	}

	var scriptPath, scriptBody string
	for _, f := range files {
		if f.Key == "script" && strings.TrimSpace(f.Script) != "" {
			scriptPath = f.FilePath
			scriptBody = f.Script
			break
		}
	}
	if scriptBody == "" {
		log.Printf("dev-save: skip job=%q (no script content in upload)", jobName)
		return
	}

	filename := filepath.Base(scriptPath)
	if filename == "" || filename == "." {
		filename = jobName + "-script.md"
	}

	body, err := json.Marshal(devSaveRequest{
		JobName:  jobName,
		Filename: filename,
		Content:  scriptBody,
	})
	if err != nil {
		log.Printf("dev-save: WARN marshal: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("dev-save: WARN request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("dev-save: WARN POST %s failed: %v (is the host catcher running?)", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("dev-save: WARN POST %s returned HTTP %d", url, resp.StatusCode)
		return
	}
	log.Printf("dev-save: W4 script sent to host => %s (%s)", url, filename)
}
