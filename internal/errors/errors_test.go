package errors

import (
	"fmt"
	"testing"
)

func TestBusinessError(t *testing.T) {
	tests := []struct {
		name     string
		err      *BusinessError
		expected string
	}{
		{
			name: "simple error",
			err: &BusinessError{
				Type:    ErrorTypeNotFound,
				Message: "item not found",
			},
			expected: "NOT_FOUND: item not found",
		},
		{
			name: "error with cause",
			err: &BusinessError{
				Type:    ErrorTypeInternal,
				Message: "database error",
				Cause:   fmt.Errorf("connection failed"),
			},
			expected: "INTERNAL: database error (caused by: connection failed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		err := NotFound("test message")
		if err.Type != ErrorTypeNotFound {
			t.Errorf("expected type %v, got %v", ErrorTypeNotFound, err.Type)
		}
		if err.Message != "test message" {
			t.Errorf("expected message 'test message', got %v", err.Message)
		}
	})

	t.Run("AlreadyExists", func(t *testing.T) {
		err := AlreadyExists("test message")
		if err.Type != ErrorTypeAlreadyExists {
			t.Errorf("expected type %v, got %v", ErrorTypeAlreadyExists, err.Type)
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		err := InvalidInput("test message")
		if err.Type != ErrorTypeInvalidInput {
			t.Errorf("expected type %v, got %v", ErrorTypeInvalidInput, err.Type)
		}
	})

	t.Run("Internal", func(t *testing.T) {
		cause := fmt.Errorf("root cause")
		err := Internal("test message", cause)
		if err.Type != ErrorTypeInternal {
			t.Errorf("expected type %v, got %v", ErrorTypeInternal, err.Type)
		}
		if err.Cause != cause {
			t.Errorf("expected cause %v, got %v", cause, err.Cause)
		}
	})

	t.Run("Auth", func(t *testing.T) {
		err := Auth("test message")
		if err.Type != ErrorTypeAuth {
			t.Errorf("expected type %v, got %v", ErrorTypeAuth, err.Type)
		}
	})
}

func TestWrap(t *testing.T) {
	t.Run("wrap nil error", func(t *testing.T) {
		result := Wrap(nil, "test message")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("wrap business error", func(t *testing.T) {
		original := NotFound("original message")
		result := Wrap(original, "wrapped message")
		if result != original {
			t.Errorf("expected same error object, got different")
		}
	})

	t.Run("wrap standard error", func(t *testing.T) {
		original := fmt.Errorf("standard error")
		result := Wrap(original, "wrapped message")

		if result.Type != ErrorTypeInternal {
			t.Errorf("expected type %v, got %v", ErrorTypeInternal, result.Type)
		}
		if result.Message != "wrapped message" {
			t.Errorf("expected message 'wrapped message', got %v", result.Message)
		}
		if result.Cause != original {
			t.Errorf("expected cause %v, got %v", original, result.Cause)
		}
	})
}

func TestUnwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := Internal("test message", cause)

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("expected %v, got %v", cause, unwrapped)
	}

	// Test error without cause
	err2 := NotFound("test message")
	if unwrapped := err2.Unwrap(); unwrapped != nil {
		t.Errorf("expected nil, got %v", unwrapped)
	}
}
