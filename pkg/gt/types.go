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

// Package gt provides a wrapper around the Gas Town CLI tool.
// The operator shells out to `gt` for all operations, keeping gt as the
// source of truth and this operator as a K8s facade.
package gt

import "time"

// RigInfo contains summary information about a rig
type RigInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	BeadsPrefix string `json:"beadsPrefix"`
	GitURL      string `json:"gitURL,omitempty"`
}

// RigStatus contains detailed status for a rig
type RigStatus struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	PolecatCount  int    `json:"polecatCount"`
	ActiveConvoys int    `json:"activeConvoys"`
	OpenBeads     int    `json:"openBeads,omitempty"`
}

// PolecatInfo contains summary information about a polecat
type PolecatInfo struct {
	Name         string `json:"name"`
	Rig          string `json:"rig"`
	Phase        string `json:"phase"`
	AssignedBead string `json:"assignedBead,omitempty"`
}

// PolecatStatus contains detailed status for a polecat
type PolecatStatus struct {
	Name          string    `json:"name"`
	Rig           string    `json:"rig"`
	Phase         string    `json:"phase"`
	AssignedBead  string    `json:"assignedBead,omitempty"`
	Branch        string    `json:"branch,omitempty"`
	WorktreePath  string    `json:"worktreePath,omitempty"`
	TmuxSession   string    `json:"tmuxSession,omitempty"`
	SessionActive bool      `json:"sessionActive"`
	LastActivity  time.Time `json:"lastActivity,omitempty"`
	CleanupStatus string    `json:"cleanupStatus,omitempty"`
}

// ConvoyInfo contains summary information about a convoy
type ConvoyInfo struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Phase        string   `json:"phase"`
	Progress     string   `json:"progress"`
	BeadCount    int      `json:"beadCount"`
	TrackedBeads []string `json:"trackedBeads,omitempty"`
}

// ConvoyStatus contains detailed status for a convoy
type ConvoyStatus struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Phase       string    `json:"phase"`
	Progress    string    `json:"progress"`
	Completed   []string  `json:"completed,omitempty"`
	Pending     []string  `json:"pending,omitempty"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
}

// HookInfo contains information about what's hooked to an assignee
type HookInfo struct {
	Assignee  string `json:"assignee"`
	BeadID    string `json:"beadID,omitempty"`
	BeadTitle string `json:"beadTitle,omitempty"`
}

// BeadStatus contains information about a bead's status
type BeadStatus struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Type   string `json:"type"`
}
