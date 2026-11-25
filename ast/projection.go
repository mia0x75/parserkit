package ast

import (
	"fmt"

	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/lex"
)

// SelectItem has one expression slot, including wildcards. Alias is a single
// identifier; Present distinguishes an absent alias from a quoted empty alias.
// Projection planning metadata does not belong in this syntax node.
type SelectItem struct {
	BaseNode
	Expr  ExprNode
	Alias lex.Ident
}

// Restore uses the caller's identifier quoting policy, not Alias.Raw.
func (n *SelectItem) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Expr == nil {
		return fmt.Errorf("restore select item: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return fmt.Errorf("restore select item expression: %w", err)
	}
	if n.Alias.Present() {
		ctx.WriteKeyword(" AS ")
		ctx.WriteIdent(n.Alias)
	}
	return ctx.Err()
}

// Accept allows Leave to replace an expression, including a wildcard.
func (n *SelectItem) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	item, ok := entered.(*SelectItem)
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

// FieldList preserves projection order without reducing items to column names.
type FieldList struct {
	BaseNode
	Items []*SelectItem
}

// Restore reports the item index while preserving the original error chain.
func (n *FieldList) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return fmt.Errorf("restore field list: nil list")
	}
	for i, item := range n.Items {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := item.Restore(ctx); err != nil {
			return fmt.Errorf("restore field list item %d: %w", i, err)
		}
	}
	return ctx.Err()
}

// Accept visits each item in source order and keeps replacements in its slot.
func (n *FieldList) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	list, ok := entered.(*FieldList)
	if !ok || list == nil {
		return entered, false
	}
	if skip {
		return v.Leave(list)
	}
	for i, item := range list.Items {
		if item == nil {
			return list, false
		}
		updated, ok := item.Accept(v)
		if !ok {
			return list, false
		}
		replacement, ok := updated.(*SelectItem)
		if !ok || replacement == nil {
			return list, false
		}
		list.Items[i] = replacement
	}
	return v.Leave(list)
}

// WildcardExpr keeps the qualifier separate from the star, so a quoted dot
// inside a name is never mistaken for a qualification boundary.
type WildcardExpr struct {
	BaseExpr
	Qualifier QualifiedName
}

// Restore treats an empty qualifier as the unqualified star.
func (n *WildcardExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return fmt.Errorf("restore wildcard: nil expression")
	}
	for _, part := range n.Qualifier.Parts {
		ctx.WriteIdent(part)
		ctx.WritePlain(".")
	}
	ctx.WritePlain("*")
	return ctx.Err()
}

// Accept keeps lexical names as values rather than visiting name wrappers.
func (n *WildcardExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, _ := v.Enter(n)
	wildcard, ok := entered.(*WildcardExpr)
	if !ok || wildcard == nil {
		return entered, false
	}
	return v.Leave(wildcard)
}

var (
	_ Node     = (*SelectItem)(nil)
	_ Node     = (*FieldList)(nil)
	_ ExprNode = (*WildcardExpr)(nil)
)
