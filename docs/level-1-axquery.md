# Level 1: axquery 包设计

> 状态：设计阶段
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

### 4.1 Selection

```go
package axquery

// Selection 是核心集合类型，类似 goquery.Selection / jQuery 对象
// 所有操作返回 *Selection，支持链式调用
type Selection struct {
    elements []ax.Element  // 底层 Level 0 元素
    root     *ax.Element   // 搜索起点（nil = 从窗口开始）
    app      *ax.Application
    opts     QueryOptions
    err      error         // 链式调用中的错误暂存
}

type QueryOptions struct {
    Timeout    time.Duration     // 单次查询超时，默认 5s
    MaxDepth   int               // 搜索深度限制，默认 20
    Strategy   TraversalStrategy // 默认 BFS
    MaxResults int               // 最大匹配数，0 = 无限
}

type TraversalStrategy int
const (
    BFS      TraversalStrategy = iota  // 宽度优先（默认）
    DFS                                 // 深度优先
    Adaptive                            // 浅层 BFS + 深层切 DFS
)
```

### 4.2 构造与遍历（学 goquery）

```go
// 核心构造
func Query(app *ax.Application, selector string, opts ...QueryOption) *Selection

// Selection 方法 — jQuery/goquery 风格链式调用

// 搜索
func (s *Selection) Find(selector string) *Selection          // 在子树内搜索（后代）
func (s *Selection) FindMatcher(m Matcher) *Selection         // 预编译选择器版本
func (s *Selection) Children() *Selection                     // 所有直接子元素
func (s *Selection) ChildrenFiltered(selector string) *Selection // 过滤的直接子元素
func (s *Selection) Parent() *Selection                       // 父元素
func (s *Selection) ParentFiltered(selector string) *Selection
func (s *Selection) Parents() *Selection                      // 所有祖先
func (s *Selection) ParentsUntil(selector string) *Selection  // 祖先直到匹配
func (s *Selection) Closest(selector string) *Selection       // 向上最近匹配
func (s *Selection) Siblings() *Selection                     // 兄弟元素
func (s *Selection) SiblingsFiltered(selector string) *Selection
func (s *Selection) Next() *Selection                         // 下一个兄弟
func (s *Selection) NextFiltered(selector string) *Selection
func (s *Selection) Prev() *Selection                         // 上一个兄弟
func (s *Selection) PrevFiltered(selector string) *Selection

// 过滤/缩减
func (s *Selection) First() *Selection
func (s *Selection) Last() *Selection
func (s *Selection) Eq(index int) *Selection
func (s *Selection) Slice(start, end int) *Selection
func (s *Selection) Filter(selector string) *Selection
func (s *Selection) FilterFunction(fn func(int, *Selection) bool) *Selection
func (s *Selection) Not(selector string) *Selection
func (s *Selection) Has(selector string) *Selection           // 保留包含匹配子元素的

// 判断
func (s *Selection) Is(selector string) bool
func (s *Selection) IsMatcher(m Matcher) bool

// 集合信息
func (s *Selection) Count() int                               // goquery 用 Length()
func (s *Selection) Err() error                               // 获取链式调用中的错误
```

### 4.3 属性读取

```go
// 通用属性
func (s *Selection) Attr(name string) (string, bool)          // 第一个元素的属性
func (s *Selection) AttrOr(name, defaultVal string) string    // 属性或默认值

// AX 快捷属性（goquery 没有这些，axquery 特有）
func (s *Selection) Text() string                             // 组合文本内容
func (s *Selection) Val() string                              // AXValue
func (s *Selection) Role() string
func (s *Selection) Title() string
func (s *Selection) Description() string
func (s *Selection) Bounds() ax.Rect
func (s *Selection) IsVisible() bool
func (s *Selection) IsEnabled() bool
func (s *Selection) IsFocused() bool
func (s *Selection) IsSelected() bool
```

