package main

import (
	"fmt"
	"log"
	"strings"
)

type artifact struct {
	Key      string `json:"key"`
	FilePath string `json:"file_path"`
	FileURL  string `json:"file_url"`
	S3URI    string `json:"s3_uri"`
	S3Key    string `json:"s3_key"`
}

type filePayload struct {
	Key      string `json:"key"`
	FilePath string `json:"file_path"`
	Script   string `json:"script,omitempty"`
}

// uploadPromoArtifacts logs each key/path and returns hardcoded URLs until S3 upload exists.
func uploadPromoArtifacts(jobName, requestID string, files []filePayload) ([]artifact, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}

	out := make([]artifact, 0, len(files))
	for _, f := range files {
		key := strings.TrimSpace(f.Key)
		filePath := strings.TrimSpace(f.FilePath)

		if key == "" {
			return nil, fmt.Errorf("each file must include key")
		}
		if filePath == "" {
			return nil, fmt.Errorf("key %q: file_path is required", key)
		}

		log.Printf("upload: key=%q path=%q job=%q request_id=%q", key, filePath, jobName, requestID)

		fileURL := fmt.Sprintf("%s/%s", strings.TrimRight(StubFileURLBase, "/"), key)
		s3Key := fmt.Sprintf("stub/%s/%s", jobName, key)

		out = append(out, artifact{
			Key:      key,
			FilePath: filePath,
			FileURL:  fileURL,
			S3URI:    fileURL,
			S3Key:    s3Key,
		})
	}

	return out, nil
}

func artifactsByKey(artifacts []artifact) map[string]artifact {
	m := make(map[string]artifact, len(artifacts))
	for _, a := range artifacts {
		m[a.Key] = a
	}
	return m
}
