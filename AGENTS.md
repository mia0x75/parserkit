# parserkit 工作契约

## 1. 核心原则与模型调度
- **技能加载**：每项 Go 编码、测试、重构或审查任务，先加载 `golang-how-to`，再加载适用专项 skill。
- **成本与 Token 优化（Sub-agent 调度）**：主 Agent 仅负责核心架构设计与复杂逻辑；对于简单、高确定性的子任务（文件只读检索、字段批量替换、常规单元测试补全、运行 `go build`/`vet`/`test`/`lint`），**优先委派给 sub agent，并指定使用 OpenCode 配置中的 `small_model` 执行**。
- **语言规范**：交流与设计文档使用中文；代码注释使用英文。
- **严禁越级交付与自动提交**：所有门禁全部通过前严禁声明任务完成；任何情况下**禁止自动执行 `git commit`、`git tag` 或 `git push`**，且不得覆盖原解析器的 `.gitignore` 忽略文件。
- **真实执行**：严禁假装工具已执行，必须真实调用命令并根据实际 stdout/stderr 输出评估状态。

## 2. 依赖与架构边界
- 公共运行时只允许标准库和本模块依赖，严禁反向导入任何方言模块。
- 保持纯语法（Pure Syntax）定位，禁止引入 `FieldType`、`Datum`、绑定结果（`ColumnBinding`/`ResultField`）或执行期元数据，也不得通过 `any`、扩展属性 map 或回调槽位变相携带。
- 公共基础结构体（`BaseNode`、`BaseStmt`、`BaseExpr` 等）不实现 `Accept` 或 `Restore`，由外层具体节点负责 Visitor 外层身份保持和方言 SQL 还原。
- 取消包级 `init()` 全局构造器注册，所有节点构造必须通过显式函数或实例级状态完成。

## 3. 统一 AST 命名与语义零混淆红线
- **标准命名**：所有公共节点与字段必须遵循统一词汇表（`Table`、`Columns`、`Assignments`、`Where`、`GroupBy`、`Having`、`OrderBy`、`Limit`、`Returning`、`Alias`、`Expr`、`Left`、`Right`），禁止出现 `TargetList`、`SelectExprs`、`RelName` 等方言俚语。
- **位置与参数编号分离**：
  - 源码范围统一使用左闭右开字节区间 `loc.Span{Start, End}`（未知位置为 `{-1, -1}`，使用 `hasSpan` 区分零值与合法空范围 `[0, 0)`）。
  - 参数占位符统一使用零基 `Index int`（未知为 `-1`），严禁将字节偏移存入 `Index`。
- **限定名称**：`QualifiedName` 必须保留 `[]lex.Ident` 分段结构与原始引号样式（`QuoteStyle`），禁止退化为带点号拼接的字符串。

## 4. 生命周期与并发契约
- 保留无解码场景下的输入子串切片复用（Zero-Copy Slicing），禁止引用会被后续 `Parse` 覆盖的可变缓冲区；单次解析生成的 AST 必须独立存活，不受同一 Parser 实例后续复用影响。
- 当前阶段（Phase 3 迁移期间）禁止更改各解析器文法产生式、优先级及 LR 状态表；禁止新增文法例外或池化优化。

## 5. 闭环验收门禁 (SOP)


```

[代码修改 / 逻辑实现]
│
▼
[Step 1: 构建、单测与静态检查] ◄────────────┐
(build / vet / test -race / lint)           │
│ (全绿: 0 issues)                          │ 发现问题
▼                                           │ 修复并回流重测
[Step 2: 质量评审 (ponytail-review)]        │
│ (通过: 0 异议)                            │
▼                                           │
[Step 3: 独立模块验证 & 任务交付] ──────────┘

```

| 门禁阶段 | 执行要求 | 通过标准 | 失败处理 |
| :--- | :--- | :--- | :--- |
| **Step 1 静态与测试** | 执行 `go build ./...`、`go vet ./...`、`go test -race -count=1 ./...`、`golangci-lint run ./...` | 退出码均为 `0`，0 issues；**禁止**未经允许添加 `//nolint` | 分析根因并就地修复，重新执行 Step 1 |
| **Step 2 质量评审** | 调用 `ponytail-review` skill 对改动进行代码与架构评审 | 评审意见为 0 条，Verdict 通过 | 根据反馈修复后，**必须返回 Step 1** 重新验证 |
| **Step 3 独立构建与交付** | 执行 `GOWORK=off go test ./...` 验证零外部依赖；清理临时调试打印后向用户交付（不自动 commit） | 独立构建与测试全绿 | 修复依赖泄漏后返回 Step 1 |

- **循环冲突熔断**：若修复 A 规则导致触发 B 规则陷入死循环，立即停止自动盲目修改，向开发者说明冲突原因与备选方案等待确认。