### 4.4 交互动作（axquery 独有）

```go
// 动作
func (s *Selection) Click() *Selection                        // 点击（返回自身支持链式）
func (s *Selection) SetValue(v string) *Selection             // 设置值
func (s *Selection) TypeText(text string) *Selection          // 键入文本
func (s *Selection) Press(key string, modifiers ...string) *Selection
func (s *Selection) Focus() *Selection
func (s *Selection) Perform(action string) *Selection         // 任意 AX action

// 滚动
func (s *Selection) ScrollIntoView() *Selection
func (s *Selection) ScrollDown(n int) *Selection
func (s *Selection) ScrollUp(n int) *Selection
```

### 4.5 遍历

```go
// 遍历（学 goquery）
func (s *Selection) Each(fn func(int, *Selection)) *Selection
func (s *Selection) EachWithBreak(fn func(int, *Selection) bool) *Selection
func (s *Selection) Map(fn func(int, *Selection) string) []string

// Go 1.23+ range iterator（学 goquery v1.10）
func (s *Selection) EachIter() iter.Seq2[int, *Selection]
```

### 4.6 等待（axquery 独有）

```go
// 等待
func (s *Selection) WaitUntil(fn func(*Selection) bool, timeout time.Duration) *Selection
func (s *Selection) WaitGone(timeout time.Duration) *Selection
func (s *Selection) WaitVisible(timeout time.Duration) *Selection
func (s *Selection) WaitEnabled(timeout time.Duration) *Selection
```

### 4.7 Matcher 接口

```go
// Matcher 接口，类似 goquery/cascadia 的 Matcher
// 预编译选择器可复用，避免重复解析
type Matcher interface {
    Match(el *ax.Element) bool
    Filter(elements []*ax.Element) []*ax.Element
}

// Compile 编译选择器字符串为 Matcher
func Compile(selector string) (Matcher, error)

// MustCompile 编译选择器，失败 panic
func MustCompile(selector string) Matcher
```

## 5. 搜索策略

### 5.1 默认 BFS

旧系统只有 DFS，导致深层元素被优先找到。UI 自动化中，用户通常想找"视觉上最浅/最近"的匹配。BFS 更符合直觉。

### 5.2 作用域搜索

```go
// 旧系统：每次从窗口根开始
findElement(criteria)  // 总是全树 DFS

// 新系统：可在子树内搜索
sheet := axquery.Query(app, "AXSheet")
sheet.Find("AXButton[title='OK']")  // 只在 sheet 子树内找
```

### 5.3 早期终止

找到足够匹配后立即停止，不继续遍历。`First()` 本质上是 `MaxResults=1`。

### 5.4 惰性子元素加载

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
├── selection.go          // Selection 类型 + 链式方法
├── query.go              // Query() 入口 + 搜索逻辑
├── traversal.go          // Find/Children/Parent/Closest 等
├── filter.go             // Filter/Not/Has/First/Last/Eq
├── property.go           // Attr/Text/Val/Role/Title 等
├── action.go             // Click/SetValue/TypeText/Press
├── iteration.go          // Each/EachWithBreak/Map
├── waiting.go            // WaitUntil/WaitGone
├── scroll.go             // ScrollIntoView/ScrollDown/ScrollUp
├── selector/             // 选择器子包（内置）
│   ├── parser.go         // 选择器字符串解析
│   ├── ast.go            // 选择器 AST 类型
│   ├── compiler.go       // AST → Matcher 编译
│   └── matcher.go        // Matcher 接口和实现
├── js/                   // JS 运行时子包
│   ├── runtime.go        // goja 运行时管理
│   ├── globals.go        // $ax/$app/$delay 等全局注入
│   ├── bridge.go         // Go Selection ↔ JS 对象桥接
│   └── executor.go       // .js 脚本加载执行
├── errors.go             // 错误类型
└── options.go            // QueryOptions / functional options
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
