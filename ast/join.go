package ast

// JoinType classifies a relation, independently of a dialect's keyword spelling.
// Unknown is not INNER: it also identifies a parser's single-source join root.
type JoinType uint8

const (
	JoinTypeUnknown JoinType = iota
	JoinTypeInner
	JoinTypeLeft
	JoinTypeRight
	JoinTypeFull
	JoinTypeCross
)

// JoinExprNode exposes shared join syntax without hiding the dialect outer.
type JoinExprNode interface {
	ResultSetNode
	JoinExprCore() *BaseJoinExpr
}

// BaseJoinExpr stores standard relation syntax but never dispatches for an outer.
// On is the expression itself; Using retains identifiers rather than bindings.
type BaseJoinExpr struct {
	BaseResultSet
	Left    ResultSetNode
	Right   ResultSetNode
	On      ExprNode
	Using   []QualifiedName
	Type    JoinType
	Natural bool
}

// JoinExprCore returns the shared data, including for a nil receiver.
func (n *BaseJoinExpr) JoinExprCore() *BaseJoinExpr { return n }
