package ast_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/loc"
)

// p43UpdateTable is a fake result set so an outer statement can hold a Table.
type p43UpdateTable struct{ ast.BaseResultSet }

func (n *p43UpdateTable) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *p43UpdateTable) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("t1")
	return ctx.Err()
}

// p43UpdateOuter is a dialect update outer: it embeds the shared base by value
// and keeps dialect-only state on the outer node.
type p43UpdateOuter struct {
	ast.BaseUpdateStmt
	Extra bool
}

func (n *p43UpdateOuter) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *p43UpdateOuter) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("UPDATE")
	return ctx.Err()
}

var _ ast.DMLNode = (*p43UpdateOuter)(nil)

// p43DeleteOuter is a minimal dialect delete outer.
type p43DeleteOuter struct{ ast.BaseDeleteStmt }

func (n *p43DeleteOuter) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *p43DeleteOuter) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DELETE")
	return ctx.Err()
}

var _ ast.DMLNode = (*p43DeleteOuter)(nil)

// p43ClauseOuter is a clause outer that embeds the shared order base by value
// and adds a dialect-only flag.
type p43ClauseOuter struct {
	ast.BaseOrderByClause
	Extra bool
}

func (n *p43ClauseOuter) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *p43ClauseOuter) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("ORDER")
	return ctx.Err()
}

var _ ast.OrderByClauseNode = (*p43ClauseOuter)(nil)

// TestP43BaseUpdateStmtContract pins the exact seven-field shape of the shared
// update base and its lack of node dispatch.
func TestP43BaseUpdateStmtContract(t *testing.T) {
	typ := reflect.TypeFor[*ast.BaseUpdateStmt]()
	if typ.Implements(reflect.TypeFor[ast.Node]()) {
		t.Error("a bare update base must not satisfy ast.Node")
	}
	fields := []struct {
		name string
		want reflect.Type
	}{
		{"BaseDML", reflect.TypeFor[ast.BaseDML]()},
		{"Table", reflect.TypeFor[ast.ResultSetNode]()},
		{"Assignments", reflect.TypeFor[[]*ast.Assignment]()},
		{"Where", reflect.TypeFor[ast.ExprNode]()},
		{"OrderBy", reflect.TypeFor[ast.OrderByClauseNode]()},
		{"Limit", reflect.TypeFor[ast.LimitNode]()},
		{"Returning", reflect.TypeFor[*ast.FieldList]()},
	}
	if typ.Elem().NumField() != len(fields) {
		t.Fatalf("BaseUpdateStmt has %d fields, want %d", typ.Elem().NumField(), len(fields))
	}
	for i, field := range fields {
		got := typ.Elem().Field(i)
		// Index 0 is the anonymous by-value BaseDML embed; the rest are named.
		if got.Name != field.name || got.Type != field.want || (i == 0) != got.Anonymous {
			t.Fatalf("BaseUpdateStmt field %d = %s %s, want %s %s", i, got.Type, got.Name, field.want, field.name)
		}
	}
}

// TestP43BaseDeleteStmtContract pins the exact seven-field shape of the shared
// delete base: single target in Table, independent source in Using.
func TestP43BaseDeleteStmtContract(t *testing.T) {
	typ := reflect.TypeFor[*ast.BaseDeleteStmt]()
	if typ.Implements(reflect.TypeFor[ast.Node]()) {
		t.Error("a bare delete base must not satisfy ast.Node")
	}
	fields := []struct {
		name string
		want reflect.Type
	}{
		{"BaseDML", reflect.TypeFor[ast.BaseDML]()},
		{"Table", reflect.TypeFor[ast.ResultSetNode]()},
		{"Using", reflect.TypeFor[ast.ResultSetNode]()},
		{"Where", reflect.TypeFor[ast.ExprNode]()},
		{"OrderBy", reflect.TypeFor[ast.OrderByClauseNode]()},
		{"Limit", reflect.TypeFor[ast.LimitNode]()},
		{"Returning", reflect.TypeFor[*ast.FieldList]()},
	}
	if typ.Elem().NumField() != len(fields) {
		t.Fatalf("BaseDeleteStmt has %d fields, want %d", typ.Elem().NumField(), len(fields))
	}
	for i, field := range fields {
		got := typ.Elem().Field(i)
		if got.Name != field.name || got.Type != field.want || (i == 0) != got.Anonymous {
			t.Fatalf("BaseDeleteStmt field %d = %s %s, want %s %s", i, got.Type, got.Name, field.want, field.name)
		}
	}
}

// TestP43BaseStmtNoDispatch pins that neither shared base may acquire node
// dispatch, which the dialect outer must own.
func TestP43BaseStmtNoDispatch(t *testing.T) {
	for _, base := range []struct {
		name string
		typ  reflect.Type
	}{
		{"BaseUpdateStmt", reflect.TypeFor[*ast.BaseUpdateStmt]()},
		{"BaseDeleteStmt", reflect.TypeFor[*ast.BaseDeleteStmt]()},
	} {
		t.Run(base.name, func(t *testing.T) {
			for _, method := range []string{"Accept", "Restore"} {
				if _, found := base.typ.MethodByName(method); found {
					t.Errorf("%s must not dispatch %s", base.name, method)
				}
			}
		})
	}
}

