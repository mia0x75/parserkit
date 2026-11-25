package ast

// BaseReferenceDef preserves the referenced table and its ordered columns.
// Action clauses remain concrete dialect nodes so traversal retains identity.
type BaseReferenceDef struct {
	BaseNode
	Table   TableRefNode
	Columns []IndexKeyPartNode
}

func (n *BaseReferenceDef) ReferenceDefCore() *BaseReferenceDef { return n }

type ReferenceDefNode interface {
	Node
	ReferenceDefCore() *BaseReferenceDef
}
