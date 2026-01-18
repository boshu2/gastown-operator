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

package client

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGVRDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		gvr      schema.GroupVersionResource
		expected schema.GroupVersionResource
	}{
		{
			name: "RigGVR",
			gvr:  RigGVR,
			expected: schema.GroupVersionResource{
				Group:    "gastown.gastown.io",
				Version:  "v1alpha1",
				Resource: "rigs",
			},
		},
		{
			name: "PolecatGVR",
			gvr:  PolecatGVR,
			expected: schema.GroupVersionResource{
				Group:    "gastown.gastown.io",
				Version:  "v1alpha1",
				Resource: "polecats",
			},
		},
		{
			name: "ConvoyGVR",
			gvr:  ConvoyGVR,
			expected: schema.GroupVersionResource{
				Group:    "gastown.gastown.io",
				Version:  "v1alpha1",
				Resource: "convoys",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.gvr != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.gvr, tt.expected)
			}
		})
	}
}

func TestNewClientFromDynamic(t *testing.T) {
	// Test that NewClientFromDynamic doesn't panic with nil client
	// In real usage, this would be a fake.NewSimpleDynamicClient
	client := NewClientFromDynamic(nil)
	if client == nil {
		t.Error("NewClientFromDynamic returned nil")
	}
}
