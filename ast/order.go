package ast

import (
	"fmt"

	"github.com/mia0x75/parserkit/format"
)

// BaseOrderItem shares sorting syntax without dispatching for an outer node.
// Desc=false covers both an omitted direction and explicit ASC.
type BaseOrderItem struct {
	BaseNode
	Expr ExprNode
	Desc bool
}

func (n *BaseOrderItem) OrderItemCore() *BaseOrderItem { return n }

// OrderItemNode admits dialect extensions without accepting grouping items.
type OrderItemNode interface {
	Node
	OrderItemCore() *BaseOrderItem
}

// OrderItem is a sorting item without dialect-specific extensions.
type OrderItem struct {
	BaseOrderItem
}

func (n *OrderItem) OrderItemCore() *BaseOrderItem {
	if n == nil {
		return nil
	}
	return &n.BaseOrderItem
}

func (n *OrderItem) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Expr == nil {
		return fmt.Errorf("restore order item: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return fmt.Errorf("restore order item expression: %w", err)
	}
	if n.Desc {
		ctx.WriteKeyword(" DESC")
	}
	return ctx.Err()
}

func (n *OrderItem) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	item, ok := entered.(*OrderItem)
	if !ok || item == nil {
		return entered, false
	}
	if skip {
		return v.Leave(item)
	}
	if item.Expr != nil {
		if !nodePresent(item.Expr) {
			return item, false
		}
		updated, ok := item.Expr.Accept(v)
		if !ok {
			return item, false
		}
		expr, ok := updated.(ExprNode)
		if !ok || !nodePresent(expr) {
			return item, false
		}
		item.Expr = expr
	}
	return v.Leave(item)
}

// BaseOrderByClause preserves item roles and order for an outer clause.
// Its interface slice supports dialect-specific ordering items, not arbitrary Nodes.
type BaseOrderByClause struct {
	BaseNode
	Items []OrderItemNode
}

func (n *BaseOrderByClause) OrderByClauseCore() *BaseOrderByClause { return n }

// OrderByClauseNode preserves the outer clause's dispatch and shared data.
type OrderByClauseNode interface {
	Node
	OrderByClauseCore() *BaseOrderByClause
}

var _ OrderItemNode = (*OrderItem)(nil)
