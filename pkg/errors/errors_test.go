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
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	err := New("test error")

	if err == nil {
		t.Fatal("expected error to be created")
	}
	if err.Message != "test error" {
		t.Errorf("expected message 'test error', got %q", err.Message)
	}
	if err.Type != ErrorTypeInternal {
		t.Errorf("expected type Internal, got %q", err.Type)
	}
	if len(err.Stack) == 0 {
		t.Error("expected stack trace to be captured")
	}
}

func TestNewf(t *testing.T) {
	err := Newf("error %d: %s", 42, "reason")

	if err.Message != "error 42: reason" {
		t.Errorf("expected formatted message, got %q", err.Message)
	}
}

func TestWrap(t *testing.T) {
	t.Run("wraps underlying error", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := Wrap(underlying, "wrapped message")

		if err.Cause != underlying {
			t.Error("expected cause to be set")
		}
		if err.Message != "wrapped message" {
			t.Errorf("expected message 'wrapped message', got %q", err.Message)
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := Wrap(nil, "message")
		if err != nil {
			t.Error("expected nil for nil input")
		}
	})
}

func TestWrapf(t *testing.T) {
	underlying := errors.New("underlying")
	err := Wrapf(underlying, "wrap %s", "formatted")

	if err.Message != "wrap formatted" {
		t.Errorf("expected 'wrap formatted', got %q", err.Message)
	}

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := Wrapf(nil, "message %s", "arg")
		if err != nil {
			t.Error("expected nil for nil input")
		}
	})
}

func TestTransient(t *testing.T) {
	underlying := errors.New("connection failed")
	err := Transient(underlying, "network error")

	if err.Type != ErrorTypeTransient {
		t.Errorf("expected type Transient, got %q", err.Type)
	}
	if !err.Retryable {
		t.Error("expected Retryable to be true")
	}

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := Transient(nil, "message")
		if err != nil {
			t.Error("expected nil for nil input")
		}
	})
}

func TestPermanent(t *testing.T) {
	underlying := errors.New("configuration error")
	err := Permanent(underlying, "invalid config")

	if err.Type != ErrorTypePermanent {
		t.Errorf("expected type Permanent, got %q", err.Type)
	}
	if err.Retryable {
		t.Error("expected Retryable to be false")
	}

	t.Run("returns nil for nil error", func(t *testing.T) {
		err := Permanent(nil, "message")
		if err != nil {
			t.Error("expected nil for nil input")
		}
	})
}

func TestValidation(t *testing.T) {
	err := Validation("invalid input")

	if err.Type != ErrorTypeValidation {
		t.Errorf("expected type Validation, got %q", err.Type)
	}
	if err.Retryable {
		t.Error("expected Retryable to be false")
	}
	if err.Message != "invalid input" {
		t.Errorf("expected message 'invalid input', got %q", err.Message)
	}
}

func TestNotFound(t *testing.T) {
	err := NotFound("Rig", "my-rig")

	if err.Type != ErrorTypeNotFound {
		t.Errorf("expected type NotFound, got %q", err.Type)
	}
	if err.Retryable {
		t.Error("expected Retryable to be false")
	}
	if !strings.Contains(err.Message, "Rig") || !strings.Contains(err.Message, "my-rig") {
		t.Errorf("expected message to contain resource and name, got %q", err.Message)
	}
	if err.Context["resource"] != "Rig" || err.Context["name"] != "my-rig" {
		t.Error("expected context to be set")
	}
}

func TestGTCLIError(t *testing.T) {
	underlying := errors.New("exit status 1")
	err := GTCLIError(underlying, "rig list")

	if err.Type != ErrorTypeGTCLI {
		t.Errorf("expected type GTCLI, got %q", err.Type)
	}
	if !err.Retryable {
		t.Error("expected Retryable to be true for CLI errors")
	}
	if err.Context["command"] != "rig list" {
		t.Error("expected command in context")
	}
}

func TestGasTownError_Error(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		underlying := errors.New("underlying")
		err := Wrap(underlying, "wrapped")

		got := err.Error()
		if !strings.Contains(got, "wrapped") || !strings.Contains(got, "underlying") {
			t.Errorf("expected error string to contain both messages, got %q", got)
		}
	})

	t.Run("without cause", func(t *testing.T) {
		err := New("just message")

		got := err.Error()
		if got != "just message" {
			t.Errorf("expected 'just message', got %q", got)
		}
	})
}

