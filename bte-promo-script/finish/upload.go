package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type filePayload struct {
	Key      string `json:"key"`
	FilePath string `json:"file_path"`
}

func readLabeledFiles(labeled []labeledFile) ([]filePayload, error) {
	files := make([]filePayload, 0, len(labeled))
	for _, lf := range labeled {
		abs, err := filepath.Abs(lf.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve path for key %q: %w", lf.Key, err)
		}
		if _, err := os.Stat(abs); err != nil {
			return nil, fmt.Errorf("file for key %q not found: %s", lf.Key, abs)
		}
		files = append(files, filePayload{Key: lf.Key, FilePath: abs})
	}
	return files, nil
}

func doUpload(labeled []labeledFile) (uploadResult, error) {
	env, err := loadJobEnv()
	if err != nil {
		return uploadResult{}, err
	}

	files, err := readLabeledFiles(labeled)
	if err != nil {
		return uploadResult{}, err
	}

	fmt.Fprintf(os.Stderr, "promo-finish upload: job=%s request_id=%s files=%d\n",
		env.JobName, env.RequestID, len(files))
	for _, f := range files {
		fmt.Fprintf(os.Stderr, "  - %s => %s\n", f.Key, f.FilePath)
	}

	client := newAPIClient(env.APIURL)
	return client.upload(env.JobName, map[string]any{
		"request_id": env.RequestID,
		"files":      files,
	})
}

func runUpload(args ...string) error {
	labeled, err := parseFileArgs(args)
	if err != nil {
		return err
	}
	result, err := doUpload(labeled)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
