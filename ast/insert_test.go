package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/loc"
)

type dialectInsert struct{ ast.BaseInsertStmt }

func (n *dialectInsert) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *dialectInsert) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("INSERT")
	return ctx.Err()
}

var _ ast.DMLNode = (*dialectInsert)(nil)

func TestBaseInsertStmtContract(t *testing.T) {
	baseType := reflect.TypeFor[*ast.BaseInsertStmt]()
	for _, method := range []string{"Accept", "Restore"} {
		if _, found := baseType.MethodByName(method); found {
			t.Errorf("BaseInsertStmt must not dispatch %s", method)
		}
	}
	if baseType.Implements(reflect.TypeFor[ast.Node]()) {
		t.Fatal("a bare insertion base must not be a node")
	}
	outerType := reflect.TypeFor[*dialectInsert]()
	if outerType.Implements(reflect.TypeFor[ast.QueryNode]()) || outerType.Implements(reflect.TypeFor[ast.ExprNode]()) {
		t.Fatal("an insertion must not acquire query or expression markers")
	}
	fields := []struct {
		name string
		want reflect.Type
	}{
		{"Table", reflect.TypeFor[ast.TableRefNode]()},
		{"Columns", reflect.TypeFor[[]ast.ExprNode]()},
		{"Values", reflect.TypeFor[[][]ast.ExprNode]()},
		{"Query", reflect.TypeFor[ast.QueryNode]()},
		{"Returning", reflect.TypeFor[*ast.FieldList]()},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			got, ok := baseType.Elem().FieldByName(field.name)
			if !ok || got.Type != field.want {
				t.Fatalf("field %s = %v, want %v", field.name, got.Type, field.want)
			}
		})
	}
}

func TestBaseInsertStmtKeepsPresenceAndSpan(t *testing.T) {
	var n dialectInsert
	if n.Span() != loc.InvalidSpan() || n.Columns != nil || n.Values != nil || n.Query != nil {
		t.Fatal("zero insertion must preserve unknown span and absent sources")
	}
	n.Columns = []ast.ExprNode{}
	n.Values = [][]ast.ExprNode{}
	n.SetSpan(loc.Span{})
	n.SetText("INSERT")
	if n.Columns == nil || n.Values == nil || n.Span() != (loc.Span{}) || n.Text() != "INSERT" {
		t.Fatal("explicit empty lists, known empty span or text were lost")
	}
	visited := false
	result, ok := n.Accept(visitor{enter: func(node ast.Node) (ast.Node, bool) {
		visited = node == &n
		return node, false
	}})
	if !ok || !visited || result != &n {
		t.Fatal("outer insertion identity was lost")
	}
}
