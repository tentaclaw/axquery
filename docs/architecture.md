# Tentaclaw 架构设计

> 状态：实现中 — axquery Phase 1-3 接近完成（选择器 + Selection 全部 + JS 运行时 + E2E 集成测试完成，剩余 Task 20 文档收尾）
> 日期：2026-03-26
> 最后更新：2026-03-27

## 1. 项目概述

Tentaclaw 是一个 macOS 辅助功能（Accessibility）自动化平台，允许用户和 AI agent 通过脚本控制任意 macOS 应用的 UI。

本文档描述 Tentaclaw 从 Rust 全面重写为 Go 的架构设计。

## 2. 四层架构

```
┌─────────────────────────────────────────────────────────────┐
│  Level 3: app (Tentaclaw 主应用)                              │
│  Wails v3 + Svelte | MCP Server | CLI | Menubar             │
├─────────────────────────────────────────────────────────────┤
│  Level 2: axblocky (npm/TS 包)                               │
│  Blockly 积木定义 | JS 代码生成 | JS→Blockly 还原             │
├─────────────────────────────────────────────────────────────┤
│  Level 1: axquery (Go 库)                                    │
│  jQuery-style Selection API | CSS-like 选择器 | JS 运行时     │
├─────────────────────────────────────────────────────────────┤
│  Level 0: ax (Go 库)                                         │
│  macOS AXUIElement 原语 | CGo + ObjC bridge                  │
└─────────────────────────────────────────────────────────────┘
```

### 各层职责

| 层 | 包名 | 语言 | 职责 |
|----|------|------|------|
| Level 0 | `github.com/tentaclaw/ax` | Go + CGo + ObjC | macOS AXUIElement 底层原语封装 |
| Level 1 | `github.com/tentaclaw/axquery` | Go | jQuery-style AX 查询 + 选择器 + goja JS 运行时 |
| Level 2 | `github.com/tentaclaw/axblocky` | TypeScript (npm) | Blockly 积木 → JS 代码生成 / JS → Blockly 还原 |
| Level 3 | `github.com/tentaclaw/app` | Go + Svelte | 主应用：Wails v3 桌面壳 + MCP Server + CLI |

### 依赖关系

```
app ──→ axquery ──→ ax
 │
 └──→ axblocky (前端侧通过 npm 引用)

axblocky 独立，只生成/解析 JS 字符串，不依赖 Go 包
```

## 3. 仓库结构

多仓库（multi-repo），每个包独立仓库：

```
github.com/tentaclaw/
├── ax/           # Level 0: macOS AX 原语
├── axquery/      # Level 1: jQuery-like 查询 + JS 运行时
├── axblocky/     # Level 2: Blockly 积木 (npm)
├── app/          # Level 3: 主应用 (Wails v3)
├── landing/      # 官网落地页
└── docs/         # 跨仓库设计文档（本目录）
```

本地开发目录：`/Users/Toby/GoglandProjects/tentaclaw/`

## 4. 进程模型

### 运行时架构

```
┌───────────────────────────────────────────────────┐
│  Tentaclaw.app (Wails v3)                          │
│  ┌───────────────────────────────────────────────┐ │
│  │  MCP Server (核心引擎 Engine)                   │ │
│  │  axquery → ax → macOS AXUIElement API         │ │
│  │                                                │ │
│  │  Transport:                                    │ │
│  │  ├── Unix Socket  ~/.tentaclaw/tentaclaw.sock │ │
│  │  ├── stdio        (本地 AI launcher 拉起)      │ │
│  │  └── Streamable HTTP  127.0.0.1:19840         │ │
│  └───────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────┐ │
│  │  Wails UI (Svelte 前端)                        │ │
│  │  通过 Wails Binding 调用 Engine                │ │
│  └───────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────┐ │
│  │  Menubar Tray (macOS 系统托盘)                  │ │
│  └───────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────┘
          ↑ Unix Socket          ↑ stdio         ↑ Streamable HTTP
          │                      │               │
     ┌────┴─────┐         ┌─────┴─────┐   ┌─────┴──────┐
     │tentaclaw │         │ Claude    │   │ 远程 AI    │
     │  CLI     │         │ Desktop   │   │ Agent      │
     └──────────┘         └───────────┘   └────────────┘
```

### 运行模式

