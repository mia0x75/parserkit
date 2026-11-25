package ast_test

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/lex"
)

func cteQuery() *fakeTableSourceSelect { return &fakeTableSourceSelect{} }

func TestCommonTableExpressionRestore(t *testing.T) {
	plain := &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{
		Name: lex.Ident{Name: "c"}, Query: cteQuery(),
	}}
	var out bytes.Buffer
	if err := plain.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "c AS (SELECT)" {
		t.Fatalf("plain CTE restore = %q, want c AS (SELECT)", got)
	}

	columned := &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{
		Name: lex.Ident{Name: "c"}, Columns: []lex.Ident{{Name: "a"}, {Name: "b"}}, Query: cteQuery(),
	}}
	out.Reset()
	if err := columned.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "c (a, b) AS (SELECT)" {
		t.Fatalf("columned CTE restore = %q, want c (a, b) AS (SELECT)", got)
	}
}

func TestCommonTableExpressionRestoreNilQueryFails(t *testing.T) {
	err := (&ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{
		Name: lex.Ident{Name: "c"},
	}}).Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
	if err == nil || !strings.Contains(err.Error(), "missing query") {
		t.Fatalf("nil query must fail with missing query: %v", err)
	}
}

func TestCommonTableExpressionAcceptVisitAndReplace(t *testing.T) {
	inner := cteQuery()
	outer := &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{
		Name: lex.Ident{Name: "c"}, Query: inner,
	}}

	var entered, left []ast.Node
	updated, ok := outer.Accept(visitor{
		enter: func(n ast.Node) (ast.Node, bool) { entered = append(entered, n); return n, false },
		leave: func(n ast.Node) (ast.Node, bool) { left = append(left, n); return n, true },
	})
	if !ok || updated != outer || len(entered) != 2 || entered[0] != outer || entered[1] != inner ||
		len(left) != 2 || left[0] != inner || left[1] != outer {
		t.Fatal("CTE visitor lost outer identity or child order")
	}

	// Child replacement must retain the CTE's statement role.
	replacement := cteQuery()
	_, ok = outer.Accept(visitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isQuery := n.(*fakeTableSourceSelect); isQuery {
				return replacement, true
			}
			return n, true
		},
	})
	if !ok || outer.Query != replacement {
		t.Fatal("inner statement replacement did not reach the CTE")
	}

	// An expression must not overwrite a statement body.
	invalid := &dialectExpr{}
	_, ok = outer.Accept(visitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isQuery := n.(*fakeTableSourceSelect); isQuery {
				return invalid, true
			}
			return n, true
		},
	})
	if ok || outer.Query != replacement {
		t.Fatalf("non-statement replacement must not overwrite the CTE Query: ok=%v", ok)
	}
}

func TestCommonTableExpressionAcceptNilQueryStops(t *testing.T) {
	// A CTE whose Query is nil must stop at itself without entering children.
	outer := &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{
		Name: lex.Ident{Name: "c"},
	}}
	entered := 0
	updated, ok := outer.Accept(visitor{
		enter: func(n ast.Node) (ast.Node, bool) { entered++; return n, false },
		leave: func(n ast.Node) (ast.Node, bool) { return n, true },
	})
	// The missing query is a stop signal: the traversal reports it without
	// descending into children.
	if ok || entered != 1 || updated != outer {
		t.Fatalf("nil query CTE must enter only itself and stop: entered=%d ok=%v updated=%v", entered, ok, updated)
	}
}

func TestCommonTableExpressionAcceptVisitorReject(t *testing.T) {
	inner := cteQuery()
	outer := &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{
		Name: lex.Ident{Name: "c"}, Query: inner,
	}}
	// Enter returns something that is not a *CommonTableExpression: the CTE
	// must return it and stop.
	invalid := &dialectExpr{}
	updated, ok := outer.Accept(visitor{enter: func(ast.Node) (ast.Node, bool) { return invalid, false }})
	if ok || updated != invalid {
		t.Fatalf("invalid Enter replacement not returned: updated=%v ok=%v", updated, ok)
	}

	// Enter that replaces with a CTE but skips children must leave it.
	skipCte := &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{
		Name: lex.Ident{Name: "d"}, Query: cteQuery(),
	}}
	updated, ok = outer.Accept(visitor{enter: func(ast.Node) (ast.Node, bool) { return skipCte, true }})
	if !ok || updated != skipCte {
		t.Fatalf("skip Enter replacement not left: updated=%v ok=%v", updated, ok)
	}

	// A nil Enter replacement stops.
	updated, ok = outer.Accept(visitor{enter: func(ast.Node) (ast.Node, bool) { return nil, false }})
	if ok || updated != nil {
		t.Fatalf("nil Enter replacement must stop: updated=%v ok=%v", updated, ok)
	}

}

