package ast

import "github.com/mia0x75/parserkit/lex"

// BaseWindowPartitionItem stores the portable expression of a partition item.
// Dialects own traversal, restoration, and dialect-specific modifiers.
type BaseWindowPartitionItem struct {
	BaseNode
	Expr ExprNode
}

func (n *BaseWindowPartitionItem) WindowPartitionItemCore() *BaseWindowPartitionItem { return n }

// WindowPartitionItemNode admits dialect-specific partition item modifiers.
type WindowPartitionItemNode interface {
	Node
	WindowPartitionItemCore() *BaseWindowPartitionItem
}

// BaseWindowPartitionClause stores window partition items in source order.
type BaseWindowPartitionClause struct {
	BaseNode
	Items []WindowPartitionItemNode
}

func (n *BaseWindowPartitionClause) WindowPartitionClauseCore() *BaseWindowPartitionClause {
	return n
}

// WindowPartitionClauseNode admits dialect-specific partition clause fields.
type WindowPartitionClauseNode interface {
	Node
	WindowPartitionClauseCore() *BaseWindowPartitionClause
}

// WindowFrameType identifies the frame unit.
type WindowFrameType uint8

const (
	// WindowFrameUnknown prevents absent frame units from silently becoming ROWS.
	WindowFrameUnknown WindowFrameType = iota
	// WindowFrameRows is a ROWS frame.
	WindowFrameRows
	// WindowFrameRange is a RANGE frame.
	WindowFrameRange
	// WindowFrameGroups is a GROUPS frame.
	WindowFrameGroups
)

// WindowFrameBoundType identifies a frame bound direction.
type WindowFrameBoundType uint8

const (
	// WindowFrameBoundUnknown prevents absent directions from becoming FOLLOWING.
	WindowFrameBoundUnknown WindowFrameBoundType = iota
	// WindowFrameFollowing is a FOLLOWING bound.
	WindowFrameFollowing
	// WindowFramePreceding is a PRECEDING bound.
	WindowFramePreceding
	// WindowFrameCurrentRow is a CURRENT ROW bound.
	WindowFrameCurrentRow
)

// WindowFrameExclusion identifies an EXCLUDE clause.
type WindowFrameExclusion uint8

const (
	// WindowFrameExcludeNone means no EXCLUDE clause.
	WindowFrameExcludeNone WindowFrameExclusion = iota
	// WindowFrameExcludeCurrentRow excludes the current row.
	WindowFrameExcludeCurrentRow
	// WindowFrameExcludeGroup excludes the current group.
	WindowFrameExcludeGroup
	// WindowFrameExcludeTies excludes ties.
	WindowFrameExcludeTies
	// WindowFrameExcludeNoOthers excludes no other rows.
	WindowFrameExcludeNoOthers
)

// BaseWindowFrameBound stores portable frame-bound syntax.
type BaseWindowFrameBound struct {
	BaseNode
	Type      WindowFrameBoundType
	Unbounded bool
	Expr      ExprNode
	Unit      lex.Ident
}

func (n *BaseWindowFrameBound) WindowFrameBoundCore() *BaseWindowFrameBound { return n }

// WindowFrameBoundNode admits dialect-specific frame-bound metadata.
type WindowFrameBoundNode interface {
	Node
	WindowFrameBoundCore() *BaseWindowFrameBound
}

// WindowFrameExtent stores the start and end bounds of a frame.
type WindowFrameExtent struct {
	Start WindowFrameBoundNode
	End   WindowFrameBoundNode
}

// BaseWindowFrameClause stores portable frame syntax.
type BaseWindowFrameClause struct {
	BaseNode
	Type      WindowFrameType
	Extent    WindowFrameExtent
	Exclusion WindowFrameExclusion
}

func (n *BaseWindowFrameClause) WindowFrameClauseCore() *BaseWindowFrameClause { return n }

// WindowFrameClauseNode admits dialect-specific frame-clause metadata.
type WindowFrameClauseNode interface {
	Node
	WindowFrameClauseCore() *BaseWindowFrameClause
}

// BaseWindowSpec stores portable window specification syntax.
type BaseWindowSpec struct {
	BaseNode
	Name        lex.Ident
	Ref         lex.Ident
	PartitionBy WindowPartitionClauseNode
	OrderBy     OrderByClauseNode
	Frame       WindowFrameClauseNode
	OnlyAlias   bool
}

// WindowSpecCore exposes the shared window syntax without dispatching.
func (n *BaseWindowSpec) WindowSpecCore() *BaseWindowSpec { return n }

// WindowSpecNode preserves the concrete dialect window identity.
type WindowSpecNode interface {
	Node
	WindowSpecCore() *BaseWindowSpec
}
