package ast

// BaseLimit shares pagination expressions without choosing a dialect's LIMIT,
// OFFSET or FETCH spelling. The outer node owns traversal and restoration.
type BaseLimit struct {
	BaseNode
	Count  ExprNode
	Offset ExprNode
}

func (n *BaseLimit) LimitCore() *BaseLimit { return n }

// LimitNode keeps pagination syntax attached to its concrete dialect outer.
type LimitNode interface {
	Node
	LimitCore() *BaseLimit
}
