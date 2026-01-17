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
	"math"
	"sync"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// DefaultBaseDelay is the initial backoff delay
	DefaultBaseDelay = 5 * time.Second

	// DefaultMaxDelay is the maximum backoff delay
	DefaultMaxDelay = 5 * time.Minute

	// DefaultMaxRetries is the number of retries before resetting
	DefaultMaxRetries = 10
)

// BackoffCalculator calculates exponential backoff delays for reconciliation errors.
type BackoffCalculator struct {
	baseDelay  time.Duration
	maxDelay   time.Duration
	maxRetries int

	// Track retry counts per resource
	mu      sync.RWMutex
	retries map[string]int
}

// NewBackoffCalculator creates a new BackoffCalculator with default settings.
func NewBackoffCalculator() *BackoffCalculator {
	return &BackoffCalculator{
		baseDelay:  DefaultBaseDelay,
		maxDelay:   DefaultMaxDelay,
		maxRetries: DefaultMaxRetries,
		retries:    make(map[string]int),
	}
}

// NewBackoffCalculatorWithConfig creates a BackoffCalculator with custom settings.
func NewBackoffCalculatorWithConfig(baseDelay, maxDelay time.Duration, maxRetries int) *BackoffCalculator {
	return &BackoffCalculator{
		baseDelay:  baseDelay,
		maxDelay:   maxDelay,
		maxRetries: maxRetries,
		retries:    make(map[string]int),
	}
}

// GetBackoffResult returns a ctrl.Result with exponential backoff delay for a resource.
// The key should be a unique identifier for the resource (e.g., namespace/name).
func (b *BackoffCalculator) GetBackoffResult(key string) ctrl.Result {
	b.mu.Lock()
	defer b.mu.Unlock()

	retries := b.retries[key]
	b.retries[key] = retries + 1

	// Calculate exponential delay: baseDelay * 2^retries
	delay := b.baseDelay * time.Duration(math.Pow(2, float64(retries)))
	delay = min(delay, b.maxDelay)

	return ctrl.Result{RequeueAfter: delay}
}

// GetRetryCount returns the current retry count for a resource.
func (b *BackoffCalculator) GetRetryCount(key string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.retries[key]
}

// ResetRetries resets the retry counter for a resource (call on successful reconciliation).
func (b *BackoffCalculator) ResetRetries(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.retries, key)
}

// ShouldGiveUp returns true if the resource has exceeded max retries.
func (b *BackoffCalculator) ShouldGiveUp(key string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.retries[key] >= b.maxRetries
}

// Cleanup removes entries for resources that no longer exist.
func (b *BackoffCalculator) Cleanup(activeKeys map[string]bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for key := range b.retries {
		if !activeKeys[key] {
			delete(b.retries, key)
		}
	}
}
