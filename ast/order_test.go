package ast_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/loc"
)

// mockDialectOrderItem is a dialect outer that embeds the shared base and
// adds a field the public node does not carry, e.g. NULLS LAST.
type mockDialectOrderItem struct {
	ast.BaseOrderItem
	Extra bool
}

func (n *mockDialectOrderItem) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*mockDialectOrderItem)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.Expr != nil {
		updated, ok := outer.Expr.Accept(v)
		if !ok {
			return outer, false
		}
		expr, ok := updated.(ast.ExprNode)
		if !ok {
			return outer, false
		}
		outer.Expr = expr
	}
	return v.Leave(outer)
}

func (n *mockDialectOrderItem) Restore(ctx *format.RestoreCtx) error {
	if n.Expr == nil {
		return errors.New("restore mock order item: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return err
	}
	if n.Extra {
		ctx.WriteKeyword(" NULLS LAST")
	}
	return ctx.Err()
}

var (
	_ ast.OrderItemNode = (*mockDialectOrderItem)(nil)
	_ ast.OrderItemNode = (*ast.OrderItem)(nil)
)

// TestOrderItemCanonicalRestore pins DESC as the only direction keyword: both
// an omitted direction and explicit ASC restore the bare expression.
func TestOrderItemCanonicalRestore(t *testing.T) {
	cases := []struct {
		name string
		desc bool
		want string
	}{
		{"omitted direction", false, "NULL"},
		{"explicit ASC keeps the bare expression", false, "NULL"},
		{"DESC", true, "NULL DESC"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: &dialectExpr{}, Desc: tc.desc}}
			if err := item.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestOrderItemRestoreFailures keeps the missing-slot root cause, the child
// root cause chain, and the first writer failure.
func TestOrderItemRestoreFailures(t *testing.T) {
	t.Run("missing expression", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{}
		err := item.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("a missing expression must fail")
		}
		if !strings.Contains(err.Error(), "restore order item: missing expression") {
			t.Fatalf("root cause must identify the missing slot: %v", err)
		}
	})

	t.Run("failing expression keeps root cause", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: &orderErrorExpr{}}}
		err := item.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("a failing expression must report an error")
		}
		if !errors.Is(err, errOrderItemExpr) {
			t.Fatalf("child root cause must stay in the chain: %v", err)
		}
	})

	t.Run("nil receiver does not panic", func(t *testing.T) {
		t.Parallel()
		var item *ast.OrderItem
		if err := item.Restore(format.NewRestoreCtx(nil, format.DialectConfig{}, format.DefaultRestoreFlags)); err == nil {
			t.Fatal("a nil order item must fail")
		}
	})

	t.Run("short write keeps root cause", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: &dialectExpr{}}}
		ctx := format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags)
		ctx.Writer = format.NewRestoreWriter(&shortWriter{})
		err := item.Restore(ctx)
		if err == nil {
			t.Fatal("a short write must report an error")
		}
		if !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("short write must stay in the chain: %v", err)
		}
	})

	t.Run("pre-existing writer error is not overwritten", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: &dialectExpr{}}}
		ctx := format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags)
		ctx.Writer = format.NewRestoreWriter(&shortWriter{})
		_, werr := ctx.Writer.Write([]byte("x"))
		if !errors.Is(werr, io.ErrShortWrite) {
			t.Fatalf("seed error must be a short write: %v", werr)
		}
		got := item.Restore(ctx)
		if got == nil {
			t.Fatal("a seeded writer error must reach the caller")
		}
		if !errors.Is(got, io.ErrShortWrite) {
			t.Fatalf("the pre-existing writer error must win, not a later one: %v", got)
		}
	})
}

