# parserkit

`github.com/mia0x75/parserkit` is a shared syntax-only library for SQL parsers.

## Scope

- Byte positions, token and identifier data, diagnostics, SQL output with write error handling, and syntax-only AST types.
- No dialect lexer or parser, type inference, name binding, or runtime metadata.
- Base structs carry syntax data only and do not implement `Accept`/`Restore`; concrete public nodes or dialect outer nodes own traversal and SQL restoration.

## Projection AST

`ast.FieldList.Items` keeps `[]*ast.SelectItem` in source order. Each item holds an expression or `*ast.WildcardExpr` in `Expr`, a single alias in `Alias lex.Ident`, and qualifier segments in `Qualifier QualifiedName`; an empty segment list means `*`. Use `Alias.Present()` to distinguish a missing alias from an empty quoted alias, and `Span()` for byte ranges. These nodes are implemented in the shared package; dialects keep type identity with aliases. Visitors walk wildcards and may replace expression slots. Restore chooses quoting from the output context instead of reusing lexical `Raw`, and carries no planning or binding metadata.

## Table Source AST

`ast.BaseTableSource` carries `Source ResultSetNode`, `Alias lex.Ident`, `Columns []lex.Ident`, and `Lateral`, without `Accept`/`Restore`. The standard concrete `ast.TableSource` embeds the base and is built by setting `BaseTableSource: ast.BaseTableSource{...}` explicitly. `Source` must be a result-set node, not an arbitrary `Node`; query nodes also implement `QueryNode`, so Restore adds parentheses for derived queries. Dialects with source-level extensions use their own outer nodes instead of inheriting the standard dispatch. `TableSourceNode.TableSourceCore()` exposes the base data so visitors see the real outer node and traverse `Source` once. `RestoreTableSource` reuses the standard source, alias, and column alias restoration; dialect-specific suffixes are emitted afterward.

## Set Operations and Query Tails

`ast.BaseSetOperation` represents one binary set operation; `Left` and `Right` are `QueryNode`s and `SetOperationType` distinguishes UNION/EXCEPT/INTERSECT with optional ALL. It has no `Accept`/`Restore`; dialect nodes keep visitor identity. `ast.BaseQueryTailStmt` combines a query, explicit parentheses, and that query level's `ORDER BY`/`LIMIT`, so parenthesized queries and set operations stay distinct and inner tails are not overwritten. `INTO`, locks, and dialect tails remain in outer nodes. A self-parenthesized subquery may embed `BaseQueryExpr` and satisfy both expression and result-set roles, but not `QueryNode`, so table sources do not add extra parentheses.

## Pagination Syntax Data

`ast.BaseLimit` holds `Count`/`Offset ExprNode` plus inherited text and span. It has no `Accept`/`Restore`; dialect outer nodes keep their LIMIT/OFFSET/FETCH restoration and extension fields. Shared data allows a missing `Count` and does not impose MySQL limits.

## Assignment AST

`ast.Assignment` is a shared concrete node with `Target ExprNode` and `Expr ExprNode`, instead of mutually exclusive single-column/tuple slots. The target may be a column reference, parenthesized row, or dialect field/subscript expression; writability is enforced by dialect grammar, not shared binding. It implements only `Node`. Visitors traverse target then value and allow replacement of either side; Restore calls each side's own rules and errors when either is missing. Copy-management keywords are not column assignments and use dialect name/value nodes.

## INSERT Syntax Data

`ast.BaseInsertStmt` embeds `BaseDML` and shares the write target `Table TableRefNode`, `Columns []ExprNode`, `Values [][]ExprNode`, `Query QueryNode`, and `Returning *FieldList`. The target is not a FROM clause or arbitrary result set; a query source must be a real query, not a table source or subquery expression. Column targets use column references or dialect field/subscript expressions, with legal shapes enforced by grammar; qualified names keep all segments. Nil and non-nil empty slices are not collapsed. VALUES and query sources are distinct; dialect SET inputs and conflict actions stay in outer nodes. The base has no `Accept`/`Restore`; outer nodes own full syntax, visitor identity, and dialect output.

## Ordering and Grouping Syntax Data

`ast.BaseOrderItem` holds `Expr`/`Desc` and is dispatched by the concrete `ast.OrderItem`. `ast.BaseGroupItem` holds only the grouping expression; MySQL GROUP BY direction and PG grouping sets belong to outer nodes. `OrderItemNode`/`GroupItemNode` expose data through `OrderItemCore()`/`GroupItemCore()` and cannot be substituted across roles. `BaseOrderByClause.Items` and `BaseGroupByClause.Items` use role-specific interface slices, allowing dialect entries but not arbitrary `Node`s. Bases have no `Accept`/`Restore`.

## UPDATE/DELETE Syntax Data

`ast.BaseUpdateStmt` embeds `BaseDML` and holds the modified relation `Table ResultSetNode`, `Assignments []*Assignment`, `Where`, `OrderBy`, `Limit`, and `Returning`. `ast.BaseDeleteStmt` separates a single target `Table` from an independent `Using ResultSetNode`; multi-target and real dialect modifiers stay in concrete outer nodes. Neither base implements `Accept`/`Restore`. `OrderByClauseNode.OrderByClauseCore()` and `LimitNode.LimitCore()` expose the original base data, but parents still call the real outer node's traversal and restoration; arbitrary `Node`s cannot be placed in these role-specific slots. Composite literals must initialize the corresponding base explicitly; this migration is not a source-compatible alias for old concrete fields.

## Query Leaf Syntax Data

`ast.BaseValuesStmt` holds `Rows []RowNode`, and `ast.BaseTableStmt` holds `Table TableRefNode`; both embed `BaseQuery` and do not implement `Accept`/`Restore`. Dialect outer nodes keep row/table concrete nodes, keyword forms, diagnostic positions, and their visitor/restore behavior. Shared slices use role interfaces, so the migration changes the static type of corresponding dialect fields. Dialect extensions belong on real outer nodes, not copied into base structs.

## Cross-Dialect AST Migration Contract

`BaseFromClause.Table` uses `ResultSetNode` so dialects can choose the real relation root; no extra FROM-list wrapper is added. `BaseWithClause.CTEs` uses `[]CTENode`, with shared fields read through `CTECore()` and dialect extensions preserved on the real outer node. A CTE's `Query StmtNode` may hold a writable body, with statement category constrained by dialect grammar; Oracle WITH FUNCTION declarations are not included.

`BaseCallExpr.Name` uses `QualifiedName`, storing each segment's name, quote, and span in `Parts []lex.Ident`, with the full `Span` kept separately. This API change does not require dialects to extend call grammar. `WindowFrameUnknown` and `WindowFrameBoundUnknown` are unknown zero values; valid units and directions must be set explicitly, while `WindowFrameExcludeNone=0` still means omitted EXCLUDE. Window bound fields now use `Unbounded`; old enum values and the `UnBounded` spelling are no longer compatible.

Nil receivers of shared concrete nodes' `Accept` return `(nil, false)` without calling the visitor. The boolean from `Enter` means skip children; when skipping, the replacement node's concrete type is still validated and `Leave` is called. Only a false from `Leave`/`Accept` stops traversal. Optional empty slots keep leaf semantics; typed-nil child slots, a missing required CTE body, and nil set-operation items stop traversal; a parent that skips a subtree does not inspect its child slots.

## Checks

```bash
make verify
GOWORK=off make verify
```

`verify` builds the library, runs `go vet`, tests without cached results, runs race tests in random order, and checks lint and generated goyacc output.
