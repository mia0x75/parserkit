package ast_test

import (
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// typeExpr mirrors a dialect outer node that owns BaseTypeName.
type typeExpr struct {
	ast.BaseTypeName
}

var _ ast.TypeNode = (*typeExpr)(nil)

func (n *typeExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*typeExpr)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

func (n *typeExpr) Restore(ctx *format.RestoreCtx) error {
	return ctx.Err()
}

// TestTypeCoreFieldsStayIndependent proves that the type name core keeps
// its built-in keywords, user-defined name, modifiers and array bounds
// independently of its span.
func TestTypeCoreFieldsStayIndependent(t *testing.T) {
	mod := &dialectExpr{}
	mod.SetText("12")
	zero := &dialectExpr{}
	zero.SetText("0")
	base := &ast.BaseTypeName{
		Name:      ast.QualifiedName{Parts: []lex.Ident{{Name: "mytype", Raw: "mytype"}}},
		Modifiers: []ast.ExprNode{mod},
		ArrayBounds: []ast.ArrayBound{
			{Size: nil, Span: loc.Span{Start: 0, End: 2}},
			{Size: zero, Span: loc.Span{Start: 2, End: 5}},
		},
	}
	if base.Name.Parts[0].Name != "mytype" {
		t.Fatal("user-defined name was lost")
	}
	if len(base.Modifiers) != 1 || base.Modifiers[0] != ast.ExprNode(mod) {
		t.Fatal("modifier must keep its expression pointer")
	}
	if len(base.ArrayBounds) != 2 {
		t.Fatal("array bounds must keep two entries")
	}
	if base.ArrayBounds[0].Size != nil {
		t.Fatal("empty array bound must have a nil size")
	}
	if base.ArrayBounds[1].Size != ast.ExprNode(zero) {
		t.Fatal("explicit array bound must keep its size pointer")
	}
	if base.ArrayBounds[1].Span != (loc.Span{Start: 2, End: 5}) {
		t.Fatal("explicit array bound span was lost")
	}
	if base.ArrayBounds[1].Size.Text() != "0" {
		t.Fatalf("explicit array bound size text = %q, want \"0\"", base.ArrayBounds[1].Size.Text())
	}
	base.SetSpan(loc.Span{Start: 0, End: 10})
	if base.Span() != (loc.Span{Start: 0, End: 10}) {
		t.Fatal("type span was lost")
	}
}

// TestTypeNameAndKeywordsAreAlternativeSpellings checks that BaseTypeName
// supports both a user-defined name and built-in keywords as separate,
// legal examples; the outer node is responsible for enforcing mutual
// exclusion.
func TestTypeNameAndKeywordsAreAlternativeSpellings(t *testing.T) {
	byName := &ast.BaseTypeName{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "mytype", Raw: "mytype"}}}}
	if len(byName.Name.Parts) != 1 || byName.Name.Parts[0].Name != "mytype" {
		t.Fatal("user-defined name spelling was lost")
	}
	if len(byName.Keywords) != 0 {
		t.Fatal("user-defined name spelling must not carry keywords")
	}

	byKeywords := &ast.BaseTypeName{Keywords: []string{"DOUBLE", "PRECISION"}}
	if len(byKeywords.Keywords) != 2 || byKeywords.Keywords[0] != "DOUBLE" || byKeywords.Keywords[1] != "PRECISION" {
		t.Fatal("built-in keyword spelling was lost")
	}
	if len(byKeywords.Name.Parts) != 0 {
		t.Fatal("built-in keyword spelling must not carry a user-defined name")
	}
}

// TestTypeCoreAccessorReturnsSameBasePointer proves that the type core
// accessor hands back the very same embedded base pointer.
func TestTypeCoreAccessorReturnsSameBasePointer(t *testing.T) {
	ty := &typeExpr{BaseTypeName: ast.BaseTypeName{}}
	if ty.TypeCore() != &ty.BaseTypeName {
		t.Fatal("type core must return the embedded base pointer")
	}
}