type writableCTEStmt struct{ ast.BaseDML }

func (n *writableCTEStmt) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *writableCTEStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain("UPDATE mock_table SET mock_column=1")
	return ctx.Err()
}

var _ ast.DMLNode = (*writableCTEStmt)(nil)

func TestCTECoreContract(t *testing.T) {
	t.Parallel()
	base := &ast.BaseCommonTableExpression{}
	cte := &ast.CommonTableExpression{}
	if base.CTECore() != base || cte.CTECore() != &cte.BaseCommonTableExpression ||
		(*ast.CommonTableExpression)(nil).CTECore() != nil {
		t.Fatal("CTE core must preserve storage identity and nil safety")
	}
	if reflect.TypeFor[*ast.BaseCommonTableExpression]().Implements(reflect.TypeFor[ast.Node]()) {
		t.Fatal("CTE base must not dispatch")
	}
	query, _ := reflect.TypeFor[ast.BaseCommonTableExpression]().FieldByName("Query")
	ctes, _ := reflect.TypeFor[ast.BaseWithClause]().FieldByName("CTEs")
	if query.Type != reflect.TypeFor[ast.StmtNode]() || ctes.Type != reflect.TypeFor[[]ast.CTENode]() {
		t.Fatal("CTEs must retain dialect roles and statement bodies")
	}
}

func TestCTEStatementReplacement(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		replacement ast.Node
		wantOK      bool
	}{
		{"query", cteQuery(), true},
		{"writable", &writableCTEStmt{}, true},
		{"expression", &dialectExpr{}, false},
		{"nil", nil, false},
		{"typed nil query", (*fakeTableSourceSelect)(nil), false},
		{"typed nil writable", (*writableCTEStmt)(nil), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := cteQuery()
			cte := &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{
				Name: lex.Ident{Name: "c"}, Query: body,
			}}
			_, ok := cte.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
				if n == body {
					return tc.replacement, true
				}
				return n, true
			}})
			if ok != tc.wantOK || !ok && cte.Query != body || ok && cte.Query != tc.replacement {
				t.Fatalf("replacement %T: ok=%v, query=%T", tc.replacement, ok, cte.Query)
			}
			if tc.name == "writable" {
				var out bytes.Buffer
				err := cte.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags))
				if err != nil || out.String() != "c AS (UPDATE mock_table SET mock_column=1)" {
					t.Fatalf("writable CTE = %q, err=%v", out.String(), err)
				}
			}
		})
	}
}

func TestCTENilBodiesAndReceiver(t *testing.T) {
	t.Parallel()
	called := false
	v := visitor{enter: func(n ast.Node) (ast.Node, bool) { called = true; return n, false }}
	if n, ok := (*ast.CommonTableExpression)(nil).Accept(v); n != nil || ok || called {
		t.Fatal("nil CTE must stop without visiting")
	}
	for _, tc := range []struct {
		name string
		body ast.StmtNode
	}{
		{"nil", nil},
		{"query", (*fakeTableSourceSelect)(nil)},
		{"writable", (*writableCTEStmt)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cte := &ast.CommonTableExpression{BaseCommonTableExpression: ast.BaseCommonTableExpression{Query: tc.body}}
			var out bytes.Buffer
			if err := cte.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)); err == nil || out.Len() != 0 {
				t.Fatalf("missing body must fail before writing: %q, err=%v", out.String(), err)
			}
			entered, left := 0, 0
			_, ok := cte.Accept(visitor{
				enter: func(n ast.Node) (ast.Node, bool) { entered++; return n, false },
				leave: func(n ast.Node) (ast.Node, bool) { left++; return n, true },
			})
			if ok || entered != 1 || left != 0 {
				t.Fatalf("missing body traversal: ok=%v enter=%d leave=%d", ok, entered, left)
			}
		})
	}
}
