package ast_test

import (
	"testing"

	"github.com/mia0x75/parserkit/ast"
)

// TestOperatorEnumsAreUnique checks that every public operator enum value is
// distinct and that OperatorUnknown is the zero value.
func TestOperatorEnumsAreUnique(t *testing.T) {
	ops := []ast.Operator{
		ast.OperatorUnknown,
		ast.OperatorLogicAnd,
		ast.OperatorLeftShift,
		ast.OperatorRightShift,
		ast.OperatorLogicOr,
		ast.OperatorGE,
		ast.OperatorLE,
		ast.OperatorEQ,
		ast.OperatorNE,
		ast.OperatorLT,
		ast.OperatorGT,
		ast.OperatorPlus,
		ast.OperatorMinus,
		ast.OperatorAnd,
		ast.OperatorOr,
		ast.OperatorMod,
		ast.OperatorXor,
		ast.OperatorDiv,
		ast.OperatorMul,
		ast.OperatorNot,
		ast.OperatorBitNeg,
		ast.OperatorIntDiv,
		ast.OperatorLogicXor,
		ast.OperatorNullEQ,
		ast.OperatorIn,
		ast.OperatorLike,
		ast.OperatorCase,
		ast.OperatorRegexp,
		ast.OperatorIsNull,
		ast.OperatorIsTruth,
		ast.OperatorIsFalsity,
		ast.OperatorJSONExtract,
		ast.OperatorJSONExtractText,
		ast.OperatorJSONPathExtract,
		ast.OperatorJSONPathExtractText,
		ast.OperatorJSONContains,
		ast.OperatorJSONContained,
		ast.OperatorJSONExists,
		ast.OperatorJSONAllExists,
		ast.OperatorJSONAnyExists,
	}
	if ast.OperatorUnknown != 0 {
		t.Fatal("OperatorUnknown must be the zero value")
	}
	seen := make(map[ast.Operator]struct{}, len(ops))
	for _, op := range ops {
		if _, dup := seen[op]; dup {
			t.Fatalf("operator %v appears more than once", op)
		}
		seen[op] = struct{}{}
	}
}

// TestBinaryAndUnaryExprsKeepOperator proves that binary and unary expression
// bases keep the operator enum they were assigned.
func TestBinaryAndUnaryExprsKeepOperator(t *testing.T) {
	bin := &ast.BaseBinaryExpr{Op: ast.OperatorPlus, Left: &dialectExpr{}, Right: &dialectExpr{}}
	if bin.Op != ast.OperatorPlus {
		t.Fatalf("binary expr lost its operator: got %v", bin.Op)
	}

	unary := &ast.BaseUnaryExpr{Op: ast.OperatorNot, Expr: &dialectExpr{}}
	if unary.Op != ast.OperatorNot {
		t.Fatalf("unary expr lost its operator: got %v", unary.Op)
	}
}
