package errfmt

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/99designs/keyring"
)

var (
	errReadAPIKey     = errors.New("read API key")
	errGeneric        = errors.New("something went wrong")
	errTestUnderlying = errors.New("underlying cause")
)

func TestFormat_Nil(t *testing.T) {
	result := Format(nil)
	if result != "" {
		t.Errorf("Format(nil) = %q, want empty string", result)
	}
}

func TestFormat_KeyNotFound(t *testing.T) {
	result := Format(keyring.ErrKeyNotFound)

	if !strings.Contains(result, "not found") {
		t.Errorf("expected 'not found' in output, got: %s", result)
	}

	if !strings.Contains(result, "auth set-credentials") {
		t.Errorf("expected auth instructions in output, got: %s", result)
	}
}

func TestFormat_WrappedKeyNotFound(t *testing.T) {
	wrapped := fmt.Errorf("%w: %w", errReadAPIKey, keyring.ErrKeyNotFound)
	result := Format(wrapped)

	if !strings.Contains(result, "not found") {
		t.Errorf("expected 'not found' for wrapped keyring error, got: %s", result)
	}
}

func TestFormat_NotExist(t *testing.T) {
	result := Format(os.ErrNotExist)

	if result == "" {
		t.Error("expected non-empty output for ErrNotExist")
	}
}

func TestFormat_GenericError(t *testing.T) {
	result := Format(errGeneric)

	if result != "something went wrong" {
		t.Errorf("expected error message, got: %s", result)
	}
}

func TestUserFacingError(t *testing.T) {
	err := NewUserFacingError("friendly message", errTestUnderlying)

	result := Format(err)
	if result != "friendly message" {
		t.Errorf("expected 'friendly message', got: %s", result)
	}

	var userErr *UserFacingError

	if !errors.As(err, &userErr) {
		t.Fatal("expected UserFacingError")
	}

	if !errors.Is(userErr.Unwrap(), errTestUnderlying) {
		t.Error("Unwrap should return the cause")
	}
}

func TestUserFacingError_Nil(t *testing.T) {
	var err *UserFacingError

	if err.Error() != "" {
		t.Error("nil UserFacingError.Error() should be empty")
	}

	if err.Unwrap() != nil {
		t.Error("nil UserFacingError.Unwrap() should be nil")
	}
}
