%{
package fixture

import "github.com/mia0x75/parserkit/loc"

func capture(lex yyLexer, span loc.Span) {
	lex.(*lexer).spans = append(lex.(*lexer).spans, span)
}
%}

%union {
	span loc.Span
}

%token ATOM
%type <span> start list optional empty

%%

start:
	list optional { capture(yylex, $<span>$) }
	| empty { capture(yylex, $<span>$) }
	;

list:
	ATOM { capture(yylex, $<span>$) }
	| list ATOM { capture(yylex, $<span>$) }
	;

optional:
	/* empty */ { capture(yylex, $<span>$) }
	| ',' { capture(yylex, $<span>$) }
	;

empty:
	/* empty */ { capture(yylex, $<span>$) }
	;
