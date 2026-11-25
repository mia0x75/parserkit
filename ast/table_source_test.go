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

var errTableSourceSource = errors.New("table source child restore failed")

// fakeTableSourceTable is a plain result-set source that already carries its
// own spelling, so TableSource must not add parentheses.
type fakeTableSourceTable struct{ ast.BaseResultSet }

func (n *fakeTableSourceTable) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *fakeTableSourceTable) Restore(ctx *format.RestoreCtx) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	ctx.WritePlain("orders")
	return ctx.Err()
}

// fakeTableSourceSelect embeds BaseQuery so its QueryNode marker keeps it a
// query, and implements the Node contract on its own.
type fakeTableSourceSelect struct {
	ast.BaseQuery
	// Test-only hook: a non-nil err makes Restore report it after writing.
	err error
}

var _ ast.QueryNode = (*fakeTableSourceSelect)(nil)

func (n *fakeTableSourceSelect) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *fakeTableSourceSelect) Restore(ctx *format.RestoreCtx) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	ctx.WritePlain("SELECT")
	if n.err != nil {
		return n.err
	}
	return ctx.Err()
}

// fakeTableSourceValues is a query source, so its QueryNode marker makes
// TableSource wrap its restore in parentheses.
type fakeTableSourceValues struct{ ast.BaseQuery }

func (n *fakeTableSourceValues) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *fakeTableSourceValues) Restore(ctx *format.RestoreCtx) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	ctx.WritePlain("VALUES")
	return ctx.Err()
}

// fakeTableSourceError is a source that fails the restore on purpose.
type fakeTableSourceError struct{ ast.BaseResultSet }

func (n *fakeTableSourceError) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *fakeTableSourceError) Restore(ctx *format.RestoreCtx) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return errTableSourceSource
}

func TestTableSourceRestoresPlainResultSets(t *testing.T) {
	cases := []struct {
		name   string
		alias  lex.Ident
		source ast.ResultSetNode
		want   string
	}{
		{"table", lex.Ident{Name: "o"}, &fakeTableSourceTable{}, "orders AS o"},
		{"bracketed query expr", lex.Ident{Name: "q"}, &fakeQueryExprOuter{}, "(SELECT) AS q"},
		{"derived select", lex.Ident{Name: "s"}, &fakeTableSourceSelect{}, "(SELECT) AS s"},
		{"values", lex.Ident{Name: "v"}, &fakeTableSourceValues{}, "(VALUES) AS v"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)
			src := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: tc.source, Alias: tc.alias}}
			if err := src.Restore(ctx); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTableSourceRestoresLateralAndColumns(t *testing.T) {
	src := &ast.TableSource{BaseTableSource: ast.BaseTableSource{
		Source:  &fakeTableSourceSelect{},
		Alias:   lex.Ident{Name: "j"},
		Columns: []lex.Ident{{Name: "a"}, {Name: "b"}},
		Lateral: true,
	}}
	var out bytes.Buffer
	if err := src.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "LATERAL (SELECT) AS j (a, b)" {
		t.Fatalf("restore = %q, want LATERAL (SELECT) AS j (a, b)", got)
	}
}

func TestTableSourceRestoreUsesCallerQuotePolicy(t *testing.T) {
	src := &ast.TableSource{BaseTableSource: ast.BaseTableSource{
		Source:  &fakeTableSourceTable{},
		Alias:   lex.Ident{Name: "o"},
		Columns: []lex.Ident{{Name: "id"}},
	}}
	cases := []struct {
		name  string
		flags format.RestoreFlags
		want  string
	}{
		{"default", format.DefaultRestoreFlags, "orders AS o (id)"},
		{"double quotes", format.DefaultRestoreFlags | format.RestoreNameDoubleQuotes, "orders AS \"o\" (\"id\")"},
		{"backticks", format.DefaultRestoreFlags | format.RestoreNameBackQuotes, "orders AS `o` (`id`)"},
		{"uppercase", format.DefaultRestoreFlags | format.RestoreNameUppercase, "orders AS O (ID)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, tc.flags)
			if err := src.Restore(ctx); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTableSourceKeepsQuotedEmptyAlias(t *testing.T) {
	alias := lex.Ident{Name: "", Raw: "``", Quote: lex.QuoteBacktick}
	if !alias.Present() {
		t.Fatal("a quoted empty identifier must be present")
	}
	var out bytes.Buffer
	ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, format.RestoreNameBackQuotes)
	src := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceTable{}, Alias: alias}}
	if err := src.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "orders AS ``" {
		t.Fatalf("restore = %q, want orders AS ``", got)
	}
}

