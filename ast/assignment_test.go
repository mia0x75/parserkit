package ast_test

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/loc"
)

var errAssignmentExpr = errors.New("assignment expr restore failed")

// assignmentErrorExpr fails on purpose to keep child root causes visible.
type assignmentErrorExpr struct{ ast.BaseExpr }

func (n *assignmentErrorExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *assignmentErrorExpr) Restore(ctx *format.RestoreCtx) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return errAssignmentExpr
}

// TestAssignmentNodeContract pins that Assignment is a plain Node with no
// expression or statement marker and exactly three fields.
func TestAssignmentNodeContract(t *testing.T) {
	assignmentType := reflect.TypeFor[*ast.Assignment]()
	nodeType := reflect.TypeFor[ast.Node]()
	if !assignmentType.Implements(nodeType) {
		t.Fatal("Assignment must satisfy ast.Node")
	}
	if assignmentType.Implements(reflect.TypeFor[ast.ExprNode]()) {
		t.Fatal("Assignment must not satisfy ast.ExprNode")
	}
	if assignmentType.Implements(reflect.TypeFor[ast.StmtNode]()) {
		t.Fatal("Assignment must not satisfy ast.StmtNode")
	}
	// A base without its own dispatch never satisfies the full Node contract,
	// so only an outer node that carries Accept and Restore may hold a slot.
	if reflect.TypeFor[*ast.BaseNode]().Implements(nodeType) {
		t.Fatal("a bare BaseNode must not satisfy ast.Node")
	}
	assignmentStruct := reflect.TypeFor[ast.Assignment]()
	if assignmentStruct.NumField() != 3 {
		t.Errorf("Assignment must have exactly 3 fields, got %d", assignmentStruct.NumField())
	}
	if f, found := assignmentStruct.FieldByName("Target"); !found || f.Type != reflect.TypeFor[ast.ExprNode]() {
		t.Errorf("Assignment.Target must be ast.ExprNode, got %v", f.Type)
	}
	if f, found := assignmentStruct.FieldByName("Expr"); !found || f.Type != reflect.TypeFor[ast.ExprNode]() {
		t.Errorf("Assignment.Expr must be ast.ExprNode, got %v", f.Type)
	}
}

// TestAssignmentSpanAndTextIndependent shows the embedded base keeps span and
// text independent of the expression slots.
func TestAssignmentSpanAndTextIndependent(t *testing.T) {
	if (&ast.Assignment{}).Span() != loc.InvalidSpan() {
		t.Fatal("zero assignment must have an unknown span")
	}
	assignment := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
	assignment.SetSpan(loc.Span{Start: 1, End: 9})
	if assignment.Span() != (loc.Span{Start: 1, End: 9}) {
		t.Fatal("the embedded base must inherit the outer span")
	}
	assignment.SetSpan(loc.Span{})
	if assignment.Span() != (loc.Span{}) {
		t.Fatal("the known empty range must be kept")
	}
	assignment.SetText("sql")
	if assignment.Text() != "sql" || assignment.Span() != (loc.Span{}) {
		t.Fatal("text and span must remain independent")
	}
}

// TestAssignmentRestoresTargetAndValue writes target=value through each side's
// own restore rules and keeps child and writer root causes in the chain.
func TestAssignmentRestoresTargetAndValue(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		var out strings.Builder
		assignment := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
		if err := assignment.Restore(format.NewRestoreCtx(&out, format.DialectConfig{}, format.DefaultRestoreFlags)); err != nil {
			t.Fatal(err)
		}
		if got := out.String(); got != "NULL=NULL" {
			t.Fatalf("restore = %q, want %q", got, "NULL=NULL")
		}
	})

	t.Run("missing target", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Expr: &dialectExpr{}}
		err := assignment.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("missing target must fail")
		}
		if !strings.Contains(err.Error(), "restore assignment: missing target or expression") {
			t.Fatalf("root cause must identify the missing slot: %v", err)
		}
	})

	t.Run("missing value", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Target: &dialectExpr{}}
		err := assignment.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if err == nil {
			t.Fatal("missing value must fail")
		}
		if !strings.Contains(err.Error(), "restore assignment: missing target or expression") {
			t.Fatalf("root cause must identify the missing slot: %v", err)
		}
	})

	t.Run("nil receiver", func(t *testing.T) {
		t.Parallel()
		var assignment *ast.Assignment
		if err := assignment.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags)); err == nil {
			t.Fatal("a nil assignment must fail")
		}
	})

	t.Run("failing target", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Target: &assignmentErrorExpr{}, Expr: &dialectExpr{}}
		err := assignment.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, errAssignmentExpr) {
			t.Fatalf("target root cause must stay in the chain: %v", err)
		}
		if !strings.Contains(err.Error(), "restore assignment target") {
			t.Fatalf("the failing side must stay named in the chain: %v", err)
		}
	})

	t.Run("failing value", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Target: &dialectExpr{}, Expr: &assignmentErrorExpr{}}
		err := assignment.Restore(format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, errAssignmentExpr) {
			t.Fatalf("value root cause must stay in the chain: %v", err)
		}
		if !strings.Contains(err.Error(), "restore assignment expression") {
			t.Fatalf("the failing side must stay named in the chain: %v", err)
		}
	})

	t.Run("nil writer", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
		err := assignment.Restore(format.NewRestoreCtx(nil, format.DialectConfig{}, format.DefaultRestoreFlags))
		if !errors.Is(err, format.ErrNilWriter) {
			t.Fatalf("writer sentinel must stay in the chain: %v", err)
		}
	})

	t.Run("short writer", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
		ctx := format.NewRestoreCtx(io.Discard, format.DialectConfig{}, format.DefaultRestoreFlags)
		short := format.NewRestoreWriter(&shortWriter{})
		ctx.Writer = short
		err := assignment.Restore(ctx)
		if !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("short write must stay in the chain: %v", err)
		}
	})
}

