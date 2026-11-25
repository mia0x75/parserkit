package ast

import (
	"fmt"

	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/lex"
)

// BaseWithClause stores syntax shared by dialects. Dialect nodes own dispatch
// and any dialect-specific WITH options.
type BaseWithClause struct {
	BaseNode
	IsRecursive bool
	CTEs        []CTENode
}

// WithClauseCore exposes the shared CTE syntax without making the base a
// dispatching AST node.
func (n *BaseWithClause) WithClauseCore() *BaseWithClause { return n }

// WithClauseNode preserves a dialect's WITH clause identity and extensions.
type WithClauseNode interface {
	Node
	WithClauseCore() *BaseWithClause
}

// BaseCommonTableExpression stores one common table expression.
type BaseCommonTableExpression struct {
	BaseNode
	Name    lex.Ident
	Columns []lex.Ident
	// Query also admits writable CTE bodies; dialect grammar limits statement kinds.
	Query StmtNode
}

// CTECore exposes shared syntax without dispatching for a dialect extension.
func (n *BaseCommonTableExpression) CTECore() *BaseCommonTableExpression { return n }

// CTENode preserves each dialect's CTE identity and extensions.
type CTENode interface {
	Node
	CTECore() *BaseCommonTableExpression
}

// CommonTableExpression is a CTE without dialect-specific extensions.
type CommonTableExpression struct {
	BaseCommonTableExpression
}

func (n *CommonTableExpression) CTECore() *BaseCommonTableExpression {
	if n == nil {
		return nil
	}
	return &n.BaseCommonTableExpression
}

func (n *CommonTableExpression) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Query) {
		return fmt.Errorf("restore common table expression: missing query")
	}
	ctx.WriteIdent(n.Name)
	if len(n.Columns) > 0 {
		ctx.WritePlain(" (")
		for i, column := range n.Columns {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(column)
		}
		ctx.WritePlain(")")
	}
	ctx.WriteKeyword(" AS ")
	ctx.WritePlain("(")
	if err := n.Query.Restore(ctx); err != nil {
		return fmt.Errorf("restore common table expression query: %w", err)
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

func (n *CommonTableExpression) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	cte, ok := entered.(*CommonTableExpression)
	if !ok || cte == nil {
		return entered, false
	}
	if skip {
		return v.Leave(cte)
	}
	if !nodePresent(cte.Query) {
		return cte, false
	}
	updated, ok := cte.Query.Accept(v)
	if !ok {
		return cte, false
	}
	query, ok := updated.(StmtNode)
	if !ok || !nodePresent(query) {
		return cte, false
	}
	cte.Query = query
	return v.Leave(cte)
}

var _ CTENode = (*CommonTableExpression)(nil)
