// Package ast defines pure syntax contracts shared by SQL dialects.
package ast

import (
	"reflect"

	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/loc"
)

type Node interface {
	Accept(Visitor) (Node, bool)
	Restore(*format.RestoreCtx) error
	Text() string
	SetText(string)
	Span() loc.Span
	SetSpan(loc.Span)
}

func nodePresent(node Node) bool {
	if node == nil {
		return false
	}
	value := reflect.ValueOf(node)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return !value.IsNil()
	default:
		return true
	}
}

type ExprNode interface {
	Node
	expression()
}

type StmtNode interface {
	Node
	statement()
}

type DMLNode interface {
	StmtNode
	dmlStatement()
}

type DDLNode interface {
	StmtNode
	ddlStatement()
}

type ResultSetNode interface {
	Node
	resultSet()
}

type QueryNode interface {
	StmtNode
	ResultSetNode
}

// Enter must preserve the concrete type, even when skipChildren is true.
// A true Enter result skips children and still calls Leave; it does not stop.
// Leave may replace a node with one accepted by its parent slot.
// A false Leave or Accept result stops traversal and is propagated by parents.
// Public concrete nodes return (nil, false) for nil receivers without visiting.
// Absent optional slots are leaves; typed-nil slots, missing required bodies,
// and nil collection entries stop traversal unless their parent is skipped.
type Visitor interface {
	Enter(Node) (Node, bool)
	Leave(Node) (Node, bool)
}