// TestAssignmentTraversalContract pins target-first order, safe slot
// replacement, enter substitution visiting the new subtree, and the
// skip/stop/wrong-type/typed-nil rules on both sides.
func TestAssignmentTraversalContract(t *testing.T) {
	t.Run("order and identity", func(t *testing.T) {
		t.Parallel()
		target := &dialectExpr{}
		expr := &dialectExpr{}
		assignment := &ast.Assignment{Target: target, Expr: expr}
		var visited []ast.Node
		updated, ok := assignment.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, false
		}})
		if !ok || updated != ast.Node(assignment) {
			t.Fatal("traversal must keep the outer assignment")
		}
		if !sameOrder(visited, []ast.Node{assignment, target, expr}) {
			t.Fatalf("enter order = %v", visited)
		}
	})

	t.Run("replace both sides", func(t *testing.T) {
		t.Parallel()
		target := &dialectExpr{}
		expr := &dialectExpr{}
		assignment := &ast.Assignment{Target: target, Expr: expr}
		newTarget := &dialectExpr{}
		newExpr := &dialectExpr{}
		updated, ok := assignment.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			switch n {
			case ast.Node(target):
				return newTarget, true
			case ast.Node(expr):
				return newExpr, true
			}
			return n, true
		}})
		if !ok || updated != ast.Node(assignment) {
			t.Fatal("replacement must keep the outer assignment")
		}
		if assignment.Target != ast.ExprNode(newTarget) || assignment.Expr != ast.ExprNode(newExpr) {
			t.Fatal("both slots must hold their replacements")
		}
	})

	t.Run("enter outer replacement visits new subtree", func(t *testing.T) {
		t.Parallel()
		outer := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
		updatedOuter := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
		var visited []ast.Node
		updated, ok := outer.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			if n == ast.Node(outer) {
				return updatedOuter, false
			}
			return n, false
		}})
		if !ok || updated != ast.Node(updatedOuter) {
			t.Fatal("enter must substitute the outer node")
		}
		// The visitor entered the outer assignment, so the replacement's
		// children are visited, not the original's.
		want := []ast.Node{outer, updatedOuter.Target, updatedOuter.Expr}
		if !sameOrder(visited, want) {
			t.Fatalf("enter replacement must visit the new subtree: %v", visited)
		}
	})

	t.Run("skip", func(t *testing.T) {
		t.Parallel()
		target := &dialectExpr{}
		expr := &dialectExpr{}
		assignment := &ast.Assignment{Target: target, Expr: expr}
		var visited []ast.Node
		updated, ok := assignment.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = append(visited, n)
			return n, n == ast.Node(assignment)
		}})
		if !ok || updated != ast.Node(assignment) {
			t.Fatal("skip must keep the outer assignment")
		}
		if !sameOrder(visited, []ast.Node{assignment}) {
			t.Fatalf("skipped children must not be visited: %v", visited)
		}
		if assignment.Target != ast.ExprNode(target) || assignment.Expr != ast.ExprNode(expr) {
			t.Fatal("skip must not change the slots")
		}
	})

	t.Run("stop at target", func(t *testing.T) {
		t.Parallel()
		target := &dialectExpr{}
		expr := &dialectExpr{}
		assignment := &ast.Assignment{Target: target, Expr: expr}
		valueEntered := false
		_, ok := assignment.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if n == ast.Node(expr) {
				valueEntered = true
				return n, n != ast.Node(expr)
			}
			return n, n != ast.Node(target)
		}})
		if ok {
			t.Fatal("stop at the target must propagate")
		}
		if valueEntered {
			t.Fatal("stop must not visit the value side")
		}
	})

	t.Run("stop at value", func(t *testing.T) {
		t.Parallel()
		target := &dialectExpr{}
		expr := &dialectExpr{}
		assignment := &ast.Assignment{Target: target, Expr: expr}
		_, ok := assignment.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			return n, n != ast.Node(expr)
		}})
		if ok {
			t.Fatal("stop at the value must propagate")
		}
		if assignment.Target != ast.ExprNode(target) {
			t.Fatal("the target slot must keep its old child")
		}
	})

	t.Run("wrong type outer", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
		_, ok := assignment.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			if n == ast.Node(assignment) {
				return &dialectExpr{}, false
			}
			return n, false
		}})
		if ok {
			t.Fatal("a wrong-type Enter on the outer assignment must stop the traversal")
		}
	})

	t.Run("typed-nil outer", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
		var nilAssignment *ast.Assignment
		updated, ok := assignment.Accept(visitor{enter: func(n ast.Node) (ast.Node, bool) {
			if n == ast.Node(assignment) {
				return ast.Node(nilAssignment), false
			}
			return n, false
		}})
		if ok {
			t.Fatal("a typed-nil Enter on the outer assignment must stop the traversal")
		}
		if updated != ast.Node(nilAssignment) {
			t.Fatalf("the typed-nil outer must stay in the result: got %T", updated)
		}
	})

	t.Run("reject target replacement", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
		updated, ok := assignment.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if _, isExpr := n.(*dialectExpr); isExpr {
				return &ast.Assignment{}, true
			}
			return n, true
		}})
		if ok {
			t.Fatal("a non-expression target replacement must stop the traversal")
		}
		if assignment.Target == nil {
			t.Fatal("the target slot must keep its old child after rejection")
		}
		if updated != ast.Node(assignment) {
			t.Fatal("the outer assignment must stay in the result")
		}
	})

	t.Run("reject value replacement", func(t *testing.T) {
		t.Parallel()
		target := &dialectExpr{}
		expr := &dialectExpr{}
		assignment := &ast.Assignment{Target: target, Expr: expr}
		updated, ok := assignment.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if n == ast.Node(expr) {
				return &ast.Assignment{}, true
			}
			return n, true
		}})
		if ok {
			t.Fatal("a non-expression value replacement must stop the traversal")
		}
		if assignment.Target != ast.ExprNode(target) {
			t.Fatal("the target slot must keep its old child")
		}
		if assignment.Expr != ast.ExprNode(expr) {
			t.Fatal("the value slot must keep its old child after rejection")
		}
		if updated != ast.Node(assignment) {
			t.Fatal("the outer assignment must stay in the result")
		}
	})

	t.Run("nil target replacement", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{Target: &dialectExpr{}, Expr: &dialectExpr{}}
		updated, ok := assignment.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if _, isExpr := n.(*dialectExpr); isExpr {
				return nil, true
			}
			return n, true
		}})
		if ok {
			t.Fatal("a nil target replacement must stop the traversal")
		}
		if assignment.Target == nil {
			t.Fatal("the target slot must keep its old child after rejection")
		}
		if updated != ast.Node(assignment) {
			t.Fatal("the outer assignment must stay in the result")
		}
	})

	t.Run("nil value replacement", func(t *testing.T) {
		t.Parallel()
		target := &dialectExpr{}
		expr := &dialectExpr{}
		assignment := &ast.Assignment{Target: target, Expr: expr}
		updated, ok := assignment.Accept(visitor{leave: func(n ast.Node) (ast.Node, bool) {
			if n == ast.Node(expr) {
				return nil, true
			}
			return n, true
		}})
		if ok {
			t.Fatal("a nil value replacement must stop the traversal")
		}
		if assignment.Target != ast.ExprNode(target) {
			t.Fatal("the target slot must keep its old child")
		}
		if assignment.Expr != ast.ExprNode(expr) {
			t.Fatal("the value slot must keep its old child after rejection")
		}
		if updated != ast.Node(assignment) {
			t.Fatal("the outer assignment must stay in the result")
		}
	})

	t.Run("nil receiver does not panic", func(t *testing.T) {
		t.Parallel()
		var assignment *ast.Assignment
		updated, ok := assignment.Accept(visitor{})
		if ok {
			t.Fatal("a nil receiver must stop the traversal")
		}
		if updated != nil {
			t.Fatal("a nil receiver must return a nil interface")
		}
	})

	t.Run("nil children keep outer", func(t *testing.T) {
		t.Parallel()
		assignment := &ast.Assignment{}
		updated, ok := assignment.Accept(visitor{})
		if !ok || updated != ast.Node(assignment) {
			t.Fatal("nil children must keep the outer assignment")
		}
	})
}

// shortWriter fails every non-empty write with a short count.
type shortWriter struct{}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, io.ErrShortWrite
}
