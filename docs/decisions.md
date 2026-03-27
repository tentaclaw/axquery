# 决策记录

> 最后更新：2026-03-27（Task 16 完成后）

## 1. 已确认决策

| # | 决策 | 理由 | 日期 |
|---|------|------|------|
| 1 | **语言：Go**（CGo + ObjC bridge） | 用户已有 `pplx-cli` 验证可行性；Go 开发效率高于 Rust | 2026-03-26 |
| 2 | **四层架构**：ax → axquery → axblocky → app | 关注点分离，各层可独立测试和演进 | 2026-03-26 |
| 3 | **脚本格式：`.js`** | 开发者友好；所有创作方式（手写/Blockly/AI）统一输出格式 | 2026-03-26 |
| 4 | **JS 引擎：`goja`** | 纯 Go 实现，无 CGo 额外依赖；ES5.1+ 足够 | 2026-03-26 |
| 5 | **选择器语法：CSS-like** | 学 goquery/cascadia；开发者熟悉；表达力强 | 2026-03-26 |
| 6 | **选择器库：内置在 axquery** | AX 选择器是特化领域，不太可能被其他项目复用 | 2026-03-26 |
| 7 | **默认遍历策略：BFS** | UI 自动化中"视觉上最浅"的元素优先匹配更符合直觉 | 2026-03-26 |
| 8 | **JS API 命名：`$` 前缀** | `$ax()`/`$app()`/`$delay()` — 简洁，jQuery 传统 | 2026-03-26 |
| 9 | **axblocky：纯 npm/TS 包** | 独立于 Go，可被任何前端框架使用 | 2026-03-26 |
| 10 | **桌面框架：Wails v3** | Go 后端 + Web 前端 + 原生窗口/系统托盘；直接支持 Svelte | 2026-03-26 |
| 11 | **前端框架：Svelte** | 旧项目已有基础；Wails 官方支持 Svelte 模板 | 2026-03-26 |
| 12 | **GUI 策略：Wails 原生窗口 + menubar** | 不需要单独的浏览器页面；不需要 HTTP server | 2026-03-26 |
| 13 | **多仓库结构** | ax / axquery / axblocky / app / landing / docs 各自独立 | 2026-03-26 |
| 14 | **CLI = MCP client** | CLI 不含业务逻辑，只翻译命令行参数为 MCP tool 调用 | 2026-03-26 |
| 15 | **MCP server 内嵌在 app 中** | 常驻运行，CLI/AI agent 连接它 | 2026-03-26 |
| 16 | **CLI ↔ MCP server：Unix socket** | `~/.tentaclaw/tentaclaw.sock` | 2026-03-26 |
| 17 | **AI agent（本地）：stdio** | 被 Claude Desktop 等直接拉起 | 2026-03-26 |
| 18 | **App 未启动时 CLI 报错** | 类似 Docker daemon 未运行时 docker 报错 | 2026-03-26 |
| 19 | **HTTP 默认仅 127.0.0.1** | 安全第一——这个 server 可以控制电脑上任意应用 | 2026-03-26 |
| 20 | **远程 transport：Streamable HTTP** | MCP 最新标准；单端口双向通信 | 2026-03-26 |
| 21 | **远程访问需 API key + TLS** | 强制认证和加密 | 2026-03-26 |
| 22 | **Matchable 接口精简化** | selector.Matchable 仅含 `GetRole()`、`GetAttr(name)`、`IsEnabled/Visible/Focused/Selected()` — 去掉了设计稿中的 `GetTitle()`/`GetDescription()`/`GetValue()`，统一走 `GetAttr` | 2026-03-26 |
| 23 | **MatchSimple 仅匹配叶节点** | `MatchSimple` 只匹配复合选择器的最后一个简单选择器；组合器（后代/子元素）延迟到 query/selection 层处理 | 2026-03-26 |
| 24 | **集合伪选择器延迟处理** | `:first`、`:last`、`:nth(n)` 被解析但在 matcher 层忽略，由 Selection 层实现 | 2026-03-26 |
| 25 | **selector.Compile 不因无效正则失败** | 无效正则模式不导致 Compile 错误，运行时 simply never match | 2026-03-26 |
| 26 | **SearchStrategy 枚举 = StrategyBFS / StrategyDFS** | 实现中去掉了设计稿的 `Adaptive` 策略，仅保留 BFS（默认）和 DFS | 2026-03-26 |
| 27 | **QueryOptions.Timeout 默认为 0（无超时）** | 设计稿预设 5s 超时，实现中改为默认无超时，由调用方按需设置 | 2026-03-26 |
| 28 | **Selection 不持有 app/root 引用** | 与设计稿不同：Selection 仅持有 `[]*ax.Element` + `error` + `selector`，不保存 app/root — 更简洁、更纯粹 | 2026-03-26 |
| 29 | **queryNode 接口实现可测试遍历** | 引入 `queryNode` 接口（extends `Matchable` + `queryChildren()` + `element()`），使 BFS/DFS 逻辑可用纯 mock 节点单测 | 2026-03-26 |
| 30 | **rootResolver 接口隔离 AX 桥接** | `rootResolver` 抽象根节点获取，`appRootResolver` 是唯一的 CGo 实现；`queryWithResolver` 是可测试入口 | 2026-03-26 |
| 31 | **axElementReader 接口隔离元素读取** | `elementAdapter` 使用 `axElementReader` 接口而非直接依赖 `*ax.Element`，允许 mock 测试属性映射逻辑 | 2026-03-26 |
| 32 | **IsVisible = !IsHidden** | ax 包的语义是 `IsHidden()`，axquery 将其反转为 `IsVisible()` 以符合选择器 `:visible` 的直觉 | 2026-03-26 |
| 33 | **子节点加载错误静默跳过** | 搜索时遇到无法读取子节点的元素（AX 错误很常见），跳过该子树继续搜索，而非中止整个查询 | 2026-03-26 |
| 34 | **elementAdapter.GetAttr 特殊映射** | `title`/`description`/`role`/`subrole`/`roleDescription` 走专用方法；其余走通用 `Attribute(name)` | 2026-03-26 |
| 35 | **错误类型用 sentinel + wrapper 双层设计** | sentinel (`ErrNotFound` 等) 配合 wrapper 结构体 (`NotFoundError` 等)；支持 `errors.Is` 和 `errors.As` | 2026-03-26 |
| 36 | **traversableNode 接口扩展 queryNode** | 引入 `traversableNode`（extends `queryNode` + `queryParent()`）实现双向遍历；Parent/Siblings/Closest 等方法需要向上访问父节点 | 2026-03-26 |
| 37 | **Selection 内部持有 nodes []queryNode** | Selection 新增 `nodes` 字段，保存产生当前 Selection 的内部节点引用；使 First/Last/Eq/Slice 结果仍可执行 traversal 操作（chaining） | 2026-03-26 |
| 38 | **遍历子节点/父节点错误静默跳过** | 与决策 #33 一致：traversal 中遇到 AX 错误时跳过该元素继续处理，而非中止整个操作 | 2026-03-26 |
| 39 | **遍历结果按指针去重** | 当多个 Selection 元素的子树可能产生重叠结果时（如 Find/Siblings），用 `map[queryNode]bool` 去重 | 2026-03-26 |
| 40 | **Siblings 不包含自身** | 与 goquery/jQuery 语义一致：Siblings() 返回同级兄弟元素但排除 Selection 中的当前元素 | 2026-03-26 |
| 41 | **Filter/Not 使用 MatchSimple 逐节点过滤** | 在当前 Selection 的 nodes 上逐个调用 `MatchSimple`，编译一次选择器复用；FilterMatcher/NotMatcher 提供预编译变体 | 2026-03-26 |
| 42 | **Has 检查后代不含自身** | 与 goquery/jQuery 语义一致：Has(sel) 保留"包含匹配后代"的元素，不检查元素自身 | 2026-03-26 |
| 43 | **Is 返回 bool 而非 Selection** | Is(sel) 是判断方法，不走链式调用；invalid selector 或空 Selection 返回 false 而非 error | 2026-03-26 |
| 44 | **Contains 基于 title 子串匹配** | Contains(text) 按 GetAttr("title") 做子串匹配，是 AX 场景下最常用的文本过滤方式；与 jQuery 的 `:contains` 语义类似 | 2026-03-26 |
| 45 | **FilterFunction 传入单元素 Selection** | 回调 fn(i, sel) 中的 sel 是包含单个元素的 Selection，保持与 Each/goquery 的一致性 | 2026-03-26 |
| 46 | **属性方法操作首元素** | 所有 property 方法（Attr/Role/Title/Text/IsVisible 等）仅操作 Selection 的第一个元素，匹配 goquery/jQuery 语义 | 2026-03-26 |
| 47 | **firstNode() 共享守卫** | property.go 中提取 `firstNode()` 方法统一处理 empty/error/nil 守卫，减少重复代码 | 2026-03-26 |
| 48 | **Text() 递归收集 title** | Text() 通过深度优先递归收集元素及后代的 `title` 属性，以空格连接；AX 场景下 title 是最通用的文本标识 | 2026-03-26 |
| 49 | **Attr("role") 特殊处理** | Attr(name) 当 name="role" 时走 GetRole() 而非 GetAttr，保持与 Role() 方法的一致性 | 2026-03-26 |
| 50 | **Bounds() 延迟到后续 Task** | Task 9 计划中的 Bounds() 暂未实现，因当前 mock 架构不自然支持几何信息；将在 action/scroll 相关 Task 中按需添加 | 2026-03-26 |
| 51 | **迭代回调传入单元素 Selection** | Each/EachWithBreak/Map 的回调 fn(i, sel) 中的 sel 是包含单个元素的 Selection，与 FilterFunction 和 goquery 保持一致 | 2026-03-26 |
| 52 | **Each/EachWithBreak 返回原 Selection** | 迭代方法返回调用者本身（不创建新 Selection），支持链式调用 `sel.Each(...).Find(...)` | 2026-03-26 |
| 53 | **EachIter 使用 iter.Seq2** | Go 1.23+ 的 range-over-func 特性，`EachIter()` 返回 `iter.Seq2[int, *Selection]`，支持 `for i, sel := range sel.EachIter()` 和 `break` | 2026-03-26 |
| 54 | **空/错误 Selection 迭代为 no-op** | 与属性方法一致：empty/error Selection 上调用 Each/Map 等不执行回调，直接返回零值 | 2026-03-26 |
| 55 | **actionable 内部接口隔离交互动作** | action 方法通过 `actionable` 接口（`press()`/`setValue()`/`performAction()`）操作节点；mock 节点直接实现，真实节点通过 `elementActionAdapter` 委托 `*ax.Element` | 2026-03-26 |
| 56 | **交互动作操作首元素** | 与属性方法一致：Click/SetValue/Focus/TypeText/Press/Perform 仅操作 Selection 的第一个元素 | 2026-03-26 |
| 57 | **交互动作返回原 Selection 支持链式** | action 方法返回 `*Selection` 自身（不创建新 Selection），出错时设置 `s.err`，后续链式调用自动短路 | 2026-03-26 |
| 58 | **Focus() 通过 AXRaise 实现** | ax 包无直接 SetFocused API，Focus() 委托 `performAction("AXRaise")` 作为最接近的通用聚焦行为 | 2026-03-26 |
| 59 | **TypeText/Press 通过包级函数变量注入** | `typeTextFn`（默认 `ax.TypeText`）和 `keyPressFn`（默认 `ax.KeyPress`）允许纯单元测试替换 CGo 键盘函数 | 2026-03-26 |
| 60 | **Press 修饰符使用字符串映射** | `Press(key, "command", "shift")` 接受人类可读的修饰符字符串，内部通过 `modifierMap` 转换为 `ax.Modifier`；支持别名（cmd/command、alt/option、ctrl/control） | 2026-03-26 |
| 61 | **空 Selection 的 action 报 ErrNotActionable** | 在空/无 actionable 节点的 Selection 上调用交互方法返回 `NotActionableError`，而非 `NotFoundError`（更精确地描述操作失败原因） | 2026-03-26 |
| 62 | **WaitUntil 通用轮询核心** | `WaitUntil(fn, timeout)` 是所有等待方法的基础；`WaitVisible`/`WaitEnabled`/`WaitGone` 均为其特化。超时返回 `TimeoutError` (wraps `ErrTimeout`) | 2026-03-26 |
| 63 | **默认轮询间隔 200ms** | `DefaultPollInterval = 200ms`，在响应性和 CPU 使用之间取平衡；作为常量暴露而非 option，简化 API | 2026-03-26 |
| 64 | **sleepFn/nowFn 包级变量注入时间** | 与 `typeTextFn`/`keyPressFn` 模式一致：`sleepFn`（默认 `time.Sleep`）和 `nowFn`（默认 `time.Now`）允许纯单元测试控制时间流逝，避免真实 sleep | 2026-03-26 |
| 65 | **WaitGone 检查 Role() == "" 判断元素消失** | 真实 AX 元素被销毁后 `Role()` 返回错误 → `GetRole()` 返回 ""。WaitGone 以此为信号而非 re-query，因为 Selection 不持有 root/resolver 引用（决策 #28） | 2026-03-26 |
| 66 | **错误 Selection 上 WaitGone 立即返回** | 已 errored 的 Selection 语义上等同于"已消失"，WaitGone 直接返回不轮询 | 2026-03-26 |
| 67 | **WaitVisible/WaitEnabled 依赖 AX 属性的实时性** | `IsHidden()`/`IsEnabled()` 每次调用都通过 AX API 实时查询（非缓存），因此轮询中无需 re-query Selection | 2026-03-26 |
| 68 | **滚动方法复用 actionable 接口** | ScrollDown/ScrollUp/ScrollIntoView 与 action 方法共用 `firstActionable()` + `actionable` 接口，保持一致的首元素操作和错误处理模式 | 2026-03-26 |
| 69 | **ScrollDown/ScrollUp 按页滚动** | `ScrollDown(n)` 执行 `AXScrollDownByPage` n 次；`ScrollUp(n)` 执行 `AXScrollUpByPage` n 次。按页滚动是 AX 最可靠的滚动原语 | 2026-03-26 |
| 70 | **ScrollIntoView 使用 AXScrollToVisible** | `ScrollIntoView()` 委托 `performAction("AXScrollToVisible")`，是 AX 原生的"确保元素在可视区域"操作 | 2026-03-26 |
| 71 | **n <= 0 时滚动为 no-op 成功** | `ScrollDown(0)` / `ScrollUp(-1)` 不执行任何操作，直接返回原 Selection。防御性设计，避免意外的反向滚动 | 2026-03-26 |
| 72 | **滚动失败快速中止** | 多次滚动（n>1）时，若中途某次 `performAction` 失败，立即设置 `s.err` 并返回，不继续后续滚动 | 2026-03-26 |
| 73 | **JS Runtime 使用 functional options** | `New(WithTimeout(...), WithOnLog(...))` 模式，与 axquery 的 `QueryOption` 保持一致；options 存储在不可导出的 `runtimeConfig` 中 | 2026-03-26 |
| 74 | **Runtime.Reset() 重建 VM 保留配置** | `Reset()` 创建新的 `goja.Runtime`，但保留 timeout/callback 等配置；用于超时恢复或测试隔离 | 2026-03-26 |
| 75 | **ScriptError 统一包装 JS 错误** | 所有 `Execute`/`ExecuteFile` 的 JS 错误（语法错误、throw、中断）统一包装为 `*ScriptError`，携带 Message + Filename + Wrapped；支持 `errors.As` 和 `errors.Unwrap` | 2026-03-26 |
| 76 | **超时通过 time.AfterFunc + vm.Interrupt 实现** | 不使用 context（goja 不支持 context 取消），改用 `time.AfterFunc` 调度 `vm.Interrupt("execution timeout")`，execute 结束后 `timer.Stop()` + `vm.ClearInterrupt()` 清理 | 2026-03-26 |
| 77 | **ExecuteFile 复用 execute 内核** | `ExecuteFile(path)` 只做 `os.ReadFile` + 委托 `execute(script, filename)`，避免代码重复；filename 传入用于 ScriptError 的源定位 | 2026-03-26 |
| 78 | **SystemBridge 接口抽象 OS 操作** | `$clipboard` 和 `$keyboard` 通过 `SystemBridge` 接口操作（`ClipboardRead/Write`、`KeyPress`、`TypeText`），测试可注入 fake bridge 验证行为和参数；默认 `defaultBridge` 委托 ax 包函数 | 2026-03-27 |
| 79 | **WithBridge functional option** | 与其他 RuntimeOption 保持一致；不传则使用 `defaultBridge{}`（ax.ClipboardRead 等） | 2026-03-27 |
| 80 | **$env/$input/$output 全局对象** | `$env` 为只读字符串 map（SetEnv 重新注入）；`$input` 为任意结构 map；`$output` 为 JS 侧可写的 goja.Object，Go 侧通过 `Output()` 导出 | 2026-03-27 |
| 81 | **console.log/warn/error 统一走 onLog callback** | 与 `$log` 共用 `emitLog(level, args)` 路径，保持单一日志出口；level 分为 "log"/"warn"/"error" | 2026-03-27 |
| 82 | **injectGlobals 在 New + Reset 中调用** | 确保 Reset 后全局变量仍可用；env/input/output 状态也被重建 | 2026-03-27 |
| 83 | **$app 先尝试 BundleID 再尝试 Name** | `ax.ApplicationFromBundleID` 优先，失败后 fallback 到 `ax.ApplicationFromName`；兼容两种命名方式 | 2026-03-27 |
| 84 | **parseModifier 支持别名** | cmd/command、alt/option、ctrl/control 均可识别；case-insensitive；未知修饰符静默忽略（返回 0） | 2026-03-27 |
| 85 | **wrapSelection 全方法桥接** | `js/bridge.go` 中 `wrapSelection` 将 Go `*Selection` 的全部公开方法（count/find/filter/each/map/click/wait/scroll 等）桥接为 goja JS 对象方法；返回 Selection 的方法递归包装，支持完整链式调用 | 2026-03-27 |
| 86 | **JS each 回调 break 检测安全** | `each` 回调返回值先检查 `!= goja.Undefined() && != goja.Null()`，再通过 `ExportType().Kind() == reflect.Bool` 判断是否显式 return false；避免 undefined 导致 nil pointer panic | 2026-03-27 |
| 87 | **errNotAFunction panic 语义** | JS 中传非函数给 each/map/filterFunction/waitUntil 时，通过 `vm.NewGoError` 抛出 Go error（goja panic 语义），被 goja 捕获为 JS 异常 | 2026-03-27 |
| 88 | **不桥接到 JS 的方法** | `Elements()`、`FilterMatcher()`、`NotMatcher()`、`EachIter()` 不暴露到 JS——前者返回 Go 指针数组无法安全导出，后三者是 Go 侧优化接口，JS 有等价替代 | 2026-03-27 |
| 89 | **NewSelection/NewSelectionError 导出构造函数** | root 包导出 `NewSelection` 和 `NewSelectionError`，让 `js` 包测试可以不依赖 AX 权限直接构造 Selection 进行 bridge 测试 | 2026-03-27 |
| 90 | **boolKind 缓存** | `var boolKind = reflect.Bool` 避免 each 热路径中重复分配 reflect.Kind 值 | 2026-03-27 |

