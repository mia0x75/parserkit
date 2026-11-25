package lex_test

import (
	"testing"

	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func TestIdentPresent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		ident lex.Ident
		want  bool
	}{
		{name: "zero value"},
		{name: "unknown span", ident: lex.Ident{Span: loc.InvalidSpan()}},
		{name: "span only", ident: lex.Ident{Span: loc.Span{Start: 2, End: 3}}},
		{name: "name only", ident: lex.Ident{Name: "MiXeD€"}, want: true},
		{name: "raw only", ident: lex.Ident{Raw: "\"\""}, want: true},
		{name: "empty backtick", ident: lex.Ident{Raw: "``", Quote: lex.QuoteBacktick}, want: true},
		{name: "empty double", ident: lex.Ident{Raw: "\"\"", Quote: lex.QuoteDouble}, want: true},
		{name: "empty bracket", ident: lex.Ident{Raw: "[]", Quote: lex.QuoteBracket}, want: true},
		{name: "empty single", ident: lex.Ident{Raw: "''", Quote: lex.QuoteSingle}, want: true},
		{name: "quote only", ident: lex.Ident{Quote: lex.QuoteDouble}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.ident.Present(); got != tt.want {
				t.Errorf("%+v.Present() = %t, want %t", tt.ident, got, tt.want)
			}
		})
	}
}

func TestLexicalSpelling(t *testing.T) {
	t.Parallel()
	const input = "AS `MiXeD``€`"
	span := loc.Span{Start: 3, End: loc.ByteOffset(len(input))}
	ident := lex.Ident{Name: "MiXeD`€", Raw: "`MiXeD``€`", Span: span, Quote: lex.QuoteBacktick}
	token := lex.Token{Kind: 517, Raw: ident.Raw, Span: span}
	raw, err := token.Span.Slice(input)
	if err != nil {
		t.Fatal(err)
	}
	if raw != token.Raw || raw != ident.Raw {
		t.Fatalf("source spelling %q does not match token or identifier", raw)
	}
	if ident.Name != "MiXeD`€" || ident.Quote != lex.QuoteBacktick || token.Kind != 517 {
		t.Fatal("lexical metadata must retain its original values")
	}
}
