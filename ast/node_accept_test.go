package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
)

func TestConcreteAcceptContract(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		new     func() ast.Node
		nilNode ast.Node
	}{
		{"assignment", func() ast.Node { return &ast.Assignment{} }, (*ast.Assignment)(nil)},
		{"order", func() ast.Node { return &ast.OrderItem{} }, (*ast.OrderItem)(nil)},
		{"select item", func() ast.Node { return &ast.SelectItem{} }, (*ast.SelectItem)(nil)},
		{"field list", func() ast.Node { return &ast.FieldList{} }, (*ast.FieldList)(nil)},
		{"wildcard", func() ast.Node { return &ast.WildcardExpr{} }, (*ast.WildcardExpr)(nil)},
		{"table source", func() ast.Node { return &ast.TableSource{} }, (*ast.TableSource)(nil)},
		{"cte", func() ast.Node {
			return &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{Query: &writableCTEStmt{}}}
		}, (*ast.CommonTableExpression)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			v := visitor{enter: func(n ast.Node) (ast.Node, bool) { calls++; return n, false }, leave: func(n ast.Node) (ast.Node, bool) { calls++; return n, true }}
			if got, ok := tc.nilNode.Accept(v); got != nil || ok || calls != 0 {
				t.Fatal("nil receiver must stop without visitor callbacks")
			}
			for _, skip := range []bool{false, true} {
				for _, replacement := range []ast.Node{nil, tc.nilNode, &dialectExpr{}} {
					left := false
					_, ok := tc.new().Accept(visitor{
						enter: func(ast.Node) (ast.Node, bool) { return replacement, skip },
						leave: func(n ast.Node) (ast.Node, bool) { left = true; return n, true },
					})
					if ok || left {
						t.Fatalf("invalid Enter replacement %T must stop before Leave, skip=%v", replacement, skip)
					}
				}
			}
			replacement := tc.new()
			entered, left := 0, 0
			got, ok := tc.new().Accept(visitor{
				enter: func(ast.Node) (ast.Node, bool) { entered++; return replacement, true },
				leave: func(n ast.Node) (ast.Node, bool) { left++; return n, true },
			})
			if !ok || got != replacement || entered != 1 || left != 1 {
				t.Fatal("skip must leave the same-type replacement once without children")
			}
			if _, ok := tc.new().Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) { return n, false }}); ok {
				t.Fatal("Leave stop must propagate")
			}
		})
	}
}

func TestAcceptRejectsTypedNilSlotsAndReplacements(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		child ast.Node
		new   func(ast.Node) ast.Node
	}{
		{"target", (*dialectExpr)(nil), func(n ast.Node) ast.Node { return &ast.Assignment{Target: n.(ast.ExprNode)} }},
		{"value", (*dialectExpr)(nil), func(n ast.Node) ast.Node { return &ast.Assignment{Expr: n.(ast.ExprNode)} }},
		{"order", (*dialectExpr)(nil), func(n ast.Node) ast.Node {
			return &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: n.(ast.ExprNode)}}
		}},
		{"projection", (*dialectExpr)(nil), func(n ast.Node) ast.Node { return &ast.SelectItem{Expr: n.(ast.ExprNode)} }},
		{"source", (*fakeTableSourceTable)(nil), func(n ast.Node) ast.Node {
			return &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: n.(ast.ResultSetNode)}}
		}},
		{"cte body", (*writableCTEStmt)(nil), func(n ast.Node) ast.Node {
			return &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{Query: n.(ast.StmtNode)}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := tc.new(tc.child)
			visited := 0
			if _, ok := root.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) { visited++; return n, false }}); ok || visited != 1 {
				t.Fatal("typed-nil child must stop before dispatch")
			}
			child := reflect.New(reflect.TypeOf(tc.child).Elem()).Interface().(ast.Node)
			for _, replacement := range []ast.Node{nil, tc.child, &ast.FieldList{}} {
				root = tc.new(child)
				if _, ok := root.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
					if n == child {
						return replacement, true
					}
					return n, true
				}}); ok {
					t.Fatalf("invalid child replacement %T must stop", replacement)
				}
			}
		})
	}
	if _, ok := (&ast.FieldList{Items: []*ast.SelectItem{nil}}).Accept(visitor{}); ok {
		t.Fatal("nil collection entry must stop")
	}
	if _, ok := (&ast.CommonTableExpression{}).Accept(visitor{}); ok {
		t.Fatal("missing required body must stop")
	}
	for _, n := range []ast.Node{&ast.Assignment{}, &ast.OrderItem{}, &ast.SelectItem{}, &ast.TableSource{}} {
		if got, ok := n.Accept(visitor{}); !ok || got != n {
			t.Fatalf("%T absent optional slot must remain a leaf", n)
		}
	}
}