func TestTableSourceRestoreFailureChain(t *testing.T) {
	t.Run("missing source", func(t *testing.T) {
		t.Parallel()
		err := (&ast.TableSource{}).Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("missing source must fail")
		}
		if !strings.Contains(err.Error(), "restore table source: missing source") {
			t.Fatalf("root cause must identify the missing source: %v", err)
		}
	})

	t.Run("column aliases without alias", func(t *testing.T) {
		t.Parallel()
		err := (&ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceTable{}, Columns: []lex.Ident{{Name: "a"}}}}).
			Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("column aliases without an alias must fail")
		}
		if !strings.Contains(err.Error(), "restore table source: column aliases require an alias") {
			t.Fatalf("root cause must forbid incomplete output: %v", err)
		}
	})

	t.Run("failing source", func(t *testing.T) {
		t.Parallel()
		err := (&ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceError{}}}).
			Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, errTableSourceSource) {
			t.Fatalf("source root cause must stay in the chain: %v", err)
		}
	})

	t.Run("nil receiver", func(t *testing.T) {
		t.Parallel()
		var src *ast.TableSource
		if err := src.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags)); err == nil {
			t.Fatal("a nil source must fail")
		}
	})

	t.Run("nil writer", func(t *testing.T) {
		t.Parallel()
		src := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceTable{}, Alias: lex.Ident{Name: "o"}}}
		err := src.Restore(format.NewRestoreCtx(nil, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, format.ErrNilWriter) {
			t.Fatalf("writer sentinel must stay in the chain: %v", err)
		}
	})

	t.Run("failing source with nil writer", func(t *testing.T) {
		t.Parallel()
		src := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceError{}}}
		err := src.Restore(format.NewRestoreCtx(nil, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, format.ErrNilWriter) {
			t.Fatalf("writer failure must reach the caller: %v", err)
		}
	})

	t.Run("failing query source", func(t *testing.T) {
		t.Parallel()
		// The query branch of Restore must keep the source error chain.
		src := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceSelect{err: errTableSourceSource}}}
		err := src.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, errTableSourceSource) {
			t.Fatalf("query source root cause must stay in the chain: %v", err)
		}
	})

	t.Run("failing writer", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)
		failing := format.NewRestoreWriter(&failingWriter{err: errTableSourceSource})
		ctx.Writer = failing
		src := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceTable{}, Alias: lex.Ident{Name: "o"}}}
		err := src.Restore(ctx)
		if !errors.Is(err, errTableSourceSource) {
			t.Fatalf("a failing writer must keep its error in the chain: %v", err)
		}
	})
}

// failingWriter fails every write with its own sentinel error.
type failingWriter struct{ err error }