## 2. 关键发现

### 2.1 旧系统超时根因

旧 Rust 系统的 `find_raw()` 和 `get_ui_tree()` 做 DFS 全树遍历，对 Mail 等大型应用（10000+ 节点）导致 8-10s 超时。

**根本原因不是语言性能，而是用法错误：**
- 不应该加载整棵树来找一个按钮
- 缺少 `ChildCount` — 无法预判子元素数量
- 缺少 `ChildrenRange` — 无法分页加载大列表
- 搜索总是从窗口根开始 — 无法限定子树范围
- 仅 DFS — 深层元素被优先找到

**新系统解决方案：** 在 Level 0 `ax` 提供精细原语，Level 1 `axquery` 实现 BFS + 作用域搜索 + 早期终止 + 分页加载。

### 2.2 Go AX 可行性

最初对 Go 重写的担忧是 Go 缺乏成熟的 macOS AX 生态。但用户已有的 `pplx-cli` 项目验证了 Go + CGo + ObjC 方案完全可行，且已实现：
- 应用激活、PID 查找、窗口等待
- AX 树遍历、按钮搜索、动作执行
- 剪贴板、键盘事件、截图
- 辅助功能信任检查

### 2.3 goquery 启发

goquery (14.9k stars) 为 HTML DOM 提供了成熟的 jQuery-like Go API。axquery 直接学习其设计：
- `Selection` 核心类型 + 链式调用
- `Find` / `Children` / `Parent` / `Closest` 遍历
- `Each` / `Map` / `Filter` 集合操作
- `Matcher` 接口 + 预编译选择器
- `Attr` / `AttrOr` / `Text` 属性读取

