// Package diagnostic attaches structured metadata to unchanged parser messages.
package diagnostic

import "github.com/mia0x75/parserkit/loc"

// Severity records the diagnostic level; its zero value is unknown.
type Severity uint8

const (
	SeverityUnknown Severity = iota
	SeverityWarning
	SeverityError
)

// Diagnostic carries a parser message and its authoritative byte range.
// Use loc.InvalidSpan for unknown positions. Cause is optional.
type Diagnostic struct {
	Severity Severity
	Code     string
	Message  string
	Span     loc.Span
	Cause    error
}

// Error returns Message unchanged, without adding code or location prefixes.
func (d *Diagnostic) Error() string {
	if d == nil {
		return ""
	}
	return d.Message
}

// Unwrap exposes Cause for errors.Is and errors.As without changing the message.
func (d *Diagnostic) Unwrap() error {
	if d == nil {
		return nil
	}
	return d.Cause
}
