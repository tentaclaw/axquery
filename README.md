# axquery

A jQuery/goquery-style query library for macOS Accessibility elements. Part of the [Tentaclaw](https://github.com/tentaclaw) automation platform.

axquery provides CSS-like selectors, chainable Selection operations, and a goja-powered JavaScript runtime for automating any macOS application through the Accessibility API.

## Architecture

```
axquery (this package)
  |
  v
ax (Level 0: macOS AXUIElement primitives via CGo + ObjC)
```

axquery is **Level 1** in the Tentaclaw stack. It depends on [`github.com/tentaclaw/ax`](https://github.com/tentaclaw/ax) for low-level AX access and exposes a high-level, developer-friendly API.

## Installation

```bash
go get github.com/tentaclaw/axquery
```

> **Requirements:** macOS with Accessibility permissions granted. The calling process must be trusted in System Settings > Privacy & Security > Accessibility.

## Quick Start

### Go API

```go
package main

import (
    "fmt"
    "log"

    "github.com/tentaclaw/ax"
    "github.com/tentaclaw/axquery"
)

func main() {
    // Open target application
    app, err := ax.ApplicationFromBundleID("com.apple.mail")
    if err != nil {
        log.Fatal(err)
    }
    defer app.Close()

    // Query for elements using CSS-like selectors
    buttons := axquery.Query(app, "AXButton")
    fmt.Printf("Found %d buttons\n", buttons.Count())

    // Chain operations
    okBtn := axquery.Query(app, `AXButton[title="OK"]:enabled`).First()
    if okBtn.Err() == nil {
        okBtn.Click()
    }

    // Traverse the UI tree
    axquery.Query(app, "AXToolbar").
        Find("AXButton").
        Each(func(i int, sel *axquery.Selection) {
            fmt.Printf("Button %d: %s\n", i, sel.Title())
        })

    // Query with options
    sel := axquery.Query(app, "AXStaticText",
        axquery.WithMaxDepth(5),
        axquery.WithMaxResults(10),
    )
    fmt.Printf("Found %d text elements (max depth 5)\n", sel.Count())
}
```

### JavaScript API

axquery includes a goja-powered JS runtime that exposes the full Selection API:

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/tentaclaw/ax"
    "github.com/tentaclaw/axquery/js"
)

func main() {
    app, err := ax.ApplicationFromBundleID("com.apple.mail")
    if err != nil {
        log.Fatal(err)
    }
    defer app.Close()

    // Create a JS runtime
    rt := js.New(
        js.WithTimeout(30 * time.Second),
        js.WithOnLog(func(level, msg string) {
            fmt.Printf("[%s] %s\n", level, msg)
        }),
    )
    rt.SetApp(app)

    // Execute a script
    script := `
        var buttons = $ax("AXButton", {maxDepth: 3});
        $output.count = buttons.count();
        $output.titles = buttons.map(function(i, btn) {
            return btn.title();
        });
    `
    if err := rt.Execute(script); err != nil {
        log.Fatal(err)
    }

    result := rt.Output()
    fmt.Printf("Found %d buttons\n", result.Map()["count"])
}
```

## Selector Syntax

axquery uses a CSS-like selector syntax tailored for AX elements:

### Role Matching

```
AXButton            -- match elements with role "AXButton"
AXStaticText        -- match elements with role "AXStaticText"
*                   -- match any role (wildcard)
```

### Attribute Matching

```
AXButton[title="OK"]           -- exact match
AXButton[title*="Send"]        -- contains substring
AXButton[title^="Re:"]         -- starts with prefix
AXButton[title$=".pdf"]        -- ends with suffix
AXButton[title~="\\d+ items"]  -- regex match
AXButton[title!="Cancel"]      -- not equal
```

Multiple attributes can be combined (AND logic):

```
AXButton[title="OK"][description*="confirm"]
```

### Pseudo-Selectors

```
AXButton:enabled    -- only enabled elements
AXButton:visible    -- only visible elements
AXButton:focused    -- only focused elements
AXButton:selected   -- only selected elements
AXButton:first      -- first match only
AXButton:last       -- last match only
AXRow:nth(3)        -- 4th match (0-indexed)
```

### Combinators

```
AXWindow AXTable              -- descendant (any depth)
AXToolbar > AXButton          -- direct child only
```

### Selector Groups

```
AXButton, AXMenuItem          -- match either role
```

### Complex Selectors

```
AXWindow AXTable > AXRow:nth(0) AXStaticText[title*="Inbox"]
```

## Selection API

All Selection methods return `*Selection`, enabling fluent chaining. Errors propagate through the chain without panicking.

### Query & Subset

| Method | Description |
|--------|-------------|
| `Query(app, sel, opts...)` | Search app's focused window |
| `First()` | First element |
| `Last()` | Last element |
| `Eq(i)` | Element at index i |
| `Slice(start, end)` | Sub-range of elements |

### Traversal

| Method | Description |
|--------|-------------|
| `Find(sel)` | Search within each element's subtree |
| `Children()` | Direct children |
| `ChildrenFiltered(sel)` | Filtered direct children |
| `Parent()` | Parent element |
| `ParentFiltered(sel)` | Filtered parent |
| `Parents()` | All ancestors |
| `ParentsUntil(sel)` | Ancestors up to match |
| `Closest(sel)` | Nearest matching ancestor |
| `Siblings()` | Sibling elements |
| `Next()` | Next sibling |
| `Prev()` | Previous sibling |

### Filtering

| Method | Description |
|--------|-------------|
| `Filter(sel)` | Keep matching elements |
| `FilterFunction(fn)` | Keep elements where fn returns true |
| `Not(sel)` | Remove matching elements |
| `Has(sel)` | Keep elements with matching descendants |
| `Is(sel)` | Test if any element matches (bool) |
| `Contains(text)` | Keep elements containing text in title |

### Properties

| Method | Description |
|--------|-------------|
| `Attr(name)` | Get attribute value |
| `AttrOr(name, def)` | Get attribute with default |
| `Role()` | AX role |
| `Title()` | AX title |
| `Description()` | AX description |
| `Val()` | AX value |
| `Text()` | Recursive text content |
| `IsVisible()` | Visibility state |
| `IsEnabled()` | Enabled state |
| `IsFocused()` | Focus state |
| `IsSelected()` | Selection state |

### Iteration

| Method | Description |
|--------|-------------|
| `Each(fn)` | Iterate with callback |
| `EachWithBreak(fn)` | Iterate with early exit |
| `Map(fn)` | Collect string results |
| `EachIter()` | Go 1.23+ range iterator |

### Actions

| Method | Description |
|--------|-------------|
| `Click()` | Press the element (AXPress) |
| `SetValue(v)` | Set AX value |
| `TypeText(text)` | Type text via keyboard events |
| `Press(key, mods...)` | Press key with modifiers |
| `Focus()` | Raise/focus the element |
| `Perform(action)` | Execute arbitrary AX action |

### Waiting

| Method | Description |
|--------|-------------|
| `WaitUntil(fn, timeout)` | Poll until condition met |
| `WaitVisible(timeout)` | Wait for element to become visible |
| `WaitEnabled(timeout)` | Wait for element to become enabled |
| `WaitGone(timeout)` | Wait for element to disappear |

### Scrolling

| Method | Description |
|--------|-------------|
| `ScrollDown(n)` | Scroll down n pages |
| `ScrollUp(n)` | Scroll up n pages |
| `ScrollIntoView()` | Scroll element into visible area |

## JS Globals

When using the `js` package runtime, the following globals are available:

| Global | Type | Description |
|--------|------|-------------|
| `$ax(selector, opts?)` | Function | Query AX elements, returns JS Selection proxy |
| `$app(nameOrBundleID)` | Function | Switch target application |
| `$delay(ms)` | Function | Synchronous delay |
| `$log(msg)` | Function | Log message |
| `$clipboard.read()` | Method | Read clipboard text |
| `$clipboard.write(text)` | Method | Write clipboard text |
| `$keyboard.type(text)` | Method | Type text |
| `$keyboard.press(key, mods...)` | Method | Press key combo |
| `$env` | Object | Read-only environment variables |
| `$input` | Object | Input parameters from Go |
| `$output` | Object | Output object, readable from Go via `Output()` |
| `console.log/warn/error` | Methods | Logging (routed to onLog callback) |

### $ax.defaults

Global query defaults, writable from JS:

```js
$ax.defaults.timeout      // 5000 (ms)
$ax.defaults.pollInterval // 200 (ms)
$ax.defaults.maxDepth     // 10 (tree levels)
$ax.defaults.maxResults   // 0 (unlimited)
```

Override per-query with inline options:

```js
$ax("AXButton", {maxDepth: 2, maxResults: 5})
```

## Executor Interface

For applications that want to stay decoupled from the JS engine:

```go
// axquery.Executor is engine-agnostic
var exec axquery.Executor = js.New(js.WithTimeout(10 * time.Second))
exec.SetApp(app)
exec.SetInput(map[string]any{"query": "AXButton"})

if err := exec.Execute(`$output.count = $ax($input.query).count()`); err != nil {
    log.Fatal(err)
}

result := exec.Output()
fmt.Printf("Count: %d\n", result.Int())
```

`axquery.Result` provides type-safe accessors:

| Method | Return Type | Description |
|--------|------------|-------------|
| `Type()` | `ResultType` | Nil/String/Int/Float/Bool/Slice/Map |
| `IsNil()` | `bool` | Check for nil |
| `String()` | `string` | String value |
| `Int()` | `int64` | Integer value |
| `Float()` | `float64` | Float value |
| `Bool()` | `bool` | Boolean value |
| `Slice()` | `[]any` | Array value |
| `StringSlice()` | `[]string` | Array as strings |
| `Map()` | `map[string]any` | Object value |
| `Raw()` | `any` | Underlying value |

## Error Handling

### Go Side

axquery uses typed errors with sentinel values for `errors.Is`:

```go
sel := axquery.Query(app, "AXButton[title=\"nonexistent\"]")
if errors.Is(sel.Err(), axquery.ErrNotFound) {
    // handle not found
}
```

| Sentinel | Error Type | Description |
|----------|-----------|-------------|
| `ErrNotFound` | `*NotFoundError` | No matching elements |
| `ErrTimeout` | `*TimeoutError` | Wait exceeded deadline |
| `ErrAmbiguous` | `*AmbiguousError` | Multiple matches where one expected |
| `ErrInvalidSelector` | `*InvalidSelectorError` | Malformed selector |
| `ErrNotActionable` | `*NotActionableError` | Action cannot be performed |

### JS Side

Terminal methods (properties, actions, waits, scrolls) throw structured error objects:

```js
try {
    $ax("AXButton[title='nonexistent']").click();
} catch (e) {
    console.log(e.code);     // "NOT_FOUND"
    console.log(e.message);  // 'axquery: no elements matching ...'
    console.log(e.selector); // "AXButton[title='nonexistent']"
}
```

Non-terminal methods (query, traversal, filter, iteration) propagate errors silently through the chain. Use `.err()` to inspect:

```js
var sel = $ax("AXButton[title='nonexistent']");
var e = sel.err();
if (e) {
    console.log(e.code);  // "NOT_FOUND"
}
```

Error codes: `NOT_FOUND`, `TIMEOUT`, `AMBIGUOUS`, `INVALID_SELECTOR`, `NOT_ACTIONABLE`, `ERROR`.

## Comparison with goquery

axquery is modeled after [goquery](https://github.com/PuerkitoBio/goquery) (14.9k stars), adapting its patterns from HTML DOM to macOS Accessibility:

| Aspect | goquery (HTML) | axquery (AX) |
|--------|---------------|--------------|
| Target | HTML DOM nodes | macOS AX elements |
| Selector | CSS (cascadia) | CSS-like (built-in) |
| Data source | `*html.Node` | `*ax.Element` |
| Roles | HTML tags (`div`, `a`) | AX roles (`AXButton`, `AXTable`) |
| Attributes | HTML attrs (`class`, `id`) | AX attrs (`title`, `description`, `value`) |
| Text | `Text()` from content | `Text()` from recursive `title` |
| Tree access | In-memory DOM | Live AX API (CGo) |
| Pseudo-selectors | CSS standard | `:visible`, `:enabled`, `:focused`, `:selected` |
| JS runtime | N/A | Built-in goja engine |
| Error model | No error propagation | Error Selection with chain propagation |

Key differences from goquery:

1. **Live tree, not in-memory** -- each query/traversal hits the macOS AX API via CGo. This means queries can fail mid-chain due to elements disappearing.
2. **Error propagation** -- Selection carries an `err` field. Terminal methods throw/return errors; non-terminal methods propagate silently.
3. **Actions** -- goquery is read-only. axquery supports Click/SetValue/TypeText/Press/Focus/Perform.
4. **Wait methods** -- Polling-based waits for dynamic UI (WaitVisible, WaitEnabled, WaitGone).
5. **Scroll methods** -- Native AX scroll actions (ScrollDown, ScrollUp, ScrollIntoView).
6. **JS runtime** -- Built-in goja engine with `$ax()` global for scripting.

## Query Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithTimeout(d)` | 0 (none) | Maximum query duration |
| `WithMaxDepth(n)` | 0 (unlimited) | Maximum tree traversal depth |
| `WithMaxResults(n)` | 0 (unlimited) | Stop after n matches |
| `WithStrategy(s)` | BFS | BFS (breadth-first) or DFS (depth-first) |

BFS is the default because in UI automation, the visually shallowest match is usually the desired one.

## Testing

```bash
# Run all unit tests (no AX permission needed)
go test ./...

# Run with verbose output
go test -v -count=1 ./...

# Run E2E tests with Mail.app (requires AX permission + Mail running)
TENTACLAW_TEST_EMAIL=you@example.com go test -v -run TestE2E -count=1 ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Current coverage: **~96%** across all packages.

## Package Structure

```
axquery/
  axquery.go          # Package doc
  query.go            # Query() entry point, BFS/DFS engine
  selection.go        # Selection type, First/Last/Eq/Slice
  traversal.go        # Find/Children/Parent/Siblings
  filter.go           # Filter/Not/Has/Is/Contains
  property.go         # Attr/Role/Title/Text/IsVisible
  iteration.go        # Each/Map/EachIter
  action.go           # Click/SetValue/TypeText/Press
  waiting.go          # WaitUntil/WaitVisible/WaitGone
  scroll.go           # ScrollDown/ScrollUp/ScrollIntoView
  options.go          # QueryOption functional options
  errors.go           # Typed errors + sentinels
  executor.go         # Executor + SystemBridge interfaces
  result.go           # Result type-safe wrapper
  integration_test.go # E2E tests (pure logic + Mail.app)
  selector/
    ast.go            # Selector AST types
    parser.go         # Recursive descent parser
    compiler.go       # Compile selector -> Matcher
    matcher.go        # Matchable + Matcher interfaces
  js/
    runtime.go        # goja Runtime, Execute, ScriptError
    globals.go        # $ax/$app/$delay/$log/$clipboard/$keyboard
    bridge.go         # Go Selection -> JS proxy object
  docs/
    architecture.md   # System architecture
    decisions.md      # Decision log (100+ entries)
    plans/            # Implementation plans
```

## License

MIT
