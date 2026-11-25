package ast

import (
	"fmt"

	"github.com/mia0x75/parserkit/format"
)

// Assignment keeps a written target expression separate from its assigned value.
// Targets may include a column, a parenthesized row, or dialect indirection.
// The dialect grammar determines which expressions are valid writable targets.
type Assignment struct {
	BaseNode
	Target ExprNode
	Expr   ExprNode
}

func (n *Assignment) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Target == nil || n.Expr == nil {
		return fmt.Errorf("restore assignment: missing target or expression")
	}
	if err := n.Target.Restore(ctx); err != nil {
		return fmt.Errorf("restore assignment target: %w", err)
	}
	ctx.WritePlain("=")
	if err := n.Expr.Restore(ctx); err != nil {
		return fmt.Errorf("restore assignment expression: %w", err)
	}
	return ctx.Err()
}

func (n *Assignment) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	assignment, ok := entered.(*Assignment)
	if !ok || assignment == nil {
		return entered, false
	}
	if skip {
		return v.Leave(assignment)
	}
	if assignment.Target != nil {
		if !nodePresent(assignment.Target) {
			return assignment, false
		}
		updated, ok := assignment.Target.Accept(v)
		if !ok {
			return assignment, false
		}
		target, ok := updated.(ExprNode)
		if !ok || !nodePresent(target) {
			return assignment, false
		}
		assignment.Target = target
	}
	if assignment.Expr != nil {
		if !nodePresent(assignment.Expr) {
			return assignment, false
		}
		updated, ok := assignment.Expr.Accept(v)
		if !ok {
			return assignment, false
		}
		expr, ok := updated.(ExprNode)
		if !ok || !nodePresent(expr) {
			return assignment, false
		}
		assignment.Expr = expr
	}
	return v.Leave(assignment)
}

var _ Node = (*Assignment)(nil)
