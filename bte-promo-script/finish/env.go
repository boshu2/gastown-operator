package main

import (
	"fmt"
	"os"
	"strings"
)

type jobEnv struct {
	APIURL    string
	JobName   string
	RequestID string
}

func loadJobEnv() (jobEnv, error) {
	e := jobEnv{
		APIURL:    strings.TrimRight(envOr("PROMO_API_URL", "http://promo-api.gastown-system.svc:8080"), "/"),
		JobName:   os.Getenv("PROMO_JOB_NAME"),
		RequestID: os.Getenv("GENAXIS_REQUEST_ID"),
	}
	if e.JobName == "" {
		return e, fmt.Errorf("PROMO_JOB_NAME is required")
	}
	if e.RequestID == "" {
		return e, fmt.Errorf("GENAXIS_REQUEST_ID is required")
	}
	return e, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
