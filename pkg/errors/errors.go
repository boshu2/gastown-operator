// Package errors provides consistent error handling for Gas Town operator controllers.
//
// This package wraps the standard errors package and provides:
// - Stack traces for debugging
// - Context propagation
// - Standardized error types for K8s conditions
// - Error categorization (transient vs permanent)
package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// Error types for categorization
const (
	// ErrorTypeTransient indicates the error may resolve on retry
	ErrorTypeTransient = "Transient"
	// ErrorTypePermanent indicates the error requires intervention
	ErrorTypePermanent = "Permanent"
	// ErrorTypeValidation indicates invalid input
	ErrorTypeValidation = "Validation"
	// ErrorTypeNotFound indicates a resource was not found
	ErrorTypeNotFound = "NotFound"
	// ErrorTypeConflict indicates a resource conflict
	ErrorTypeConflict = "Conflict"
	// ErrorTypeInternal indicates an internal error
	ErrorTypeInternal = "Internal"
	// ErrorTypeGTCLI indicates an error from the gt CLI
	ErrorTypeGTCLI = "GTCLI"
)

// GasTownError is the base error type with context and stack trace.
type GasTownError struct {
	// Cause is the underlying error
	Cause error
	// Message is the human-readable error message
	Message string
	// Type categorizes the error
	Type string
	// Context contains key-value pairs for debugging
	Context map[string]string
	// Stack is the call stack at error creation
	Stack []uintptr
	// Retryable indicates if the operation should be retried
	Retryable bool
}

// Error implements the error interface.
func (e *GasTownError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *GasTownError) Unwrap() error {
	return e.Cause
}

// StackTrace returns a formatted stack trace.
func (e *GasTownError) StackTrace() string {
	if len(e.Stack) == 0 {
		return ""
	}

	var sb strings.Builder
	frames := runtime.CallersFrames(e.Stack)
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			sb.WriteString(fmt.Sprintf("  %s\n    %s:%d\n", frame.Function, frame.File, frame.Line))
		}
		if !more {
			break
		}
	}
	return sb.String()
}

// WithContext adds context to the error.
func (e *GasTownError) WithContext(key, value string) *GasTownError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// captureStack captures the current call stack.
func captureStack(skip int) []uintptr {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	return pcs[:n]
}

// New creates a new GasTownError with a message.
func New(message string) *GasTownError {
	return &GasTownError{
		Message: message,
		Type:    ErrorTypeInternal,
		Stack:   captureStack(1),
	}
}

// Newf creates a new GasTownError with a formatted message.
func Newf(format string, args ...interface{}) *GasTownError {
	return &GasTownError{
		Message: fmt.Sprintf(format, args...),
		Type:    ErrorTypeInternal,
		Stack:   captureStack(1),
	}
}

// Wrap wraps an error with additional context.
func Wrap(err error, message string) *GasTownError {
	if err == nil {
		return nil
	}
	return &GasTownError{
		Cause:   err,
		Message: message,
		Type:    ErrorTypeInternal,
		Stack:   captureStack(1),
	}
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, format string, args ...interface{}) *GasTownError {
	if err == nil {
		return nil
	}
	return &GasTownError{
		Cause:   err,
		Message: fmt.Sprintf(format, args...),
		Type:    ErrorTypeInternal,
		Stack:   captureStack(1),
	}
}

// Transient creates a transient (retryable) error.
func Transient(err error, message string) *GasTownError {
	e := Wrap(err, message)
	if e == nil {
		return nil
	}
	e.Type = ErrorTypeTransient
	e.Retryable = true
	return e
}

// Permanent creates a permanent (non-retryable) error.
func Permanent(err error, message string) *GasTownError {
	e := Wrap(err, message)
	if e == nil {
		return nil
	}
	e.Type = ErrorTypePermanent
	e.Retryable = false
	return e
}

// Validation creates a validation error.
func Validation(message string) *GasTownError {
	return &GasTownError{
		Message:   message,
		Type:      ErrorTypeValidation,
		Retryable: false,
		Stack:     captureStack(1),
	}
}

// NotFound creates a not-found error.
func NotFound(resource, name string) *GasTownError {
	return &GasTownError{
		Message:   fmt.Sprintf("%s %q not found", resource, name),
		Type:      ErrorTypeNotFound,
		Retryable: false,
		Stack:     captureStack(1),
		Context:   map[string]string{"resource": resource, "name": name},
	}
}

// GTCLIError creates an error from gt CLI execution.
func GTCLIError(err error, command string) *GasTownError {
	return &GasTownError{
		Cause:     err,
		Message:   fmt.Sprintf("gt CLI command failed: %s", command),
		Type:      ErrorTypeGTCLI,
		Retryable: true, // gt CLI errors are usually transient
		Stack:     captureStack(1),
		Context:   map[string]string{"command": command},
	}
}

// IsRetryable checks if an error should be retried.
func IsRetryable(err error) bool {
	var ge *GasTownError
	if errors.As(err, &ge) {
		return ge.Retryable
	}
	return false
}

// IsType checks if an error is of a specific type.
func IsType(err error, errType string) bool {
	var ge *GasTownError
	if errors.As(err, &ge) {
		return ge.Type == errType
	}
	return false
}

// IsNotFound checks if an error is a not-found error.
func IsNotFound(err error) bool {
	return IsType(err, ErrorTypeNotFound)
}

// IsValidation checks if an error is a validation error.
func IsValidation(err error) bool {
	return IsType(err, ErrorTypeValidation)
}

// IsGTCLIError checks if an error is from the gt CLI.
func IsGTCLIError(err error) bool {
	return IsType(err, ErrorTypeGTCLI)
}

// ToConditionReason converts an error type to a K8s condition reason.
func ToConditionReason(err error) string {
	var ge *GasTownError
	if errors.As(err, &ge) {
		switch ge.Type {
		case ErrorTypeTransient:
			return "TransientError"
		case ErrorTypePermanent:
			return "PermanentError"
		case ErrorTypeValidation:
			return "ValidationError"
		case ErrorTypeNotFound:
			return "ResourceNotFound"
		case ErrorTypeConflict:
			return "ResourceConflict"
		case ErrorTypeGTCLI:
			return "GTCLIError"
		default:
			return "InternalError"
		}
	}
	return "UnknownError"
}
