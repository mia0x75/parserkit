package ast

// BaseValuesStmt stores the rows of a table value constructor. The dialect
// node owns row spelling, traversal and restoration.
type BaseValuesStmt struct {
	BaseQuery
	// Rows are the value constructor rows in source order.
	Rows []RowNode
}
