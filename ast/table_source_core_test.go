package ast_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/format"
	"github.com/mia0x75/parserkit/lex"
)

func TestTableSourceCoreContract(t *testing.T) {
	t.Parallel()
	var base *ast.BaseTableSource
	var standard *ast.TableSource
	if base.TableSourceCore() != nil || standard.TableSourceCore() != nil {
		t.Fatal("nil source core is not nil")
	}
	n := &ast.TableSource{BaseTableSource: ast.BaseTableSource{Source: &fakeTableSourceTable{}}}
	var role ast.TableSourceNode = n
	if role.TableSourceCore() != &n.BaseTableSource || n.BaseTableSource.TableSourceCore() != &n.BaseTableSource {
		t.Fatal("source core copied the original data")
	}
	role.TableSourceCore().Alias = lex.Ident{Name: "o"}
	if n.Alias.Name != "o" {
		t.Fatal("source core mutation did not reach the outer node")
	}
	for _, typ := range []reflect.Type{reflect.TypeFor[ast.BaseTableSource](), reflect.TypeFor[*ast.BaseTableSource]()} {
		for _, method := range []string{"Accept", "Restore"} {
			if _, ok := typ.MethodByName(method); ok {
				t.Fatalf("%v unexpectedly dispatches %s", typ, method)
			}
		}
		if typ.Implements(reflect.TypeFor[ast.Node]()) || typ.Implements(reflect.TypeFor[ast.TableSourceNode]()) {
			t.Fatalf("%v must not be a complete source node", typ)
		}
	}
}

func ExampleRestoreTableSource() {
	base := ast.BaseTableSource{Source: &fakeTableSourceTable{}, Alias: lex.Ident{Name: "o"}}
	var out strings.Builder
	if err := ast.RestoreTableSource(format.NewRestoreCtx(&out, format.DialectConfig{IdentifierQuote: lex.QuoteDouble}, format.DefaultRestoreFlags), &base); err != nil {
		panic(err)
	}
	fmt.Println(out.String())
	// Output: orders AS "o"
}
