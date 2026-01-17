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

package health

import (
	"net/http"
	"testing"
	"time"
)

func TestNewChecker(t *testing.T) {
	c := NewChecker()
	if c == nil {
		t.Fatal("NewChecker returned nil")
	}
	if c.checkTimeout != 5*time.Second {
		t.Errorf("expected checkTimeout 5s, got %v", c.checkTimeout)
	}
	if !c.gtHealthy {
		t.Error("expected gtHealthy to be true initially")
	}
}

func TestHealthzCheck(t *testing.T) {
	c := NewChecker()
	checker := c.HealthzCheck()

	// Healthz should always return nil (healthy)
	if err := checker(nil); err != nil {
		t.Errorf("healthz check failed: %v", err)
	}
}

func TestReadyzCheck_Healthy(t *testing.T) {
	c := NewChecker()
	c.SetGTHealthy(true)

	checker := c.ReadyzCheck()
	if err := checker(nil); err != nil {
		t.Errorf("readyz check failed when healthy: %v", err)
	}
}

func TestReadyzCheck_Unhealthy(t *testing.T) {
	c := NewChecker()
	c.SetGTHealthy(false)

	checker := c.ReadyzCheck()
	if err := checker(nil); err == nil {
		t.Error("readyz check should fail when gt CLI is unhealthy")
	}
}

func TestReadyzCheck_StaleHealth(t *testing.T) {
	c := NewChecker()
	c.SetGTHealthy(false)

	// Simulate stale health check by manipulating lastGTCheck
	c.mu.Lock()
	c.lastGTCheck = time.Now().Add(-10 * time.Minute)
	c.mu.Unlock()

	checker := c.ReadyzCheck()
	// Stale health info should not fail the check
	if err := checker(nil); err != nil {
		t.Errorf("readyz check should not fail with stale health info: %v", err)
	}
}

func TestGTCLICheck(t *testing.T) {
	tests := []struct {
		name        string
		healthy     bool
		stale       bool
		expectError bool
	}{
		{
			name:        "healthy",
			healthy:     true,
			stale:       false,
			expectError: false,
		},
		{
			name:        "unhealthy",
			healthy:     false,
			stale:       false,
			expectError: true,
		},
		{
			name:        "stale unhealthy",
			healthy:     false,
			stale:       true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChecker()
			c.SetGTHealthy(tt.healthy)

			if tt.stale {
				c.mu.Lock()
				c.lastGTCheck = time.Now().Add(-10 * time.Minute)
				c.mu.Unlock()
			}

			checker := c.GTCLICheck()
			err := checker(nil)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLeaderCheck(t *testing.T) {
	tests := []struct {
		name        string
		isLeader    bool
		expectError bool
	}{
		{
			name:        "is leader",
			isLeader:    true,
			expectError: false,
		},
		{
			name:        "not leader",
			isLeader:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := LeaderCheck(func() bool { return tt.isLeader })
			err := checker(&http.Request{})

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSetGTHealthy_UpdatesTimestamp(t *testing.T) {
	c := NewChecker()

	before := time.Now()
	c.SetGTHealthy(true)
	after := time.Now()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastGTCheck.Before(before) || c.lastGTCheck.After(after) {
		t.Error("lastGTCheck was not updated correctly")
	}
}

func TestHealthError(t *testing.T) {
	e := &healthError{msg: "test error"}
	if e.Error() != "test error" {
		t.Errorf("expected 'test error', got '%s'", e.Error())
	}
}
