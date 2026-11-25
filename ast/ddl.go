package ast

// BaseColumnDef holds the shared syntax of a column declaration. Dialects
// own their column options and visitor dispatch.
type BaseColumnDef struct {
	BaseNode
	// QualifiedName preserves legacy containers; dialect grammar limits declarations.
	Name QualifiedName
	Type TypeNode
}

// ColumnDefCore exposes the shared column declaration without dispatching.
func (n *BaseColumnDef) ColumnDefCore() *BaseColumnDef { return n }

// ColumnDefNode preserves the identity of a dialect's column declaration.
type ColumnDefNode interface {
	Node
	ColumnDefCore() *BaseColumnDef
}

// BaseCreateTableStmt stores the common table declaration and CTAS source.
type BaseCreateTableStmt struct {
	BaseDDL
	Table       TableRefNode
	Columns     []ColumnDefNode
	Constraints []ConstraintNode
	Query       QueryNode
	IfNotExists bool
	Temporary   bool
}
