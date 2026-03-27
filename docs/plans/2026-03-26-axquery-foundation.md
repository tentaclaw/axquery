# axquery 核心包实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 `github.com/tentaclaw/axquery` — 基于 goquery 风格的 macOS AX 查询库，提供 CSS-like 选择器、Selection 链式 API、以及 goja JS 运行时。

**Architecture:** axquery 依赖 Level 0 `ax` 包，分三大模块：(1) 选择器解析编译 (selector 子包)，(2) Selection API (核心 Go 库)，(3) JS 运行时 (js 子包)。选择器解析器使用手写递归下降解析器，编译为 Matcher 接口。Selection 持有 ax.Element 切片，所有操作返回新 Selection 实现链式调用。JS 运行时通过 goja 将 Selection 暴露为 JS 代理对象。

**Tech Stack:** Go 1.22+, github.com/tentaclaw/ax, github.com/dop251/goja (ES5.1+ JS 引擎)

**前置条件:** `ax` 包 Phase 1-6 (Task 1-14) 必须先完成，axquery 依赖 ax 的所有核心类型。

---

## Phase 1: 项目脚手架 + 选择器解析器

### Task 1: 初始化 Go module ✅ `5823185`

**Files:**
- Create: `go.mod`
- Create: `axquery.go`

**Step 1: 初始化 module**

Run:
```
cd /Users/Toby/GoglandProjects/tentaclaw/axquery
go mod init github.com/tentaclaw/axquery
```

**Step 2: 添加 ax 依赖**

`go.mod` 中添加 replace 指令（本地开发期间）:
```
require github.com/tentaclaw/ax v0.0.0

replace github.com/tentaclaw/ax => ../ax
```

**Step 3: 创建包入口**

Create `axquery.go`:
```go
// Package axquery provides a jQuery/goquery-style query API for macOS Accessibility elements.
//
// It builds on the ax package (Level 0) to provide CSS-like selectors, chainable
// Selection operations, and a goja-powered JavaScript runtime for automation scripts.
//
// Core concepts:
//   - Selection: a collection of AX elements with chainable methods
//   - Selector: CSS-like syntax for matching AX elements (e.g., "AXButton[title='OK']")
//   - Matcher: compiled selector for reusable matching
package axquery
```

**Step 4: 验证编译**

Run: `go build ./...`
Expected: 成功

**Step 5: Commit**

```
git init
git add -A
git commit -m "feat(axquery): init go module"
```

---

### Task 2: 选择器 AST 类型 ✅ `38251a7`

**Files:**
- Create: `selector/ast.go`
- Create: `selector/ast_test.go`

**Step 1: 写 AST 测试**

Create `selector/ast_test.go`:
```go
package selector

import "testing"

func TestSelectorString(t *testing.T) {
    // 基本选择器
    s := &SimpleSelector{Role: "AXButton"}
    if s.String() != "AXButton" {
        t.Fatalf("expected AXButton, got %s", s.String())
    }

    // 带属性的选择器
    s2 := &SimpleSelector{
        Role: "AXButton",
        Attrs: []AttrMatcher{
            {Name: "title", Op: OpEquals, Value: "OK"},
        },
    }
    want := `AXButton[title="OK"]`
    if s2.String() != want {
        t.Fatalf("expected %s, got %s", want, s2.String())
    }

    // 通配符
    s3 := &SimpleSelector{
        Role: "*",
        Attrs: []AttrMatcher{
            {Name: "title", Op: OpContains, Value: "Send"},
        },
    }
    want3 := `*[title*="Send"]`
    if s3.String() != want3 {
        t.Fatalf("expected %s, got %s", want3, s3.String())
    }
}

func TestAttrOp_String(t *testing.T) {
    cases := []struct {
        op   AttrOp
        want string
    }{
        {OpEquals, "="},
        {OpContains, "*="},
        {OpPrefix, "^="},
        {OpSuffix, "$="},
        {OpRegex, "~="},
        {OpNotEquals, "!="},
    }
    for _, c := range cases {
        if c.op.String() != c.want {
            t.Errorf("op %d: got %q, want %q", c.op, c.op.String(), c.want)
        }
    }
}

func TestCombinator_String(t *testing.T) {
    cases := []struct {
        c    Combinator
        want string
    }{
        {CombDescendant, " "},
        {CombChild, " > "},
    }
    for _, c := range cases {
        if c.c.String() != c.want {
            t.Errorf("combinator %d: got %q, want %q", c.c, c.c.String(), c.want)
        }
    }
}
```

**Step 2: 运行测试确认失败**

Run: `go test -v ./selector/...`
Expected: 编译失败 — 类型未定义

**Step 3: 实现 ast.go**

