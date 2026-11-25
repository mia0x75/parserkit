package ast

// BaseIndexKeyPart stores either a column name or an expression. PrefixLength
// and Desc describe the written key part, not resolved index metadata.
type BaseIndexKeyPart struct {
	BaseNode
	Column       QualifiedName
	Expr         ExprNode
	PrefixLength int
	Desc         bool
}

func (n *BaseIndexKeyPart) IndexKeyPartCore() *BaseIndexKeyPart { return n }

// IndexKeyPartNode admits dialect key syntax, not arbitrary AST nodes.
type IndexKeyPartNode interface {
	Node
	IndexKeyPartCore() *BaseIndexKeyPart
}
