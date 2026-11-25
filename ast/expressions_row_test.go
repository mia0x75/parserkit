package ast_test

import (
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/loc"
)

type rowBaseExpr struct {
	ast.BaseRowExpr
}

var _ ast.RowNode = (*rowBaseExpr)(nil)

func (n *rowBaseExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*rowBaseExpr)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	for i, child := range outer.Items {
		if child == nil {
			continue
		}
		updated, ok := child.Accept(v)
		if !ok {
			return outer, false
		}
		expr, ok := updated.(ast.ExprNode)
		if !ok {
			return outer, false
		}
		outer.Items[i] = expr
	}
	return v.Leave(outer)
}

func (n *rowBaseExpr) Restore(ctx *format.RestoreCtx) error { return ctx.Err() }

func TestRowCoreKeepsItemsAndIdentity(t *testing.T) {
	child := &dialectExpr{}
	row := &rowBaseExpr{BaseRowExpr: ast.BaseRowExpr{Items: []ast.ExprNode{child, child}}}
	core := row.RowCore()
	if core != &row.BaseRowExpr {
		t.Fatal("row core must return the original embedded base")
	}
	if len(core.Items) != 2 || core.Items[0] != child || core.Items[1] != child {
		t.Fatal("row core must preserve each expression slot")
	}
	row.SetSpan(loc.Span{Start: 3, End: 12})
	if core.Span() != row.Span() {
		t.Fatal("row core lost the outer span")
	}
	var entered []ast.Node
	got, ok := row.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
		entered = append(entered, n)
		return n, false
	}})
	if !ok || got != row || !sameOrder(entered, []ast.Node{row, child, child}) {
		t.Fatal("row must dispatch its outer identity and visit shared items by slot")
	}
	replacement := &dialectExpr{}
	core.Items[1] = replacement
	if row.Items[1] != replacement {
		t.Fatal("row core must not return a copy")
	}
}
