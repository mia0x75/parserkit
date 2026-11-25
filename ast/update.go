package ast

// BaseUpdateStmt shares update syntax; the outer node owns dialect dispatch.
type BaseUpdateStmt struct {
	BaseDML
	// Table is the complete relation being modified, including joins.
	Table       ResultSetNode
	Assignments []*Assignment
	Where       ExprNode
	OrderBy     OrderByClauseNode
	Limit       LimitNode
	Returning   *FieldList
}
