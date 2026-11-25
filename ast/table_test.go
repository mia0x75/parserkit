package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
)

func TestBaseTableStmtContract(t *testing.T) {
	typ := reflect.TypeFor[*ast.BaseTableStmt]()
	if typ.Implements(reflect.TypeFor[ast.Node]()) {
		t.Fatal("a bare TABLE base must not satisfy ast.Node")
	}
	want := []struct {
		name      string
		typ       reflect.Type
		anonymous bool
	}{
		{"BaseQuery", reflect.TypeFor[ast.BaseQuery](), true},
		{"Table", reflect.TypeFor[ast.TableRefNode](), false},
	}
	if typ.Elem().NumField() != len(want) {
		t.Fatalf("BaseTableStmt fields = %d, want %d", typ.Elem().NumField(), len(want))
	}
	for i, field := range want {
		got := typ.Elem().Field(i)
		if got.Name != field.name || got.Type != field.typ || got.Anonymous != field.anonymous {
			t.Fatalf("BaseTableStmt field %d = %s %s, want %s %s", i, got.Type, got.Name, field.typ, field.name)
		}
	}
	for _, method := range []string{"Accept", "Restore"} {
		if _, ok := typ.MethodByName(method); ok {
			t.Fatalf("BaseTableStmt must not define %s", method)
		}
	}
}
