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

// Standard Condition Types for Gas Town Controllers
//
// NAMING CONVENTION: Condition types use unprefixed names because the condition
// is already scoped to its parent resource. This follows Kubernetes conventions:
//
//   - Pod has conditions like "Ready", "ContainersReady", "PodScheduled"
//   - Deployment has "Available", "Progressing", "ReplicaFailure"
//
// We use the same pattern. A Polecat with condition type "Ready" is unambiguous
// because you're looking at a Polecat's status.conditions.
//
// Standard condition types aligned with Kubernetes conventions:
//   - Ready: The resource is fully operational and ready to serve its purpose
//   - Progressing: An operation is in progress (e.g., merge, deployment)
//   - Degraded: The resource is operational but in a degraded state
//   - Available: A subset of Ready, indicates minimum viable operation
//
// Resource-specific condition types:
//   - Complete: Work has finished (Convoy)
//   - Working: Actively processing work (Polecat)
//   - Exists: External resource exists in gt CLI (Rig)
//   - Healthy: Monitoring is functioning (Witness)
//   - NotificationSent: Completion notification delivered (Convoy)
//
// When adding new condition types:
//  1. Prefer standard Kubernetes names when semantically appropriate
//  2. Use unprefixed names (e.g., "Ready" not "PolecatReady")
//  3. Document the meaning in the const definition
const (
	// ConditionReady indicates the resource is fully operational.
	// This is the primary "health" condition for most resources.
	ConditionReady = "Ready"

	// ConditionDegraded indicates the resource is operational but impaired.
	// Use when the resource can function but with reduced capability.
	ConditionDegraded = "Degraded"

	// ConditionProgressing indicates an operation is in progress.
	// Set to True during transitions, False when stable.
	ConditionProgressing = "Progressing"
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