Create `selector/ast.go`:
```go
// Package selector implements a CSS-like selector parser for AX elements.
// It is internal to axquery and not published as a separate package.
package selector

import (
    "fmt"
    "strings"
)

// AttrOp 属性匹配操作符
type AttrOp int

const (
    OpEquals    AttrOp = iota // =  精确匹配
    OpContains                // *= 包含
    OpPrefix                  // ^= 前缀
    OpSuffix                  // $= 后缀
    OpRegex                   // ~= 正则
    OpNotEquals               // != 不等于
)

var attrOpStrings = map[AttrOp]string{
    OpEquals:    "=",
    OpContains:  "*=",
    OpPrefix:    "^=",
    OpSuffix:    "$=",
    OpRegex:     "~=",
    OpNotEquals: "!=",
}

func (op AttrOp) String() string {
    if s, ok := attrOpStrings[op]; ok {
        return s
    }
    return fmt.Sprintf("unknown_op(%d)", int(op))
}

// Combinator 组合器
type Combinator int

const (
    CombDescendant Combinator = iota // 空格 - 后代
    CombChild                         // >    - 直接子元素
)

func (c Combinator) String() string {
    switch c {
    case CombDescendant:
        return " "
    case CombChild:
        return " > "
    default:
        return fmt.Sprintf("unknown_comb(%d)", int(c))
    }
}

// PseudoType 伪选择器
type PseudoType int

const (
    PseudoFirst    PseudoType = iota
    PseudoLast
    PseudoNth
    PseudoVisible
    PseudoEnabled
    PseudoFocused
    PseudoSelected
)

// Pseudo 伪选择器
type Pseudo struct {
    Type PseudoType
    N    int // 用于 :nth(N)
}

// AttrMatcher 单个属性匹配条件
type AttrMatcher struct {
    Name  string // 属性名 (如 "title", "description")
    Op    AttrOp // 匹配操作符
    Value string // 匹配值
}

func (am AttrMatcher) String() string {
    return fmt.Sprintf(`[%s%s"%s"]`, am.Name, am.Op, am.Value)
}

// SimpleSelector 表示单个选择器（角色 + 属性 + 伪选择器）
type SimpleSelector struct {
    Role    string        // AX 角色名，"*" 表示通配
    Attrs   []AttrMatcher // 属性匹配条件
    Pseudos []Pseudo      // 伪选择器
}

func (s *SimpleSelector) String() string {
    var b strings.Builder
    b.WriteString(s.Role)
    for _, a := range s.Attrs {
        b.WriteString(a.String())
    }
    for _, p := range s.Pseudos {
        switch p.Type {
        case PseudoFirst:
            b.WriteString(":first")
        case PseudoLast:
            b.WriteString(":last")
        case PseudoNth:
            b.WriteString(fmt.Sprintf(":nth(%d)", p.N))
        case PseudoVisible:
            b.WriteString(":visible")
        case PseudoEnabled:
            b.WriteString(":enabled")
        case PseudoFocused:
            b.WriteString(":focused")
        case PseudoSelected:
            b.WriteString(":selected")
        }
    }
    return b.String()
}

// CompoundStep 组合选择器中的一步（组合器 + 简单选择器）
type CompoundStep struct {
    Combinator Combinator
    Selector   *SimpleSelector
}

// CompoundSelector 组合选择器（如 "AXWindow AXTable > AXRow"）
type CompoundSelector struct {
    Head  *SimpleSelector  // 第一个选择器
    Steps []CompoundStep   // 后续步骤
}

// SelectorGroup 选择器组（逗号分隔，如 "AXButton, AXMenuItem"）
type SelectorGroup struct {
    Selectors []*CompoundSelector
}
```

**Step 4: 运行测试确认通过**

Run: `go test -v ./selector/...`
Expected: PASS

**Step 5: Commit**

```
git add -A
git commit -m "feat(axquery): add selector AST types"
```

---

### Task 3: 选择器解析器 ✅ `a4580e5`

**Files:**
- Create: `selector/parser.go`
- Create: `selector/parser_test.go`

**Step 1: 写解析器测试（覆盖所有语法）**

