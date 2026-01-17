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
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// Checker provides custom health checking for the operator.
type Checker struct {
	mu           sync.RWMutex
	lastGTCheck  time.Time
	gtHealthy    bool
	checkTimeout time.Duration
}

// NewChecker creates a new health checker.
func NewChecker() *Checker {
	return &Checker{
		checkTimeout: 5 * time.Second,
		gtHealthy:    true, // Assume healthy until proven otherwise
	}
}

// SetGTHealthy updates the GT CLI health status.
// Call this from reconcilers when GT CLI operations succeed or fail.
func (c *Checker) SetGTHealthy(healthy bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gtHealthy = healthy
	c.lastGTCheck = time.Now()
}

// HealthzCheck returns a healthz.Checker for the /healthz endpoint.
// Liveness checks should always pass if the process is running - they indicate
// the process hasn't deadlocked and should not be killed. No external dependency
// checks are performed here; those belong in readiness checks.
func (c *Checker) HealthzCheck() healthz.Checker {
	return func(_ *http.Request) error {
		// Liveness: process is alive, return healthy unconditionally
		return nil
	}
}

// ReadyzCheck returns a healthz.Checker for the /readyz endpoint.
// This checks if the operator is ready to serve traffic.
//
// Behavior:
// - If GT CLI was recently checked and is unhealthy: NOT READY (fail)
// - If GT CLI was recently checked and is healthy: READY (pass)
// - If no recent GT CLI check (>5 min): READY (pass, optimistic)
//
// The optimistic approach for stale checks prevents cascading failures during
// startup or when GT CLI checks are delayed. Use GTCLICheck() for stricter checking.
func (c *Checker) ReadyzCheck() healthz.Checker {
	return func(_ *http.Request) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		// Stale health info (>5 min old): be optimistic, assume healthy
		// This prevents blocking startup when GT CLI hasn't been probed yet
		if time.Since(c.lastGTCheck) > 5*time.Minute {
			return nil
		}

		// Recent check showed GT CLI is unhealthy
		if !c.gtHealthy {
			return &healthError{msg: "gt CLI is not healthy"}
		}

		return nil
	}
}

// GTCLICheck returns a healthz.Checker specifically for GT CLI health.
// Register this separately for granular health reporting (e.g., /readyz?verbose).
//
// Behavior mirrors ReadyzCheck: optimistic on stale data, fails on recent unhealthy.
func (c *Checker) GTCLICheck() healthz.Checker {
	return func(_ *http.Request) error {
		c.mu.RLock()
		defer c.mu.RUnlock()

		// Stale health info: be optimistic
		if time.Since(c.lastGTCheck) > 5*time.Minute {
			return nil
		}

		if !c.gtHealthy {
			return &healthError{msg: "gt CLI is not responding"}
		}

		return nil
	}
}

// LeaderCheck returns a healthz.Checker that reports healthy only when leader.
// Useful for debugging leader election state via /readyz?verbose.
// Non-leaders are still "healthy" for liveness, but this check can expose election status.
func LeaderCheck(isLeader func() bool) healthz.Checker {
	return func(_ *http.Request) error {
		if !isLeader() {
			return &healthError{msg: "not the leader"}
		}
		return nil
	}
}

// healthError implements error interface for health check failures.
type healthError struct {
	msg string
}

func (e *healthError) Error() string {
	return e.msg
}