func TestGasTownError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying")
	err := Wrap(underlying, "wrapped")

	if err.Unwrap() != underlying {
		t.Error("Unwrap should return underlying error")
	}
}

func TestGasTownError_StackTrace(t *testing.T) {
	err := New("test")

	trace := err.StackTrace()
	if trace == "" {
		t.Error("expected non-empty stack trace")
	}
	if !strings.Contains(trace, "errors_test.go") {
		t.Errorf("expected stack trace to contain test file, got %q", trace)
	}
}

func TestGasTownError_WithContext(t *testing.T) {
	err := New("test")
	err = err.WithContext("key1", "value1").WithContext("key2", "value2")

	if err.Context["key1"] != "value1" {
		t.Error("expected key1 to be set")
	}
	if err.Context["key2"] != "value2" {
		t.Error("expected key2 to be set")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"transient error", Transient(errors.New("test"), "msg"), true},
		{"permanent error", Permanent(errors.New("test"), "msg"), false},
		{"validation error", Validation("invalid"), false},
		{"not found error", NotFound("Rig", "name"), false},
		{"gt cli error", GTCLIError(errors.New("test"), "cmd"), true},
		{"plain error", errors.New("plain"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		errType  string
		expected bool
	}{
		{"transient matches", Transient(errors.New("test"), "msg"), ErrorTypeTransient, true},
		{"transient doesn't match permanent", Transient(errors.New("test"), "msg"), ErrorTypePermanent, false},
		{"validation matches", Validation("invalid"), ErrorTypeValidation, true},
		{"plain error doesn't match", errors.New("plain"), ErrorTypeInternal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsType(tt.err, tt.errType); got != tt.expected {
				t.Errorf("IsType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	t.Run("returns true for NotFound error", func(t *testing.T) {
		err := NotFound("Rig", "name")
		if !IsNotFound(err) {
			t.Error("expected IsNotFound to return true")
		}
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		err := Validation("invalid")
		if IsNotFound(err) {
			t.Error("expected IsNotFound to return false")
		}
	})
}

func TestIsValidation(t *testing.T) {
	t.Run("returns true for Validation error", func(t *testing.T) {
		err := Validation("invalid")
		if !IsValidation(err) {
			t.Error("expected IsValidation to return true")
		}
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		err := NotFound("Rig", "name")
		if IsValidation(err) {
			t.Error("expected IsValidation to return false")
		}
	})
}

func TestIsGTCLIError(t *testing.T) {
	t.Run("returns true for GTCLIError", func(t *testing.T) {
		err := GTCLIError(errors.New("test"), "cmd")
		if !IsGTCLIError(err) {
			t.Error("expected IsGTCLIError to return true")
		}
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		err := Validation("invalid")
		if IsGTCLIError(err) {
			t.Error("expected IsGTCLIError to return false")
		}
	})
}

func TestToConditionReason(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"transient", Transient(errors.New("test"), "msg"), "TransientError"},
		{"permanent", Permanent(errors.New("test"), "msg"), "PermanentError"},
		{"validation", Validation("invalid"), "ValidationError"},
		{"not found", NotFound("Rig", "name"), "ResourceNotFound"},
		{"gt cli", GTCLIError(errors.New("test"), "cmd"), "GTCLIError"},
		{"internal", New("internal"), "InternalError"},
		{"plain error", errors.New("plain"), "UnknownError"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToConditionReason(tt.err); got != tt.expected {
				t.Errorf("ToConditionReason() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	// Test that wrapped errors can be unwrapped properly
	underlying := errors.New("root cause")
	wrapped := Wrap(underlying, "level 1")
	wrapped2 := Wrap(wrapped, "level 2")

	if !errors.Is(wrapped2, underlying) {
		t.Error("expected errors.Is to find underlying error")
	}

	var gtErr *GasTownError
	if !errors.As(wrapped2, &gtErr) {
		t.Error("expected errors.As to find GasTownError")
	}
}
