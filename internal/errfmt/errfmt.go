package errfmt

import (
	"errors"
	"os"
	"strings"

	"github.com/99designs/keyring"
	"github.com/alecthomas/kong"
)

func Format(err error) string {
	if err == nil {
		return ""
	}

	// Handle Kong parse errors with better messaging
	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return formatParseError(parseErr)
	}

	if errors.Is(err, keyring.ErrKeyNotFound) {
		return "Secret not found in keyring. Run: <cli> auth set-key --stdin"
	}

	if errors.Is(err, os.ErrNotExist) {
		return err.Error()
	}

	var userErr *UserFacingError
	if errors.As(err, &userErr) {
		return userErr.Message
	}

	return err.Error()
}

// UserFacingError forces a specific message, while preserving the underlying cause.
type UserFacingError struct {
	Message string
	Cause   error
}

func (e *UserFacingError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

func (e *UserFacingError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

func NewUserFacingError(message string, cause error) error {
	return &UserFacingError{Message: message, Cause: cause}
}

// formatParseError enhances Kong parse errors with helpful hints.
func formatParseError(err *kong.ParseError) string {
	msg := err.Error()

	// If Kong already provided a suggestion, use it as-is
	if strings.Contains(msg, "did you mean") {
		return msg
	}

	// For unknown flag errors without suggestions, add a help hint
	if strings.HasPrefix(msg, "unknown flag") {
		return msg + "\nRun with --help to see available flags"
	}

	// For missing required flags
	if strings.Contains(msg, "missing") || strings.Contains(msg, "required") {
		return msg + "\nRun with --help to see usage"
	}

	return msg
}
