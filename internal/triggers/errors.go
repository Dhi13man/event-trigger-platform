package triggers

import "fmt"

// ValidationError represents user-facing validation issues.
type ValidationError struct {
	msg string
}

func (e ValidationError) Error() string {
	return e.msg
}

// NewValidationError creates a new validation error.
func NewValidationError(format string, args ...interface{}) error {
	return ValidationError{msg: fmt.Sprintf(format, args...)}
}
