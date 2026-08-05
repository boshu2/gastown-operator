package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func runWebhook(inputJSON string) error {
	env, err := loadJobEnv()
	if err != nil {
		return err
	}

	inputJSON = strings.TrimSpace(inputJSON)
	if inputJSON == "" {
		return fmt.Errorf("upload result JSON is required (output of promo-finish upload)")
	}

	var upload uploadResult
	if err := json.Unmarshal([]byte(inputJSON), &upload); err != nil {
		return fmt.Errorf("parse upload JSON: %w", err)
	}
	if len(upload.Artifacts) == 0 {
		return fmt.Errorf("upload JSON must include artifacts[] from upload step")
	}
	for i, a := range upload.Artifacts {
		if a.Key == "" {
			return fmt.Errorf("artifacts[%d] must include key (from upload step)", i)
		}
		if a.FileURL == "" || a.S3Key == "" {
			return fmt.Errorf("artifacts[%d] must include file_url and s3_key", i)
		}
	}
	if upload.RequestID != "" && upload.RequestID != env.RequestID {
		return fmt.Errorf("request_id mismatch: env=%s json=%s", env.RequestID, upload.RequestID)
	}

	fmt.Fprintf(os.Stderr, "promo-finish webhook: job=%s request_id=%s artifacts=%d\n",
		env.JobName, env.RequestID, len(upload.Artifacts))
	for _, a := range upload.Artifacts {
		fmt.Fprintf(os.Stderr, "  - %s => %s\n", a.Key, a.FileURL)
	}

	client := newAPIClient(env.APIURL)
	return client.webhook(env.JobName, webhookInput{
		RequestID: env.RequestID,
		Artifacts: upload.Artifacts,
	})
}
