package ast_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

var errProjectionExpr = errors.New("projection expr restore failed")

// Keep both expression and writer failures reachable through parent wrappers.
type projectionErrorExpr struct{ ast.BaseExpr }

func (n *projectionErrorExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *projectionErrorExpr) Restore(ctx *format.RestoreCtx) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return errProjectionExpr
}

func TestSelectItemRestoreUsesCallerQuotePolicy(t *testing.T) {
	expr := &dialectExpr{}
	alias := lex.Ident{Name: "total", Raw: "`total`", Quote: lex.QuoteBacktick}
	item := &ast.SelectItem{Expr: expr, Alias: alias}

	cases := []struct {
		name  string
		flags format.RestoreFlags
		want  string
	}{
		{"default", format.DefaultRestoreFlags, "NULL AS total"},
		{"double quotes", format.DefaultRestoreFlags | format.RestoreNameDoubleQuotes, "NULL AS \"total\""},
		{"backticks", format.DefaultRestoreFlags | format.RestoreNameBackQuotes, "NULL AS `total`"},
		{"uppercase", format.DefaultRestoreFlags | format.RestoreNameUppercase, "NULL AS TOTAL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, tc.flags)
			if err := item.Restore(ctx); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSelectItemEmptyAndQuotedEmptyAliases(t *testing.T) {
	t.Run("no alias", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)
		if err := (&ast.SelectItem{Expr: &dialectExpr{}}).Restore(ctx); err != nil {
			t.Fatal(err)
		}
		if got := out.String(); got != "NULL" {
			t.Fatalf("restore = %q, want %q", got, "NULL")
		}
	})

	aliasSources := []struct {
		name  string
		alias lex.Ident
	}{
		{"single quote", lex.Ident{Name: "", Raw: "''", Quote: lex.QuoteSingle}},
		{"double quote", lex.Ident{Name: "", Raw: `""`, Quote: lex.QuoteDouble}},
		{"backtick", lex.Ident{Name: "", Raw: "``", Quote: lex.QuoteBacktick}},
	}
	for _, src := range aliasSources {
		t.Run("empty alias "+src.name, func(t *testing.T) {
			t.Parallel()
			if !src.alias.Present() {
				t.Fatal("a quoted empty identifier must be present")
			}
			var out bytes.Buffer
			ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, format.RestoreNameBackQuotes)
			item := &ast.SelectItem{Expr: &dialectExpr{}, Alias: src.alias}
			if err := item.Restore(ctx); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != "null AS ``" {
				t.Fatalf("restore = %q, want %q", got, "null AS ``")
			}
		})
	}

	t.Run("missing expression", func(t *testing.T) {
		t.Parallel()
		item := &ast.SelectItem{Alias: lex.Ident{Name: "total"}}
		err := item.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("missing expression must fail")
		}
		if !strings.Contains(err.Error(), "restore select item: missing expression") {
			t.Fatalf("root cause must identify the missing expression: %v", err)
		}
	})
}

func TestSelectItemRestoreFailureChain(t *testing.T) {
	t.Run("child expression", func(t *testing.T) {
		t.Parallel()
		item := &ast.SelectItem{Expr: &projectionErrorExpr{}}
		err := item.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, errProjectionExpr) {
			t.Fatalf("child root cause must stay in the chain: %v", err)
		}
	})

	t.Run("writer sentinel", func(t *testing.T) {
		t.Parallel()
		item := &ast.SelectItem{Expr: &dialectExpr{}}
		err := item.Restore(format.NewRestoreCtx(nil, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, format.ErrNilWriter) {
			t.Fatalf("writer sentinel must stay in the chain: %v", err)
		}
	})

	t.Run("writer failure in error expr", func(t *testing.T) {
		t.Parallel()
		item := &ast.SelectItem{Expr: &projectionErrorExpr{}}
		err := item.Restore(format.NewRestoreCtx(nil, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, format.ErrNilWriter) {
			t.Fatalf("writer failure must reach the caller: %v", err)
		}
	})
}

func TestFieldListRestoresItemsInSourceOrder(t *testing.T) {
	var out bytes.Buffer
	items := []*ast.SelectItem{
		{Expr: &dialectExpr{}},
		{Expr: &dialectExpr{}, Alias: lex.Ident{Name: "a"}},
	}
	list := &ast.FieldList{Items: items}
	ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)
	if err := list.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "NULL, NULL AS a" {
		t.Fatalf("restore = %q, want %q", got, "NULL, NULL AS a")
	}

	var discard bytes.Buffer
	failing := &ast.FieldList{Items: []*ast.SelectItem{
		{Expr: &dialectExpr{}},
		{Expr: &projectionErrorExpr{}},
	}}
	err := failing.Restore(format.NewRestoreCtx(&discard, format.DialectConfig{}, format.DefaultRestoreFlags))
	if err == nil {
		t.Fatal("a failing item must report an error")
	}
	if !errors.Is(err, errProjectionExpr) {
		t.Fatalf("item failure must keep the child root cause: %v", err)
	}
	if !strings.Contains(err.Error(), "restore field list item 1") {
		t.Fatalf("item index must stay in the chain: %v", err)
	}
}

