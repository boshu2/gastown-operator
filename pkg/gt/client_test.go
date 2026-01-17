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

package gt

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default settings", func(t *testing.T) {
		client := NewClient("/test/town")

		if client.TownRoot() != "/test/town" {
			t.Errorf("expected TownRoot to be /test/town, got %s", client.TownRoot())
		}
		if client.GTPath() != "gt" {
			t.Errorf("expected GTPath to be gt, got %s", client.GTPath())
		}
		if client.Timeout() != DefaultTimeout {
			t.Errorf("expected Timeout to be %v, got %v", DefaultTimeout, client.Timeout())
		}
	})

	t.Run("respects GT_PATH environment variable", func(t *testing.T) {
		_ = os.Setenv("GT_PATH", "/custom/gt")
		defer func() { _ = os.Unsetenv("GT_PATH") }()

		client := NewClient("/test/town")
		if client.GTPath() != "/custom/gt" {
			t.Errorf("expected GTPath to be /custom/gt, got %s", client.GTPath())
		}
	})
}

func TestNewClientWithConfig(t *testing.T) {
	t.Run("uses provided configuration", func(t *testing.T) {
		cfg := ClientConfig{
			GTPath:   "/my/gt",
			TownRoot: "/my/town",
			Timeout:  60 * time.Second,
		}
		client := NewClientWithConfig(cfg)

		if client.GTPath() != "/my/gt" {
			t.Errorf("expected GTPath to be /my/gt, got %s", client.GTPath())
		}
		if client.TownRoot() != "/my/town" {
			t.Errorf("expected TownRoot to be /my/town, got %s", client.TownRoot())
		}
		if client.Timeout() != 60*time.Second {
			t.Errorf("expected Timeout to be 60s, got %v", client.Timeout())
		}
	})

	t.Run("applies defaults for zero values", func(t *testing.T) {
		cfg := ClientConfig{
			TownRoot: "/my/town",
		}
		client := NewClientWithConfig(cfg)

		if client.GTPath() != "gt" {
			t.Errorf("expected GTPath to default to gt, got %s", client.GTPath())
		}
		if client.Timeout() != DefaultTimeout {
			t.Errorf("expected Timeout to default to %v, got %v", DefaultTimeout, client.Timeout())
		}
	})

	t.Run("with circuit breaker", func(t *testing.T) {
		cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
		cfg := ClientConfig{
			TownRoot:       "/my/town",
			CircuitBreaker: cb,
		}
		client := NewClientWithConfig(cfg)

		if client.CircuitBreaker() != cb {
			t.Error("expected CircuitBreaker to be set")
		}
	})
}

func TestClientTimeout(t *testing.T) {
	t.Run("returns configured timeout", func(t *testing.T) {
		client := NewClientWithConfig(ClientConfig{
			TownRoot: "/test",
			Timeout:  45 * time.Second,
		})

		if client.Timeout() != 45*time.Second {
			t.Errorf("expected 45s, got %v", client.Timeout())
		}
	})

	t.Run("returns default timeout when not configured", func(t *testing.T) {
		client := NewClient("/test")

		if client.Timeout() != DefaultTimeout {
			t.Errorf("expected %v, got %v", DefaultTimeout, client.Timeout())
		}
	})
}

func TestClientRunWithCircuitBreaker(t *testing.T) {
	t.Run("fails fast when circuit is open", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     time.Hour, // Long timeout so it stays open
		})

		// Force circuit open
		cb.RecordFailure()

		client := NewClientWithConfig(ClientConfig{
			GTPath:         "nonexistent-command",
			TownRoot:       "/test",
			CircuitBreaker: cb,
		})

		ctx := context.Background()
		_, err := client.RigList(ctx)

		if err == nil {
			t.Error("expected error when circuit is open")
		}
	})
}

func TestClientRunTimeout(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		client := NewClientWithConfig(ClientConfig{
			GTPath:   "sleep", // Use sleep to simulate long-running command
			TownRoot: "/test",
			Timeout:  100 * time.Millisecond,
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := client.RigList(ctx)
		if err == nil {
			t.Error("expected error on cancelled context")
		}
	})
}
