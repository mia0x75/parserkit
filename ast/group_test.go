package ast_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/loc"
)

// mockDialectGroupItem is a dialect outer that embeds the shared base and
// carries direction modifiers that belong to grouping syntax, not sorting.
type mockDialectGroupItem struct {
	ast.BaseGroupItem
	Desc  bool
	Extra bool
}

func (n *mockDialectGroupItem) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*mockDialectGroupItem)
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

func (n *mockDialectGroupItem) Restore(ctx *format.RestoreCtx) error {
	if n.Expr == nil {
		return errors.New("restore mock group item: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return err
	}
	if n.Desc {
		ctx.WriteKeyword(" DESC")
	}
	if n.Extra {
		ctx.WriteKeyword(" WITH ROLLUP")
	}
	return ctx.Err()
}

var (
	_ ast.GroupItemNode = (*mockDialectGroupItem)(nil)
)

// TestGroupItemNodeContracts pins the role marker, the dialect slice support,
// and that the bare bases carry no dispatch and no outer identity.
func TestGroupItemNodeContracts(t *testing.T) {
	nodeType := reflect.TypeFor[ast.Node]()
	// No shared concrete GroupItem is promised: a dialect outer that embeds
	// BaseGroupItem and owns the dispatch satisfies the role.
	if !reflect.TypeFor[*mockDialectGroupItem]().Implements(reflect.TypeFor[ast.GroupItemNode]()) {
		t.Error("a dialect outer must satisfy ast.GroupItemNode")
	}
	if _, found := reflect.TypeFor[*mockDialectGroupItem]().MethodByName("GroupItemCore"); !found {
		t.Error("a dialect outer must expose the inherited role accessor")
	}

	items := []ast.GroupItemNode{&mockDialectGroupItem{BaseGroupItem: ast.BaseGroupItem{Expr: &dialectExpr{}}}}
	var clause ast.BaseGroupByClause
	clause.Items = items
	if len(clause.Items) != 1 || clause.Items[0] != ast.GroupItemNode(items[0]) {
		t.Fatal("an interface slice must hold dialect group items by role")
	}

	for _, base := range []struct {
		name string
		typ  reflect.Type
	}{
		{"BaseGroupItem", reflect.TypeFor[*ast.BaseGroupItem]()},
		{"BaseGroupByClause", reflect.TypeFor[*ast.BaseGroupByClause]()},
	} {
		if _, found := base.typ.MethodByName("Accept"); found {
			t.Errorf("%s must not dispatch Accept", base.name)
		}
		if _, found := base.typ.MethodByName("Restore"); found {
			t.Errorf("%s must not dispatch Restore", base.name)
		}
	}
	if reflect.TypeFor[*ast.BaseGroupItem]().Implements(nodeType) {
		t.Error("a bare BaseGroupItem must not satisfy ast.Node")
	}
}

// TestGroupDialectOuterDispatch pins that the mock outer keeps its own
// visitor identity and that dialect fields survive both dispatch and restore.
func TestGroupDialectOuterDispatch(t *testing.T) {
	expr := &dialectExpr{}
	item := &mockDialectGroupItem{
		BaseGroupItem: ast.BaseGroupItem{Expr: expr},
		Desc:          true,
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
			return n, true
		},
	})
	if !ok || updated != ast.Node(item) {
		t.Fatal("traversal must keep the mock outer group item")
	}
	if !sameOrder(entered, []ast.Node{item, expr}) {
		t.Fatalf("enter order = %v, want the mock outer then its child", entered)
	}
	if !sameOrder(left, []ast.Node{expr, item}) {
		t.Fatalf("leave order = %v, want the child then the mock outer", left)
	}

	var out bytes.Buffer
	if err := item.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)); err != nil {
		t.Fatal(err)
	}
	// The dialect direction and extra fields must still be written.
	if got := out.String(); got != "NULL DESC WITH ROLLUP" {
		t.Fatalf("restore = %q, want %q", got, "NULL DESC WITH ROLLUP")
	}
}

// TestOrderItemAndGroupItemRolesDoNotInterchange pins that a sorting item is
// not a grouping item and vice versa: the role accessors stay separate.
func TestOrderItemAndGroupItemRolesDoNotInterchange(t *testing.T) {
	orderType := reflect.TypeFor[*ast.OrderItem]()
	groupType := reflect.TypeFor[*mockDialectGroupItem]()
	if orderType.Implements(reflect.TypeFor[ast.GroupItemNode]()) {
		t.Error("an order item must not satisfy ast.GroupItemNode")
	}
	if groupType.Implements(reflect.TypeFor[ast.OrderItemNode]()) {
		t.Error("a group item must not satisfy ast.OrderItemNode")
	}

	var items []ast.GroupItemNode
	items = append(items, &mockDialectGroupItem{})
	var clause ast.BaseGroupByClause
	clause.Items = items
	if clause.Items[0].GroupItemCore() == nil {
		t.Fatal("the role accessor must reach the embedded base")
	}
}

// TestGroupByBaseSpanAndTextIndependent shows the embedded base keeps span
// and text independent of the role slots.
func TestGroupByBaseSpanAndTextIndependent(t *testing.T) {
	clause := &ast.BaseGroupByClause{}
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