func TestFieldListTraversalOrderAndReplacement(t *testing.T) {
	replaceItem := &ast.SelectItem{Expr: &dialectExpr{}}
	replaceExpr := &dialectExpr{}

	t.Run("source order identity", func(t *testing.T) {
		t.Parallel()
		expr := &dialectExpr{}
		wildcard := &ast.WildcardExpr{Qualifier: ast.QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}
		item0 := &ast.SelectItem{Expr: expr}
		item1 := &ast.SelectItem{Expr: wildcard}
		list := &ast.FieldList{Items: []*ast.SelectItem{item0, item1}}
		var entered []ast.Node
		updated, ok := list.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			entered = append(entered, n)
			return n, false
		}})
		if !ok || updated != list {
			t.Fatal("traversal must keep the outer list")
		}
		want := []ast.Node{list, item0, expr, item1, wildcard}
		if !sameOrder(entered, want) {
			t.Fatalf("enter order = %v, want %v", entered, want)
		}
	})

	t.Run("skip", func(t *testing.T) {
		t.Parallel()
		expr := &dialectExpr{}
		list := &ast.FieldList{Items: []*ast.SelectItem{{Expr: expr}}}
		var visited []ast.Node
		updated, ok := list.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, true
		}})
		if !ok || updated != list {
			t.Fatal("skip must keep the outer list")
		}
		if len(visited) != 1 || visited[0] != list {
			t.Fatalf("skipped subtree must not be visited: %v", visited)
		}
	})

	t.Run("skip child", func(t *testing.T) {
		t.Parallel()
		expr := &dialectExpr{}
		item := &ast.SelectItem{Expr: expr}
		list := &ast.FieldList{Items: []*ast.SelectItem{item}}
		var visited []ast.Node
		updated, ok := list.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, n == item
		}})
		if !ok || updated != list {
			t.Fatal("child skip must keep the outer list")
		}
		want := []ast.Node{list, item}
		if !sameOrder(visited, want) {
			t.Fatalf("skipped child subtree was visited: %v", visited)
		}
	})

	t.Run("stop", func(t *testing.T) {
		t.Parallel()
		first := &dialectExpr{}
		second := &dialectExpr{}
		item0 := &ast.SelectItem{Expr: first}
		item1 := &ast.SelectItem{Expr: second}
		list := &ast.FieldList{Items: []*ast.SelectItem{item0, item1}}
		var visited []ast.Node
		updated, ok := list.Accept(visitor{
			enter: func(n ast.Node) (ast.Node, bool) {
				visited = append(visited, n)
				return n, false
			},
			leave: func(n ast.Node) (ast.Node, bool) {
				return n, n != first
			},
		})
		if ok {
			t.Fatal("child stop must propagate")
		}
		if updated != list {
			t.Fatal("stop must keep the outer list")
		}
		// leave(first) returned false, so SelectItem.Accept stopped;
		// the list's remaining items were not visited.
		want := []ast.Node{list, item0, first}
		if !sameOrder(visited, want) {
			t.Fatalf("traversal must stop at the failing child: %v", visited)
		}
	})

	t.Run("replace item", func(t *testing.T) {
		t.Parallel()
		original := &ast.SelectItem{Expr: &dialectExpr{}}
		list := &ast.FieldList{Items: []*ast.SelectItem{original}}
		updated, ok := list.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if _, isItem := n.(*ast.SelectItem); isItem {
				return replaceItem, true
			}
			return n, true
		}})
		if !ok || updated != list {
			t.Fatal("item replacement must keep the outer list")
		}
		if list.Items[0] != replaceItem || list.Items[0] == original {
			t.Fatal("slot must hold the replacement item")
		}
	})

	t.Run("replace wildcard expr", func(t *testing.T) {
		t.Parallel()
		wildcard := &ast.WildcardExpr{Qualifier: ast.QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}
		item := &ast.SelectItem{Expr: wildcard}
		updated, ok := item.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if n == wildcard {
				return replaceExpr, true
			}
			return n, true
		}})
		if !ok || updated != item {
			t.Fatal("expression replacement must keep the outer item")
		}
		if item.Expr != replaceExpr {
			t.Fatal("item expression slot must hold the replacement")
		}
	})

	t.Run("invalid item replacement", func(t *testing.T) {
		t.Parallel()
		// Leave returns a non-SelectItem for the item slot, which must
		// stop the traversal at FieldList.Accept.
		item := &ast.SelectItem{Expr: &dialectExpr{}}
		list := &ast.FieldList{Items: []*ast.SelectItem{item}}
		_, ok := list.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if _, isItem := n.(*ast.SelectItem); isItem {
				return &ast.WildcardExpr{}, true
			}
			return n, true
		}})
		if ok {
			t.Fatal("a non-item replacement must stop the traversal")
		}
	})

	t.Run("invalid expr replacement", func(t *testing.T) {
		t.Parallel()
		// Leave returns a non-ExprNode for the expr slot, which must
		// stop the traversal at SelectItem.Accept.
		wildcard := &ast.WildcardExpr{}
		item := &ast.SelectItem{Expr: wildcard}
		_, ok := item.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			return &ast.FieldList{}, true
		}})
		if ok {
			t.Fatal("a non-expression replacement must stop the traversal")
		}
	})

	t.Run("enter wrong type on list", func(t *testing.T) {
		t.Parallel()
		list := &ast.FieldList{Items: []*ast.SelectItem{{Expr: &dialectExpr{}}}}
		_, ok := list.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			return &ast.SelectItem{}, false
		}})
		if ok {
			t.Fatal("a wrong-type Enter on the list must stop the traversal")
		}
	})

	t.Run("enter wrong type on item", func(t *testing.T) {
		t.Parallel()
		item := &ast.SelectItem{Expr: &dialectExpr{}}
		_, ok := item.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			if _, isItem := n.(*ast.SelectItem); isItem {
				return &ast.FieldList{}, false
			}
			return n, false
		}})
		if ok {
			t.Fatal("a wrong-type Enter on the item must stop the traversal")
		}
	})
}

