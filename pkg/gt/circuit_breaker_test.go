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
	"testing"
	"time"
)

func TestCircuitBreakerState(t *testing.T) {
	t.Run("starts in closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
		if cb.State() != CircuitClosed {
			t.Errorf("expected CircuitClosed, got %v", cb.State())
		}
	})

	t.Run("opens after failure threshold", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 3,
			ResetTimeout:     time.Hour,
		})

		// Record failures below threshold
		cb.RecordFailure()
		cb.RecordFailure()
		if cb.State() != CircuitClosed {
			t.Error("circuit should still be closed")
		}

		// Record failure at threshold
		cb.RecordFailure()
		if cb.State() != CircuitOpen {
			t.Error("circuit should be open after reaching threshold")
		}
	})

	t.Run("transitions to half-open after reset timeout", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     10 * time.Millisecond,
		})

		cb.RecordFailure()
		if cb.State() != CircuitOpen {
			t.Error("circuit should be open")
		}

		// Wait for reset timeout
		time.Sleep(20 * time.Millisecond)

		if cb.State() != CircuitHalfOpen {
			t.Errorf("circuit should be half-open, got %v", cb.State())
		}
	})

	t.Run("closes on success in half-open state", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     10 * time.Millisecond,
		})

		cb.RecordFailure()
		time.Sleep(20 * time.Millisecond)

		// Verify half-open
		if cb.State() != CircuitHalfOpen {
			t.Fatal("expected half-open state")
		}

		// Allow request transitions to half-open internally
		cb.AllowRequest()
		cb.RecordSuccess()

		if cb.State() != CircuitClosed {
			t.Errorf("circuit should be closed after success, got %v", cb.State())
		}
	})

	t.Run("re-opens on failure in half-open state", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     10 * time.Millisecond,
		})

		cb.RecordFailure()
		time.Sleep(20 * time.Millisecond)

		// Allow request and then fail
		cb.AllowRequest()
		cb.RecordFailure()

		if cb.State() != CircuitOpen {
			t.Errorf("circuit should be open after half-open failure, got %v", cb.State())
		}
	})
}

func TestCircuitBreakerAllowRequest(t *testing.T) {
	t.Run("allows requests when closed", func(t *testing.T) {
		cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

		for i := 0; i < 10; i++ {
			if !cb.AllowRequest() {
				t.Error("should allow requests when closed")
			}
		}
	})

	t.Run("denies requests when open", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     time.Hour,
		})

		cb.RecordFailure()

		if cb.AllowRequest() {
			t.Error("should deny requests when open")
		}
	})

	t.Run("allows limited requests in half-open", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     10 * time.Millisecond,
			HalfOpenMaxCalls: 2,
		})

		cb.RecordFailure()
		time.Sleep(20 * time.Millisecond)

		// Should allow HalfOpenMaxCalls requests
		if !cb.AllowRequest() {
			t.Error("should allow first request in half-open")
		}
		if !cb.AllowRequest() {
			t.Error("should allow second request in half-open")
		}
		if cb.AllowRequest() {
			t.Error("should deny third request in half-open")
		}
	})
}

func TestCircuitBreakerRecordSuccess(t *testing.T) {
	t.Run("resets failure count on success", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 5,
			ResetTimeout:     time.Hour,
		})

		// Record some failures
		cb.RecordFailure()
		cb.RecordFailure()
		cb.RecordFailure()

		// Success should reset
		cb.RecordSuccess()

		stats := cb.Stats()
		if stats.FailureCount != 0 {
			t.Errorf("failure count should be 0, got %d", stats.FailureCount)
		}
	})
}

func TestCircuitBreakerStats(t *testing.T) {
	t.Run("tracks statistics", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 10,
			ResetTimeout:     time.Hour,
		})

		cb.RecordFailure()
		cb.RecordFailure()
		cb.RecordSuccess()

		stats := cb.Stats()
		if stats.State != CircuitClosed {
			t.Errorf("expected closed, got %v", stats.State)
		}
		if stats.FailureCount != 0 {
			t.Errorf("expected 0 failures after success, got %d", stats.FailureCount)
		}
		if stats.SuccessCount != 1 {
			t.Errorf("expected 1 success, got %d", stats.SuccessCount)
		}
	})
}

func TestCircuitBreakerReset(t *testing.T) {
	t.Run("resets to initial state", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 1,
			ResetTimeout:     time.Hour,
		})

		cb.RecordFailure()
		if cb.State() != CircuitOpen {
			t.Fatal("expected open state")
		}

		cb.Reset()

		if cb.State() != CircuitClosed {
			t.Error("expected closed state after reset")
		}
		stats := cb.Stats()
		if stats.FailureCount != 0 {
			t.Error("expected 0 failures after reset")
		}
	})
}

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.expected)
		}
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()

	if cfg.FailureThreshold != 5 {
		t.Errorf("expected FailureThreshold 5, got %d", cfg.FailureThreshold)
	}
	if cfg.ResetTimeout != 30*time.Second {
		t.Errorf("expected ResetTimeout 30s, got %v", cfg.ResetTimeout)
	}
	if cfg.HalfOpenMaxCalls != 1 {
		t.Errorf("expected HalfOpenMaxCalls 1, got %d", cfg.HalfOpenMaxCalls)
	}
}
