package ast

import (
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

type Ident = lex.Ident

// QualifiedName keeps quoted dots inside one part rather than flattening them.
type QualifiedName struct {
	Parts []Ident
	Span  loc.Span
}

// Syntax accessors expose shared data without dispatching on an embedded base.
type ColumnRefNode interface {
	ExprNode
	ColumnRefCore() *BaseColumnRef
}

type TableRefNode interface {
	ResultSetNode
	TableRefCore() *BaseTableRef
}

type BaseColumnRef struct {
	BaseExpr
	Name QualifiedName
}

func (n *BaseColumnRef) ColumnRefCore() *BaseColumnRef { return n }

type BaseTableRef struct {
	BaseResultSet
	Name QualifiedName
}

func (n *BaseTableRef) TableRefCore() *BaseTableRef { return n }
