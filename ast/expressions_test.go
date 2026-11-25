package ast_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// binaryBaseExpr is the outer dialect node that owns the shared binary base.
// It carries the dispatch itself so the base never has to.
type binaryBaseExpr struct {
	ast.BaseBinaryExpr
}

var _ ast.ExprNode = (*binaryBaseExpr)(nil)

func (n *binaryBaseExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*binaryBaseExpr)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	left, ok := outer.acceptSlot(outer.Left, v)
	if !ok {
		return outer, false
	}
	outer.Left = left
	right, ok := outer.acceptSlot(outer.Right, v)
	if !ok {
		return outer, false
	}
	outer.Right = right
	return v.Leave(outer)
}

// acceptSlot visits one child slot and converts the result back to an
// expression, so a replaced child keeps its slot type.
func (n *binaryBaseExpr) acceptSlot(child ast.ExprNode, v ast.Visitor) (ast.ExprNode, bool) {
	if child == nil {
		return nil, true
	}
	updated, ok := child.Accept(v)
	if !ok {
		return nil, false
	}
	next, ok := updated.(ast.ExprNode)
	if !ok {
		return nil, false
	}
	return next, true
}

func (n *binaryBaseExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain("(")
	if n.Left != nil {
		if err := n.Left.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain("+")
	if n.Right != nil {
		if err := n.Right.Restore(ctx); err != nil {
			return err
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// spanHolder is the part of ast.Node the bases can supply on their own.
type spanHolder interface {
	Span() loc.Span
	SetSpan(loc.Span)
}

func sameOrder(got, want []ast.Node) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestExpressionBasesCarryNoDispatch checks that expression bases stay
// free of node dispatch and never satisfy the expression marker alone.
func TestExpressionBasesCarryNoDispatch(t *testing.T) {
	exprNode := reflect.TypeFor[ast.ExprNode]()
	bases := []reflect.Type{
		reflect.TypeFor[*ast.BaseLiteralExpr](),
		reflect.TypeFor[*ast.BaseParameterExpr](),
		reflect.TypeFor[*ast.BaseBinaryExpr](),
		reflect.TypeFor[*ast.BaseUnaryExpr](),
		reflect.TypeFor[*ast.BaseRowExpr](),
	}
	for _, base := range bases {
		if _, found := base.MethodByName("Accept"); found {
			t.Errorf("%v must not dispatch Accept", base)
		}
		if _, found := base.MethodByName("Restore"); found {
			t.Errorf("%v must not dispatch Restore", base)
		}
		// reflect only lists exported methods, so the unexported expression
		// marker is proven by the outer fixture, which inherits it through
		// the base chain.
		if base.Implements(exprNode) {
			t.Errorf("%v must not satisfy ast.ExprNode", base)
		}
	}
	outer := reflect.TypeFor[*binaryBaseExpr]()
	if !outer.Implements(exprNode) {
		t.Errorf("%v must satisfy ast.ExprNode", outer)
	}
	if _, found := outer.MethodByName("Accept"); !found {
		t.Errorf("%v must dispatch Accept", outer)
	}
	if _, found := outer.MethodByName("Restore"); !found {
		t.Errorf("%v must dispatch Restore", outer)
	}
}

func TestBaseCallExprQualifiedName(t *testing.T) {
	t.Parallel()
	name := ast.QualifiedName{Parts: []lex.Ident{{Name: "mock_schema"}, {Name: "mock_func"}},
		Span: loc.Span{Start: 0, End: 21}}
	call := ast.BaseCallExpr{Name: name}
	field, _ := reflect.TypeFor[ast.BaseCallExpr]().FieldByName("Name")
	if field.Type != reflect.TypeFor[ast.QualifiedName]() || !reflect.DeepEqual(call.Name, name) {
		t.Fatal("call names must retain qualified parts and their range")
	}
}

func TestLiteralKindSequence(t *testing.T) {
	kinds := []ast.LiteralKind{
		ast.LiteralUnknown,
		ast.LiteralNull,
		ast.LiteralBoolean,
		ast.LiteralInteger,
		ast.LiteralFloat,
		ast.LiteralDecimal,
		ast.LiteralString,
		ast.LiteralBytes,
		ast.LiteralHex,
		ast.LiteralBit,
	}
	for i, kind := range kinds {
		if uint8(kind) != uint8(i) {
			t.Errorf("%v must be %d, got %d", kind, i, uint8(kind))
		}
	}
}

func TestBaseLiteralExprKeepsSpelling(t *testing.T) {
	const highPrecision = "123456789012345678901234567890.1234567890123456789012345678901234567890"
	cases := []struct {
		name  string
		kind  ast.LiteralKind
		raw   string
		value string
	}{
		{"null", ast.LiteralNull, "NULL", ""},
		{"boolean", ast.LiteralBoolean, "true", "true"},
		{"high precision decimal", ast.LiteralDecimal, highPrecision, highPrecision},
		{"leading zeros", ast.LiteralInteger, "007", "7"},
		{"trailing zeros", ast.LiteralFloat, "1.500", "1.5"},
		{"hex", ast.LiteralHex, "0x1F", "1f"},
		{"escaped string", ast.LiteralString, "'a''b'", "a'b"},
		{"bytes", ast.LiteralBytes, "B'0101'", "0101"},
		{"bit", ast.LiteralBit, "b'1010'", "1010"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lit := &ast.BaseLiteralExpr{Kind: tc.kind, Raw: tc.raw, Value: tc.value}
			if lit.Kind != tc.kind {
				t.Fatalf("kind changed: got %v want %v", lit.Kind, tc.kind)
			}
			if lit.Raw != tc.raw {
				t.Fatalf("raw spelling changed: got %q want %q", lit.Raw, tc.raw)
			}
			if lit.Value != tc.value {
				t.Fatalf("value changed: got %q want %q", lit.Value, tc.value)
			}
		})
	}
}

func TestExpressionBaseSpanStates(t *testing.T) {
	bases := []func() spanHolder{
		func() spanHolder { return &ast.BaseLiteralExpr{} },
		func() spanHolder { return &ast.BaseParameterExpr{} },
		func() spanHolder { return &ast.BaseBinaryExpr{} },
		func() spanHolder { return &ast.BaseUnaryExpr{} },
		func() spanHolder { return &ast.BaseRowExpr{} },
		func() spanHolder { return &binaryBaseExpr{} },
	}
	for _, newBase := range bases {
		node := newBase()
		if node.Span() != loc.InvalidSpan() {
			t.Fatalf("%T must start with an unknown span", node)
		}
		node.SetSpan(loc.Span{})
		if node.Span() != (loc.Span{}) {
			t.Fatalf("%T lost the known empty range", node)
		}
		node.SetSpan(loc.Span{Start: 4, End: 4})
		if node.Span() != (loc.Span{Start: 4, End: 4}) {
			t.Fatalf("%T lost the valid empty range", node)
		}
		node.SetSpan(loc.Span{Start: 5, End: 2})
		if node.Span() != loc.InvalidSpan() {
			t.Fatalf("%T must fall back to an unknown span", node)
		}
	}
}

// TestBaseParameterExprIndexIsIndependentOfSpan proves that a parameter's
// zero-based numbering never mixes with its source span.
func TestBaseParameterExprIndexIsIndependentOfSpan(t *testing.T) {
	unknown := &ast.BaseParameterExpr{Index: -1}
	unknown.SetSpan(loc.Span{Start: 0, End: 3})
	if unknown.Index != -1 {
		t.Fatalf("index changed with the span: got %d want -1", unknown.Index)
	}
	if unknown.Span() != (loc.Span{Start: 0, End: 3}) {
		t.Fatalf("span changed with the index: got %v", unknown.Span())
	}
	unknown.SetSpan(loc.InvalidSpan())
	if unknown.Index != -1 || unknown.Span() != loc.InvalidSpan() {
		t.Fatalf("unknown index and unknown span must stay independent")
	}

	zero := &ast.BaseParameterExpr{Index: 0}
	if zero.Span() != loc.InvalidSpan() {
		t.Fatalf("index 0 must not imply a known span")
	}
	zero.SetSpan(loc.Span{Start: 7, End: 7})
	if zero.Index != 0 {
		t.Fatalf("index 0 must stay distinct from the unknown index")
	}
	if zero.Span() != (loc.Span{Start: 7, End: 7}) {
		t.Fatalf("span must stay independent of the index")
	}

	large := &ast.BaseParameterExpr{Index: 1000}
	large.SetSpan(loc.Span{Start: 0, End: 2})
	if large.Index != 1000 || large.Span() != (loc.Span{Start: 0, End: 2}) {
		t.Fatalf("index must not carry byte offsets")
	}
}

func TestNewBaseParameterExprKeepsIndex(t *testing.T) {
	unknown := ast.NewBaseParameterExpr(ast.ParameterIndexUnknown)
	if unknown.Index != -1 {
		t.Fatalf("unknown index = %d, want -1", unknown.Index)
	}
	first := ast.NewBaseParameterExpr(0)
	if first.Index != 0 {
		t.Fatalf("first index = %d, want 0", first.Index)
	}
}

// TestBaseUnaryExprCarriesExpr checks that the unary base keeps its child
// expression slot and allows it to stay empty.
func TestBaseUnaryExprCarriesExpr(t *testing.T) {
	child := &dialectExpr{}
	unary := &ast.BaseUnaryExpr{Expr: child}
	if unary.Expr != ast.ExprNode(child) {
		t.Fatal("unary base must keep the child expression")
	}
	unary.Expr = nil
	if unary.Expr != nil {
		t.Fatal("unary base must allow an empty child slot")
	}
}

func TestBinaryBaseTraversalKeepsOuterIdentity(t *testing.T) {
	left := &dialectExpr{}
	right := &dialectExpr{}
	root := &binaryBaseExpr{BaseBinaryExpr: ast.BaseBinaryExpr{Left: left, Right: right}}

	var entered, departed []ast.Node
	updated, ok := root.Accept(visitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			entered = append(entered, n)
			return n, false
		},
		leave: func(n ast.Node) (ast.Node, bool) {
			departed = append(departed, n)
			return n, true
		},
	})
	if !ok {
		t.Fatal("traversal must succeed")
	}
	if updated != ast.Node(root) {
		t.Fatalf("outer identity lost: got %T", updated)
	}
	if !sameOrder(entered, []ast.Node{ast.Node(root), ast.Node(left), ast.Node(right)}) {
		t.Fatalf("wrong enter order: %v", entered)
	}
	if !sameOrder(departed, []ast.Node{ast.Node(left), ast.Node(right), ast.Node(root)}) {
		t.Fatalf("wrong leave order: %v", departed)
	}
}

func TestBinaryBaseReplacesVisitedChildren(t *testing.T) {
	left := &dialectExpr{}
	right := &dialectExpr{}
	root := &binaryBaseExpr{BaseBinaryExpr: ast.BaseBinaryExpr{Left: left, Right: right}}
	leftReplacement := &dialectExpr{}
	rightReplacement := &dialectExpr{}

	updated, ok := root.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
		switch n {
		case ast.Node(left):
			return leftReplacement, true
		case ast.Node(right):
			return rightReplacement, true
		}
		return n, true
	}})
	if !ok || updated != ast.Node(root) {
		t.Fatal("replacement must keep the outer node")
	}
	if root.Left != ast.ExprNode(leftReplacement) {
		t.Fatal("left slot was not replaced")
	}
	if root.Right != ast.ExprNode(rightReplacement) {
		t.Fatal("right slot was not replaced")
	}
}

