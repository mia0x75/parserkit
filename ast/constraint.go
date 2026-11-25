package ast

import "github.com/mia0x75/parserkit/lex"

// ConstraintKind describes portable syntax, not enforcement or index metadata.
type ConstraintKind uint8

const (
	ConstraintKindUnknown ConstraintKind = iota
	ConstraintKindPrimaryKey
	ConstraintKindUnique
	ConstraintKindForeignKey
	ConstraintKindCheck
	ConstraintKindIndex
)

// BaseConstraint keeps the CONSTRAINT symbol separate from dialect index names.
// The concrete dialect maps its sole kind field through ConstraintKind.
type BaseConstraint struct {
	BaseNode
	Name  lex.Ident
	Keys  []IndexKeyPartNode
	Expr  ExprNode
	Refer ReferenceDefNode
}

func (n *BaseConstraint) ConstraintCore() *BaseConstraint { return n }

type ConstraintNode interface {
	Node
	ConstraintCore() *BaseConstraint
	ConstraintKind() ConstraintKind
}