Create `selector/parser_test.go`:
```go
package selector

import "testing"

func TestParse_SimpleRole(t *testing.T) {
    g, err := Parse("AXButton")
    if err != nil { t.Fatal(err) }
    if len(g.Selectors) != 1 { t.Fatalf("expected 1 selector, got %d", len(g.Selectors)) }
    if g.Selectors[0].Head.Role != "AXButton" { t.Fatalf("expected AXButton, got %s", g.Selectors[0].Head.Role) }
}

func TestParse_Wildcard(t *testing.T) {
    g, err := Parse("*[title=\"OK\"]")
    if err != nil { t.Fatal(err) }
    if g.Selectors[0].Head.Role != "*" { t.Fatalf("expected *, got %s", g.Selectors[0].Head.Role) }
    if len(g.Selectors[0].Head.Attrs) != 1 { t.Fatal("expected 1 attr") }
    if g.Selectors[0].Head.Attrs[0].Name != "title" { t.Fatal("wrong attr name") }
    if g.Selectors[0].Head.Attrs[0].Op != OpEquals { t.Fatal("wrong op") }
    if g.Selectors[0].Head.Attrs[0].Value != "OK" { t.Fatal("wrong value") }
}

func TestParse_AllAttrOps(t *testing.T) {
    cases := []struct {
        input string
        op    AttrOp
    }{
        {`AXButton[title="OK"]`, OpEquals},
        {`AXButton[title*="Send"]`, OpContains},
        {`AXButton[title^="Re:"]`, OpPrefix},
        {`AXButton[title$="btn"]`, OpSuffix},
        {`AXButton[title~="\\d+"]`, OpRegex},
        {`AXButton[title!="No"]`, OpNotEquals},
    }
    for _, c := range cases {
        g, err := Parse(c.input)
        if err != nil { t.Fatalf("Parse(%q): %v", c.input, err) }
        if g.Selectors[0].Head.Attrs[0].Op != c.op {
            t.Errorf("Parse(%q): expected op %v, got %v", c.input, c.op, g.Selectors[0].Head.Attrs[0].Op)
        }
    }
}

func TestParse_MultipleAttrs(t *testing.T) {
    g, err := Parse(`AXButton[title="OK"][description*="confirm"]`)
    if err != nil { t.Fatal(err) }
    if len(g.Selectors[0].Head.Attrs) != 2 { t.Fatalf("expected 2 attrs, got %d", len(g.Selectors[0].Head.Attrs)) }
}

func TestParse_Descendant(t *testing.T) {
    g, err := Parse("AXWindow AXTable")
    if err != nil { t.Fatal(err) }
    cs := g.Selectors[0]
    if cs.Head.Role != "AXWindow" { t.Fatal("wrong head") }
    if len(cs.Steps) != 1 { t.Fatal("expected 1 step") }
    if cs.Steps[0].Combinator != CombDescendant { t.Fatal("expected descendant combinator") }
    if cs.Steps[0].Selector.Role != "AXTable" { t.Fatal("wrong step selector") }
}

func TestParse_Child(t *testing.T) {
    g, err := Parse("AXSheet > AXButton[title=\"OK\"]")
    if err != nil { t.Fatal(err) }
    cs := g.Selectors[0]
    if cs.Head.Role != "AXSheet" { t.Fatal("wrong head") }
    if len(cs.Steps) != 1 { t.Fatal("expected 1 step") }
    if cs.Steps[0].Combinator != CombChild { t.Fatal("expected child combinator") }
    if cs.Steps[0].Selector.Role != "AXButton" { t.Fatal("wrong step role") }
}

func TestParse_Group(t *testing.T) {
    g, err := Parse("AXButton, AXMenuItem")
    if err != nil { t.Fatal(err) }
    if len(g.Selectors) != 2 { t.Fatalf("expected 2 selectors, got %d", len(g.Selectors)) }
    if g.Selectors[0].Head.Role != "AXButton" { t.Fatal("wrong first") }
    if g.Selectors[1].Head.Role != "AXMenuItem" { t.Fatal("wrong second") }
}

func TestParse_Pseudos(t *testing.T) {
    cases := []struct {
        input string
        ptype PseudoType
    }{
        {"AXButton:first", PseudoFirst},
        {"AXButton:last", PseudoLast},
        {"AXButton:visible", PseudoVisible},
        {"AXButton:enabled", PseudoEnabled},
        {"AXButton:focused", PseudoFocused},
        {"AXButton:selected", PseudoSelected},
    }
    for _, c := range cases {
        g, err := Parse(c.input)
        if err != nil { t.Fatalf("Parse(%q): %v", c.input, err) }
        if len(g.Selectors[0].Head.Pseudos) != 1 {
            t.Fatalf("Parse(%q): expected 1 pseudo", c.input)
        }
        if g.Selectors[0].Head.Pseudos[0].Type != c.ptype {
            t.Errorf("Parse(%q): wrong pseudo type", c.input)
        }
    }
}

func TestParse_NthPseudo(t *testing.T) {
    g, err := Parse("AXRow:nth(3)")
    if err != nil { t.Fatal(err) }
    p := g.Selectors[0].Head.Pseudos[0]
    if p.Type != PseudoNth { t.Fatal("expected PseudoNth") }
    if p.N != 3 { t.Fatalf("expected N=3, got %d", p.N) }
}

func TestParse_Complex(t *testing.T) {
    input := `AXWindow AXTable > AXRow:nth(0) AXStaticText`
    g, err := Parse(input)
    if err != nil { t.Fatal(err) }
    cs := g.Selectors[0]
    if cs.Head.Role != "AXWindow" { t.Fatal("wrong head") }
    if len(cs.Steps) != 3 { t.Fatalf("expected 3 steps, got %d", len(cs.Steps)) }
    // AXWindow (descendant) AXTable (child) AXRow:nth(0) (descendant) AXStaticText
    if cs.Steps[0].Combinator != CombDescendant { t.Fatal("step0: expected descendant") }
    if cs.Steps[0].Selector.Role != "AXTable" { t.Fatal("step0: wrong role") }
    if cs.Steps[1].Combinator != CombChild { t.Fatal("step1: expected child") }
    if cs.Steps[1].Selector.Role != "AXRow" { t.Fatal("step1: wrong role") }
    if cs.Steps[2].Combinator != CombDescendant { t.Fatal("step2: expected descendant") }
    if cs.Steps[2].Selector.Role != "AXStaticText" { t.Fatal("step2: wrong role") }
}

func TestParse_Errors(t *testing.T) {
    invalid := []string{
        "",
        "[title=",
        "AXButton[",
        "AXButton[title]",
        ">",
        "AXButton >",
    }
    for _, input := range invalid {
        _, err := Parse(input)
        if err == nil {
            t.Errorf("Parse(%q): expected error, got nil", input)
        }
    }
}

// 支持单引号
func TestParse_SingleQuotes(t *testing.T) {
    g, err := Parse("AXButton[title='OK']")
    if err != nil { t.Fatal(err) }
    if g.Selectors[0].Head.Attrs[0].Value != "OK" { t.Fatal("wrong value") }
}
```

