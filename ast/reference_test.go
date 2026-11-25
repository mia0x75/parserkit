package ast_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
)

// TestReferenceBaseContract pins the shared BaseReferenceDef and
// BaseIndexKeyPart contract: the bases carry no Accept or Restore dispatch,
// the Core accessors return the embedded base pointer, typed nil outers
// report a nil core, and the role fields keep their declared node types.
func TestReferenceBaseContract(t *testing.T) {
	t.Parallel()

	t.Run("bases carry no dispatch", func(t *testing.T) {
		t.Parallel()
		for _, base := range []reflect.Type{
			reflect.TypeFor[ast.BaseIndexKeyPart](),
			reflect.TypeFor[ast.BaseReferenceDef](),
		} {
			for _, method := range []string{"Accept", "Restore"} {
				if m, found := base.MethodByName(method); found {
					t.Errorf("%v must not declare %s: %v", base, method, m.Type)
				}
			}
		}
	})

	t.Run("core accessors return the embedded base", func(t *testing.T) {
		t.Parallel()
		base := &ast.BaseIndexKeyPart{}
		if base.IndexKeyPartCore() != base {
			t.Fatal("IndexKeyPartCore must return the embedded base pointer")
		}
		ref := &ast.BaseReferenceDef{}
		if ref.ReferenceDefCore() != ref {
			t.Fatal("ReferenceDefCore must return the embedded base pointer")
		}
	})

	t.Run("typed nil outer reports a nil core", func(t *testing.T) {
		t.Parallel()
		if (*referenceKeyPart)(nil).IndexKeyPartCore() != nil {
			t.Fatal("typed nil key part must report a nil core")
		}
		if (*referenceDefNode)(nil).ReferenceDefCore() != nil {
			t.Fatal("typed nil reference def must report a nil core")
		}
	})

	t.Run("role fields keep their node types", func(t *testing.T) {
		t.Parallel()
		for _, slot := range []struct {
			base reflect.Type
			name string
			want reflect.Type
		}{
			{reflect.TypeFor[ast.BaseReferenceDef](), "Table", reflect.TypeFor[ast.TableRefNode]()},
			{reflect.TypeFor[ast.BaseReferenceDef](), "Columns", reflect.TypeFor[[]ast.IndexKeyPartNode]()},
			{reflect.TypeFor[ast.BaseIndexKeyPart](), "Column", reflect.TypeFor[ast.QualifiedName]()},
			{reflect.TypeFor[ast.BaseIndexKeyPart](), "Expr", reflect.TypeFor[ast.ExprNode]()},
			{reflect.TypeFor[ast.BaseIndexKeyPart](), "PrefixLength", reflect.TypeFor[int]()},
			{reflect.TypeFor[ast.BaseIndexKeyPart](), "Desc", reflect.TypeFor[bool]()},
		} {
			if field, ok := slot.base.FieldByName(slot.name); !ok || field.Type != slot.want {
				t.Errorf("%v.%s must have type %v", slot.base, slot.name, slot.want)
			}
		}
	})
}

// referenceKeyPart mirrors a dialect outer node that owns BaseIndexKeyPart.
type referenceKeyPart struct {
	ast.BaseIndexKeyPart
}

// IndexKeyPartCore implements the shared key role contract: typed nil
// reports a nil core instead of crashing.
func (n *referenceKeyPart) IndexKeyPartCore() *ast.BaseIndexKeyPart {
	if n == nil {
		return nil
	}
	return &n.BaseIndexKeyPart
}

// Accept implements the shared Node contract.
func (n *referenceKeyPart) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*referenceKeyPart)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

// Restore implements the shared Node contract.
func (n *referenceKeyPart) Restore(ctx *format.RestoreCtx) error {
	return ctx.Err()
}

// referenceDefNode mirrors a dialect outer node that owns BaseReferenceDef.
type referenceDefNode struct {
	ast.BaseReferenceDef
}

// ReferenceDefCore implements the shared reference role contract: typed nil
// reports a nil core instead of crashing.
func (n *referenceDefNode) ReferenceDefCore() *ast.BaseReferenceDef {
	if n == nil {
		return nil
	}
	return &n.BaseReferenceDef
}

// Accept implements the shared Node contract.
func (n *referenceDefNode) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*referenceDefNode)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

// Restore implements the shared Node contract.
func (n *referenceDefNode) Restore(ctx *format.RestoreCtx) error {
	return ctx.Err()
}

var (
	_ ast.IndexKeyPartNode = (*referenceKeyPart)(nil)
	_ ast.ReferenceDefNode = (*referenceDefNode)(nil)
)

// TestReferenceBaseRoles proves the shared role interfaces accept any
// conforming node, not only concrete dialect types.
func TestReferenceBaseRoles(t *testing.T) {
	t.Parallel()

	part := &referenceKeyPart{}
	def := &referenceDefNode{
		BaseReferenceDef: ast.BaseReferenceDef{
			Table:   &tableRefExpr{},
			Columns: []ast.IndexKeyPartNode{&referenceKeyPart{}},
		},
	}
	var key ast.IndexKeyPartNode = part
	if key.IndexKeyPartCore() == nil {
		t.Fatal("conforming key node lost its core")
	}
	var ref ast.ReferenceDefNode = def
	if ref.ReferenceDefCore() == nil {
		t.Fatal("conforming reference node lost its core")
	}
}
