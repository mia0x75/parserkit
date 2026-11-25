package ast

import (
	"fmt"

	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/lex"
)

// TableSourceNode exposes shared source syntax without hiding dialect identity.
type TableSourceNode interface {
	ResultSetNode
	TableSourceCore() *BaseTableSource
}

// BaseTableSource holds source syntax without dispatching on a dialect's behalf.
// Source stays one result-set node; dialect nodes that already bracket their
// own spelling (for example a subquery expression) must not implement
// QueryNode, so a bare SELECT or VALUES source gets its parentheses here.
type BaseTableSource struct {
	BaseResultSet
	Source  ResultSetNode
	Alias   lex.Ident
	Columns []lex.Ident
	Lateral bool
}

// TableSourceCore returns the shared source data, including for a nil receiver.
func (n *BaseTableSource) TableSourceCore() *BaseTableSource { return n }

// TableSource is a standard table, view, subquery or VALUES source.
type TableSource struct {
	BaseTableSource
}

// TableSourceCore returns the shared data while preserving the outer identity.
func (n *TableSource) TableSourceCore() *BaseTableSource {
	if n == nil {
		return nil
	}
	return &n.BaseTableSource
}

// Restore writes LATERAL, the source with parentheses when needed, the alias
// and optional column aliases using the caller's quote policy.
func (n *TableSource) Restore(ctx *format.RestoreCtx) error {
	return RestoreTableSource(ctx, n.TableSourceCore())
}

// RestoreTableSource writes the shared source syntax, not dialect suffixes.
// It calls the source's own Restore and does not construct a dispatch wrapper.
func RestoreTableSource(ctx *format.RestoreCtx, n *BaseTableSource) error {
	if n == nil || n.Source == nil {
		return fmt.Errorf("restore table source: missing source")
	}
	if len(n.Columns) > 0 && !n.Alias.Present() {
		return fmt.Errorf("restore table source: column aliases require an alias")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if n.Lateral {
		ctx.WriteKeyword("LATERAL")
		ctx.WritePlain(" ")
	}
	if query, ok := n.Source.(QueryNode); ok {
		ctx.WritePlain("(")
		if err := query.Restore(ctx); err != nil {
			return fmt.Errorf("restore table source query: %w", err)
		}
		ctx.WritePlain(")")
	} else if err := n.Source.Restore(ctx); err != nil {
		return fmt.Errorf("restore table source source: %w", err)
	}
	if n.Alias.Present() {
		ctx.WriteKeyword(" AS ")
		ctx.WriteIdent(n.Alias)
	}
	if len(n.Columns) > 0 {
		ctx.WritePlain(" (")
		for i, col := range n.Columns {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(col)
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept walks the source once; Enter replacements that are not result sets
// stop the traversal instead of panicking. A nil source counts as a leaf.
func (n *TableSource) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*TableSource)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.Source != nil {
		if !nodePresent(outer.Source) {
			return outer, false
		}
		updated, ok := outer.Source.Accept(v)
		if !ok {
			return outer, false
		}
		source, ok := updated.(ResultSetNode)
		if !ok || !nodePresent(source) {
			return outer, false
		}
		outer.Source = source
	}
	return v.Leave(outer)
}

var (
	_ Node            = (*TableSource)(nil)
	_ ResultSetNode   = (*TableSource)(nil)
	_ TableSourceNode = (*TableSource)(nil)
)