// TestOrderItemTraversalContract pins outer identity, same-type Enter
// replacement, skip, expression replacement with stop, and the wrong-type
// outer exit.
func TestOrderItemTraversalContract(t *testing.T) {
	expr := &dialectExpr{}
	replaceExpr := &dialectExpr{}

	t.Run("outer identity", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: expr}}
		var visited []ast.Node
		updated, ok := item.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, false
		}})
		if !ok || updated != ast.Node(item) {
			t.Fatal("traversal must keep the outer order item")
		}
		if !sameOrder(visited, []ast.Node{item, expr}) {
			t.Fatalf("enter order = %v", visited)
		}
	})

	t.Run("enter same-type replacement", func(t *testing.T) {
		t.Parallel()
		original := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: &dialectExpr{}}}
		replacement := &ast.OrderItem{}
		updated, ok := original.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			return replacement, true
		}})
		if !ok || updated != ast.Node(replacement) {
			t.Fatal("Enter must substitute a same-type outer item")
		}
	})

	t.Run("skip keeps slots", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: expr}}
		var visited []ast.Node
		updated, ok := item.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, n == ast.Node(item)
		}})
		if !ok || updated != ast.Node(item) {
			t.Fatal("skip must keep the outer order item")
		}
		if !sameOrder(visited, []ast.Node{item}) {
			t.Fatalf("skipped children must not be visited: %v", visited)
		}
		if item.Expr != ast.ExprNode(expr) {
			t.Fatal("skip must not change the expression slot")
		}
	})

	t.Run("replace expression", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: expr}}
		updated, ok := item.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if n == ast.Node(expr) {
				return replaceExpr, true
			}
			return n, true
		}})
		if !ok || updated != ast.Node(item) {
			t.Fatal("expression replacement must keep the outer item")
		}
		if item.Expr != ast.ExprNode(replaceExpr) {
			t.Fatal("the expression slot must hold the replacement")
		}
	})

	t.Run("stop at expression", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: expr}}
		_, ok := item.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			return n, n != ast.Node(expr)
		}})
		if ok {
			t.Fatal("stop at the expression must propagate")
		}
		if item.Expr != ast.ExprNode(expr) {
			t.Fatal("stop must keep the old child")
		}
	})

	t.Run("reject expression replacement", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: expr}}
		updated, ok := item.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if n == ast.Node(expr) {
				return &ast.FieldList{}, true
			}
			return n, true
		}})
		if ok {
			t.Fatal("a non-expression replacement must stop the traversal")
		}
		if item.Expr != ast.ExprNode(expr) {
			t.Fatal("the expression slot must keep its old child after rejection")
		}
		if updated != ast.Node(item) {
			t.Fatal("the outer item must stay in the result")
		}
	})

	t.Run("wrong-type outer stops traversal", func(t *testing.T) {
		t.Parallel()
		item := &ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: expr}}
		replacement := &ast.FieldList{}
		updated, ok := item.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			return replacement, false
		}})
		if ok {
			t.Fatal("a wrong-type Enter must stop the traversal")
		}
		if updated != ast.Node(replacement) {
			t.Fatalf("the wrong-type outer must stay in the result: got %T", updated)
		}
	})

	t.Run("nil receiver does not panic", func(t *testing.T) {
		t.Parallel()
		var item *ast.OrderItem
		updated, ok := item.Accept(visitor{})
		if ok {
			t.Fatal("a nil receiver must stop the traversal")
		}
		if updated != nil {
			t.Fatal("a nil receiver must return a nil interface")
		}
	})
}

// TestOrderDialectOuterDispatch pins that the mock outer keeps its own
// visitor identity and that its Extra field still writes through restore.
func TestOrderDialectOuterDispatch(t *testing.T) {
	expr := &dialectExpr{}
	replaceExpr := &dialectExpr{}
	item := &mockDialectOrderItem{
		BaseOrderItem: ast.BaseOrderItem{Expr: expr},
		Extra:         true,
	}

	var entered, left []ast.Node
	updated, ok := item.Accept(visitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			entered = append(entered, n)
			return n, false
		},
		leave: func(n ast.Node) (ast.Node, bool) {
			left = append(left, n)
			if n == ast.Node(expr) {
				return replaceExpr, true
			}
			return n, true
		},
	})
	if !ok || updated != ast.Node(item) {
		t.Fatal("traversal must keep the mock outer order item")
	}
	if !sameOrder(entered, []ast.Node{item, expr}) {
		t.Fatalf("enter order = %v, want the mock outer then its child", entered)
	}
	// Enter the child expr, the visitor's leave swaps it for replaceExpr; the
	// mock outer then leaves. The replacement is never entered or left again.
	if !sameOrder(left, []ast.Node{expr, item}) {
		t.Fatalf("leave order = %v, want the child then the mock outer", left)
	}
	if item.Expr != ast.ExprNode(replaceExpr) {
		t.Fatal("the expression slot must hold the replacement")
	}

	var out bytes.Buffer
	ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)
	ctx.Flags = format.RestoreKeywordLowercase
	if err := item.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	// The dialect Extra field must still be written after the base expr.
	if got := out.String(); got != "null nulls last" {
		t.Fatalf("restore = %q, want %q", got, "null nulls last")
	}
}