func TestProjectionSpanStates(t *testing.T) {
	nodes := []ast.Node{&ast.SelectItem{}, &ast.FieldList{}, &ast.WildcardExpr{}}
	for _, n := range nodes {
		if n.Span() != loc.InvalidSpan() {
			t.Fatalf("%T must start with an unknown span", n)
		}
		n.SetSpan(loc.Span{})
		if n.Span() != (loc.Span{}) {
			t.Fatalf("%T lost the known empty range [0, 0)", n)
		}
	}
}

func TestProjectionNodeContracts(t *testing.T) {
	itemType := reflect.TypeFor[*ast.SelectItem]()
	if !itemType.Implements(reflect.TypeFor[ast.Node]()) {
		t.Error("SelectItem must satisfy ast.Node")
	}
	listType := reflect.TypeFor[*ast.FieldList]()
	if !listType.Implements(reflect.TypeFor[ast.Node]()) {
		t.Error("FieldList must satisfy ast.Node")
	}
	wildcardType := reflect.TypeFor[*ast.WildcardExpr]()
	if !wildcardType.Implements(reflect.TypeFor[ast.ExprNode]()) {
		t.Error("WildcardExpr must satisfy ast.ExprNode")
	}

	// Local field checks: SelectItem has no planning metadata.
	itemStruct := reflect.TypeFor[ast.SelectItem]()
	if itemStruct.NumField() != 3 {
		t.Errorf("SelectItem must have exactly 3 fields (BaseNode, Expr, Alias), got %d", itemStruct.NumField())
	}
	if f, found := itemStruct.FieldByName("Alias"); !found || f.Type != reflect.TypeFor[lex.Ident]() {
		t.Errorf("SelectItem.Alias must be lex.Ident, got %v", f.Type)
	}
	if f, found := itemStruct.FieldByName("Expr"); !found || f.Type != reflect.TypeFor[ast.ExprNode]() {
		t.Errorf("SelectItem.Expr must be ast.ExprNode, got %v", f.Type)
	}
	for _, bad := range []string{"Auxiliary", "Offset", "AsName", "WildCard"} {
		if _, found := itemStruct.FieldByName(bad); found {
			t.Errorf("SelectItem must not carry %s", bad)
		}
	}

	listStruct := reflect.TypeFor[ast.FieldList]()
	if listStruct.NumField() != 2 {
		t.Errorf("FieldList must have exactly 2 fields (BaseNode, Items), got %d", listStruct.NumField())
	}
	if f, found := listStruct.FieldByName("Items"); !found || f.Type != reflect.TypeFor[[]*ast.SelectItem]() {
		t.Errorf("FieldList.Items must be []*ast.SelectItem, got %v", f.Type)
	}
	for _, bad := range []string{"Auxiliary", "Offset", "AsName", "WildCard"} {
		if _, found := listStruct.FieldByName(bad); found {
			t.Errorf("FieldList must not carry %s", bad)
		}
	}

	for _, base := range []struct {
		name string
		typ  reflect.Type
	}{
		{"BaseNode", reflect.TypeFor[*ast.BaseNode]()},
		{"BaseExpr", reflect.TypeFor[*ast.BaseExpr]()},
	} {
		for _, method := range []string{"Accept", "Restore"} {
			if _, found := base.typ.MethodByName(method); found {
				t.Errorf("%s must not dispatch %s", base.name, method)
			}
		}
	}
}

