package format

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/lex"
)

type failingWriter struct {
	calls int
	err   error
	short bool
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.short {
		return len(p) - 1, nil
	}
	return 0, w.err
}

func TestRestoreWriter(t *testing.T) {
	sentinel := errors.New("test write failed")
	tests := []struct {
		name   string
		writer *failingWriter
		want   error
	}{
		{name: "failure", writer: &failingWriter{err: sentinel}, want: sentinel},
		{name: "short write", writer: &failingWriter{short: true}, want: io.ErrShortWrite},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewRestoreWriter(tt.writer)
			if _, err := writer.WriteString("first"); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if _, err := writer.Write([]byte("later")); !errors.Is(err, tt.want) {
				t.Fatalf("later error = %v, want %v", err, tt.want)
			}
			if tt.writer.calls != 1 {
				t.Fatalf("writer called %d times after first failure", tt.writer.calls)
			}
		})
	}
	t.Run("nil and zero writer", func(t *testing.T) {
		var nilBuilder *strings.Builder
		// A typed nil boxed into io.Writer survives the == nil check and
		// would panic on Write; it must report ErrNilWriter instead.
		writers := []RestoreWriter{{}, NewRestoreWriter(nil), NewRestoreWriter(io.Writer(nilBuilder))}
		for i := range writers {
			if _, err := writers[i].WriteString(""); !errors.Is(err, ErrNilWriter) {
				t.Fatalf("nil writer %d error = %v", i, err)
			}
			if err := writers[i].Err(); !errors.Is(err, ErrNilWriter) {
				t.Fatalf("nil writer %d Err() = %v", i, err)
			}
		}
	})
}

func TestDialectOutput(t *testing.T) {
	tests := []struct {
		name  string
		quote lex.QuoteStyle
		input string
		want  string
	}{
		{name: "backtick", quote: lex.QuoteBacktick, input: "a`b", want: "`a``b`"},
		{name: "double", quote: lex.QuoteDouble, input: `a"b`, want: `"a""b"`},
		{name: "bracket", quote: lex.QuoteBracket, input: "a]b", want: "[a]]b]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			ctx := NewRestoreCtx(&out, DialectConfig{IdentifierQuote: tt.quote}, DefaultRestoreFlags)
			ctx.WriteKeyword("select")
			ctx.WritePlain(" ")
			ctx.WriteName(tt.input)
			ctx.WritePlain(" ")
			ctx.WriteString("a'b")
			if ctx.Err() != nil || out.String() != "SELECT "+tt.want+" 'a''b'" {
				t.Fatalf("output = %q, error = %v", out.String(), ctx.Err())
			}
		})
	}
}

func TestQuotedIdentCase(t *testing.T) {
	var out strings.Builder
	ctx := NewRestoreCtx(&out, DialectConfig{IdentifierQuote: lex.QuoteDouble, IdentifierCase: CaseLower}, 0)
	ctx.WriteIdent(lex.Ident{Name: "MiXeD", Raw: `"stale"`, Quote: lex.QuoteDouble})
	ctx.WritePlain(" ")
	ctx.WriteIdent(lex.Ident{Name: "MiXeD"})
	if ctx.Err() != nil || out.String() != `"MiXeD" "mixed"` {
		t.Fatalf("output = %q, error = %v", out.String(), ctx.Err())
	}
}

func TestInvalidConfigAndPropagation(t *testing.T) {
	ctx := NewRestoreCtx(io.Discard, DialectConfig{StringQuote: lex.QuoteBracket}, 0)
	ctx.WriteString("value")
	if !errors.Is(ctx.Err(), ErrInvalidConfig) {
		t.Fatalf("invalid config error = %v", ctx.Err())
	}
	sentinel := errors.New("test failed")
	w := &failingWriter{err: sentinel}
	ctx = NewRestoreCtx(w, DialectConfig{IdentifierQuote: lex.QuoteDouble}, DefaultRestoreFlags)
	ctx.WriteName("name")
	ctx.WritePlainf("%d", 1)
	ctx.WriteString("later")
	if !errors.Is(ctx.Err(), sentinel) || w.calls != 1 {
		t.Fatalf("failure not preserved: error = %v, calls = %d", ctx.Err(), w.calls)
	}
}
