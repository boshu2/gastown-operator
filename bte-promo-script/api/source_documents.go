package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	AnnotationSourceDocuments = "gastown.io/source-documents"
	maxSourceDocuments        = 100
)

func normalizeSourceDocuments(in []string) ([]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	if len(in) > maxSourceDocuments {
		return nil, fmt.Errorf("source_documents: max %d URLs", maxSourceDocuments)
	}

	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for i, raw := range in {
		u := strings.TrimSpace(raw)
		if u == "" {
			return nil, fmt.Errorf("source_documents[%d]: empty URL", i)
		}
		parsed, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("source_documents[%d]: invalid URL", i)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, fmt.Errorf("source_documents[%d]: URL must be http or https", i)
		}
		if parsed.Host == "" {
			return nil, fmt.Errorf("source_documents[%d]: URL must include host", i)
		}
		if _, dup := seen[u]; dup {
			return nil, fmt.Errorf("source_documents[%d]: duplicate URL", i)
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out, nil
}

func sourceDocumentsAnnotation(urls []string) (string, error) {
	if len(urls) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(urls)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func showSourceDir(jobName string) string {
	return fmt.Sprintf("%s/show source/%s", DefaultPromoWorkspacePath, jobName)
}

func sourceDocumentsTaskBlock(jobName string, count int) string {
	if count == 0 {
		return ""
	}
	dir := showSourceDir(jobName)
	return fmt.Sprintf(`
SOURCE EPISODE SCRIPTS (optional canon input — read before W2):
Directory: %s
Files: ep001.txt through ep%03d.txt (extracted from source_documents URLs)
Use these for episode-level canon when generating the promo.
`, dir, count)
}
