package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"
)

func TestBaseLimitSyntaxContract(t *testing.T) {
	base := reflect.TypeFor[*ast.BaseLimit]()
	for _, name := range []string{"Accept", "Restore"} {
		if _, found := base.MethodByName(name); found {
			t.Fatalf("BaseLimit must leave %s to the dialect outer", name)
		}
	}
	if base.Implements(reflect.TypeFor[ast.Node]()) {
		t.Fatal("BaseLimit must not dispatch as a complete node")
	}
	var limit ast.BaseLimit
	if limit.Span() != loc.InvalidSpan() || limit.Count != nil || limit.Offset != nil {
		t.Fatal("zero pagination syntax must have no expressions and an unknown span")
	}
	count, offset := &dialectExpr{}, &dialectExpr{}
	limit.Count, limit.Offset = count, offset
	span := loc.Span{Start: 3, End: 17}
	limit.SetSpan(span)
	limit.SetText("LIMIT 2 OFFSET 1")
	if limit.Count != count || limit.Offset != offset || limit.Span() != span || limit.Text() != "LIMIT 2 OFFSET 1" {
		t.Fatal("pagination expressions, source span and source text must remain independent")
	}
}
