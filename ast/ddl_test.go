package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
)

func TestDDLCoreSyntaxRoles(t *testing.T) {
	for _, base := range []reflect.Type{reflect.TypeFor[*ast.BaseColumnDef](), reflect.TypeFor[*ast.BaseCreateTableStmt]()} {
		for _, method := range []string{"Accept", "Restore"} {
			if _, ok := base.MethodByName(method); ok {
				t.Errorf("%s must not implement %s", base, method)
			}
		}
	}
	column := &ast.BaseColumnDef{}
	if column.ColumnDefCore() != column {
		t.Fatal("column core did not preserve identity")
	}
	for _, slot := range []struct {
		base reflect.Type
		name string
		want reflect.Type
	}{
		{reflect.TypeFor[ast.BaseCreateTableStmt](), "Table", reflect.TypeFor[ast.TableRefNode]()},
		{reflect.TypeFor[ast.BaseCreateTableStmt](), "Columns", reflect.TypeFor[[]ast.ColumnDefNode]()},
		{reflect.TypeFor[ast.BaseCreateTableStmt](), "Query", reflect.TypeFor[ast.QueryNode]()},
		{reflect.TypeFor[ast.BaseColumnDef](), "Type", reflect.TypeFor[ast.TypeNode]()},
	} {
		if field, ok := slot.base.FieldByName(slot.name); !ok || field.Type != slot.want {
			t.Errorf("%v.%s must have type %v", slot.base, slot.name, slot.want)
		}
	}
}