| 模式 | 启动方式 | UI | 说明 |
|------|----------|-----|------|
| 桌面模式（默认） | 双击 Tentaclaw.app | Wails 窗口 + menubar 托盘 | 用户日常使用 |
| MCP 模式 | 被 AI launcher 拉起 | headless | 仅 MCP Server |
| CLI | `tentaclaw run script.js` | 无 | MCP client，连接已运行的 app |

### CLI 与 App 的关系

CLI 是纯 MCP client，**不包含业务逻辑**。类似 Docker CLI 与 Docker daemon 的关系：

1. 用户启动 Tentaclaw.app → MCP server 开始监听 Unix socket
2. CLI 连接 `~/.tentaclaw/tentaclaw.sock` → 发送 MCP tool 调用
3. App 未启动时 CLI 报错："请先启动 Tentaclaw.app"

### 多 Transport 通信

| Transport | 监听地址 | 客户端 | 说明 |
|-----------|----------|--------|------|
| Unix Socket | `~/.tentaclaw/tentaclaw.sock` | CLI | 本地 |
| stdio | 被外部进程启动时激活 | Claude Desktop 等 | 本地 |
| Streamable HTTP | 默认 `127.0.0.1:19840` | 本地/远程 agent | 远程需配置 |

**安全策略：**
- HTTP 默认仅监听 `127.0.0.1`（仅本地）
- 开放远程需用户显式配置绑定 `0.0.0.0`
- 远程访问强制要求 API key 认证 + TLS 加密
- 网络穿透（Tailscale/WireGuard/端口映射）由用户自行处理

## 5. 脚本格式

所有自动化脚本统一使用 **`.js`（JavaScript）** 格式：

- 用户手写 → `.js`
- Blockly 生成 → `.js`
- AI agent 生成 → `.js`
- 三种创作方式产出相同格式，可互相编辑

JS 运行时使用 **goja**（纯 Go 实现的 ES5.1+ 引擎）。

## 6. 核心创新

### 对比旧系统

| 维度 | 旧系统 (Rust) | 新系统 (Go) |
|------|---------------|-------------|
| 语言 | Rust | Go + CGo + ObjC |
| 脚本格式 | JSON step 数组 | `.js` |
| 选择器 | 4 个扁平 AND 字段 | CSS-like 结构化选择器 |
| 查询模型 | 每次从窗口根 DFS 全树 | 作用域搜索 + BFS + 分页 |
| 元素操作 | 独立 MCP tool 调用 | `$ax(...).click()` 链式调用 |
| 控制流 | 自定义 If/ForEach step | 原生 JS if/for/while |
| 桌面框架 | Tauri (旧) | Wails v3 |
| 前端框架 | Svelte | Svelte（保持） |
| GUI 策略 | 浏览器打开 web UI | Wails 原生窗口 + menubar |
| MCP tools | 20+ 个独立 tool | ~7 个（JS 表达大部分操作） |
| CLI | 直接调用引擎 | MCP client，连接 app |

## 7. 测试策略

| 层 | 测试类型 | 工具 | CI 要求 |
|----|---------|------|---------|
| ax | 单元 + 集成（需真实 macOS AX） | `go test` | macOS runner |
| axquery | 选择器解析单元 + mock AX 集成 + 真实 app E2E | `go test` | macOS runner |
| axblocky | 积木定义单元 + 代码生成快照 + 往返测试 | `vitest` | 任意 |
| app | API 测试 + MCP tool 测试 + E2E | `go test` + Playwright/Maestro | macOS runner |

### axquery 测试策略（已验证可行）

axquery 采用**接口抽象**实现纯单元测试，避免对真实 macOS AX 的依赖：