func (w *failingWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

func TestTableSourceTraversalContract(t *testing.T) {
	// Each subtest builds its own source, because a successful replacement
	// rewrites the shared slot for later subtests.
	newSource := func() *fakeTableSourceSelect { return &fakeTableSourceSelect{} }

	t.Run("source identity and order", func(t *testing.T) {
		t.Parallel()
		source := newSource()
		outer := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: source}}
		var visited []ast.Node
		updated, ok := outer.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, false
		}})
		if !ok || updated != ast.Node(outer) {
			t.Fatal("traversal must keep the outer source")
		}
		if !sameOrder(visited, []ast.Node{outer, source}) {
			t.Fatalf("enter order = %v", visited)
		}
	})

	t.Run("replace source", func(t *testing.T) {
		t.Parallel()
		source := newSource()
		outer := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: source}}
		outerReplacement := newSource()
		updated, ok := outer.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if n == source {
				return outerReplacement, true
			}
			return n, true
		}})
		if !ok || updated != ast.Node(outer) {
			t.Fatal("source replacement must keep the outer node")
		}
		if outer.Source != ast.ResultSetNode(outerReplacement) || outer.Source == ast.ResultSetNode(source) {
			t.Fatal("source slot must hold the replacement")
		}
	})

	t.Run("skip", func(t *testing.T) {
		t.Parallel()
		source := newSource()
		outer := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: source}}
		var visited []ast.Node
		updated, ok := outer.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, n == ast.Node(outer)
		}})
		if !ok || updated != ast.Node(outer) {
			t.Fatal("skip must keep the outer node")
		}
		if !sameOrder(visited, []ast.Node{outer}) {
			t.Fatalf("skipped child must not be visited: %v", visited)
		}
		if outer.Source != ast.ResultSetNode(source) {
			t.Fatal("skip must not change the source slot")
		}
	})

	t.Run("child stop", func(t *testing.T) {
		t.Parallel()
		source := newSource()
		outer := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: source}}
		_, ok := outer.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			return n, n != ast.Node(source)
		}})
		if ok {
			t.Fatal("child stop must propagate")
		}
	})

	t.Run("invalid enter type", func(t *testing.T) {
		t.Parallel()
		outer := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: newSource()}}
		_, ok := outer.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			if n == ast.Node(outer) {
				return &fakeTableSourceTable{}, false
			}
			return n, false
		}})
		if ok {
			t.Fatal("a wrong-type Enter on the outer node must stop the traversal")
		}
	})

	t.Run("invalid source replacement", func(t *testing.T) {
		t.Parallel()
		// The Leave for the source slot returns a non-result-set node, so the
		// outer type check must stop the traversal and keep the old source.
		outer := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: newSource()}}
		updated, ok := outer.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if _, isSource := n.(*fakeTableSourceSelect); isSource {
				return &dialectExpr{}, true
			}
			return n, true
		}})
		if ok {
			t.Fatal("a non-result-set source replacement must stop the traversal")
		}
		if _, isSource := outer.Source.(*fakeTableSourceSelect); !isSource {
			t.Fatal("the old source must stay in the slot")
		}
		if updated != ast.Node(outer) {
			t.Fatal("the outer node must stay in the result")
		}
	})

	// A BaseQueryExpr outer satisfies ResultSetNode, so a Visitor may replace
	// the Source slot with it; it is not a QueryNode, so no double brackets.
	t.Run("replace source with bracketed query expr", func(t *testing.T) {
		t.Parallel()
		outer := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceTable{}}}
		replacement := &fakeQueryExprOuter{}
		updated, ok := outer.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if _, isSource := n.(*fakeTableSourceTable); isSource {
				return replacement, true
			}
			return n, true
		}})
		if !ok || updated != ast.Node(outer) {
			t.Fatal("query expr replacement must keep the outer node")
		}
		if outer.Source != ast.ResultSetNode(replacement) {
			t.Fatal("the query expr must land in the source slot")
		}
	})

	t.Run("nil source counts as leaf", func(t *testing.T) {
		t.Parallel()
		var visited []ast.Node
		leaf := &ast.TableSource{}
		updated, ok := leaf.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, false
		}})
		if !ok || updated != ast.Node(leaf) {
			t.Fatal("a nil source must keep the outer node")
		}
		if len(visited) != 1 || visited[0] != ast.Node(leaf) {
			t.Fatalf("nil source has no children to visit: %v", visited)
		}
	})

	t.Run("nil receiver does not panic", func(t *testing.T) {
		t.Parallel()
		var src *ast.TableSource
		updated, ok := src.Accept(visitor{})
		if ok || updated != nil {
			t.Fatal("a nil receiver must stop the traversal")
		}
	})
}

func TestTableSourceSpanAndTextInheritance(t *testing.T) {
	if (&ast.TableSource{}).Span() != loc.InvalidSpan() {
		t.Fatal("zero table source must have an unknown span")
	}
	outer := &ast.TableSource{}
	outer.SetSpan(loc.Span{Start: 1, End: 9})
	// Span is promoted from the embedded base, so both reads must agree.
	if outer.Span() != (loc.Span{Start: 1, End: 9}) {
		t.Fatal("the embedded base must inherit the outer span")
	}
	outer.SetSpan(loc.Span{})
	if outer.Span() != (loc.Span{}) {
		t.Fatal("the known empty range must be kept")
	}
	outer.SetText("sql")
	if outer.Text() != "sql" || outer.Span() != (loc.Span{}) {
		t.Fatal("text and span must remain independent")
	}
}