**Step 2: 运行测试确认失败**

Run: `go test -v ./selector/...`
Expected: 编译失败 — Parse 未定义

**Step 3: 实现 parser.go**

手写递归下降解析器。主要函数:
- `Parse(input string) (*SelectorGroup, error)` — 入口
- `parseCompound(tokens) (*CompoundSelector, error)` — 解析组合选择器
- `parseSimple(tokens) (*SimpleSelector, error)` — 解析简单选择器
- `parseAttr(tokens) (AttrMatcher, error)` — 解析属性条件
- `parsePseudo(tokens) (Pseudo, error)` — 解析伪选择器

关键实现细节:
- 先做 tokenizer (role/attribute-bracket/combinator/pseudo/comma)
- 然后递归下降解析 token 流

**Step 4: 运行测试**

Run: `go test -v ./selector/...`
Expected: 全部 PASS

**Step 5: Commit**

```
git add -A
git commit -m "feat(axquery): implement selector parser with all operators"
```

---

### Task 4: 选择器 Matcher 编译 ✅ `79090fa`

**Files:**
- Create: `selector/matcher.go`
- Create: `selector/compiler.go`
- Create: `selector/matcher_test.go`

**Step 1: 写 Matcher 测试**

Create `selector/matcher_test.go`:
```go
package selector

import "testing"

// MockElement 实现 Element 接口用于测试
type MockElement struct {
    role        string
    title       string
    description string
    value       string
    enabled     bool
    visible     bool
    focused     bool
    selected    bool
}

func (m *MockElement) GetRole() string        { return m.role }
func (m *MockElement) GetTitle() string       { return m.title }
func (m *MockElement) GetDescription() string { return m.description }
func (m *MockElement) GetValue() string       { return m.value }
func (m *MockElement) GetAttr(name string) string {
    switch name {
    case "title": return m.title
    case "description": return m.description
    case "value": return m.value
    case "role": return m.role
    default: return ""
    }
}
func (m *MockElement) IsEnabled() bool  { return m.enabled }
func (m *MockElement) IsVisible() bool  { return m.visible }
func (m *MockElement) IsFocused() bool  { return m.focused }
func (m *MockElement) IsSelected() bool { return m.selected }

func TestMatcher_SimpleRole(t *testing.T) {
    m, err := Compile("AXButton")
    if err != nil { t.Fatal(err) }
    btn := &MockElement{role: "AXButton", enabled: true, visible: true}
    text := &MockElement{role: "AXStaticText"}
    if !m.MatchSimple(btn) { t.Fatal("expected match for AXButton") }
    if m.MatchSimple(text) { t.Fatal("expected no match for AXStaticText") }
}

func TestMatcher_AttrEquals(t *testing.T) {
    m, err := Compile(`AXButton[title="OK"]`)
    if err != nil { t.Fatal(err) }
    ok := &MockElement{role: "AXButton", title: "OK"}
    no := &MockElement{role: "AXButton", title: "Cancel"}
    if !m.MatchSimple(ok) { t.Fatal("expected match") }
    if m.MatchSimple(no) { t.Fatal("expected no match") }
}

func TestMatcher_AttrContains(t *testing.T) {
    m, err := Compile(`AXButton[title*="end"]`)
    if err != nil { t.Fatal(err) }
    ok := &MockElement{role: "AXButton", title: "Send Message"}
    no := &MockElement{role: "AXButton", title: "Cancel"}
    if !m.MatchSimple(ok) { t.Fatal("expected match") }
    if m.MatchSimple(no) { t.Fatal("expected no match") }
}

func TestMatcher_AttrPrefix(t *testing.T) {
    m, err := Compile(`*[title^="Re:"]`)
    if err != nil { t.Fatal(err) }
    ok := &MockElement{role: "AXWindow", title: "Re: Hello"}
    no := &MockElement{role: "AXWindow", title: "Fwd: Hello"}
    if !m.MatchSimple(ok) { t.Fatal("expected match") }
    if m.MatchSimple(no) { t.Fatal("expected no match") }
}

func TestMatcher_AttrRegex(t *testing.T) {
    m, err := Compile(`AXStaticText[title~="\\d+ unread"]`)
    if err != nil { t.Fatal(err) }
    ok := &MockElement{role: "AXStaticText", title: "42 unread messages"}
    no := &MockElement{role: "AXStaticText", title: "no unread"}
    if !m.MatchSimple(ok) { t.Fatal("expected match") }
    if m.MatchSimple(no) { t.Fatal("expected no match") }
}

func TestMatcher_PseudoEnabled(t *testing.T) {
    m, err := Compile("AXButton:enabled")
    if err != nil { t.Fatal(err) }
    ok := &MockElement{role: "AXButton", enabled: true}
    no := &MockElement{role: "AXButton", enabled: false}
    if !m.MatchSimple(ok) { t.Fatal("expected match for enabled") }
    if m.MatchSimple(no) { t.Fatal("expected no match for disabled") }
}

func TestMatcher_Wildcard(t *testing.T) {
    m, err := Compile("*")
    if err != nil { t.Fatal(err) }
    if !m.MatchSimple(&MockElement{role: "AXButton"}) { t.Fatal("wildcard should match anything") }
    if !m.MatchSimple(&MockElement{role: "AXWindow"}) { t.Fatal("wildcard should match anything") }
}
```

