package format

import (
	"errors"
	"io"

	"github.com/mia0x75/parserkit/lex"
)

type CaseMode uint8

const (
	CasePreserve CaseMode = iota
	CaseUpper
	CaseLower
)

type DialectConfig struct {
	IdentifierQuote lex.QuoteStyle
	KeywordCase     CaseMode
	IdentifierCase  CaseMode
	StringQuote     lex.QuoteStyle
	EscapeBackslash bool
}

type RestoreFlags uint64

const (
	RestoreStringSingleQuotes RestoreFlags = 1 << iota
	RestoreStringDoubleQuotes
	RestoreStringEscapeBackslash
	RestoreKeywordUppercase
	RestoreKeywordLowercase
	RestoreNameUppercase
	RestoreNameLowercase
	RestoreNameDoubleQuotes
	RestoreNameBackQuotes
	RestoreSpacesAroundBinaryOperation
)

const DefaultRestoreFlags = RestoreStringSingleQuotes | RestoreKeywordUppercase

var ErrInvalidConfig = errors.New("format: invalid dialect configuration")

type RestoreCtx struct {
	Writer    RestoreWriter
	Flags     RestoreFlags
	Dialect   DialectConfig
	JoinLevel int
}

func validIdentifierQuote(quote lex.QuoteStyle) bool {
	switch quote {
	case lex.QuoteUnquoted, lex.QuoteBacktick, lex.QuoteDouble, lex.QuoteBracket, lex.QuoteSingle:
		return true
	default:
		return false
	}
}

func validStringQuote(quote lex.QuoteStyle) bool {
	switch quote {
	case lex.QuoteUnquoted, lex.QuoteSingle, lex.QuoteDouble:
		return true
	default:
		return false
	}
}

func validCaseMode(mode CaseMode) bool {
	switch mode {
	case CasePreserve, CaseUpper, CaseLower:
		return true
	default:
		return false
	}
}

func NewRestoreCtx(out io.Writer, dialect DialectConfig, flags RestoreFlags) *RestoreCtx {
	ctx := &RestoreCtx{Writer: NewRestoreWriter(out), Flags: flags, Dialect: dialect}
	if !validIdentifierQuote(dialect.IdentifierQuote) || !validStringQuote(dialect.StringQuote) ||
		!validCaseMode(dialect.KeywordCase) || !validCaseMode(dialect.IdentifierCase) {
		if ctx.Writer.err == nil {
			ctx.Writer.err = ErrInvalidConfig
		}
	}
	return ctx
}

func (ctx *RestoreCtx) Err() error {
	return ctx.Writer.Err()
}

func (flags RestoreFlags) HasSpacesAroundBinaryOperationFlag() bool {
	return flags&RestoreSpacesAroundBinaryOperation != 0
}
