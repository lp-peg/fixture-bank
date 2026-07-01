// Package ferr provides the unified error model used across fixture-bank.
//
// DSL_SPEC.md requires that every user-facing failure carries an error_type
// so callers (CLI users or, eventually, MCP tool callers) can branch on the
// failure category instead of parsing free-form messages.
package ferr

import "fmt"

// Error is a fixture-bank domain error tagged with a stable error_type.
type Error struct {
	ErrorType string
	Message   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorType, e.Message)
}

// New creates an Error with a formatted message.
func New(errorType, format string, args ...any) *Error {
	return &Error{ErrorType: errorType, Message: fmt.Sprintf(format, args...)}
}

// Known error_type values. Keep this list in sync with DSL_SPEC.md.
const (
	TypeSyntaxError          = "syntax_error"
	TypeSchemaMismatch       = "schema_mismatch"
	TypeEmptyPool            = "empty_pool"
	TypeUniqueRetryExhausted = "unique_retry_exhausted"
	TypeUnknownRef           = "unknown_ref"
	TypeDBError              = "db_error"
	TypeUnsupportedFormat    = "unsupported_format"
	TypeFixtureNotFound      = "fixture_not_found"
)