**Step 2: 运行测试确认失败**

**Step 3: 定义 Matchable 接口和实现 compiler**

`selector/matcher.go`:
```go
package selector

// Matchable 是选择器可以匹配的元素接口
// ax.Element 会实现此接口（通过适配器）
type Matchable interface {
    GetRole() string
    GetTitle() string
    GetDescription() string
    GetValue() string
    GetAttr(name string) string
    IsEnabled() bool
    IsVisible() bool
    IsFocused() bool
    IsSelected() bool
}

// Matcher 预编译的选择器匹配器
type Matcher interface {
    // MatchSimple 判断单个元素是否匹配（不含组合器/伪选择器中的集合操作）
    MatchSimple(el Matchable) bool
}
```

`selector/compiler.go`:
```go
package selector

import (
    "regexp"
    "strings"
)

// Compile 将选择器字符串编译为 Matcher
func Compile(sel string) (CompiledSelector, error) {
    group, err := Parse(sel)
    if err != nil { return CompiledSelector{}, err }
    return compileGroup(group)
}

// CompiledSelector 编译后的选择器
type CompiledSelector struct {
    group *SelectorGroup
    // 预编译的正则表达式缓存
    regexCache map[string]*regexp.Regexp
}

func (cs CompiledSelector) MatchSimple(el Matchable) bool {
    // 对 group 中的每个 compound selector，只检查最后一个 simple selector
    // （组合器需要在 Selection 层处理，这里只做叶子匹配）
    for _, compound := range cs.group.Selectors {
        last := compound.Head
        if len(compound.Steps) > 0 {
            last = compound.Steps[len(compound.Steps)-1].Selector
        }
        if matchSimpleSelector(last, el, cs.regexCache) {
            return true
        }
    }
    return false
}

func matchSimpleSelector(s *SimpleSelector, el Matchable, regexCache map[string]*regexp.Regexp) bool {
    // 角色匹配
    if s.Role != "*" && el.GetRole() != s.Role {
        return false
    }
    // 属性匹配
    for _, attr := range s.Attrs {
        val := el.GetAttr(attr.Name)
        if !matchAttr(attr, val, regexCache) {
            return false
        }
    }
    // 伪选择器 (布尔型)
    for _, p := range s.Pseudos {
        switch p.Type {
        case PseudoEnabled:
            if !el.IsEnabled() { return false }
        case PseudoVisible:
            if !el.IsVisible() { return false }
        case PseudoFocused:
            if !el.IsFocused() { return false }
        case PseudoSelected:
            if !el.IsSelected() { return false }
        // first/last/nth 在 Selection 层处理
        }
    }
    return true
}

func matchAttr(am AttrMatcher, val string, regexCache map[string]*regexp.Regexp) bool {
    switch am.Op {
    case OpEquals:
        return val == am.Value
    case OpContains:
        return strings.Contains(val, am.Value)
    case OpPrefix:
        return strings.HasPrefix(val, am.Value)
    case OpSuffix:
        return strings.HasSuffix(val, am.Value)
    case OpNotEquals:
        return val != am.Value
    case OpRegex:
        re, ok := regexCache[am.Value]
        if !ok {
            var err error
            re, err = regexp.Compile(am.Value)
            if err != nil { return false }
            regexCache[am.Value] = re
        }
        return re.MatchString(val)
    }
    return false
}

func compileGroup(group *SelectorGroup) (CompiledSelector, error) {
    return CompiledSelector{
        group:      group,
        regexCache: make(map[string]*regexp.Regexp),
    }, nil
}
```

**Step 4: 运行测试确认通过**

Run: `go test -v ./selector/...`
Expected: 全部 PASS

**Step 5: Commit**

```
git add -A
git commit -m "feat(axquery): implement selector compiler and matcher"
```

---

## Phase 2: Selection 核心

### Task 5: Selection 类型 + 构造 + 基础方法 ✅ `5810571`

**Files:**
- Create: `selection.go`
- Create: `options.go`
- Create: `errors.go`

**Step 1: 实现基础 Selection 类型**

`errors.go` — axquery 级别的错误类型:
- NotFoundError (选择器无匹配)
- TimeoutError
- AmbiguousError (匹配多个但需要单个)
- InvalidSelectorError
- NotActionableError

`options.go` — QueryOptions 和 functional option 模式:
- WithTimeout(d)
- WithMaxDepth(n)
- WithStrategy(s)
- WithMaxResults(n)