- **selector 包**：纯逻辑，100% 可单测（解析器、AST、编译器、匹配器）
- **query 引擎**：通过 `queryNode` 接口 + mock 节点实现纯单元测试
- **elementAdapter**：通过 `axElementReader` 接口注入 mock，测试 AX 方法映射
- **traversal 层**：通过 `traversableNode` 接口（extends `queryNode` + `queryParent()`）实现双向遍历，纯 mock 单测
- **filter 层**：通过 `selector.Compile` + `MatchSimple` 在 Selection 节点上做过滤/排除/判断，纯 mock 单测
- **等待层**：通过可注入的 `sleepFn`/`nowFn` 包级变量实现纯单元测试，mock 节点的属性（visible/enabled/role）可在测试中动态修改以模拟状态变化
- **JS 运行时层**：通过 `goja` 纯 Go JS 引擎实现，单元测试直接执行 JS 代码验证结果；Runtime 的 `Reset()` 支持测试隔离；`SystemBridge` 接口允许 fake bridge 替换 clipboard/keyboard 操作，实现纯行为测试；`wrapSelection` 将 Go `*Selection` 包装为 goja JS 对象，暴露全部 Selection 方法（count/find/filter/each/map/click/wait 等），支持链式调用和 error-selection 安全传播；终端方法（属性读取/actions/waits/scrolls）在 error selection 上抛出结构化 JS 异常 `{code, message, selector, ...}`，非终端方法保持链式传播；`Executor` 接口和 `Result` 类型将 JS 引擎细节隐藏在公共 API 之后，app 层仅依赖 `axquery.Executor` 接口，不接触 goja 类型
- **CGo 桥接层**（`appRootResolver`、`Query`）：薄封装，仅在集成测试中覆盖
- **E2E 集成测试**：`integration_test.go` 包含两类测试：(1) 纯逻辑 E2E 测试（多步 JS 脚本 → `$output` → Result 验证），不需 AX 权限；(2) Mail.app 真实 E2E 测试（打开 Mail → 查询 AX 树 → 验证元素），需要 macOS AX 权限和 `TENTACLAW_TEST_EMAIL` 环境变量
- **查询深度限制**：`$ax.defaults.maxDepth = 10` 和 `$ax.defaults.maxResults = 0`（无限制）防止对 Mail 等大型 AX 树的无界遍历；JS 侧可通过 `$ax("AXButton", {maxDepth: 2})` 内联覆盖

当前覆盖率：**selector 97.1% / root 96.0% / js 95.1% / 总计 ~96.0%**

## 8. 实现进度

| 层 | 包名 | 状态 | 说明 |
|----|------|------|------|
| Level 0 | `ax` | ✅ 可用 | Phase 1-6 完成，axquery 已依赖 |
| Level 1 | `axquery` | 🚧 实现中 | Phase 1-3 接近完成（选择器 + Selection + JS 运行时 + E2E 测试），剩余 Task 20 文档收尾 |
| Level 2 | `axblocky` | ⬜ 未开始 | 等待 axquery JS 运行时完成 |
| Level 3 | `app` | ⬜ 未开始 | 等待 axquery + axblocky |

### axquery 已完成的模块

| Task | 模块 | 文件 | 覆盖率 |
|------|------|------|--------|
| 1 | Go module 初始化 | `go.mod`, `axquery.go` | — |
| 2 | 选择器 AST | `selector/ast.go` | 100% |
| 3 | 选择器解析器 | `selector/parser.go` | 96.9% |
| 4 | 选择器匹配/编译 | `selector/matcher.go`, `selector/compiler.go` | 97.1% |
| 5 | Selection + Errors + Options | `selection.go`, `errors.go`, `options.go` | 100% |
| 6 | Query 引擎 (BFS/DFS) | `query.go` | 93.9% |
| 7 | Selection 遍历方法 | `traversal.go` | 95.1% |
| 8 | Selection 过滤方法 | `filter.go` | 95.1% |
| 9 | Selection 属性读取 | `property.go` | 95.4% |
| 10 | Selection 迭代方法 | `iteration.go` | 95.6% |
| 11 | Selection 交互动作 | `action.go` | 94.9% |
| 12 | Selection 等待方法 | `waiting.go` | 95.5% |
| 13 | Selection 滚动方法 | `scroll.go` | 95.1% |
| 14 | goja JS 运行时脚手架 | `js/runtime.go` | 95.8% |
| 15 | JS 全局函数注入 + SystemBridge | `js/globals.go`, `js/runtime.go` | 95.5% |
| 16 | JS Selection 代理对象 | `js/bridge.go` | 96.2% |
| 17 | JS 错误处理 + $ax.defaults | `js/bridge.go`, `js/globals.go` | 96.8% |
| 18 | Executor 抽象 + Result 类型 | `executor.go`, `result.go`, `js/runtime.go` 重构 | 96.0% |
| 19 | E2E 集成测试 + 查询深度限制 | `integration_test.go`, `js/globals.go` (`$ax.defaults.maxDepth/maxResults`) | 96.0% |

## 9. 相关文档

- [Level 0: ax 包设计](./level-0-ax.md)
- [Level 1: axquery 包设计](./level-1-axquery.md)
- [Level 2: axblocky 包设计](./level-2-axblocky.md)
- [Level 3: app 设计](./level-3-app.md)
- [决策记录](./decisions.md)
