# Level 1: axquery 包设计

> 状态：实现中 — Phase 1 完成（选择器），Phase 2 进行中（Selection 核心 + 遍历 + 过滤 + 属性读取 + 迭代方法已完成）
> 包路径：`github.com/tentaclaw/axquery`
> 语言：Go
> 参考：[goquery](https://github.com/PuerkitoBio/goquery)（API 风格）、[cascadia](https://github.com/andybalholm/cascadia)（选择器模型）

## 1. 定位

`axquery` 是整个 Tentaclaw 的**核心创新层**。它在 Level 0 `ax`（底层原语）之上，提供：

1. **jQuery/goquery 风格的 Selection API** — Go 库
2. **CSS-like AX 选择器** — 内置解析器（不独立发包）
3. **goja JS 运行时** — 将 Go API 暴露为 `$ax(...)` JS 全局函数
4. **脚本执行引擎** — 加载、运行、管理 `.js` 自动化脚本

## 2. 设计参考：goquery

我们直接学习 goquery 的 API 设计模式：

| goquery 模式 | axquery 对应 |
|---|---|
| `Document` 入口，内含根 `Selection` | `App` / `Window` 入口 |
| `Selection` 核心类型，所有操作返回 `*Selection` | 完全一致 |
| `Find(selector)` 子树搜索 | 完全一致 |
| `Children()` / `ChildrenFiltered(selector)` | 采纳 |
| `Parent()` / `ParentsFiltered()` / `ParentsUntil()` | 采纳 |
| `Closest(selector)` 向上最近匹配 | 采纳 |
| `First()` / `Last()` / `Eq(i)` / `Slice(start,end)` | 采纳 |
| `Filter(selector)` / `FilterFunction(fn)` / `Not(selector)` | 采纳 |
| `Each(fn)` / `EachWithBreak(fn)` / `Map(fn)` | 采纳 |
| `Attr(name)` / `AttrOr(name, default)` / `Text()` | 采纳 + AX 快捷属性 |
| `Is(selector)` 判断匹配 | 采纳 |
| `XxxMatcher()` 预编译选择器变体 | 采纳 |
| `cascadia` 独立选择器库 | 内置在 axquery（AX 选择器太特化） |

**关键差异（axquery 独有，goquery 不需要）：**
- goquery 操作**静态 HTML DOM**；axquery 操作**活的 AX 树**（随时变化）
- goquery 不需要超时；axquery 每个操作都可能超时
- goquery 不需要交互动作；axquery 需要 Click/SetValue/TypeText 等
- goquery 的树是预加载的；axquery 的树需要**按需惰性加载**

## 3. 选择器语法

CSS-like 风格，针对 AX 元素属性设计：

### 3.1 基本选择器

```
AXButton                          // 按角色（AXRole 精确匹配）
AXButton[title="Send"]            // 角色 + 属性精确匹配
*[title="OK"]                     // 任意角色
```

### 3.2 属性匹配操作符

| 操作符 | 含义 | 例子 |
|--------|------|------|
| `=` | 精确匹配 | `[title="OK"]` |
| `*=` | 包含 | `[title*="Send"]` |
| `^=` | 前缀 | `[title^="Re:"]` |
| `$=` | 后缀 | `[description$="button"]` |
| `~=` | 正则匹配 | `[title~="\\d+ unread"]` |
| `!=` | 不等于 | `[role!="AXStaticText"]` |

### 3.3 组合器

| 符号 | 含义 | 例子 |
|------|------|------|
| `A B` | B 是 A 的后代（任意深度） | `AXWindow AXTable` |
| `A > B` | B 是 A 的直接子元素 | `AXSheet > AXButton` |
| `A, B` | OR — 匹配 A 或 B | `AXButton, AXMenuItem` |

### 3.4 伪选择器

```
AXButton:first                    // 第一个匹配
AXButton:last                     // 最后一个匹配
AXRow:nth(3)                      // 第 N 个（0-based）
AXButton:visible                  // 可见的
AXButton:enabled                  // 启用的
AXButton:focused                  // 聚焦的
AXButton:selected                 // 选中的
```

### 3.5 完整示例

```
AXSheet > AXButton[title="OK"]                    // sheet 内直接子元素中标题为 OK 的按钮
AXWindow AXTable AXRow:nth(0) AXStaticText         // 窗口 → 表格 → 第一行 → 所有静态文本
AXButton[title*="Send"]:enabled                    // 标题包含 Send 且启用的按钮
AXMenuItem[title~="^New\\s"], AXButton[title="New"] // 标题匹配正则的菜单项 或 标题为 New 的按钮
```

## 4. Go 核心类型

### 4.1 Selection（已实现 ✅）

```go
package axquery

// Selection 是核心集合类型（已实现 — selection.go）
// 与设计稿的关键差异：Selection 不持有 app/root 引用，仅持有元素切片
// Task 7 新增：nodes 字段保存内部 queryNode 引用，支持遍历链式调用
type Selection struct {
    elems    []*ax.Element  // 底层 Level 0 元素
    nodes    []queryNode    // 内部遍历节点（traversal 方法产生时填充）
    err      error          // 链式调用中的错误暂存
    selector string         // 产生此 Selection 的选择器
}
```

> **注意：** 设计稿中 Selection 包含 `root *ax.Element`、`app *ax.Application`、`opts QueryOptions`。
> 实现中去掉了这些字段 — Selection 更纯粹，只负责持有元素和错误。查询配置在 Query 入口层处理。

### 4.2 QueryOptions（已实现 ✅）

```go
type SearchStrategy int
const (
    StrategyBFS SearchStrategy = iota  // 宽度优先（默认）
    StrategyDFS                         // 深度优先
    // 注意：设计稿中的 Adaptive 策略已移除
)

type QueryOptions struct {
    Timeout    time.Duration  // 0 = 无超时（设计稿默认 5s，实现中改为 0）
    MaxDepth   int            // 0 = 无限制
    MaxResults int            // 0 = 无限制
    Strategy   SearchStrategy // 默认 BFS
}

// Functional options
func WithTimeout(d time.Duration) QueryOption
func WithMaxDepth(n int) QueryOption
func WithMaxResults(n int) QueryOption
func WithStrategy(s SearchStrategy) QueryOption
```

### 4.3 错误类型（已实现 ✅）

```go
// Sentinel errors — 支持 errors.Is()
var (
    ErrNotFound        = errors.New("not found")
    ErrTimeout         = errors.New("timeout")
    ErrAmbiguous       = errors.New("ambiguous")
    ErrInvalidSelector = errors.New("invalid selector")
    ErrNotActionable   = errors.New("not actionable")
)

// Wrapper types — 支持 errors.As()，每个都实现 Unwrap() 返回对应 sentinel
type NotFoundError struct{ Selector string }
type TimeoutError struct{ Selector, Duration string }
type AmbiguousError struct{ Selector string; Count int }
type InvalidSelectorError struct{ Selector, Reason string }
type NotActionableError struct{ Action, Reason string }
```

### 4.4 Matcher 接口（已实现 ✅）

```go
package selector

// Matchable — 精简接口（与设计稿的关键差异：去掉了 GetTitle/GetDescription/GetValue）
type Matchable interface {
    GetRole() string
    GetAttr(name string) string     // 统一属性访问，取代多个专用 getter
    IsEnabled() bool
    IsVisible() bool
    IsFocused() bool
    IsSelected() bool
}

// Matcher — 编译后的选择器
type Matcher interface {
    MatchSimple(el Matchable) bool  // 仅匹配叶节点简单选择器
    Group() *SelectorGroup          // 返回 AST 供 query 层处理组合器
}

// 公共 API（在 selector 包中）
func Compile(sel string) (CompiledSelector, error)
func MustCompile(sel string) CompiledSelector
```

> **与设计稿差异：** 设计稿的 `Matcher` 接口在根包（axquery），含 `Match(*ax.Element)` 和 `Filter([]*ax.Element)`。
> 实现中 `Matcher` 在 `selector` 包，用 `MatchSimple(Matchable)` — 不直接依赖 `*ax.Element`，更可测试。

### 4.5 构造与缩减（已实现 ✅）

```go
// 核心构造
func Query(app *ax.Application, selector string, opts ...QueryOption) *Selection

// 缩减方法（已实现）
func (s *Selection) Count() int
func (s *Selection) IsEmpty() bool
func (s *Selection) Err() error
func (s *Selection) Selector() string
func (s *Selection) Elements() []*ax.Element
func (s *Selection) First() *Selection
func (s *Selection) Last() *Selection
func (s *Selection) Eq(index int) *Selection
func (s *Selection) Slice(start, end int) *Selection
```

### 4.6 遍历方法（已实现 ✅ — Task 7）

```go
func (s *Selection) Find(selector string) *Selection
func (s *Selection) Children() *Selection
func (s *Selection) ChildrenFiltered(selector string) *Selection
func (s *Selection) Parent() *Selection
func (s *Selection) ParentFiltered(selector string) *Selection
func (s *Selection) Parents() *Selection
func (s *Selection) ParentsUntil(selector string) *Selection
func (s *Selection) Closest(selector string) *Selection
func (s *Selection) Siblings() *Selection
func (s *Selection) Next() *Selection
func (s *Selection) Prev() *Selection
```

> **内部架构：** 遍历方法通过 `traversableNode` 接口（extends `queryNode` + `queryParent()`）实现双向树遍历。
> Selection 内部持有 `nodes []queryNode`，使得 First/Last/Eq/Slice 产生的子 Selection 仍可继续 traversal 链式调用。
> 所有遍历结果通过指针去重避免重复，AX 错误静默跳过。

### 4.7 过滤/判断（已实现 ✅ — Task 8）

```go
func (s *Selection) Filter(selector string) *Selection
func (s *Selection) FilterMatcher(m selector.Matcher) *Selection
func (s *Selection) FilterFunction(fn func(int, *Selection) bool) *Selection
func (s *Selection) Not(selector string) *Selection
func (s *Selection) NotMatcher(m selector.Matcher) *Selection
func (s *Selection) Has(selector string) *Selection
func (s *Selection) Is(selector string) bool
func (s *Selection) Contains(text string) *Selection
```

> **实现细节：** Filter/Not 使用 `selector.Compile` 编译后对每个 node 调用 `MatchSimple` 过滤。
> FilterMatcher/NotMatcher 接受预编译的 Matcher，避免重复编译。
> Has 检查后代（不含自身），使用递归 BFS 搜索子树。
> Is 返回 bool，invalid selector 或空 Selection 返回 false。
> Contains 按 title 属性子串匹配。

### 4.8 属性读取（已实现 ✅ — Task 9）

所有属性方法操作 Selection 的**第一个元素**（goquery/jQuery 语义），空/错误 Selection 返回零值。

```go
func (s *Selection) Attr(name string) string       // 命名属性；name="role" 时走 GetRole()
func (s *Selection) AttrOr(name, defaultVal string) string
func (s *Selection) Role() string                   // GetRole() 快捷方式
func (s *Selection) Title() string                  // Attr("title")
func (s *Selection) Description() string            // Attr("description")
func (s *Selection) Val() string                    // Attr("value")
func (s *Selection) Text() string                   // 递归收集 title 属性，空格连接
func (s *Selection) IsVisible() bool
func (s *Selection) IsEnabled() bool
func (s *Selection) IsFocused() bool
func (s *Selection) IsSelected() bool
```

> **内部架构：** 所有方法共享 `firstNode()` 守卫（处理 nil/empty/error），减少重复。
> Text() 通过 `collectText()` 深度优先递归收集 title。
> Bounds() 延迟到后续 Task（action/scroll 需要时实现）。

#### 4.8.2 遍历回调（已实现 ✅ — Task 10）

所有迭代方法的回调接收**单元素 Selection**，与 FilterFunction 和 goquery 保持一致。
空/错误 Selection 上调用迭代方法为 no-op。

```go
func (s *Selection) Each(fn func(int, *Selection)) *Selection          // 返回原 Selection，支持链式
func (s *Selection) EachWithBreak(fn func(int, *Selection) bool) *Selection  // fn 返回 false 停止
func (s *Selection) Map(fn func(int, *Selection) string) []string      // 收集回调返回的字符串
func (s *Selection) EachIter() iter.Seq2[int, *Selection]              // Go 1.23+ range-over-func
```

> **内部架构：** 迭代方法通过 `s.getNodes()` 获取节点列表，为每个节点创建单元素 Selection（`newSelectionFromNodes`），
> 保留节点身份以支持后续 traversal 链式调用。EachIter() 利用 Go 1.23+ 的 `iter.Seq2` 支持 `for i, sel := range` 和 `break`。

#### 4.8.3 交互动作

```go
func (s *Selection) Click() *Selection
func (s *Selection) SetValue(v string) *Selection
func (s *Selection) TypeText(text string) *Selection
func (s *Selection) Press(key string, modifiers ...string) *Selection
func (s *Selection) Focus() *Selection
func (s *Selection) Perform(action string) *Selection
```

#### 4.8.4 等待

```go
func (s *Selection) WaitUntil(fn func(*Selection) bool, timeout time.Duration) *Selection
func (s *Selection) WaitGone(timeout time.Duration) *Selection
func (s *Selection) WaitVisible(timeout time.Duration) *Selection
func (s *Selection) WaitEnabled(timeout time.Duration) *Selection
```

## 5. Query 引擎架构（已实现 ✅）

### 5.1 内部架构

```
Query(app, sel, opts...)                    ← 公共入口（薄 CGo 桥接）
  └─ queryWithResolver(resolver, sel, opts) ← 可测试入口
       ├─ resolver.resolveRoot()            ← 获取根节点
       └─ queryFromRoot(root, sel, opts)    ← 核心查询逻辑
            ├─ selector.Compile(sel)        ← 编译选择器
            └─ searchBFS / searchDFS        ← 遍历 queryNode 树

Selection traversal methods:                ← Task 7 新增
  └─ traversableNode (extends queryNode + queryParent())
       ├─ findInSubtrees()                  ← Find
       ├─ getChildren/getChildrenFiltered() ← Children
       ├─ getParents/getAncestors()         ← Parent/Parents
       ├─ getClosest()                      ← Closest
       └─ getSiblings/getNext/getPrev()     ← Siblings/Next/Prev
```

**关键接口：**
- `queryNode` — 统一遍历接口（extends `selector.Matchable`）
- `rootResolver` — 根节点获取策略
- `axElementReader` — ax.Element 属性读取抽象

### 5.2 默认 BFS（已实现 ✅）

旧系统只有 DFS，导致深层元素被优先找到。UI 自动化中，用户通常想找"视觉上最浅/最近"的匹配。BFS 更符合直觉。

### 5.3 作用域搜索（已实现 ✅）

```go
// 旧系统：每次从窗口根开始
findElement(criteria)  // 总是全树 DFS

// 新系统：可在子树内搜索
sheet := axquery.Query(app, "AXSheet")
sheet.Find("AXButton[title='OK']")  // 只在 sheet 子树内找
```

### 5.4 早期终止（已实现 ✅）

找到足够匹配后立即停止，不继续遍历。`First()` 本质上是 `MaxResults=1`。

### 5.5 惰性子元素加载（计划中）

利用 Level 0 的 `ChildCount()` 和 `ChildrenRange()` 原语：
- 先看子元素数量
- 大列表分页加载
- 不匹配的子树提前剪枝

## 6. JS 运行时（goja）

### 6.1 全局 API

```javascript
// 核心选择器
$ax(selector)                     // 返回 Selection 代理对象
$ax(selector, { timeout: 10000 }) // 带选项

// 应用管理
$app('com.apple.mail')            // 切换目标应用（Bundle ID）
$app('Mail')                      // 或应用名

// 系统工具
$delay(ms)                        // 延迟
$log(...)                         // 日志
$screenshot(filename?)            // 截图

// 剪贴板
$clipboard.read()
$clipboard.write(text)

// 键盘（不针对特定元素）
$keyboard.press(key, ...modifiers)
$keyboard.type(text)

// 脚本 IO
$env                              // 环境变量（只读）
$input                            // 脚本输入参数
$output                           // 脚本输出设置

// 控制台
console.log / console.warn / console.error
```

### 6.2 完整使用示例

```javascript
// 邮件摘要提取
$app('com.apple.mail');

let rows = $ax('AXTable > AXRow');
let summaries = [];

rows.each(function(i, row) {
    if (i >= 10) return false; // 只取前 10 封

    let subject = row.find('AXStaticText[description*="subject"]').text();
    let sender  = row.find('AXStaticText[description*="from"]').text();
    let date    = row.find('AXStaticText[description*="date"]').text();

    summaries.push({
        subject: subject,
        sender: sender,
        date: date
    });
});

$output.summaries = summaries;
$log('提取了 ' + summaries.length + ' 封邮件摘要');
```

```javascript
// 自动回复邮件
$app('com.apple.mail');

// 等待邮件窗口就绪
$ax('AXWindow').waitUntil(function(el) {
    return el.count() > 0;
}, 5000);

// 点击回复按钮
$ax('AXButton[title*="Reply"]').click();
$delay(500);

// 等待编辑窗口出现
$ax('AXSheet, AXWindow[title^="Re:"]').waitUntil(function(el) {
    return el.count() > 0;
}, 5000);

// 输入回复内容
$ax('AXTextArea').typeText('感谢你的邮件，我已收到。');

// 发送
$ax('AXButton[title="Send"]').click();
```

### 6.3 默认选项

```javascript
// 全局默认值（可在脚本开头修改）
$ax.defaults.timeout = 5000;       // 查询超时 ms
$ax.defaults.maxDepth = 20;        // 搜索深度
$ax.defaults.strategy = 'bfs';     // 遍历策略: bfs / dfs / adaptive

// 单次覆盖
$ax('AXButton[title="Send"]', { timeout: 10000 }).click();
```

## 7. 错误处理

### 7.1 Go 侧

Selection 方法链不中断——错误暂存在 `Selection.err` 中，通过 `Err()` 获取：

```go
sel := axquery.Query(app, "AXButton[title='Send']")
sel.Click()
if sel.Err() != nil {
    // 处理错误
}
```

### 7.2 JS 侧

JS 中使用标准 try-catch，错误有结构化的 `code` 字段：

```javascript
try {
    $ax('AXButton[title="Send"]').click();
} catch (e) {
    switch (e.code) {
        case 'NOT_FOUND':        // 选择器无匹配
        case 'TIMEOUT':          // 查询超时
        case 'AMBIGUOUS':        // 匹配多个但需要单个（e.count 有数量）
        case 'NOT_ACTIONABLE':   // 元素存在但不可操作
        case 'AX_ERROR':         // 底层 AX API 错误
        case 'PERMISSION_DENIED':// 无辅助功能权限
        case 'INVALID_SELECTOR': // 选择器语法错误
    }
}
```

## 8. 包结构

```
axquery/
├── go.mod
├── axquery.go            // 包入口 + 文档注释                    ✅
├── selection.go          // Selection 类型 + 缩减方法             ✅
├── query.go              // Query 入口 + BFS/DFS 搜索引擎        ✅
│                         //   queryNode, elementAdapter,
│                         //   rootResolver, queryWithResolver
├── options.go            // QueryOptions + functional options     ✅
├── errors.go             // 错误类型 (sentinel + wrapper)         ✅
├── traversal.go          // Find/Children/Parent/Closest 等       ✅
├── filter.go             // Filter/Not/Has/Is/Contains            ✅
├── property.go           // Attr/Text/Val/Role/Title 等           ✅
├── action.go             // Click/SetValue/TypeText/Press          计划
├── iteration.go          // Each/EachWithBreak/Map/EachIter          ✅
├── waiting.go            // WaitUntil/WaitGone                     计划
├── scroll.go             // ScrollIntoView/ScrollDown/ScrollUp     计划
├── selector/             // 选择器子包                             ✅
│   ├── ast.go            // 选择器 AST 类型                       ✅
│   ├── parser.go         // 递归下降解析器                         ✅
│   ├── compiler.go       // AST → CompiledSelector 编译            ✅
│   └── matcher.go        // Matchable + Matcher 接口               ✅
├── js/                   // JS 运行时子包                          计划
│   ├── runtime.go
│   ├── globals.go
│   ├── bridge.go
│   └── executor.go
├── *_test.go             // 各文件对应测试                         ✅ (~95.6% root / 97.1% selector)
└── docs/
    ├── architecture.md
    ├── decisions.md
    ├── level-1-axquery.md
    └── plans/
        └── 2026-03-26-axquery-foundation.md
```

## 9. 与旧系统对比

| 维度 | 旧系统 (Rust/JSON step) | 新系统 (axquery JS) |
|------|------------------------|---------------------|
| 脚本格式 | JSON step 数组 | `.js` 文件 |
| 选择器 | 4 个扁平 AND 字段 + substring | CSS-like 结构化选择器 |
| 匹配方式 | role 精确 + 其余 substring | 精确/包含/前缀/后缀/正则 |
| 组合器 | 无 | 后代 / 直接子元素 / OR |
| 作用域 | 始终从窗口根开始 | 可在子树内搜索 |
| 遍历 | 仅 DFS | BFS / DFS / Adaptive |
| 集合操作 | forEach step + 位置路径 | `.each()` / `.map()` / `.filter()` |
| 元素引用 | 每次重新遍历全树 | Selection 持有引用，子树复用 |
| 错误类型 | 字符串 | 结构化枚举 |
| 控制流 | 自定义 If/ForEach step | 原生 JS if/for/while/try-catch |
| MCP tools | 20+ 个 | ~7 个（JS 表达大部分操作） |

## 10. 相关文档

- [架构总览](./architecture.md)
- [Level 0: ax 包设计](./level-0-ax.md)
- [Level 2: axblocky 包设计](./level-2-axblocky.md)
