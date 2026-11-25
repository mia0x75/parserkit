package ast

// BaseGroupItem is grouping syntax, not a sorting item. Direction modifiers and
// grouping sets belong to dialect-specific outer nodes.
type BaseGroupItem struct {
	BaseNode
	Expr ExprNode
}

func (n *BaseGroupItem) GroupItemCore() *BaseGroupItem { return n }

// GroupItemNode admits grouping extensions without accepting ordering items.
type GroupItemNode interface {
	Node
	GroupItemCore() *BaseGroupItem
}

// BaseGroupByClause shares grouping items; the outer clause owns its modifiers.
type BaseGroupByClause struct {
	BaseNode
	Items []GroupItemNode
}

// GroupByClauseCore exposes the shared grouping syntax without dispatching.
func (n *BaseGroupByClause) GroupByClauseCore() *BaseGroupByClause { return n }

// GroupByClauseNode preserves the concrete dialect clause identity.
type GroupByClauseNode interface {
	Node
	GroupByClauseCore() *BaseGroupByClause
}
