package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func TestConstraintCoreContract(t *testing.T) {
	base := &ast.BaseConstraint{}
	if base.ConstraintCore() != base {
		t.Fatal("core identity changed")
	}
	for _, method := range []string{"Accept", "Restore"} {
		if _, ok := reflect.TypeFor[*ast.BaseConstraint]().MethodByName(method); ok {
			t.Fatalf("shared base dispatches %s", method)
		}
	}
	for _, slot := range []struct {
		base reflect.Type
		name string
		want reflect.Type
	}{
		{reflect.TypeFor[ast.BaseConstraint](), "Name", reflect.TypeFor[lex.Ident]()},
		{reflect.TypeFor[ast.BaseConstraint](), "Keys", reflect.TypeFor[[]ast.IndexKeyPartNode]()},
		{reflect.TypeFor[ast.BaseConstraint](), "Expr", reflect.TypeFor[ast.ExprNode]()},
		{reflect.TypeFor[ast.BaseConstraint](), "Refer", reflect.TypeFor[ast.ReferenceDefNode]()},
		{reflect.TypeFor[ast.BaseCreateTableStmt](), "Constraints", reflect.TypeFor[[]ast.ConstraintNode]()},
	} {
		if field, ok := slot.base.FieldByName(slot.name); !ok || field.Type != slot.want {
			t.Fatalf("%v.%s must be %v", slot.base, slot.name, slot.want)
		}
	}
	if _, ok := reflect.TypeFor[ast.BaseConstraint]().FieldByName("Kind"); ok {
		t.Fatal("kind is mapped by the dialect, not stored twice")
	}
}
