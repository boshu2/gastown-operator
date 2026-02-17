/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestIsSecretStale(t *testing.T) {
	tests := []struct {
		name          string
		syncTimestamp string
		expectStale   bool
	}{
		{
			name:          "empty timestamp is stale",
			syncTimestamp: "",
			expectStale:   true,
		},
		{
			name:          "invalid timestamp is stale",
			syncTimestamp: "not-a-timestamp",
			expectStale:   true,
		},
		{
			name:          "fresh secret (1 day old) is not stale",
			syncTimestamp: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			expectStale:   false,
		},
		{
			name:          "secret at threshold boundary (89 days) is not stale",
			syncTimestamp: time.Now().Add(-89 * 24 * time.Hour).Format(time.RFC3339),
			expectStale:   false,
		},
		{
			name:          "secret just past threshold (91 days) is stale",
			syncTimestamp: time.Now().Add(-91 * 24 * time.Hour).Format(time.RFC3339),
			expectStale:   true,
		},
		{
			name:          "very old secret (365 days) is stale",
			syncTimestamp: time.Now().Add(-365 * 24 * time.Hour).Format(time.RFC3339),
			expectStale:   true,
		},
		{
			name:          "future timestamp is not stale",
			syncTimestamp: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			expectStale:   false,
		},
		{
			name:          "just now is not stale",
			syncTimestamp: time.Now().Format(time.RFC3339),
			expectStale:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSecretStale(tt.syncTimestamp)
			if got != tt.expectStale {
				t.Errorf("IsSecretStale(%q) = %v, want %v", tt.syncTimestamp, got, tt.expectStale)
			}
		})
	}
}

func TestRecordSecretAge(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		secretName    string
		secretType    string
		syncTimestamp string
		wantErr       bool
		wantRecorded  bool
	}{
		{
			name:          "empty timestamp records nothing",
			namespace:     "default",
			secretName:    "my-secret",
			secretType:    "Opaque",
			syncTimestamp: "",
			wantErr:       false,
			wantRecorded:  false,
		},
		{
			name:          "invalid timestamp returns error",
			namespace:     "default",
			secretName:    "my-secret",
			secretType:    "Opaque",
			syncTimestamp: "invalid",
			wantErr:       true,
			wantRecorded:  false,
		},
		{
			name:          "valid timestamp records age",
			namespace:     "test-ns",
			secretName:    "test-secret",
			secretType:    "kubernetes.io/tls",
			syncTimestamp: time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
			wantErr:       false,
			wantRecorded:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gastownSecretAge.Reset()

			err := RecordSecretAge(tt.namespace, tt.secretName, tt.secretType, tt.syncTimestamp)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordSecretAge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantRecorded {
				val := testutil.ToFloat64(gastownSecretAge.WithLabelValues(tt.namespace, tt.secretName, tt.secretType))
				if val <= 0 {
					t.Errorf("expected positive age metric, got %f", val)
				}
			}
		})
	}
}

func TestSecretStaleThresholdDays(t *testing.T) {
	if DefaultSecretStaleThresholdDays != 90 {
		t.Errorf("expected DefaultSecretStaleThresholdDays=90, got %d", DefaultSecretStaleThresholdDays)
	}
	if SecretStaleThresholdDays != DefaultSecretStaleThresholdDays {
		t.Errorf("expected SecretStaleThresholdDays to default to %d, got %d",
			DefaultSecretStaleThresholdDays, SecretStaleThresholdDays)
	}
}

func TestIsSecretStaleCustomThreshold(t *testing.T) {
	original := SecretStaleThresholdDays
	defer func() { SecretStaleThresholdDays = original }()

	// Set a 30-day threshold
	SecretStaleThresholdDays = 30

	// 45 days old — stale with 30-day threshold, not stale with default 90-day
	ts := time.Now().Add(-45 * 24 * time.Hour).Format(time.RFC3339)
	if !IsSecretStale(ts) {
		t.Error("expected 45-day-old secret to be stale with 30-day threshold")
	}

	// 15 days old — not stale even with 30-day threshold
	ts = time.Now().Add(-15 * 24 * time.Hour).Format(time.RFC3339)
	if IsSecretStale(ts) {
		t.Error("expected 15-day-old secret to not be stale with 30-day threshold")
	}
}
