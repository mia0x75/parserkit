package ast

// SetOperationType identifies a set operation and its duplicate policy.
// Unknown is the zero value so an unset operator cannot be mistaken for UNION.
type SetOperationType uint8

const (
	SetOperationUnknown SetOperationType = iota
	SetOperationUnion
	SetOperationUnionAll
	SetOperationExcept
	SetOperationExceptAll
	SetOperationIntersect
	SetOperationIntersectAll
)

// String returns the SQL spelling of the set operation.
func (o SetOperationType) String() string {
	switch o {
	case SetOperationUnion:
		return "UNION"
	case SetOperationUnionAll:
		return "UNION ALL"
	case SetOperationExcept:
		return "EXCEPT"
	case SetOperationExceptAll:
		return "EXCEPT ALL"
	case SetOperationIntersect:
		return "INTERSECT"
	case SetOperationIntersectAll:
		return "INTERSECT ALL"
	default:
		return ""
	}
}

// BaseSetOperation stores one binary set-operation node. It deliberately does
// not implement Accept or Restore; the dialect node owns dispatch and SQL
// spelling.
type BaseSetOperation struct {
	BaseQuery
	Left     QueryNode
	Right    QueryNode
	Operator SetOperationType
}

// SetOperationCore exposes the shared binary operation data.
func (n *BaseSetOperation) SetOperationCore() *BaseSetOperation {
	return n
}

// SetOperationNode preserves the concrete dialect node's identity.
type SetOperationNode interface {
	QueryNode
	SetOperationCore() *BaseSetOperation
}

// BaseQueryTail stores the portable ORDER BY and LIMIT tail of one query
// level. Dialects own any additional tail clauses such as INTO or locking.
type BaseQueryTail struct {
	BaseNode
	OrderBy OrderByClauseNode
	Limit   LimitNode
}

// QueryTailCore exposes the shared query-tail data.
func (n *BaseQueryTail) QueryTailCore() *BaseQueryTail {
	return n
}

// BaseQueryTailStmt stores a query and the tail attached to that query level.
// Parenthesized distinguishes an explicit grouping from an ordinary wrapper.
// It does not implement Accept or Restore.
type BaseQueryTailStmt struct {
	BaseQuery
	Query         QueryNode
	Tail          BaseQueryTail
	Parenthesized bool
}

// QueryTailStmtCore exposes the shared query-wrapper data.
func (n *BaseQueryTailStmt) QueryTailStmtCore() *BaseQueryTailStmt {
	return n
}

// QueryTailNode preserves the concrete dialect node's identity.
type QueryTailNode interface {
	QueryNode
	QueryTailStmtCore() *BaseQueryTailStmt
}