`selection.go` — 核心 Selection 结构:
- Selection 持有 []*ax.Element
- 所有操作返回 *Selection
- 错误暂存在 err 字段
- Count(), Err(), IsEmpty()
- First(), Last(), Eq(i), Slice(start, end)

**Step 2: 写测试 + 运行**

**Step 3: Commit**

```
git add -A
git commit -m "feat(axquery): add Selection type, options, and error types"
```

---

### Task 6: Query 入口 + 搜索引擎 (BFS) ✅ `3882738`

**Files:**
- Create: `query.go`
- Create: `query_test.go`

**Step 1: 实现 BFS 搜索**

`query.go`:
```go
// Query 是 axquery 的核心入口函数
// 在应用的窗口中搜索匹配选择器的元素
func Query(app *ax.Application, selector string, opts ...QueryOption) *Selection

// queryBFS 宽度优先搜索实现
func queryBFS(root *ax.Element, matcher CompiledSelector, opts QueryOptions) ([]*ax.Element, error)
```

关键实现:
- 从 app.FocusedWindow() 或 app.MainWindow() 获取根
- BFS 队列 + ax.Children() 逐层展开
- 每个节点用 matcher.MatchSimple() 检测
- 达到 MaxResults 或 MaxDepth 后停止
- Timeout 用 context.WithTimeout 控制

**Step 2: 集成测试（需要真实 macOS AX）**

```go
func TestQuery_FinderButtons(t *testing.T) {
    if !ax.IsTrusted(false) { t.Skip("no AX permission") }
    app, _ := ax.ApplicationFromBundleID("com.apple.finder")
    if app == nil { t.Skip("Finder not available") }
    defer app.Close()

    sel := Query(app, "AXButton")
    if sel.Err() != nil { t.Fatal(sel.Err()) }
    if sel.Count() == 0 { t.Fatal("expected at least one button") }
    t.Logf("Found %d buttons", sel.Count())
}
```

**Step 3: Commit**

```
git add -A
git commit -m "feat(axquery): implement BFS query engine"
```

---

### Task 7: Selection 遍历方法 (Find, Children, Parent, Closest) ✅

**Files:**
- Create: `traversal.go`
- Create: `traversal_test.go`

实现 goquery 风格遍历:
- `Find(selector)` — 在当前 Selection 的每个元素子树内搜索
- `Children()` / `ChildrenFiltered(selector)`
- `Parent()` / `ParentFiltered(selector)`
- `Parents()` / `ParentsUntil(selector)`
- `Closest(selector)`
- `Siblings()` / `Next()` / `Prev()` (需要 Parent + Children)

**Commit:**
```
git add -A
git commit -m "feat(axquery): add traversal methods (Find/Children/Parent/Closest)"
```

---

### Task 8: Selection 过滤方法 (Filter, Not, Has) ✅

**Files:**
- Create: `filter.go`
- Create: `filter_test.go`

实现:
- `Filter(selector)` / `FilterFunction(fn)` / `FilterMatcher(m)`
- `Not(selector)` / `NotMatcher(m)`
- `Has(selector)` — 保留包含匹配后代元素的
- `Is(selector)` — bool 判断
- `Contains(text)` — 按 title 子串过滤

**Commit:**
```
git add -A
git commit -m "feat(axquery): add filter methods (Filter/Not/Has/Is/Contains)"
```

---

### Task 9: Selection 属性读取 ✅

**Files:**
- Create: `property.go`
- Create: `property_test.go`

实现:
- `Attr(name)` / `AttrOr(name, default)`
- `Text()` — 组合 AXValue + AXTitle + 递归子元素文本
- `Val()` — AXValue
- `Role()` / `Title()` / `Description()`
- `Bounds()` — ax.Rect
- `IsVisible()` / `IsEnabled()` / `IsFocused()` / `IsSelected()`

**Commit:**
```
git add -A
git commit -m "feat(axquery): add property methods (Attr/Text/Val/Role etc.)"
```

---

### Task 10: Selection 遍历回调 (Each, Map) ✅

**Files:**
- Create: `iteration.go`
- Create: `iteration_test.go`

实现:
- `Each(fn func(int, *Selection)) *Selection`
- `EachWithBreak(fn func(int, *Selection) bool) *Selection`
- `Map(fn func(int, *Selection) string) []string`
- `EachIter() iter.Seq2[int, *Selection]` (Go 1.23+)

**Commit:**
```
git add -A
git commit -m "feat(axquery): add iteration methods (Each/Map/EachIter)"
```

---

### Task 11: Selection 交互动作 ✅

**Files:**
- Create: `action.go`
- Create: `action_test.go`

实现:
- `Click()` — 对第一个元素执行 AXPress
- `SetValue(v string)` — 设置 AXValue
- `TypeText(text string)` — Focus + ax.TypeText
- `Press(key, modifiers...)` — ax.KeyPress
- `Focus()` — 设置 AXFocused=true
- `Perform(action string)` — 任意 AX action

所有方法返回 `*Selection` 支持链式调用。出错时存入 `s.err`。

**Commit:**
```
git add -A
git commit -m "feat(axquery): add interaction methods (Click/SetValue/TypeText)"
```

---

### Task 12: Selection 等待方法 ✅

