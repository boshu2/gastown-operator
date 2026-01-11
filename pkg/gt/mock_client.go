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

// MockClient is a mock implementation of ClientInterface for testing.
type MockClient struct {
	// Rig operation mocks
	RigListFunc   func(ctx context.Context) ([]RigInfo, error)
	RigStatusFunc func(ctx context.Context, name string) (*RigStatus, error)
	RigExistsFunc func(ctx context.Context, name string) (bool, error)

	// Polecat operation mocks
	SlingFunc         func(ctx context.Context, beadID, rig string) error
	PolecatListFunc   func(ctx context.Context, rig string) ([]PolecatInfo, error)
	PolecatStatusFunc func(ctx context.Context, rig, name string) (*PolecatStatus, error)
	PolecatNukeFunc   func(ctx context.Context, rig, name string, force bool) error
	PolecatResetFunc  func(ctx context.Context, rig, name string) error
	PolecatExistsFunc func(ctx context.Context, rig, name string) (bool, error)

	// Convoy operation mocks
	ConvoyCreateFunc func(ctx context.Context, description string, beadIDs []string) (string, error)
	ConvoyStatusFunc func(ctx context.Context, id string) (*ConvoyStatus, error)
	ConvoyListFunc   func(ctx context.Context) ([]ConvoyInfo, error)

	// Hook operation mocks
	HookFunc       func(ctx context.Context, beadID, assignee string) error
	HookStatusFunc func(ctx context.Context, assignee string) (*HookInfo, error)

	// Beads operation mocks
	BeadStatusFunc func(ctx context.Context, beadID string) (*BeadStatus, error)

	// Mail operation mocks
	MailSendFunc func(ctx context.Context, address, subject, message string) error
}

// Verify MockClient implements ClientInterface at compile time
var _ ClientInterface = (*MockClient)(nil)

// RigList implements ClientInterface
func (m *MockClient) RigList(ctx context.Context) ([]RigInfo, error) {
	if m.RigListFunc != nil {
		return m.RigListFunc(ctx)
	}
	return nil, nil
}

// RigStatus implements ClientInterface
func (m *MockClient) RigStatus(ctx context.Context, name string) (*RigStatus, error) {
	if m.RigStatusFunc != nil {
		return m.RigStatusFunc(ctx, name)
	}
	return &RigStatus{}, nil
}

// RigExists implements ClientInterface
func (m *MockClient) RigExists(ctx context.Context, name string) (bool, error) {
	if m.RigExistsFunc != nil {
		return m.RigExistsFunc(ctx, name)
	}
	return false, nil
}

// Sling implements ClientInterface
func (m *MockClient) Sling(ctx context.Context, beadID, rig string) error {
	if m.SlingFunc != nil {
		return m.SlingFunc(ctx, beadID, rig)
	}
	return nil
}

// PolecatList implements ClientInterface
func (m *MockClient) PolecatList(ctx context.Context, rig string) ([]PolecatInfo, error) {
	if m.PolecatListFunc != nil {
		return m.PolecatListFunc(ctx, rig)
	}
	return nil, nil
}

// PolecatStatus implements ClientInterface
func (m *MockClient) PolecatStatus(ctx context.Context, rig, name string) (*PolecatStatus, error) {
	if m.PolecatStatusFunc != nil {
		return m.PolecatStatusFunc(ctx, rig, name)
	}
	return &PolecatStatus{}, nil
}

// PolecatNuke implements ClientInterface
func (m *MockClient) PolecatNuke(ctx context.Context, rig, name string, force bool) error {
	if m.PolecatNukeFunc != nil {
		return m.PolecatNukeFunc(ctx, rig, name, force)
	}
	return nil
}

// PolecatReset implements ClientInterface
func (m *MockClient) PolecatReset(ctx context.Context, rig, name string) error {
	if m.PolecatResetFunc != nil {
		return m.PolecatResetFunc(ctx, rig, name)
	}
	return nil
}

// PolecatExists implements ClientInterface
func (m *MockClient) PolecatExists(ctx context.Context, rig, name string) (bool, error) {
	if m.PolecatExistsFunc != nil {
		return m.PolecatExistsFunc(ctx, rig, name)
	}
	return false, nil
}

// ConvoyCreate implements ClientInterface
func (m *MockClient) ConvoyCreate(ctx context.Context, description string, beadIDs []string) (string, error) {
	if m.ConvoyCreateFunc != nil {
		return m.ConvoyCreateFunc(ctx, description, beadIDs)
	}
	return "mock-convoy-id", nil
}

// ConvoyStatus implements ClientInterface
func (m *MockClient) ConvoyStatus(ctx context.Context, id string) (*ConvoyStatus, error) {
	if m.ConvoyStatusFunc != nil {
		return m.ConvoyStatusFunc(ctx, id)
	}
	return &ConvoyStatus{}, nil
}

// ConvoyList implements ClientInterface
func (m *MockClient) ConvoyList(ctx context.Context) ([]ConvoyInfo, error) {
	if m.ConvoyListFunc != nil {
		return m.ConvoyListFunc(ctx)
	}
	return nil, nil
}

// Hook implements ClientInterface
func (m *MockClient) Hook(ctx context.Context, beadID, assignee string) error {
	if m.HookFunc != nil {
		return m.HookFunc(ctx, beadID, assignee)
	}
	return nil
}

// HookStatus implements ClientInterface
func (m *MockClient) HookStatus(ctx context.Context, assignee string) (*HookInfo, error) {
	if m.HookStatusFunc != nil {
		return m.HookStatusFunc(ctx, assignee)
	}
	return &HookInfo{}, nil
}

// BeadStatus implements ClientInterface
func (m *MockClient) BeadStatus(ctx context.Context, beadID string) (*BeadStatus, error) {
	if m.BeadStatusFunc != nil {
		return m.BeadStatusFunc(ctx, beadID)
	}
	return &BeadStatus{}, nil
}

// MailSend implements ClientInterface
func (m *MockClient) MailSend(ctx context.Context, address, subject, message string) error {
	if m.MailSendFunc != nil {
		return m.MailSendFunc(ctx, address, subject, message)
	}
	return nil
}
