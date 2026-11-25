package ast

// BaseCallExpr stores the callee and expression arguments. Dialect-specific
// modifiers are not arguments and remain with the concrete node.
type BaseCallExpr struct {
	BaseExpr
	Name QualifiedName
	Args []ExprNode
}

// LiteralKind describes SQL syntax, not an evaluated value or inferred type.
type LiteralKind uint8

const (
	LiteralUnknown LiteralKind = iota
	LiteralNull
	LiteralBoolean
	LiteralInteger
	LiteralFloat
	LiteralDecimal
	LiteralString
	LiteralBytes
	LiteralHex
	LiteralBit
)

// BaseLiteralExpr keeps numeric spelling as text to avoid precision loss.
// Raw is the original spelling; Value is decoded text or a canonical spelling.
// Hex and bit values contain digits without prefixes or quotes.
// Dialect-specific introducers and delimiters belong to the outer node.
type BaseLiteralExpr struct {
	BaseExpr
	Kind  LiteralKind
	Raw   string
	Value string
}

// ParameterIndexUnknown marks a parameter whose source order is not known yet.
const ParameterIndexUnknown = -1

// BaseParameterExpr separates zero-based numbering from the source span.
// The zero Index is a valid first parameter; use ParameterIndexUnknown when
// the index is not known at construction time.
type BaseParameterExpr struct {
	BaseExpr
	Index int
}

// NewBaseParameterExpr creates a syntax parameter with the given source index.
func NewBaseParameterExpr(index int) *BaseParameterExpr {
	return &BaseParameterExpr{Index: index}
}

type BaseBinaryExpr struct {
	BaseExpr
	Op    Operator
	Left  ExprNode
	Right ExprNode
}

type BaseUnaryExpr struct {
	BaseExpr
	Op   Operator
	Expr ExprNode
}

type RowNode interface {
	ExprNode
	RowCore() *BaseRowExpr
}

// The outer node preserves dialect-specific row delimiters and keywords.
type BaseRowExpr struct {
	BaseExpr
	Items []ExprNode
}

func (n *BaseRowExpr) RowCore() *BaseRowExpr { return n }
