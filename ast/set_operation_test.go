package ast

import (
	"reflect"
	"testing"
)

func TestSetOperationTypeString(t *testing.T) {
	tests := []struct {
		operation SetOperationType
		want      string
	}{
		{SetOperationUnknown, ""},
		{SetOperationUnion, "UNION"},
		{SetOperationUnionAll, "UNION ALL"},
		{SetOperationExcept, "EXCEPT"},
		{SetOperationExceptAll, "EXCEPT ALL"},
		{SetOperationIntersect, "INTERSECT"},
		{SetOperationIntersectAll, "INTERSECT ALL"},
	}
	for _, tt := range tests {
		if got := tt.operation.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.operation, got, tt.want)
		}
	}
}

func TestSetOperationBasesHaveNoDispatchMethods(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(BaseSetOperation{}),
		reflect.TypeOf(BaseQueryTail{}),
		reflect.TypeOf(BaseQueryTailStmt{}),
	} {
		if _, ok := typ.MethodByName("Accept"); ok {
			t.Fatalf("%v unexpectedly implements Accept", typ)
		}
		if _, ok := typ.MethodByName("Restore"); ok {
			t.Fatalf("%v unexpectedly implements Restore", typ)
		}
	}
}

func TestSetOperationCoreNilSafety(t *testing.T) {
	var operation *BaseSetOperation
	if operation.SetOperationCore() != nil {
		t.Fatal("nil BaseSetOperation core is not nil")
	}
	var wrapper *BaseQueryTailStmt
	if wrapper.QueryTailStmtCore() != nil {
		t.Fatal("nil BaseQueryTailStmt core is not nil")
	}
}