// TestP43StmtSpanStates shows a zero-value base keeps an unknown span while a
// known empty [0,0) range and the source text remain independent.
func TestP43StmtSpanStates(t *testing.T) {
	update := p43UpdateOuter{}
	if update.Span() != loc.InvalidSpan() || update.Table != nil || update.Assignments != nil ||
		update.Where != nil || update.OrderBy != nil || update.Limit != nil || update.Returning != nil {
		t.Fatal("a zero update must keep an unknown span and absent slots")
	}
	update.SetSpan(loc.Span{})
	update.SetText("UPDATE t1 SET x = 1")
	if update.Span() != (loc.Span{}) || update.Text() != "UPDATE t1 SET x = 1" {
		t.Fatal("a known empty span and text must be kept independent")
	}
	update.SetSpan(loc.Span{Start: 2, End: 1})
	if update.Span() != loc.InvalidSpan() || update.Text() != "UPDATE t1 SET x = 1" {
		t.Fatal("an invalid range must fall back without touching text")
	}

	del := p43DeleteOuter{}
	if del.Span() != loc.InvalidSpan() || del.Table != nil || del.Using != nil {
		t.Fatal("a zero delete must keep an unknown span and absent relations")
	}
	del.SetSpan(loc.Span{})
	del.SetText("DELETE FROM t1")
	if del.Span() != (loc.Span{}) || del.Text() != "DELETE FROM t1" {
		t.Fatal("a known empty span and text must be kept independent")
	}
}

// TestP43ClauseCoreIdentity pins that the strong role interfaces resolve the
// shared data through their Core accessors, that a nil base receiver is safe,
// and that a by-value embedded base never satisfies ast.Node.
func TestP43ClauseCoreIdentity(t *testing.T) {
	if reflect.TypeFor[*ast.BaseOrderByClause]().Implements(reflect.TypeFor[ast.Node]()) {
		t.Error("a bare order-by base must not satisfy ast.Node")
	}
	if reflect.TypeFor[*ast.BaseLimit]().Implements(reflect.TypeFor[ast.Node]()) {
		t.Error("a bare limit base must not satisfy ast.Node")
	}
	var clause *ast.BaseOrderByClause
	if clause.OrderByClauseCore() != nil {
		t.Fatal("a nil order-by base must return a nil core")
	}
	var limit *ast.BaseLimit
	if limit.LimitCore() != nil {
		t.Fatal("a nil limit base must return a nil core")
	}
	for _, base := range []struct {
		name string
		typ  reflect.Type
	}{
		{"BaseOrderByClause", reflect.TypeFor[*ast.BaseOrderByClause]()},
		{"BaseLimit", reflect.TypeFor[*ast.BaseLimit]()},
	} {
		for _, method := range []string{"Accept", "Restore"} {
			if _, found := base.typ.MethodByName(method); found {
				t.Errorf("%s must not dispatch %s", base.name, method)
			}
		}
	}

	outer := &p43ClauseOuter{
		BaseOrderByClause: ast.BaseOrderByClause{Items: []ast.OrderItemNode{&ast.OrderItem{}}},
		Extra:             true,
	}
	if outer.OrderByClauseCore() != &outer.BaseOrderByClause {
		t.Fatal("an embedded clause outer must resolve its shared core in place")
	}
	if !outer.Extra {
		t.Fatal("the dialect-only extension must stay on the outer node, not the shared base")
	}
}

// TestP43StmtOuterDispatch shows a dialect outer keeps its own visitor
// identity, holds shared slot data by the strong roles, and restores.
func TestP43StmtOuterDispatch(t *testing.T) {
	table := &p43UpdateTable{}
	assign := &ast.Assignment{}
	outer := &p43UpdateOuter{
		BaseUpdateStmt: ast.BaseUpdateStmt{
			Table:       table,
			Assignments: []*ast.Assignment{assign},
		},
		Extra: true,
	}
	if outer.Table != ast.ResultSetNode(table) || outer.Assignments[0] != assign {
		t.Fatal("shared slot data must keep the held values on the outer")
	}
	visited := false
	updated, ok := outer.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
		visited = n == ast.Node(outer)
		return n, false
	}})
	if !ok || !visited || updated != ast.Node(outer) {
		t.Fatal("an outer update must keep its own visitor identity")
	}
	if !outer.Extra {
		t.Fatal("the dialect-only extension must survive traversal on the outer node")
	}
	var out bytes.Buffer
	if err := outer.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)); err != nil {
		t.Fatal(err)
	}
	if out.String() != "UPDATE" {
		t.Fatal("an outer update must restore through its own dispatch")
	}

	del := &p43DeleteOuter{}
	if del.Table != nil || del.Using != nil {
		t.Fatal("a zero delete must keep target and source absent")
	}
	visited = false
	updated, ok = del.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
		visited = n == ast.Node(del)
		return n, false
	}})
	if !ok || !visited || updated != ast.Node(del) {
		t.Fatal("an outer delete must keep its own visitor identity")
	}
}
