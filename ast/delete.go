package ast

// BaseDeleteStmt separates a single target from an independent source relation.
// Dialects own multi-target lists and concrete traversal and restoration.
type BaseDeleteStmt struct {
	BaseDML
	Table     ResultSetNode
	Using     ResultSetNode
	Where     ExprNode
	OrderBy   OrderByClauseNode
	Limit     LimitNode
	Returning *FieldList
}
