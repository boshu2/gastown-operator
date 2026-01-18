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
	"sync"
	"time"
)

// CircuitState represents the current state of the circuit breaker.
type CircuitState int

const (
	// CircuitClosed means the circuit is operating normally.
	CircuitClosed CircuitState = iota
	// CircuitOpen means the circuit is open and requests should fail fast.
	CircuitOpen
	// CircuitHalfOpen means the circuit is testing if the underlying service has recovered.
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for the circuit breaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	// Default: 5
	FailureThreshold int

	// ResetTimeout is how long to wait before attempting to close an open circuit.
	// Default: 30s
	ResetTimeout time.Duration

	// HalfOpenMaxCalls is the number of calls allowed in half-open state.
	// Default: 1
	HalfOpenMaxCalls int
}

// DefaultCircuitBreakerConfig returns sensible defaults for circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern for gt CLI calls.
// It helps prevent cascading failures when the gt CLI is unavailable.
type CircuitBreaker struct {
	mu     sync.RWMutex
	config CircuitBreakerConfig

	state           CircuitState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	halfOpenCalls   int
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}
	if config.HalfOpenMaxCalls <= 0 {
		config.HalfOpenMaxCalls = 1
	}
	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.currentState()
}

// currentState returns the current state, checking if we should transition from open to half-open.
// Must be called with at least a read lock held.
func (cb *CircuitBreaker) currentState() CircuitState {
	if cb.state == CircuitOpen {
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) >= cb.config.ResetTimeout {
			return CircuitHalfOpen
		}
	}
	return cb.state
}

// AllowRequest checks if a request should be allowed through.
// Returns true if the request can proceed, false if it should fail fast.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()

	switch state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		return false

	case CircuitHalfOpen:
		// Transition state if we were open
		if cb.state == CircuitOpen {
			cb.state = CircuitHalfOpen
			cb.halfOpenCalls = 0
		}
		// Allow limited calls in half-open state
		if cb.halfOpenCalls < cb.config.HalfOpenMaxCalls {
			cb.halfOpenCalls++
			return true
		}
		return false

	default:
		return true
	}
}

// RecordSuccess records a successful call.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitHalfOpen:
		cb.successCount++
		// One success in half-open state closes the circuit
		cb.state = CircuitClosed
		cb.failureCount = 0
		cb.successCount = 0
		cb.halfOpenCalls = 0

	case CircuitClosed:
		// Reset failure count on success
		cb.failureCount = 0
		cb.successCount++
	}
}

// RecordFailure records a failed call.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitHalfOpen:
		// Failure in half-open state re-opens the circuit
		cb.state = CircuitOpen
		cb.failureCount = cb.config.FailureThreshold
		cb.halfOpenCalls = 0

	case CircuitClosed:
		cb.failureCount++
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = CircuitOpen
		}
	}
}

// Stats returns current circuit breaker statistics.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:           cb.currentState(),
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailureTime: cb.lastFailureTime,
	}
}

// CircuitBreakerStats holds statistics about the circuit breaker.
type CircuitBreakerStats struct {
	State           CircuitState
	FailureCount    int
	SuccessCount    int
	LastFailureTime time.Time
}

// Reset resets the circuit breaker to its initial closed state.
// This is useful for testing or manual intervention.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenCalls = 0
	cb.lastFailureTime = time.Time{}
}
