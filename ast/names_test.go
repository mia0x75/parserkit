package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// nameBases are the public base types that must stay free of node dispatch.
var nameBases = []reflect.Type{
	reflect.TypeFor[*ast.BaseNode](),
	reflect.TypeFor[*ast.BaseExpr](),
	reflect.TypeFor[*ast.BaseQueryExpr](),
	reflect.TypeFor[*ast.BaseStmt](),
	reflect.TypeFor[*ast.BaseDML](),
	reflect.TypeFor[*ast.BaseDDL](),
	reflect.TypeFor[*ast.BaseResultSet](),
	reflect.TypeFor[*ast.BaseQuery](),
	reflect.TypeFor[*ast.BaseColumnRef](),
	reflect.TypeFor[*ast.BaseTableRef](),
	reflect.TypeFor[*ast.BaseTypeName](),
}

// TestNoBaseTypeDispatchesNodes pins the shared contract that every public
// base type keeps its span and text accessors but never Accept or Restore.
func TestNoBaseTypeDispatchesNodes(t *testing.T) {
	for _, base := range nameBases {
		if _, found := base.MethodByName("Accept"); found {
			t.Errorf("%v must not dispatch Accept", base)
		}
		if _, found := base.MethodByName("Restore"); found {
			t.Errorf("%v must not dispatch Restore", base)
		}
	}
}

// TestQualifiedNameKeepsPartStyle proves that a qualified name keeps the dot
// segmentation, decoded names, raw spellings, quote styles and span.
func TestQualifiedNameKeepsPartStyle(t *testing.T) {
	const input = "`a.b`.\"tab\""
	want := []lex.Ident{
		{Name: "a.b", Raw: "`a.b`", Span: loc.Span{Start: 0, End: 5}, Quote: lex.QuoteBacktick},
		{Name: "tab", Raw: "\"tab\"", Span: loc.Span{Start: 6, End: 11}, Quote: lex.QuoteDouble},
	}
	qn := ast.QualifiedName{Parts: want, Span: loc.Span{Start: 0, End: 11}}
	if len(qn.Parts) != 2 {
		t.Fatalf("qualified name must keep exactly two parts, got %d", len(qn.Parts))
	}
	for i, got := range qn.Parts {
		if got.Name != want[i].Name || got.Raw != want[i].Raw || got.Quote != want[i].Quote || got.Span != want[i].Span {
			t.Fatalf("part %d lost style: got %+v want %+v", i, got, want[i])
		}
		gotSpan, err := got.Span.Slice(input)
		if err != nil {
			t.Fatalf("part %d span slice: %v", i, err)
		}
		if gotSpan != got.Raw {
			t.Fatalf("part %d span slice %q does not match raw %q", i, gotSpan, got.Raw)
		}
	}
	if qn.Span != (loc.Span{Start: 0, End: 11}) {
		t.Fatalf("qualified span changed: got %v", qn.Span)
	}
}

// columnRefExpr mirrors a dialect outer node that owns BaseColumnRef.
type columnRefExpr struct {
	ast.BaseColumnRef
}

var _ ast.ColumnRefNode = (*columnRefExpr)(nil)

func (n *columnRefExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*columnRefExpr)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

func (n *columnRefExpr) Restore(ctx *format.RestoreCtx) error {
	return ctx.Err()
}

// TestColumnRefCoreReturnsSameBasePointer proves that the outer column ref
// core accessor hands back the very same embedded base pointer.
func TestColumnRefCoreReturnsSameBasePointer(t *testing.T) {
	ref := &columnRefExpr{BaseColumnRef: ast.BaseColumnRef{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "col", Raw: "col"}}}}}
	if ref.ColumnRefCore() != &ref.BaseColumnRef {
		t.Fatal("column ref core must return the embedded base pointer")
	}
	if len(ref.Name.Parts) != 1 || ref.Name.Parts[0].Name != "col" {
		t.Fatal("name slot was lost")
	}
}

// tableRefExpr mirrors a dialect outer node that owns BaseTableRef.
type tableRefExpr struct {
	ast.BaseTableRef
}

var _ ast.TableRefNode = (*tableRefExpr)(nil)

func (n *tableRefExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*tableRefExpr)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

func (n *tableRefExpr) Restore(ctx *format.RestoreCtx) error {
	return ctx.Err()
}

// TestTableRefCoreReturnsSameBasePointer proves that the outer table ref
// core accessor hands back the very same embedded base pointer.
func TestTableRefCoreReturnsSameBasePointer(t *testing.T) {
	ref := &tableRefExpr{BaseTableRef: ast.BaseTableRef{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "tbl", Raw: "tbl"}}}}}
	if ref.TableRefCore() != &ref.BaseTableRef {
		t.Fatal("table ref core must return the embedded base pointer")
	}
	if len(ref.Name.Parts) != 1 || ref.Name.Parts[0].Name != "tbl" {
		t.Fatal("name slot was lost")
	}
}

// TestColumnRefCoreVisitsThroughOuterDispatch shows that the outer column
// ref node visits through its own dispatch and keeps the embedded base.
func TestColumnRefCoreVisitsThroughOuterDispatch(t *testing.T) {
	root := &columnRefExpr{BaseColumnRef: ast.BaseColumnRef{
		Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "col", Raw: "col"}}},
	}}
	updated, ok := root.Accept(visitor{})
	if !ok {
		t.Fatal("traversal must succeed")
	}
	if updated != ast.Node(root) {
		t.Fatal("outer node identity was lost")
	}
	if len(root.Name.Parts) != 1 {
		t.Fatal("name slot was lost")
	}
}
