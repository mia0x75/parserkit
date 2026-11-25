package ast_test

import (
	"testing"

	"github.com/mia0x75/parserkit/ast"
)

func TestWindowFrameZeroValues(t *testing.T) {
	t.Parallel()
	var frame ast.BaseWindowFrameClause
	var bound ast.BaseWindowFrameBound
	if frame.Type != ast.WindowFrameUnknown || bound.Type != ast.WindowFrameBoundUnknown {
		t.Fatal("absent frame units and directions must remain unknown")
	}
	if frame.Exclusion != ast.WindowFrameExcludeNone || bound.Unbounded {
		t.Fatal("zero exclusion and boundedness must retain their omission semantics")
	}
	for _, unit := range []ast.WindowFrameType{ast.WindowFrameRows, ast.WindowFrameRange, ast.WindowFrameGroups} {
		if unit == frame.Type {
			t.Fatal("a valid frame unit must differ from the zero value")
		}
	}
	for _, direction := range []ast.WindowFrameBoundType{ast.WindowFrameFollowing, ast.WindowFramePreceding, ast.WindowFrameCurrentRow} {
		if direction == bound.Type {
			t.Fatal("a valid bound direction must differ from the zero value")
		}
	}
}