### 2.4 MCP tools 精简

因为有了 `execute_js` tool（执行任意 JS 代码），AI agent 可以在一个 tool 调用中完成以前需要 5+ 个 tool 调用的操作。MCP tools 从 20+ 个精简到 ~7 个。

### 2.5 axquery 可测试性架构

在实现 Task 6（Query 引擎）时发现，`*ax.Element` 的所有方法返回 `(value, error)` 且涉及 CGo 调用，无法在纯单元测试中使用。解决方案是引入三层内部接口：

```
queryNode（遍历接口）
├── queryChildren() → []queryNode
├── element() → *ax.Element
└── embeds selector.Matchable

axElementReader（属性读取接口）
├── Role/Title/Description/... → (string, error)
├── IsEnabled/IsHidden/... → (bool, error)
└── Attribute(name) → (*ax.Value, error)

rootResolver（根节点获取接口）
└── resolveRoot() → (queryNode, error)
```

**成果：** query 引擎的 BFS/DFS 逻辑、elementAdapter 的属性映射、queryWithResolver 的根解析都可以纯 mock 单测。CGo 依赖仅存在于两个薄封装 (`appRootResolver.resolveRoot` 和 `Query`)，它们只在集成测试中覆盖。

总覆盖率 95.5%，满足 95%+ 要求。