// TestOrderItemNodeContracts pins the role marker, the dialect slice support,
// and that the bare bases carry no dispatch and no outer identity.
func TestOrderItemNodeContracts(t *testing.T) {
	nodeType := reflect.TypeFor[ast.Node]()
	orderItemType := reflect.TypeFor[*ast.OrderItem]()
	if !orderItemType.Implements(nodeType) {
		t.Error("OrderItem must satisfy ast.Node")
	}
	if !orderItemType.Implements(reflect.TypeFor[ast.OrderItemNode]()) {
		t.Error("OrderItem must satisfy ast.OrderItemNode")
	}
	if _, found := orderItemType.MethodByName("OrderItemCore"); !found {
		t.Error("OrderItem must expose the role accessor")
	}

	items := []ast.OrderItemNode{&ast.OrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: &dialectExpr{}}}, &mockDialectOrderItem{BaseOrderItem: ast.BaseOrderItem{Expr: &dialectExpr{}}}}
	var clause ast.BaseOrderByClause
	clause.Items = items
	if len(clause.Items) != 2 || clause.Items[0] != ast.OrderItemNode(items[0]) || clause.Items[1] != ast.OrderItemNode(items[1]) {
		t.Fatal("an interface slice must hold dialect order items by role")
	}

	for _, base := range []struct {
		name string
		typ  reflect.Type
	}{
		{"BaseOrderItem", reflect.TypeFor[*ast.BaseOrderItem]()},
		{"BaseOrderByClause", reflect.TypeFor[*ast.BaseOrderByClause]()},
	} {
		if _, found := base.typ.MethodByName("Accept"); found {
			t.Errorf("%s must not dispatch Accept", base.name)
		}
		if _, found := base.typ.MethodByName("Restore"); found {
			t.Errorf("%s must not dispatch Restore", base.name)
		}
	}
	// A bare base never satisfies the full Node contract: only the outer
	// concrete node that owns Accept and Restore may hold a slot.
	if reflect.TypeFor[*ast.BaseOrderItem]().Implements(nodeType) {
		t.Error("a bare BaseOrderItem must not satisfy ast.Node")
	}

	// The dialect outer keeps its own visitor identity: the base provides the
	// role accessor, never the dispatch.
	if _, found := reflect.TypeFor[*mockDialectOrderItem]().MethodByName("OrderItemCore"); !found {
		t.Error("a dialect outer must expose the inherited role accessor")
	}
}

// TestOrderItemBaseSpanAndTextIndependent shows the embedded base keeps span
// and text independent of the role slots.
func TestOrderItemBaseSpanAndTextIndependent(t *testing.T) {
	clause := &ast.BaseOrderByClause{}
	if clause.Span() != loc.InvalidSpan() {
		t.Fatal("a zero clause must have an unknown span")
	}
	clause.SetSpan(loc.Span{})
	if clause.Span() != (loc.Span{}) {
		t.Fatal("the known empty range must be kept")
	}
	clause.SetText("sql")
	if clause.Text() != "sql" || clause.Span() != (loc.Span{}) {
		t.Fatal("text and span must remain independent")
	}
}

var errOrderItemExpr = errors.New("order item expr restore failed")

// orderErrorExpr fails on purpose to keep child root causes visible.
type orderErrorExpr struct{ ast.BaseExpr }

func (n *orderErrorExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *orderErrorExpr) Restore(ctx *format.RestoreCtx) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return errOrderItemExpr
}
