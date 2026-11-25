package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"
)

func TestBaseJoinExprContract(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeFor[*ast.BaseJoinExpr]()
	for _, method := range []string{"Accept", "Restore"} {
		if _, ok := typ.MethodByName(method); ok {
			t.Fatalf("base dispatches %s", method)
		}
	}
	var n ast.BaseJoinExpr
	if n.JoinExprCore() != &n || n.Type != ast.JoinTypeUnknown || n.Span() != loc.InvalidSpan() {
		t.Fatal("invalid zero value or copied core")
	}
	for _, span := range []loc.Span{{Start: 0, End: 0}, {Start: 3, End: 12}, loc.InvalidSpan()} {
		n.SetSpan(span)
		if n.Span() != span {
			t.Fatalf("got span %v, want %v", n.Span(), span)
		}
	}
}