## 3. 旧系统审计摘要

### 3.1 旧系统架构

- 语言：Rust
- 脚本格式：JSON step 数组（`define_steps!` 宏 + `serde_json::Value`）
- 选择器：4 个扁平 AND 字段（role/title/description/value），role 精确其余 substring
- 遍历：仅 DFS，`TreeWalker` + `TreeVisitor` trait
- 桌面框架：Tauri（旧）
- AX 封装：`accessibility` + `accessibility-sys` crate（非 AccessKit）

### 3.2 旧系统痛点

| 问题 | 影响 |
|------|------|
| 选择器表达力弱 | 无法表达"AXSheet 内的 AXButton"等结构化查询 |
| 每次从根 DFS | 大应用超时 |
| 无分页/计数 | 大列表（邮件 1000+ 行）导致超时 |
| JSON step DSL | 弱类型，难读难写 |
| 双重遍历 | elementIndex 需要两次全树遍历 |
| 字符串错误 | 无法区分"未找到"和"超时" |

### 3.3 保留的优点

- Mail 摘要提取已能工作
- 带标注截图功能完善
- Blockly 编辑器基础可复用思路
- MCP server 架构方向正确

## 4. 技术选型对比

### 4.1 桌面框架对比

| 框架 | Menubar | Web UI | 维护状态 | 选择理由 |
|------|---------|--------|---------|---------|
| **Wails v3** ✅ | 一流支持 | 核心架构 | 活跃 alpha | Go 后端 + Web 前端 + 原生壳 |
| systray | 仅 tray/menu | 需自行搭建 | 活跃 | 太底层，需要大量胶水 |
| Fyne | 有 | 原生 widget，非 web | 活跃 | 不是 web-first |
| go-app | 无 | PWA | 活跃 | 不是桌面应用 |
| go-astilectron | 可能 | Electron | 老旧 | 维护不足 |

