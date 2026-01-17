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

package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateGitURL(t *testing.T) {
	tests := []struct {
		name    string
		gitURL  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid SSH URL",
			gitURL:  "git@github.com:org/repo.git",
			wantErr: false,
		},
		{
			name:    "valid SSH URL without .git",
			gitURL:  "git@github.com:org/repo",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL",
			gitURL:  "https://github.com/org/repo.git",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL without .git",
			gitURL:  "https://github.com/org/repo",
			wantErr: false,
		},
		{
			name:    "valid HTTP URL",
			gitURL:  "http://internal.example.com/org/repo.git",
			wantErr: false,
		},
		{
			name:    "valid file URL",
			gitURL:  "file:///home/user/repo.git",
			wantErr: false,
		},
		{
			name:    "empty URL",
			gitURL:  "",
			wantErr: true,
			errMsg:  "gitURL is required",
		},
		{
			name:    "invalid URL - no protocol",
			gitURL:  "github.com/org/repo",
			wantErr: true,
			errMsg:  "gitURL must be SSH",
		},
		{
			name:    "invalid URL - HTTPS with no host",
			gitURL:  "https:///repo.git",
			wantErr: true,
			errMsg:  "URL must have a host",
		},
		{
			name:    "invalid URL - HTTPS with no path",
			gitURL:  "https://github.com/",
			wantErr: true,
			errMsg:  "URL must include a repository path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitURL(tt.gitURL)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateBeadsPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid 2-char prefix",
			prefix:  "ap",
			wantErr: false,
		},
		{
			name:    "valid 4-char prefix",
			prefix:  "prod",
			wantErr: false,
		},
		{
			name:    "valid 10-char prefix",
			prefix:  "production",
			wantErr: false,
		},
		{
			name:    "empty prefix",
			prefix:  "",
			wantErr: true,
			errMsg:  "beadsPrefix is required",
		},
		{
			name:    "too short - 1 char",
			prefix:  "a",
			wantErr: true,
			errMsg:  "must be 2-10 lowercase letters",
		},
		{
			name:    "too long - 11 chars",
			prefix:  "abcdefghijk",
			wantErr: true,
			errMsg:  "must be 2-10 lowercase letters",
		},
		{
			name:    "contains uppercase",
			prefix:  "ABC",
			wantErr: true,
			errMsg:  "must be 2-10 lowercase letters",
		},
		{
			name:    "contains numbers",
			prefix:  "ap123",
			wantErr: true,
			errMsg:  "must be 2-10 lowercase letters",
		},
		{
			name:    "contains special chars",
			prefix:  "ap-test",
			wantErr: true,
			errMsg:  "must be 2-10 lowercase letters",
		},
		{
			name:    "reserved prefix - hq",
			prefix:  "hq",
			wantErr: true,
			errMsg:  "prefix \"hq\" is reserved",
		},
		{
			name:    "reserved prefix - system",
			prefix:  "system",
			wantErr: true,
			errMsg:  "prefix \"system\" is reserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBeadsPrefix(tt.prefix)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateLocalPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid absolute path",
			path:    "/home/user/projects/myrig",
			wantErr: false,
		},
		{
			name:    "valid short absolute path",
			path:    "/tmp/rig",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "localPath is required",
		},
		{
			name:    "relative path",
			path:    "relative/path",
			wantErr: true,
			errMsg:  "must be an absolute path",
		},
		{
			name:    "path traversal attempt",
			path:    "/home/../etc/passwd",
			wantErr: true,
			errMsg:  "path cannot contain '..'",
		},
		{
			name:    "path too long",
			path:    "/" + string(make([]byte, 260)),
			wantErr: true,
			errMsg:  "path length exceeds 255 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLocalPath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateNamepoolTheme(t *testing.T) {
	tests := []struct {
		name    string
		theme   string
		wantErr bool
	}{
		{
			name:    "valid theme - mad-max",
			theme:   "mad-max",
			wantErr: false,
		},
		{
			name:    "valid theme - minerals",
			theme:   "minerals",
			wantErr: false,
		},
		{
			name:    "valid theme - wasteland",
			theme:   "wasteland",
			wantErr: false,
		},
		{
			name:    "invalid theme",
			theme:   "unknown-theme",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNamepoolTheme(tt.theme)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRigCustomValidator_ValidateCreate(t *testing.T) {
	validator := &RigCustomValidator{}
	ctx := context.Background()

	tests := []struct {
		name        string
		rig         *Rig
		wantErr     bool
		wantWarning bool
	}{
		{
			name: "valid rig",
			rig: &Rig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
				Spec: RigSpec{
					GitURL:      "git@github.com:org/repo.git",
					BeadsPrefix: "test",
					LocalPath:   "/home/user/rig",
				},
			},
			wantErr: false,
		},
		{
			name: "valid rig with high maxPolecats",
			rig: &Rig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
				Spec: RigSpec{
					GitURL:      "git@github.com:org/repo.git",
					BeadsPrefix: "test",
					LocalPath:   "/home/user/rig",
					Settings: RigSettings{
						MaxPolecats: 75,
					},
				},
			},
			wantErr:     false,
			wantWarning: true, // Should warn about high maxPolecats
		},
		{
			name: "invalid git URL",
			rig: &Rig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
				Spec: RigSpec{
					GitURL:      "not-a-valid-url",
					BeadsPrefix: "test",
					LocalPath:   "/home/user/rig",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid beads prefix",
			rig: &Rig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
				Spec: RigSpec{
					GitURL:      "git@github.com:org/repo.git",
					BeadsPrefix: "X",
					LocalPath:   "/home/user/rig",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid local path",
			rig: &Rig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
				Spec: RigSpec{
					GitURL:      "git@github.com:org/repo.git",
					BeadsPrefix: "test",
					LocalPath:   "relative/path",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid namepool theme",
			rig: &Rig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
				Spec: RigSpec{
					GitURL:      "git@github.com:org/repo.git",
					BeadsPrefix: "test",
					LocalPath:   "/home/user/rig",
					Settings: RigSettings{
						NamepoolTheme: "invalid-theme",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := validator.ValidateCreate(ctx, tt.rig)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.wantWarning {
				assert.NotEmpty(t, warnings)
			}
		})
	}
}

func TestRigCustomValidator_ValidateUpdate(t *testing.T) {
	validator := &RigCustomValidator{}
	ctx := context.Background()

	oldRig := &Rig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
		Spec: RigSpec{
			GitURL:      "git@github.com:org/repo.git",
			BeadsPrefix: "test",
			LocalPath:   "/home/user/rig",
		},
	}

	tests := []struct {
		name    string
		newRig  *Rig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid update - change git URL",
			newRig: &Rig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
				Spec: RigSpec{
					GitURL:      "git@github.com:org/new-repo.git",
					BeadsPrefix: "test",
					LocalPath:   "/home/user/rig",
				},
			},
			wantErr: false,
		},
		{
			name: "valid update - change local path",
			newRig: &Rig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
				Spec: RigSpec{
					GitURL:      "git@github.com:org/repo.git",
					BeadsPrefix: "test",
					LocalPath:   "/home/user/new-rig",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid update - change beads prefix (immutable)",
			newRig: &Rig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
				Spec: RigSpec{
					GitURL:      "git@github.com:org/repo.git",
					BeadsPrefix: "newprefix",
					LocalPath:   "/home/user/rig",
				},
			},
			wantErr: true,
			errMsg:  "spec.beadsPrefix is immutable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidateUpdate(ctx, oldRig, tt.newRig)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRigCustomValidator_ValidateDelete(t *testing.T) {
	validator := &RigCustomValidator{}
	ctx := context.Background()

	rig := &Rig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-rig"},
		Spec: RigSpec{
			GitURL:      "git@github.com:org/repo.git",
			BeadsPrefix: "test",
			LocalPath:   "/home/user/rig",
		},
	}

	warnings, err := validator.ValidateDelete(ctx, rig)
	require.NoError(t, err)
	assert.Nil(t, warnings)
}

func TestRigCustomValidator_WrongType(t *testing.T) {
	validator := &RigCustomValidator{}
	ctx := context.Background()

	// Pass a non-Rig object
	wrongObj := &Polecat{}

	_, err := validator.ValidateCreate(ctx, wrongObj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected a Rig")

	_, err = validator.ValidateUpdate(ctx, wrongObj, wrongObj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected a Rig")

	_, err = validator.ValidateDelete(ctx, wrongObj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected a Rig")
}
