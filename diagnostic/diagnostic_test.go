package diagnostic_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/mia0x75/parserkit/diagnostic"
	"github.com/mia0x75/parserkit/loc"
)

func TestDiagnosticError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		diag diagnostic.Diagnostic
		want string
	}{
		{name: "zero value"},
		{
			name: "original dialect message",
			diag: diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Code:     "syntax",
				Message:  "ERROR: syntax error at or near \"SELECT\"",
				Span:     loc.Span{Start: 5, End: 11},
				Cause:    errors.New("underlying parser error"),
			},
			want: "ERROR: syntax error at or near \"SELECT\"",
		},
		{
			name: "unknown position",
			diag: diagnostic.Diagnostic{Message: "unexpected token", Span: loc.InvalidSpan()},
			want: "unexpected token",
		},
		{
			name: "cause without message",
			diag: diagnostic.Diagnostic{Cause: errors.New("cause")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.diag.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiagnosticUnwrap(t *testing.T) {
	t.Parallel()
	cause := &strconv.NumError{Func: "ParseInt", Num: "x", Err: strconv.ErrSyntax}
	diag := diagnostic.Diagnostic{
		Severity: diagnostic.SeverityWarning,
		Code:     "invalid_number",
		Message:  "invalid number",
		Span:     loc.InvalidSpan(),
		Cause:    cause,
	}
	wrapped := fmt.Errorf("parse: %w", &diag)
	if !errors.Is(wrapped, cause) || !errors.Is(wrapped, strconv.ErrSyntax) {
		t.Fatal("errors.Is must find the direct and nested causes")
	}
	if errors.Is(wrapped, strconv.ErrRange) {
		t.Fatal("errors.Is must not match unrelated causes")
	}
	var got *diagnostic.Diagnostic
	if !errors.As(wrapped, &got) {
		t.Fatal("errors.As must find the diagnostic")
	}
	if got.Severity != diag.Severity || got.Code != diag.Code || got.Span != loc.InvalidSpan() {
		t.Fatalf("errors.As lost diagnostic metadata: %+v", got)
	}
	if got.Span.Valid() || got.Span.Empty() {
		t.Fatal("unknown diagnostic position must not become an empty range")
	}
	var gotCause *strconv.NumError
	if !errors.As(wrapped, &gotCause) || gotCause != cause {
		t.Fatal("errors.As must find the typed cause")
	}
	if got != &diag {
		t.Fatal("errors.As must retain diagnostic identity")
	}
	var zero diagnostic.Diagnostic
	if zero.Unwrap() != nil {
		t.Fatal("a diagnostic without a cause must unwrap to nil")
	}
	var absent *diagnostic.Diagnostic
	if absent.Error() != "" || absent.Unwrap() != nil {
		t.Fatal("a nil diagnostic must not panic")
	}
}
