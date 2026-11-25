package format

import (
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/lex"
)

func TestDialectConfigValidatesByExplicitSet(t *testing.T) {
	t.Parallel()
	valid := []DialectConfig{
		{},
		{IdentifierQuote: lex.QuoteBacktick, StringQuote: lex.QuoteSingle, KeywordCase: CaseUpper, IdentifierCase: CasePreserve},
		{IdentifierQuote: lex.QuoteDouble, StringQuote: lex.QuoteDouble, KeywordCase: CaseLower, IdentifierCase: CaseLower},
		{IdentifierQuote: lex.QuoteBracket, StringQuote: lex.QuoteUnquoted, KeywordCase: CasePreserve, IdentifierCase: CaseUpper},
	}
	for _, dialect := range valid {
		if err := NewRestoreCtx(&strings.Builder{}, dialect, 0).Err(); err != nil {
			t.Errorf("valid config %+v rejected: %v", dialect, err)
		}
	}
	invalid := []DialectConfig{
		{IdentifierQuote: lex.QuoteStyle(5)},
		{StringQuote: lex.QuoteBracket},
		{KeywordCase: CaseMode(3)},
		{IdentifierCase: CaseMode(9)},
	}
	for _, dialect := range invalid {
		if err := NewRestoreCtx(&strings.Builder{}, dialect, 0).Err(); err != ErrInvalidConfig {
			t.Errorf("invalid config %+v returned %v, want ErrInvalidConfig", dialect, err)
		}
	}
}

func TestRestoreFlagsAndDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		flags   RestoreFlags
		dialect DialectConfig
		want    string
	}{
		{name: "no flags or dialect quoting", want: `select a'b MiXeD`},
		{name: "MySQL default", flags: DefaultRestoreFlags | RestoreNameBackQuotes, want: "SELECT 'a''b' `MiXeD`"},
		{name: "PG default", flags: DefaultRestoreFlags | RestoreNameDoubleQuotes, want: `SELECT 'a''b' "MiXeD"`},
		{name: "flag priority", flags: RestoreStringSingleQuotes | RestoreStringDoubleQuotes | RestoreKeywordUppercase | RestoreKeywordLowercase | RestoreNameUppercase | RestoreNameLowercase | RestoreNameDoubleQuotes | RestoreNameBackQuotes, want: `SELECT 'a''b' "MIXED"`},
		{name: "dialect default", dialect: DialectConfig{IdentifierQuote: lex.QuoteBracket, StringQuote: lex.QuoteDouble, KeywordCase: CaseLower}, want: `select "a'b" [MiXeD]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			ctx := NewRestoreCtx(&out, tt.dialect, tt.flags)
			ctx.WriteKeyword("select")
			ctx.WritePlain(" ")
			ctx.WriteString("a'b")
			ctx.WritePlain(" ")
			ctx.WriteName("MiXeD")
			if err := ctx.Err(); err != nil || out.String() != tt.want {
				t.Fatalf("output = %q, error = %v, want %q", out.String(), err, tt.want)
			}
		})
	}
}
