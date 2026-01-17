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

package errors

import (
	"testing"
	"time"
)

func TestNewBackoffCalculator(t *testing.T) {
	b := NewBackoffCalculator()

	if b.baseDelay != DefaultBaseDelay {
		t.Errorf("expected baseDelay %v, got %v", DefaultBaseDelay, b.baseDelay)
	}
	if b.maxDelay != DefaultMaxDelay {
		t.Errorf("expected maxDelay %v, got %v", DefaultMaxDelay, b.maxDelay)
	}
	if b.maxRetries != DefaultMaxRetries {
		t.Errorf("expected maxRetries %d, got %d", DefaultMaxRetries, b.maxRetries)
	}
}

func TestBackoffCalculator_GetBackoffResult(t *testing.T) {
	b := NewBackoffCalculatorWithConfig(1*time.Second, 30*time.Second, 10)

	// First retry: 1s * 2^0 = 1s
	result := b.GetBackoffResult("test-resource")
	if result.RequeueAfter != 1*time.Second {
		t.Errorf("expected 1s, got %v", result.RequeueAfter)
	}

	// Second retry: 1s * 2^1 = 2s
	result = b.GetBackoffResult("test-resource")
	if result.RequeueAfter != 2*time.Second {
		t.Errorf("expected 2s, got %v", result.RequeueAfter)
	}

	// Third retry: 1s * 2^2 = 4s
	result = b.GetBackoffResult("test-resource")
	if result.RequeueAfter != 4*time.Second {
		t.Errorf("expected 4s, got %v", result.RequeueAfter)
	}

	// Fourth retry: 1s * 2^3 = 8s
	result = b.GetBackoffResult("test-resource")
	if result.RequeueAfter != 8*time.Second {
		t.Errorf("expected 8s, got %v", result.RequeueAfter)
	}

	// Fifth retry: 1s * 2^4 = 16s
	result = b.GetBackoffResult("test-resource")
	if result.RequeueAfter != 16*time.Second {
		t.Errorf("expected 16s, got %v", result.RequeueAfter)
	}

	// Sixth retry: 1s * 2^5 = 32s, but capped at 30s
	result = b.GetBackoffResult("test-resource")
	if result.RequeueAfter != 30*time.Second {
		t.Errorf("expected 30s (max), got %v", result.RequeueAfter)
	}
}

func TestBackoffCalculator_ResetRetries(t *testing.T) {
	b := NewBackoffCalculatorWithConfig(1*time.Second, 30*time.Second, 10)

	// Build up some retries
	_ = b.GetBackoffResult("test-resource")
	_ = b.GetBackoffResult("test-resource")
	_ = b.GetBackoffResult("test-resource")

	if b.GetRetryCount("test-resource") != 3 {
		t.Errorf("expected 3 retries, got %d", b.GetRetryCount("test-resource"))
	}

	// Reset and verify
	b.ResetRetries("test-resource")
	if b.GetRetryCount("test-resource") != 0 {
		t.Errorf("expected 0 retries after reset, got %d", b.GetRetryCount("test-resource"))
	}

	// Next backoff should start from 1s again
	result := b.GetBackoffResult("test-resource")
	if result.RequeueAfter != 1*time.Second {
		t.Errorf("expected 1s after reset, got %v", result.RequeueAfter)
	}
}

func TestBackoffCalculator_ShouldGiveUp(t *testing.T) {
	b := NewBackoffCalculatorWithConfig(1*time.Second, 30*time.Second, 3)

	// Should not give up initially
	if b.ShouldGiveUp("test-resource") {
		t.Error("should not give up with 0 retries")
	}

	// Exhaust retries
	_ = b.GetBackoffResult("test-resource")
	_ = b.GetBackoffResult("test-resource")
	_ = b.GetBackoffResult("test-resource")

	// Should give up now
	if !b.ShouldGiveUp("test-resource") {
		t.Error("should give up after max retries")
	}

	// Reset and verify
	b.ResetRetries("test-resource")
	if b.ShouldGiveUp("test-resource") {
		t.Error("should not give up after reset")
	}
}

func TestBackoffCalculator_Cleanup(t *testing.T) {
	b := NewBackoffCalculator()

	// Add retries for multiple resources
	_ = b.GetBackoffResult("resource-1")
	_ = b.GetBackoffResult("resource-2")
	_ = b.GetBackoffResult("resource-3")

	// Cleanup, keeping only resource-2
	activeKeys := map[string]bool{
		"resource-2": true,
	}
	b.Cleanup(activeKeys)

	// Verify cleanup
	if b.GetRetryCount("resource-1") != 0 {
		t.Error("resource-1 should have been cleaned up")
	}
	if b.GetRetryCount("resource-2") != 1 {
		t.Errorf("resource-2 should still have 1 retry, got %d", b.GetRetryCount("resource-2"))
	}
	if b.GetRetryCount("resource-3") != 0 {
		t.Error("resource-3 should have been cleaned up")
	}
}

func TestBackoffCalculator_IndependentResources(t *testing.T) {
	b := NewBackoffCalculatorWithConfig(1*time.Second, 30*time.Second, 10)

	// Backoff for resource-1
	_ = b.GetBackoffResult("resource-1")
	_ = b.GetBackoffResult("resource-1")

	// Backoff for resource-2 should start fresh
	result := b.GetBackoffResult("resource-2")
	if result.RequeueAfter != 1*time.Second {
		t.Errorf("expected 1s for new resource, got %v", result.RequeueAfter)
	}

	// resource-1 should continue from where it left off
	result = b.GetBackoffResult("resource-1")
	if result.RequeueAfter != 4*time.Second {
		t.Errorf("expected 4s for continued resource, got %v", result.RequeueAfter)
	}
}
