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

package controller

import (
	"context"
	"time"
)

// Requeue intervals for controller reconciliation.
// Using named constants makes it easier to tune timeouts and ensures consistency.
const (
	// RequeueShort is used for rapid re-checks during active state transitions.
	// Use when expecting a fast state change (e.g., waiting for pod to start).
	RequeueShort = 10 * time.Second

	// RequeueDefault is the standard interval for most re-sync operations.
	// Use for normal periodic syncing with external systems.
	RequeueDefault = 30 * time.Second

	// RequeueLong is used when recovery or retry is expected to take longer.
	// Use after errors or when entering a degraded state.
	RequeueLong = 1 * time.Minute

	// RequeueRetryTransient is used for retrying transient errors.
	// Slightly shorter than RequeueLong to enable faster recovery.
	RequeueRetryTransient = 10 * time.Second
)

// Timeout constants for external system calls.
const (
	// GTClientTimeout is the maximum time to wait for gt CLI operations.
	// Prevents hung CLI calls from blocking reconciliation indefinitely.
	GTClientTimeout = 60 * time.Second
)

// WithGTClientTimeout returns a context with the standard GT client timeout.
// The returned cancel function MUST be called to release resources, typically via defer.
// Example:
//
//	ctx, cancel := WithGTClientTimeout(ctx)
//	defer cancel()
//	status, err := r.GTClient.RigStatus(ctx, name)
func WithGTClientTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, GTClientTimeout)
}
