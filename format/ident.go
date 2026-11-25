package format

import (
	"fmt"
	"strings"

	"github.com/mia0x75/parserkit/lex"
)

func fold(value string, mode CaseMode) string {
	switch mode {
	case CaseUpper:
		return strings.ToUpper(value)
	case CaseLower:
		return strings.ToLower(value)
	default:
		return value
	}
}

func (ctx *RestoreCtx) WritePlain(value string) {
	ctx.Writer.writeString(value)
}

func (ctx *RestoreCtx) WritePlainf(pattern string, args ...any) {
	if ctx.Err() == nil {
		ctx.WritePlain(fmt.Sprintf(pattern, args...))
	}
}

func (ctx *RestoreCtx) WriteKeyword(value string) {
	if ctx.Err() != nil {
		return
	}
	mode := ctx.Dialect.KeywordCase
	switch {
	case ctx.Flags&RestoreKeywordUppercase != 0:
		mode = CaseUpper
	case ctx.Flags&RestoreKeywordLowercase != 0:
		mode = CaseLower
	}
	ctx.WritePlain(fold(value, mode))
}

func (ctx *RestoreCtx) nameCase(quoted bool) CaseMode {
	switch {
	case ctx.Flags&RestoreNameUppercase != 0:
		return CaseUpper
	case ctx.Flags&RestoreNameLowercase != 0:
		return CaseLower
	case quoted:
		return CasePreserve
	default:
		return ctx.Dialect.IdentifierCase
	}
}

func (ctx *RestoreCtx) nameQuote() lex.QuoteStyle {
	switch {
	case ctx.Flags&RestoreNameDoubleQuotes != 0:
		return lex.QuoteDouble
	case ctx.Flags&RestoreNameBackQuotes != 0:
		return lex.QuoteBacktick
	default:
		return ctx.Dialect.IdentifierQuote
	}
}

func (ctx *RestoreCtx) WriteName(name string) {
	if ctx.Err() == nil {
		ctx.writeQuoted(fold(name, ctx.nameCase(false)), ctx.nameQuote())
	}
}

func (ctx *RestoreCtx) WriteIdent(ident lex.Ident) {
	if ctx.Err() == nil {
		ctx.writeQuoted(fold(ident.Name, ctx.nameCase(ident.Quote != lex.QuoteUnquoted)), ctx.nameQuote())
	}
}

func (ctx *RestoreCtx) WriteString(value string) {
	if ctx.Err() != nil {
		return
	}
	if ctx.Dialect.EscapeBackslash || ctx.Flags&RestoreStringEscapeBackslash != 0 {
		value = strings.ReplaceAll(value, `\`, `\\`)
	}
	quote := ctx.Dialect.StringQuote
	switch {
	case ctx.Flags&RestoreStringSingleQuotes != 0:
		quote = lex.QuoteSingle
	case ctx.Flags&RestoreStringDoubleQuotes != 0:
		quote = lex.QuoteDouble
	}
	ctx.writeQuoted(value, quote)
}

func (ctx *RestoreCtx) writeQuoted(value string, quote lex.QuoteStyle) {
	if !validIdentifierQuote(quote) {
		if ctx.Writer.err == nil {
			ctx.Writer.err = ErrInvalidConfig
		}
		return
	}
	var open, close string
	switch quote {
	case lex.QuoteUnquoted:
	case lex.QuoteBacktick:
		open, close = "`", "`"
	case lex.QuoteDouble:
		open, close = `"`, `"`
	case lex.QuoteBracket:
		open, close = "[", "]"
	case lex.QuoteSingle:
		open, close = "'", "'"
	default:
		if ctx.Writer.err == nil {
			ctx.Writer.err = ErrInvalidConfig
		}
		return
	}
	if close != "" {
		value = strings.ReplaceAll(value, close, close+close)
	}
	ctx.WritePlain(open)
	ctx.WritePlain(value)
	ctx.WritePlain(close)
}
