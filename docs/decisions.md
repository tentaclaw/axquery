# 决策记录

> 最后更新：2026-03-26

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
- [ ] axquery 包：选择器解析器
- [ ] axquery 包：Selection 类型 + 基本遍历（Find/Children/Parent）
- [ ] axquery 包：BFS 搜索引擎
- [ ] axquery 包：属性读取 + 交互动作
- [ ] axquery 包：goja JS 运行时 + $ax() 全局 API

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
