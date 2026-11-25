package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
)

func TestBaseSelectStmtIsPureStorage(t *testing.T) {
	base := reflect.TypeFor[*ast.BaseSelectStmt]()
	if base.Implements(reflect.TypeFor[ast.Node]()) {
		t.Fatal("BaseSelect must not implement Node")
	}
	want := []struct {
		name      string
		typ       reflect.Type
		anonymous bool
	}{
		{"BaseQuery", reflect.TypeFor[ast.BaseQuery](), true},
		{"Distinct", reflect.TypeFor[bool](), false},
		{"Fields", reflect.TypeFor[*ast.FieldList](), false},
		{"From", reflect.TypeFor[ast.FromClauseNode](), false},
		{"Where", reflect.TypeFor[ast.ExprNode](), false},
		{"GroupBy", reflect.TypeFor[ast.GroupByClauseNode](), false},
		{"Having", reflect.TypeFor[ast.ExprNode](), false},
		{"WindowSpecs", reflect.TypeFor[[]ast.WindowSpecNode](), false},
		{"OrderBy", reflect.TypeFor[ast.OrderByClauseNode](), false},
		{"Limit", reflect.TypeFor[ast.LimitNode](), false},
		{"With", reflect.TypeFor[ast.WithClauseNode](), false},
	}
	if base.Elem().NumField() != len(want) {
		t.Fatalf("BaseSelectStmt fields = %d, want %d", base.Elem().NumField(), len(want))
	}
	for i, field := range want {
		got := base.Elem().Field(i)
		if got.Name != field.name || got.Type != field.typ || got.Anonymous != field.anonymous {
			t.Fatalf("BaseSelectStmt field %d = %s %s, want %s %s", i, got.Type, got.Name, field.typ, field.name)
		}
	}
	for _, method := range []string{"Accept", "Restore"} {
		if _, ok := base.MethodByName(method); ok {
			t.Fatalf("BaseSelect must not define %s", method)
		}
	}
}

func TestSelectCoreAccessorsPreserveIdentity(t *testing.T) {
	from := &ast.BaseFromClause{}
	group := &ast.BaseGroupByClause{}
	with := &ast.BaseWithClause{}
	window := &ast.BaseWindowSpec{}
	if from.FromClauseCore() != from ||
		group.GroupByClauseCore() != group || with.WithClauseCore() != with ||
		window.WindowSpecCore() != window {
		t.Fatal("core accessor did not preserve identity")
	}
	for _, typ := range []reflect.Type{
		reflect.TypeFor[*ast.BaseFromClause](),
		reflect.TypeFor[*ast.BaseGroupByClause](),
		reflect.TypeFor[*ast.BaseWithClause](),
		reflect.TypeFor[*ast.BaseWindowSpec](),
	} {
		if typ.Implements(reflect.TypeFor[ast.Node]()) {
			t.Errorf("%s must not implement Node", typ)
		}
		for _, method := range []string{"Accept", "Restore"} {
			if _, ok := typ.MethodByName(method); ok {
				t.Errorf("%s must not define %s", typ, method)
			}
		}
	}
}

func TestBaseFromClauseAcceptsSingleSource(t *testing.T) {
	t.Parallel()
	source := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceTable{}}}
	clause := &ast.BaseFromClause{Table: source}
	if clause.Table != source || clause.FromClauseCore() != clause {
		t.Fatal("FROM must retain its source without a synthetic join")
	}
	field, ok := reflect.TypeFor[ast.BaseFromClause]().FieldByName("Table")
	if !ok || field.Type != reflect.TypeFor[ast.ResultSetNode]() {
		t.Fatal("FROM must admit any result-set node")
	}
}
