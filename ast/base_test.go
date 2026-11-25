package ast_test

import (
	"io"
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/loc"
)

type dialectExpr struct {
	ast.BaseExpr
	Child ast.ExprNode
}

func (n *dialectExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*dialectExpr)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.Child != nil {
		updated, ok := outer.Child.Accept(v)
		if !ok {
			return outer, false
		}
		child, ok := updated.(ast.ExprNode)
		if !ok {
			return outer, false
		}
		outer.Child = child
	}
	return v.Leave(outer)
}

func (n *dialectExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("null")
	return ctx.Err()
}

var _ ast.ExprNode = (*dialectExpr)(nil)

type visitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

func (v visitor) Enter(n ast.Node) (ast.Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v visitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func TestBaseNodeSpan(t *testing.T) {
	var n ast.BaseNode
	if n.Span() != loc.InvalidSpan() {
		t.Fatal("zero base node must have an unknown span")
	}
	n.SetSpan(loc.Span{})
	if n.Span() != (loc.Span{}) {
		t.Fatal("known empty range was lost")
	}
	n.SetText("text")
	if n.Text() != "text" || n.Span() != (loc.Span{}) {
		t.Fatal("text and span must remain independent")
	}
	n.SetSpan(loc.Span{Start: 2, End: 1})
	if n.Span() != loc.InvalidSpan() || n.Text() != "text" {
		t.Fatal("invalid span must become unknown without changing text")
	}
	for _, base := range []reflect.Type{reflect.TypeFor[*ast.BaseNode](), reflect.TypeFor[*ast.BaseExpr](), reflect.TypeFor[*ast.BaseQuery]()} {
		if _, found := base.MethodByName("Accept"); found {
			t.Fatalf("%v must not dispatch Accept", base)
		}
		if _, found := base.MethodByName("Restore"); found {
			t.Fatalf("%v must not dispatch Restore", base)
		}
	}
}

// fakeQueryExprOuter is a dialect expression that yields a result set, such
// as a parenthesized subquery: it must be usable wherever either marker is
// expected, but never as a statement.
type fakeQueryExprOuter struct{ ast.BaseQueryExpr }

var _ ast.ExprNode = (*fakeQueryExprOuter)(nil)
var _ ast.ResultSetNode = (*fakeQueryExprOuter)(nil)

func (n *fakeQueryExprOuter) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *fakeQueryExprOuter) Restore(ctx *format.RestoreCtx) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	ctx.WritePlain("(SELECT)")
	return ctx.Err()
}

func TestBaseQueryExprDualMarkerContract(t *testing.T) {
	// An outer node carrying BaseQueryExpr must satisfy both the expression
	// and result-set interfaces, while staying out of the statement/query
	// families; a bare base has neither marker satisfied through Node.
	outerType := reflect.TypeFor[*fakeQueryExprOuter]()
	if !outerType.Implements(reflect.TypeFor[ast.ExprNode]()) {
		t.Error("a BaseQueryExpr outer must satisfy ast.ExprNode")
	}
	if !outerType.Implements(reflect.TypeFor[ast.ResultSetNode]()) {
		t.Error("a BaseQueryExpr outer must satisfy ast.ResultSetNode")
	}
	if outerType.Implements(reflect.TypeFor[ast.StmtNode]()) {
		t.Error("a BaseQueryExpr outer must not satisfy ast.StmtNode")
	}
	if outerType.Implements(reflect.TypeFor[ast.QueryNode]()) {
		t.Error("a BaseQueryExpr outer must not satisfy ast.QueryNode")
	}
	for _, base := range []reflect.Type{reflect.TypeFor[*ast.BaseQueryExpr](), reflect.TypeFor[*ast.BaseResultSet]()} {
		if _, found := base.MethodByName("Accept"); found {
			t.Fatalf("%v must not dispatch Accept", base)
		}
		if _, found := base.MethodByName("Restore"); found {
			t.Fatalf("%v must not dispatch Restore", base)
		}
	}
	if reflect.TypeFor[*ast.BaseQueryExpr]().Implements(reflect.TypeFor[ast.ExprNode]()) {
		t.Error("a bare BaseQueryExpr must not satisfy ast.ExprNode")
	}
	if reflect.TypeFor[*ast.BaseQueryExpr]().Implements(reflect.TypeFor[ast.ResultSetNode]()) {
		t.Error("a bare BaseQueryExpr must not satisfy ast.ResultSetNode")
	}
}

func TestOuterVisitorIdentity(t *testing.T) {
	child := &dialectExpr{}
	root := &dialectExpr{Child: child}
	var entered, left []ast.Node
	updated, ok := root.Accept(visitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			entered = append(entered, n)
			return n, false
		},
		leave: func(n ast.Node) (ast.Node, bool) {
			left = append(left, n)
			return n, true
		},
	})
	if !ok || updated != root || len(entered) != 2 || entered[0] != root || entered[1] != child ||
		len(left) != 2 || left[0] != child || left[1] != root {
		t.Fatal("visitor lost outer identity or extension child order")
	}
}

func TestVisitorSkipReplaceAndStop(t *testing.T) {
	child := &dialectExpr{}
	root := &dialectExpr{Child: child}
	replacement := &dialectExpr{}
	updated, ok := root.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
		return replacement, true
	}})
	if !ok || updated != replacement {
		t.Fatal("Enter replacement or skip was lost")
	}
	if _, ok := root.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
		return n, n != child
	}}); ok {
		t.Fatal("child abort was not propagated")
	}
	if err := root.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags)); err != nil {
		t.Fatal(err)
	}
}
