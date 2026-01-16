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

import "context"

// ClientInterface defines the interface for gt CLI operations.
// This interface allows for mocking in tests.
// nolint:dupl // Interface definition duplicated in MockClient for testing
type ClientInterface interface {
	// Rig operations
	RigList(ctx context.Context) ([]RigInfo, error)
	RigStatus(ctx context.Context, name string) (*RigStatus, error)
	RigExists(ctx context.Context, name string) (bool, error)

	// Polecat operations
	Sling(ctx context.Context, beadID, rig string) error
	PolecatList(ctx context.Context, rig string) ([]PolecatInfo, error)
	PolecatStatus(ctx context.Context, rig, name string) (*PolecatStatus, error)
	PolecatNuke(ctx context.Context, rig, name string, force bool) error
	PolecatReset(ctx context.Context, rig, name string) error
	PolecatExists(ctx context.Context, rig, name string) (bool, error)

	// Convoy operations
	ConvoyCreate(ctx context.Context, description string, beadIDs []string) (string, error)
	ConvoyStatus(ctx context.Context, id string) (*ConvoyStatus, error)
	ConvoyList(ctx context.Context) ([]ConvoyInfo, error)

	// Hook operations
	Hook(ctx context.Context, beadID, assignee string) error
	HookStatus(ctx context.Context, assignee string) (*HookInfo, error)

	// Beads operations
	BeadStatus(ctx context.Context, beadID string) (*BeadStatus, error)

	// Mail operations
	MailSend(ctx context.Context, address, subject, message string) error
}

// Verify Client implements ClientInterface at compile time
var _ ClientInterface = (*Client)(nil)
