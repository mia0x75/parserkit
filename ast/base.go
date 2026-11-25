package ast

import "github.com/mia0x75/parserkit/loc"

// BaseNode deliberately has no Accept or Restore: promoted dispatch would lose
// the outer dialect node's identity or omit its extension fields.
type BaseNode struct {
	text    string
	span    loc.Span
	hasSpan bool
}

func (n *BaseNode) Text() string {
	return n.text
}

func (n *BaseNode) SetText(text string) {
	n.text = text
}

func (n *BaseNode) Span() loc.Span {
	if !n.hasSpan {
		return loc.InvalidSpan()
	}
	return n.span
}

func (n *BaseNode) SetSpan(span loc.Span) {
	if !span.Valid() {
		n.span = loc.InvalidSpan()
		n.hasSpan = false
		return
	}
	n.span = span
	n.hasSpan = true
}

type BaseExpr struct{ BaseNode }

func (*BaseExpr) expression() {}

// BaseQueryExpr is an expression that also yields a result set, such as a
// parenthesized subquery. It carries both markers so a dialect can use it
// wherever either an expression or a result set is expected.
type BaseQueryExpr struct{ BaseExpr }

func (*BaseQueryExpr) resultSet() {}

type BaseStmt struct{ BaseNode }

func (*BaseStmt) statement() {}

type BaseDML struct{ BaseStmt }

func (*BaseDML) dmlStatement() {}

type BaseDDL struct{ BaseStmt }

func (*BaseDDL) ddlStatement() {}

type BaseResultSet struct{ BaseNode }

func (*BaseResultSet) resultSet() {}

type BaseQuery struct{ BaseDML }

func (*BaseQuery) resultSet() {}

// Bases satisfy category markers, but must never acquire node dispatch methods.
var (
	_ interface{ expression() }   = (*BaseExpr)(nil)
	_ interface{ statement() }    = (*BaseStmt)(nil)
	_ interface{ dmlStatement() } = (*BaseDML)(nil)
	_ interface{ ddlStatement() } = (*BaseDDL)(nil)
	_ interface{ resultSet() }    = (*BaseQueryExpr)(nil)
	_ interface{ resultSet() }    = (*BaseResultSet)(nil)
	_ interface{ resultSet() }    = (*BaseQuery)(nil)
)