**Files:**
- Create: `waiting.go`
- Create: `waiting_test.go`

实现:
- `WaitUntil(fn, timeout)` — 轮询直到条件满足
- `WaitGone(timeout)` — 等待元素消失
- `WaitVisible(timeout)` — 等待可见
- `WaitEnabled(timeout)` — 等待启用

轮询间隔默认 200ms，可配置。

**Commit:**
```
git add -A
git commit -m "feat(axquery): add wait methods (WaitUntil/WaitGone/WaitVisible)"
```

---

### Task 13: Selection 滚动方法 ✅

**Files:**
- Create: `scroll.go`
- Create: `scroll_test.go`

实现:
- `ScrollIntoView()` — 确保元素在可视区域
- `ScrollDown(n)` / `ScrollUp(n)` — 通过 AX action
- 依赖 ax 包的 PerformAction("AXScrollDownByPage")

**Commit:**
```
git add -A
git commit -m "feat(axquery): add scroll methods"
```

---

## Phase 3: JS 运行时

### Task 14: goja 运行时脚手架 ✅ `pending`

**Files:**
- Create: `js/runtime.go`
- Create: `js/runtime_test.go`

**Step 1: 添加 goja 依赖**

```
go get github.com/dop251/goja
```

**Step 2: 实现 Runtime 类型**

`js/runtime.go`:
```go
package js

import (
    "github.com/dop251/goja"
    "github.com/tentaclaw/ax"
)

// Runtime 管理 goja JS 运行时
type Runtime struct {
    vm      *goja.Runtime
    app     *ax.Application
    options RuntimeOptions
}

type RuntimeOptions struct {
    Timeout   time.Duration // 脚本总超时
    OnLog     func(level, msg string)
    OnError   func(err error)
}

func New(opts ...RuntimeOption) *Runtime
func (r *Runtime) SetApp(app *ax.Application)
func (r *Runtime) Execute(script string) (goja.Value, error)
func (r *Runtime) ExecuteFile(path string) (goja.Value, error)
```

**Step 3: 测试基本 JS 执行**

```go
func TestRuntime_BasicJS(t *testing.T) {
    rt := New()
    val, err := rt.Execute("1 + 2")
    if err != nil { t.Fatal(err) }
    if val.ToInteger() != 3 { t.Fatalf("expected 3, got %v", val) }
}
```

**Step 4: Commit**

```
git add -A
git commit -m "feat(axquery): add goja JS runtime scaffold"
```

---

### Task 15: JS 全局函数注入 ($ax, $app, $delay, $log) ✅

**Files:**
- Create: `js/globals.go`
- Create: `js/globals_test.go`

**Step 1: 实现全局注入**

`js/globals.go`:
```go
func (r *Runtime) injectGlobals() {
    r.vm.Set("$ax", r.jsAx)
    r.vm.Set("$app", r.jsApp)
    r.vm.Set("$delay", r.jsDelay)
    r.vm.Set("$log", r.jsLog)
    r.vm.Set("$screenshot", r.jsScreenshot)
    r.vm.Set("$clipboard", r.jsClipboardObj())
    r.vm.Set("$keyboard", r.jsKeyboardObj())
    r.vm.Set("$env", r.env)
    r.vm.Set("$input", r.input)
    r.vm.Set("$output", r.vm.NewObject())
    r.injectConsole()
}

// $ax(selector, opts?) -> JS Selection 代理
func (r *Runtime) jsAx(call goja.FunctionCall) goja.Value {
    selector := call.Argument(0).String()
    sel := axquery.Query(r.app, selector)
    return r.wrapSelection(sel)
}

// $app(nameOrBundleID) -> 切换目标应用
func (r *Runtime) jsApp(call goja.FunctionCall) goja.Value { ... }

// $delay(ms) -> 同步延迟
func (r *Runtime) jsDelay(call goja.FunctionCall) goja.Value { ... }
```

**Step 2: 测试**

```go
func TestRuntime_DollarLog(t *testing.T) {
    var logged string
    rt := New(WithOnLog(func(level, msg string) { logged = msg }))
    rt.Execute(`$log("hello from JS")`)
    if logged != "hello from JS" { t.Fatalf("expected log, got %q", logged) }
}
```

**Step 3: Commit**

```
git add -A
git commit -m "feat(axquery): inject JS globals ($ax/$app/$delay/$log)"
```

---

### Task 16: JS Selection 代理对象 ✅

**Files:**
- Create: `js/bridge.go`
- Create: `js/bridge_test.go`

**Step 1: 实现 wrapSelection**

将 Go *axquery.Selection 包装为 goja JS 对象，暴露所有 Selection 方法:
- `.find(selector)` -> 返回新的 JS Selection
- `.click()` -> 调用 Click()
- `.text()` -> 返回 Text()
- `.count()` -> 返回 Count()
- `.each(fn)` -> 回调 JS 函数
- `.attr(name)` -> 返回属性值
- 等等...

关键: goja 使用 `Object.Set()` 将方法绑定到对象。

**Step 2: 集成测试**