func TestBinaryBaseSkipsSubtrees(t *testing.T) {
	t.Run("outer skip", func(t *testing.T) {
		left := &dialectExpr{Child: &dialectExpr{}}
		right := &dialectExpr{}
		root := &binaryBaseExpr{BaseBinaryExpr: ast.BaseBinaryExpr{Left: left, Right: right}}

		var visited []ast.Node
		updated, ok := root.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, true
		}})
		if !ok || updated != ast.Node(root) {
			t.Fatal("skip must keep the outer node")
		}
		if !sameOrder(visited, []ast.Node{ast.Node(root)}) {
			t.Fatalf("skipped subtree must not be visited: %v", visited)
		}
		if root.Left != ast.ExprNode(left) || root.Right != ast.ExprNode(right) {
			t.Fatal("skip must not change the slots")
		}
	})

	t.Run("child skip", func(t *testing.T) {
		grandchild := &dialectExpr{}
		left := &dialectExpr{Child: grandchild}
		right := &dialectExpr{}
		root := &binaryBaseExpr{BaseBinaryExpr: ast.BaseBinaryExpr{Left: left, Right: right}}

		var visited []ast.Node
		updated, ok := root.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, n == ast.Node(left)
		}})
		if !ok || updated != ast.Node(root) {
			t.Fatal("child skip must keep the outer node")
		}
		if !sameOrder(visited, []ast.Node{ast.Node(root), ast.Node(left), ast.Node(right)}) {
			t.Fatalf("skipped child subtree was visited: %v", visited)
		}
	})
}

func TestBinaryBasePropagatesChildStop(t *testing.T) {
	left := &dialectExpr{}
	right := &dialectExpr{}
	root := &binaryBaseExpr{BaseBinaryExpr: ast.BaseBinaryExpr{Left: left, Right: right}}

	var visited []ast.Node
	updated, ok := root.Accept(visitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, false
		},
		leave: func(n ast.Node) (ast.Node, bool) {
			return n, n != ast.Node(left)
		},
	})
	if ok {
		t.Fatal("child stop must propagate")
	}
	if updated != ast.Node(root) {
		t.Fatal("stop must keep the outer node")
	}
	if !sameOrder(visited, []ast.Node{ast.Node(root), ast.Node(left)}) {
		t.Fatalf("traversal must stop at the failing child: %v", visited)
	}
}

func TestBinaryBaseRestoresThroughItsOwnDispatch(t *testing.T) {
	var out bytes.Buffer
	root := &binaryBaseExpr{BaseBinaryExpr: ast.BaseBinaryExpr{
		Left:  &dialectExpr{},
		Right: &dialectExpr{},
	}}
	ctx := format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)
	if err := root.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if out.String() != "(NULL+NULL)" {
		t.Fatalf("unexpected restore output: %q", out.String())
	}
}
