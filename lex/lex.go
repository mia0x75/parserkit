// Package lex carries lexical spelling and byte ranges without dialect token IDs.
package lex

import "github.com/mia0x75/parserkit/loc"

// QuoteStyle records the original delimiters without validating dialect rules.
type QuoteStyle uint8

const (
	QuoteUnquoted QuoteStyle = iota
	QuoteBacktick
	QuoteDouble
	QuoteBracket
	QuoteSingle
)

// Ident keeps the decoded Name and complete original Raw spelling, including
// delimiters. Names are not normalized. Use loc.InvalidSpan for unknown ranges.
type Ident struct {
	Name  string
	Raw   string
	Span  loc.Span
	Quote QuoteStyle
}

// Present distinguishes an absent zero value from a quoted empty identifier.
// A span alone does not establish presence.
func (i Ident) Present() bool {
	return i.Name != "" || i.Raw != "" || i.Quote != QuoteUnquoted
}

// Token carries a dialect's own Kind and the complete original Raw spelling.
// Use loc.InvalidSpan for unknown ranges.
type Token struct {
	Kind int
	Raw  string
	Span loc.Span
}