func TestProjectionNilChildrenKeepErrors(t *testing.T) {
	t.Run("nil item expression", func(t *testing.T) {
		t.Parallel()
		item := &ast.SelectItem{}
		err := item.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("a missing expression must fail")
		}
	})

	t.Run("nil list item", func(t *testing.T) {
		t.Parallel()
		list := &ast.FieldList{Items: []*ast.SelectItem{nil}}
		err := list.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("a nil slot must fail")
		}
		if !strings.Contains(err.Error(), "restore field list item 0") {
			t.Fatalf("the nil slot must fail with its index: %v", err)
		}
	})

	t.Run("nil wildcard", func(t *testing.T) {
		t.Parallel()
		var wildcard *ast.WildcardExpr
		err := wildcard.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("a missing wildcard must fail")
		}
	})

	t.Run("nil list receiver", func(t *testing.T) {
		t.Parallel()
		var list *ast.FieldList
		err := list.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("a nil list must fail")
		}
		if !strings.Contains(err.Error(), "restore field list: nil list") {
			t.Fatalf("the nil list root cause must stay in the chain: %v", err)
		}
	})

	t.Run("nil writer", func(t *testing.T) {
		t.Parallel()
		writer := format.NewRestoreWriter(nil)
		if err := writer.Err(); !errors.Is(err, format.ErrNilWriter) {
			t.Fatalf("the writer sentinel must be reachable: %v", err)
		}
		ctx := format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags)
		ctx.Writer = writer
		for _, node := range []ast.Node{
			&ast.SelectItem{Expr: &dialectExpr{}},
			&ast.FieldList{},
			&ast.WildcardExpr{},
		} {
			if err := node.Restore(ctx); !errors.Is(err, format.ErrNilWriter) {
				t.Fatalf("%T restore must keep the writer sentinel: %v", node, err)
			}
		}
	})

	t.Run("nil receiver Accept does not panic", func(t *testing.T) {
		t.Parallel()
		var (
			item     *ast.SelectItem
			list     *ast.FieldList
			wildcard *ast.WildcardExpr
		)
		for _, n := range []ast.Node{item, list, wildcard} {
			_, _ = n.Accept(visitor{})
		}
	})
}

func TestWildcardExprRestore(t *testing.T) {
	cases := []struct {
		name      string
		qualifier ast.QualifiedName
		flags     format.RestoreFlags
		want      string
	}{
		{"plain", ast.QualifiedName{}, 0, "*"},
		{"qualified", ast.QualifiedName{Parts: []lex.Ident{{Name: "t"}}}, 0, "t.*"},
		{"quoted dot backtick", ast.QualifiedName{Parts: []lex.Ident{
			{Name: "a.b", Raw: "`a.b`", Quote: lex.QuoteBacktick},
		}}, format.RestoreNameBackQuotes, "`a.b`.*"},
		{"escaped backtick in name", ast.QualifiedName{Parts: []lex.Ident{
			{Name: "a`b", Raw: "`a``b`", Quote: lex.QuoteBacktick},
		}}, format.RestoreNameBackQuotes, "`a``b`.*"},
		{"multi part", ast.QualifiedName{Parts: []lex.Ident{
			{Name: "db"},
			{Name: "tbl"},
		}}, 0, "db.tbl.*"},
		{"escaped double quote", ast.QualifiedName{Parts: []lex.Ident{
			{Name: `a"b`, Raw: `"a""b"`, Quote: lex.QuoteDouble},
		}}, format.RestoreNameDoubleQuotes, `"a""b".*`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, tc.flags)
			if err := (&ast.WildcardExpr{Qualifier: tc.qualifier}).Restore(ctx); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func ExampleFieldList() {
	list := &ast.FieldList{Items: []*ast.SelectItem{
		{Expr: &ast.WildcardExpr{Qualifier: ast.QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}},
		{Expr: &ast.WildcardExpr{}},
	}}
	var out bytes.Buffer
	if err := list.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.RestoreNameDoubleQuotes)); err != nil {
		panic(err)
	}
	fmt.Println(out.String())
	// Output:
	// "t".*, *
}