### 4.2 JS 引擎对比

| 引擎 | 类型 | ES 版本 | 依赖 | 选择理由 |
|------|------|---------|------|---------|
| **goja** ✅ | 纯 Go | ES5.1+ | 无 | 零额外依赖，足够用 |
| v8go | V8 binding | ES2022+ | 需要 V8 | 太重 |
| quickjs-go | QuickJS binding | ES2020+ | 需要 C | 性能好但增加复杂性 |

## 5. 待实现事项

### Phase 1: 基础层（ax + axquery 核心）
- [ ] ax 包：CGo/ObjC bridge 基础
- [ ] ax 包：Element/Application/Value 类型
- [ ] ax 包：属性读取 + 子元素访问 + 动作
- [ ] ax 包：ChildCount + ChildrenRange（性能关键）
- [x] axquery 包：选择器解析器（Task 2-4, 97.1% coverage）
- [x] axquery 包：Selection 类型 + 基本缩减（Task 5, 100% coverage）
- [x] axquery 包：BFS/DFS 搜索引擎（Task 6, 95.9% total coverage）
- [x] axquery 包：Selection 遍历方法（Find/Children/Parent）— Task 7, 95.1% coverage
- [x] axquery 包：Selection 过滤方法（Filter/Not/Has/Is/Contains）— Task 8, 95.1% coverage
- [x] axquery 包：属性读取（Attr/Text/Val/Role/Title/IsVisible等）— Task 9, 95.4% coverage
- [x] axquery 包：迭代回调（Each/EachWithBreak/Map/EachIter）— Task 10, 95.6% coverage
- [x] axquery 包：交互动作（Click/SetValue/TypeText/Press/Focus/Perform）— Task 11, 94.9% coverage
- [x] axquery 包：等待方法（WaitUntil/WaitGone/WaitVisible/WaitEnabled）— Task 12, 95.5% coverage
- [x] axquery 包：滚动方法（ScrollIntoView/ScrollDown/ScrollUp）— Task 13, 95.1% coverage
- [x] axquery 包：goja JS 运行时脚手架（Runtime/Execute/ExecuteFile）— Task 14, 95.8% coverage
- [ ] axquery 包：JS 全局函数注入（$ax/$app/$delay/$log）— Task 15, next

### Phase 2: 应用层（app 基础）
- [ ] app：Wails v3 项目初始化
- [ ] app：Engine 核心 + 脚本执行
- [ ] app：MCP server + tools 定义
- [ ] app：Unix socket transport
- [ ] app：CLI MCP client
- [ ] app：Explore 基础功能

### Phase 3: 完善
- [ ] app：Streamable HTTP transport + 认证
- [ ] app：stdio transport
- [ ] app：Menubar 系统托盘
- [ ] app：Svelte UI（Explore + 脚本管理）
- [ ] axblocky：积木定义 + 代码生成
- [ ] axblocky：JS → Blockly 还原
- [ ] app：Blockly 编辑器集成

### Phase 4: 打磨
- [ ] 截图标注
- [ ] 权限检测/引导
- [ ] 配置管理
- [ ] i18n
- [ ] 文档/官网

## 6. 相关文档

- [架构总览](./architecture.md)
- [Level 0: ax 包设计](./level-0-ax.md)
- [Level 1: axquery 包设计](./level-1-axquery.md)
- [Level 2: axblocky 包设计](./level-2-axblocky.md)
- [Level 3: app 设计](./level-3-app.md)
