package ast

import "github.com/mia0x75/parserkit/loc"

// TypeNode carries written type syntax, never a resolved runtime type.
type TypeNode interface {
	Node
	TypeCore() *BaseTypeName
}

// An absent Size distinguishes [] from an explicitly written [0].
type ArrayBound struct {
	Size ExprNode
	Span loc.Span
}

// Name and Keywords are alternative spellings, not two copies of a type name.
type BaseTypeName struct {
	BaseNode
	Name        QualifiedName
	Keywords    []string
	Modifiers   []ExprNode
	ArrayBounds []ArrayBound
}

func (n *BaseTypeName) TypeCore() *BaseTypeName { return n }