```go
func TestRuntime_AxQuery(t *testing.T) {
    if !ax.IsTrusted(false) { t.Skip("no AX permission") }
    app, _ := ax.ApplicationFromBundleID("com.apple.finder")
    if app == nil { t.Skip("Finder not available") }
    defer app.Close()

    rt := New()
    rt.SetApp(app)
    val, err := rt.Execute(`$ax("AXButton").count()`)
    if err != nil { t.Fatal(err) }
    count := val.ToInteger()
    t.Logf("Finder buttons via JS: %d", count)
    if count <= 0 { t.Fatal("expected >0 buttons") }
}
```

**Step 3: Commit**

```
git add -A
git commit -m "feat(axquery): bridge Go Selection to JS proxy object"
```

---

### Task 17: JS 错误处理 + console ✅

**Files:**
- Modified: `js/globals.go` — 添加 `injectAx()`, `$ax.defaults` 对象
- Modified: `js/bridge.go` — 结构化错误转换, 终端方法抛出, `.err()` 返回结构化对象

实现:
- 结构化错误抛出: terminal 方法（属性读取/actions/waits/scrolls）在 error selection 上抛出 `{code: 'NOT_FOUND', message: '...', selector: '...'}`
- 错误类型映射: NotFoundError→NOT_FOUND, TimeoutError→TIMEOUT, AmbiguousError→AMBIGUOUS, InvalidSelectorError→INVALID_SELECTOR, NotActionableError→NOT_ACTIONABLE, 其他→ERROR
- `.err()` 返回结构化对象（与 throw 相同 shape），无错误时返回 null
- 非终端方法（query/traversal/subset/filter/inspection/iteration）保持链式不抛出
- console.log/warn/error 已在 Task 15 中实现（复用 emitLog 路径）
- `$ax.defaults` = `{timeout: 5000, pollInterval: 200}`，可写，Reset 后保持

**Commit:**
```
git add -A
git commit -m "feat(axquery): add JS error handling and console bridge"
```

---

### Task 18: 脚本执行器

**Files:**
- Create: `js/executor.go`
- Create: `js/executor_test.go`

实现:
- `ExecuteFile(path string) (map[string]interface{}, error)` — 加载 .js 文件执行
- `Execute(script string) (map[string]interface{}, error)` — 直接执行字符串
- 返回值: $output 对象内容
- 超时控制: context + goja.Interrupt
- 输入参数: 通过 $input 传入

**Commit:**
```
git add -A
git commit -m "feat(axquery): add script executor with timeout and I/O"
```

---

## Phase 4: 端到端验证

### Task 19: 端到端集成测试

**Files:**
- Create: `integration_test.go`

完整测试:
1. 创建 Runtime
2. 设置 Finder 为目标应用
3. 执行 JS 脚本查询按钮
4. 验证返回结果
5. 执行 JS 脚本读取窗口属性
6. 验证属性值

```go
func TestE2E_JSQueryFinder(t *testing.T) {
    script := `
        $app('com.apple.finder');
        var buttons = $ax('AXButton');
        var count = buttons.count();
        var titles = [];
        buttons.each(function(i, btn) {
            if (i >= 5) return false;
            titles.push(btn.title());
        });
        $output.count = count;
        $output.titles = titles;
    `
    rt := js.New()
    result, err := rt.Execute(script)
    // 验证 result
}
```

**Commit:**
```
git add -A
git commit -m "test(axquery): add E2E integration tests"
```

---

### Task 20: README + 文档

**Files:**
- Create: `README.md`

包含:
- 包用途和定位
- 安装
- Go API 示例
- JS 脚本示例
- 选择器语法参考
- 与 goquery 的对比

**Commit:**
```
git add -A
git commit -m "docs(axquery): add README with examples"
```

---

## 实现顺序总结

| Phase | Tasks | 预估时间 | 状态 |
|-------|-------|---------|------|
| 1: 脚手架+选择器 | Task 1-4 | 2-3 hours | ✅ 完成 |
| 2: Selection 核心 | Task 5-13 | 4-6 hours | ✅ 完成 |
| 3: JS 运行时 | Task 14-18 | 3-4 hours | 🔄 进行中 (3/5) |
| 4: 集成测试+文档 | Task 19-20 | 1-2 hours | ⬜ 未开始 |
| **总计** | **20 Tasks** | **~10-15 hours** | **16/20 完成** |

## 依赖关系

```
前置: ax 包 Phase 1-6 完成

Task 1 (module init)
  -> Task 2 (AST)
    -> Task 3 (parser)
      -> Task 4 (matcher/compiler)
        -> Task 5 (Selection type)
          -> Task 6 (Query/BFS engine) [核心]
            -> Task 7 (traversal)
            -> Task 8 (filter)
            -> Task 9 (property)
            -> Task 10 (iteration)
            -> Task 11 (action)
            -> Task 12 (waiting)
            -> Task 13 (scroll)
          -> Task 14 (goja scaffold)
            -> Task 15 (globals)
              -> Task 16 (JS bridge) [核心]
                -> Task 17 (JS errors)
                -> Task 18 (executor)
        -> Task 19 (E2E tests)
  -> Task 20 (docs)
```

Task 7-13 之间没有严格依赖，可以并行开发。Task 14-18 之间是线性的。