func TestTableSourceNodeContract(t *testing.T) {
	nodeType := reflect.TypeFor[ast.Node]()
	if !reflect.TypeFor[*ast.TableSource]().Implements(nodeType) {
		t.Error("TableSource must satisfy ast.Node")
	}
	// A base without its own dispatch satisfies the result-set marker but not
	// the full Node contract (no Accept or Restore), so it cannot become a
	// source; only an outer node that carries dispatch does.
	if reflect.TypeFor[*ast.BaseResultSet]().Implements(reflect.TypeFor[ast.ResultSetNode]()) {
		t.Error("a bare BaseResultSet must not satisfy ast.ResultSetNode")
	}
	if !reflect.TypeFor[*fakeTableSourceSelect]().Implements(reflect.TypeFor[ast.ResultSetNode]()) {
		t.Error("an outer node must satisfy ast.ResultSetNode through its base chain")
	}
	// An ordinary expression does not carry the result-set marker.
	if reflect.TypeFor[*dialectExpr]().Implements(reflect.TypeFor[ast.ResultSetNode]()) {
		t.Error("an expression must not satisfy ast.ResultSetNode")
	}
	// A dual-marker query expression satisfies the source slot but, lacking a
	// statement marker, must not enter the QueryNode parenthesis branch.
	if !reflect.TypeFor[*fakeQueryExprOuter]().Implements(reflect.TypeFor[ast.ResultSetNode]()) {
		t.Error("a BaseQueryExpr outer must satisfy ast.ResultSetNode")
	}
	if reflect.TypeFor[*fakeQueryExprOuter]().Implements(reflect.TypeFor[ast.QueryNode]()) {
		t.Error("a BaseQueryExpr outer must not satisfy ast.QueryNode")
	}

	baseType := reflect.TypeFor[ast.BaseTableSource]()
	wantBase := []struct {
		name     string
		typ      reflect.Type
		embedded bool
	}{
		{"BaseResultSet", reflect.TypeFor[ast.BaseResultSet](), true},
		{"Source", reflect.TypeFor[ast.ResultSetNode](), false},
		{"Alias", reflect.TypeFor[lex.Ident](), false},
		{"Columns", reflect.TypeFor[[]lex.Ident](), false},
		{"Lateral", reflect.TypeFor[bool](), false},
	}
	if baseType.NumField() != len(wantBase) {
		t.Fatalf("BaseTableSource must have exactly 5 fields, got %d", baseType.NumField())
	}
	for i, want := range wantBase {
		if field := baseType.Field(i); field.Name != want.name || field.Type != want.typ || field.Anonymous != want.embedded {
			t.Errorf("BaseTableSource field %d = %+v, want %+v", i, field, want)
		}
	}

	// The standard concrete source embeds exactly one value base and nothing else.
	srcType := reflect.TypeFor[ast.TableSource]()
	if srcType.NumField() != 1 {
		t.Fatalf("TableSource must have exactly 1 field, got %d", srcType.NumField())
	}
	if f := srcType.Field(0); f.Name != "BaseTableSource" || f.Type != reflect.TypeFor[ast.BaseTableSource]() || !f.Anonymous {
		t.Errorf("TableSource must embed ast.BaseTableSource by value, got %+v", f)
	}
	if f, found := srcType.FieldByName("Source"); !found || f.Type != reflect.TypeFor[ast.ResultSetNode]() {
		t.Errorf("TableSource must promote Source as ast.ResultSetNode, got %v", f.Type)
	}
	if f, found := srcType.FieldByName("Alias"); !found || f.Type != reflect.TypeFor[lex.Ident]() {
		t.Errorf("TableSource must promote Alias as lex.Ident, got %v", f.Type)
	}
	if f, found := srcType.FieldByName("Columns"); !found || f.Type != reflect.TypeFor[[]lex.Ident]() {
		t.Errorf("TableSource must promote Columns as []lex.Ident, got %v", f.Type)
	}
	if f, found := srcType.FieldByName("Lateral"); !found || f.Type != reflect.TypeFor[bool]() {
		t.Errorf("TableSource must promote Lateral as bool, got %v", f.Type)
	}
	for _, bad := range []string{"Core", "Wrapper", "Auxiliary", "AsName", "TableAlias"} {
		if _, found := srcType.FieldByName(bad); found {
			t.Errorf("TableSource must not carry %s", bad)
		}
	}
}

func ExampleTableSource() {
	src := &ast.TableSource{BaseTableSource: ast.BaseTableSource{
		Source:  &fakeTableSourceSelect{},
		Alias:   lex.Ident{Name: "s"},
		Columns: []lex.Ident{{Name: "id"}, {Name: "name"}},
		Lateral: true,
	}}
	var out bytes.Buffer
	if err := src.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)); err != nil {
		panic(err)
	}
	fmt.Println(out.String())
	// Output:
	// LATERAL (SELECT) AS s (id, name)
}
