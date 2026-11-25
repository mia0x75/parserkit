package ast

// BaseTableStmt stores the table reference of a TABLE query. The dialect node
// owns its concrete identity, traversal and restoration.
type BaseTableStmt struct {
	BaseQuery
	// Table is the source table reference.
	Table TableRefNode
}
