package ast

// BaseFromClause stores the relation introduced by FROM. The concrete clause
// remains responsible for dialect-specific join restoration and traversal.
type BaseFromClause struct {
	BaseNode
	// Table admits a single source or a dialect's complete relation tree.
	Table ResultSetNode
}

// FromClauseCore exposes the shared relation without dispatching.
func (n *BaseFromClause) FromClauseCore() *BaseFromClause { return n }

// FromClauseNode preserves the concrete dialect clause identity.
type FromClauseNode interface {
	Node
	FromClauseCore() *BaseFromClause
}

// BaseSelectStmt stores portable query-block syntax. It intentionally has no
// Accept or Restore method; concrete dialect statements own node identity.
type BaseSelectStmt struct {
	BaseQuery
	// Distinct requests duplicate elimination; false is the default ALL form.
	Distinct bool
	// Fields is the query projection in source order.
	Fields *FieldList
	// From is absent when the SELECT has no explicit FROM clause.
	From FromClauseNode
	// Where is the query's row-filter expression.
	Where ExprNode
	// GroupBy contains the query's grouping items.
	GroupBy GroupByClauseNode
	// Having is the post-grouping filter expression.
	Having ExprNode
	// WindowSpecs contains named window definitions in source order.
	WindowSpecs []WindowSpecNode
	// OrderBy is the query's ordering clause.
	OrderBy OrderByClauseNode
	// Limit is the dialect-specific pagination node through its shared role.
	Limit LimitNode
	// With is the optional common table expression clause.
	With WithClauseNode
}
