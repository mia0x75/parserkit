package ast

// BaseInsertStmt shares insertion syntax without dispatching on a dialect's
// behalf. The outer statement owns traversal, restoration and conflict clauses.
type BaseInsertStmt struct {
	BaseDML
	// Table is the write target, not a FROM source or a join.
	Table TableRefNode
	// Columns are syntactic write targets: column references, or a dialect's
	// field/subscript targets. The grammar restricts which expressions occur.
	// A nil list is absent; a non-nil empty list retains explicit ().
	Columns []ExprNode
	// Values and Query are alternative sources. Nil and non-nil empty slices
	// remain distinct; dialect-specific sources such as SET belong outside.
	Values [][]ExprNode
	Query  QueryNode
	// Returning is the optional projection of inserted rows.
	Returning *FieldList
}
