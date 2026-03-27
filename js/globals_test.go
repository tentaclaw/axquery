package js

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tentaclaw/ax"
)

// ---------------------------------------------------------------------------
// $log — maps to onLog callback
// ---------------------------------------------------------------------------

func TestDollarLog_CallsOnLog(t *testing.T) {
	var logs []string
	rt := New(WithOnLog(func(level, msg string) {
		logs = append(logs, level+":"+msg)
	}))
	err := rt.Execute(`$log("hello from JS")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0] != "log:hello from JS" {
		t.Fatalf("expected 'log:hello from JS', got %q", logs[0])
	}
}

func TestDollarLog_WithoutCallback_NoPanic(t *testing.T) {
	rt := New() // no onLog callback
	err := rt.Execute(`$log("silent")`)
	if err != nil {
		t.Fatalf("$log without callback should not error: %v", err)
	}
}

func TestDollarLog_MultipleArgs(t *testing.T) {
	var logs []string
	rt := New(WithOnLog(func(level, msg string) {
		logs = append(logs, msg)
	}))
	err := rt.Execute(`$log("a", "b", "c")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}
	if logs[0] != "a b c" {
		t.Fatalf("expected 'a b c', got %q", logs[0])
	}
}

// ---------------------------------------------------------------------------
// $delay — synchronous sleep
// ---------------------------------------------------------------------------

func TestDollarDelay_Sleeps(t *testing.T) {
	rt := New()
	start := time.Now()
	err := rt.Execute(`$delay(50)`)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if elapsed < 40*time.Millisecond {
		t.Fatalf("expected >=40ms delay, got %v", elapsed)
	}
}

func TestDollarDelay_ZeroMs(t *testing.T) {
	rt := New()
	err := rt.Execute(`$delay(0)`)
	if err != nil {
		t.Fatalf("$delay(0) should not error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// console.log / console.warn / console.error
// ---------------------------------------------------------------------------

func TestConsoleLog_CallsOnLog(t *testing.T) {
	var logs []string
	rt := New(WithOnLog(func(level, msg string) {
		logs = append(logs, level+":"+msg)
	}))
	err := rt.Execute(`console.log("info msg")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(logs) != 1 || logs[0] != "log:info msg" {
		t.Fatalf("expected 'log:info msg', got %v", logs)
	}
}

func TestConsoleWarn_CallsOnLog(t *testing.T) {
	var logs []string
	rt := New(WithOnLog(func(level, msg string) {
		logs = append(logs, level+":"+msg)
	}))
	err := rt.Execute(`console.warn("warn msg")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(logs) != 1 || logs[0] != "warn:warn msg" {
		t.Fatalf("expected 'warn:warn msg', got %v", logs)
	}
}

func TestConsoleError_CallsOnLog(t *testing.T) {
	var logs []string
	rt := New(WithOnLog(func(level, msg string) {
		logs = append(logs, level+":"+msg)
	}))
	err := rt.Execute(`console.error("err msg")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(logs) != 1 || logs[0] != "error:err msg" {
		t.Fatalf("expected 'error:err msg', got %v", logs)
	}
}

func TestConsoleLog_MultipleArgs(t *testing.T) {
	var logs []string
	rt := New(WithOnLog(func(level, msg string) {
		logs = append(logs, msg)
	}))
	err := rt.Execute(`console.log("x", 42, true)`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(logs) != 1 || logs[0] != "x 42 true" {
		t.Fatalf("expected 'x 42 true', got %v", logs)
	}
}

func TestConsoleLog_WithoutCallback_NoPanic(t *testing.T) {
	rt := New()
	err := rt.Execute(`console.log("silent")`)
	if err != nil {
		t.Fatalf("console.log without callback should not error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// $env — environment variables
// ---------------------------------------------------------------------------

func TestDollarEnv_ReadValue(t *testing.T) {
	rt := New()
	rt.SetEnv(map[string]string{"FOO": "bar"})
	err := rt.Execute(`$output = $env.FOO`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().String() != "bar" {
		t.Fatalf("expected 'bar', got %v", rt.Output().Raw())
	}
}

func TestDollarEnv_DefaultEmpty(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = typeof $env`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().String() != "object" {
		t.Fatalf("expected $env to be an object, got %v", rt.Output().Raw())
	}
}

// ---------------------------------------------------------------------------
// $input / $output
// ---------------------------------------------------------------------------

func TestDollarInput_ReadValue(t *testing.T) {
	rt := New()
	rt.SetInput(map[string]any{"name": "test", "count": 42})
	err := rt.Execute(`$output = $input.name + ":" + $input.count`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().String() != "test:42" {
		t.Fatalf("expected 'test:42', got %v", rt.Output().Raw())
	}
}

func TestDollarOutput_WriteAndRead(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output.result = "done"; $output.code = 0;`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	out := rt.Output().Map()
	if out["result"] != "done" {
		t.Fatalf("expected output.result='done', got %v", out["result"])
	}
	codeVal, ok := out["code"]
	if !ok {
		t.Fatal("expected output.code to exist")
	}
	// goja exports numbers as int64 or float64
	switch v := codeVal.(type) {
	case int64:
		if v != 0 {
			t.Fatalf("expected code=0, got %v", v)
		}
	case float64:
		if v != 0 {
			t.Fatalf("expected code=0, got %v", v)
		}
	default:
		t.Fatalf("unexpected code type %T: %v", codeVal, codeVal)
	}
}

func TestDollarOutput_DefaultEmpty(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = typeof $output`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().String() != "object" {
		t.Fatalf("expected $output to be an object, got %v", rt.Output().Raw())
	}
}

// ---------------------------------------------------------------------------
// fakeBridge — records calls for behavioral testing
// ---------------------------------------------------------------------------

type fakeBridge struct {
	clipboardContent  string
	clipboardReadErr  error
	clipboardWritten  string
	clipboardWriteErr error
	keyPresses        []fakeKeyPress
	keyPressErr       error
	typedTexts        []string
	typeTextErr       error
}

type fakeKeyPress struct {
	key  string
	mods []ax.Modifier
}

func (f *fakeBridge) ClipboardRead() (string, error) {
	return f.clipboardContent, f.clipboardReadErr
}
func (f *fakeBridge) ClipboardWrite(text string) error {
	f.clipboardWritten = text
	return f.clipboardWriteErr
}
func (f *fakeBridge) KeyPress(key string, mods ...ax.Modifier) error {
	f.keyPresses = append(f.keyPresses, fakeKeyPress{key, mods})
	return f.keyPressErr
}
func (f *fakeBridge) TypeText(text string) error {
	f.typedTexts = append(f.typedTexts, text)
	return f.typeTextErr
}

// ---------------------------------------------------------------------------
// $clipboard — behavioral tests via fake bridge
// ---------------------------------------------------------------------------

func TestDollarClipboard_Read_ReturnsContent(t *testing.T) {
	fb := &fakeBridge{clipboardContent: "from clipboard"}
	rt := New(WithBridge(fb))
	err := rt.Execute(`$output = $clipboard.read()`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().String() != "from clipboard" {
		t.Fatalf("expected 'from clipboard', got %v", rt.Output().Raw())
	}
}

func TestDollarClipboard_Read_Error(t *testing.T) {
	fb := &fakeBridge{clipboardReadErr: fmt.Errorf("no pasteboard")}
	rt := New(WithBridge(fb))
	err := rt.Execute(`$clipboard.read()`)
	if err == nil {
		t.Fatal("expected error from $clipboard.read(), got nil")
	}
}

func TestDollarClipboard_Write_CallsBridge(t *testing.T) {
	fb := &fakeBridge{}
	rt := New(WithBridge(fb))
	err := rt.Execute(`$clipboard.write("hello JS")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if fb.clipboardWritten != "hello JS" {
		t.Fatalf("expected bridge to receive 'hello JS', got %q", fb.clipboardWritten)
	}
}

func TestDollarClipboard_Write_Error(t *testing.T) {
	fb := &fakeBridge{clipboardWriteErr: fmt.Errorf("write failed")}
	rt := New(WithBridge(fb))
	err := rt.Execute(`$clipboard.write("x")`)
	if err == nil {
		t.Fatal("expected error from $clipboard.write(), got nil")
	}
}

// ---------------------------------------------------------------------------
// $keyboard — behavioral tests via fake bridge
// ---------------------------------------------------------------------------

func TestDollarKeyboard_Press_CallsBridge(t *testing.T) {
	fb := &fakeBridge{}
	rt := New(WithBridge(fb))
	err := rt.Execute(`$keyboard.press("a")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(fb.keyPresses) != 1 {
		t.Fatalf("expected 1 keyPress, got %d", len(fb.keyPresses))
	}
	if fb.keyPresses[0].key != "a" {
		t.Fatalf("expected key 'a', got %q", fb.keyPresses[0].key)
	}
	if len(fb.keyPresses[0].mods) != 0 {
		t.Fatalf("expected no modifiers, got %v", fb.keyPresses[0].mods)
	}
}

func TestDollarKeyboard_Press_WithModifiers(t *testing.T) {
	fb := &fakeBridge{}
	rt := New(WithBridge(fb))
	err := rt.Execute(`$keyboard.press("c", "command")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(fb.keyPresses) != 1 {
		t.Fatalf("expected 1 keyPress, got %d", len(fb.keyPresses))
	}
	if fb.keyPresses[0].key != "c" {
		t.Fatalf("expected key 'c', got %q", fb.keyPresses[0].key)
	}
	if len(fb.keyPresses[0].mods) != 1 || fb.keyPresses[0].mods[0] != ax.ModCommand {
		t.Fatalf("expected [ModCommand], got %v", fb.keyPresses[0].mods)
	}
}

func TestDollarKeyboard_Press_Error(t *testing.T) {
	fb := &fakeBridge{keyPressErr: fmt.Errorf("key failed")}
	rt := New(WithBridge(fb))
	err := rt.Execute(`$keyboard.press("a")`)
	if err == nil {
		t.Fatal("expected error from $keyboard.press(), got nil")
	}
}

func TestDollarKeyboard_Type_CallsBridge(t *testing.T) {
	fb := &fakeBridge{}
	rt := New(WithBridge(fb))
	err := rt.Execute(`$keyboard.type("hello world")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(fb.typedTexts) != 1 || fb.typedTexts[0] != "hello world" {
		t.Fatalf("expected typed 'hello world', got %v", fb.typedTexts)
	}
}

func TestDollarKeyboard_Type_Error(t *testing.T) {
	fb := &fakeBridge{typeTextErr: fmt.Errorf("type failed")}
	rt := New(WithBridge(fb))
	err := rt.Execute(`$keyboard.type("x")`)
	if err == nil {
		t.Fatal("expected error from $keyboard.type(), got nil")
	}
}

// ---------------------------------------------------------------------------
// $ax — query function (stub until Task 16 bridge)
// ---------------------------------------------------------------------------

func TestDollarAx_IsFunction(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = typeof $ax`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().String() != "function" {
		t.Fatalf("expected $ax to be a function, got %v", rt.Output().Raw())
	}
}

func TestDollarAx_WithoutApp_ReturnsError(t *testing.T) {
	rt := New() // no app set
	err := rt.Execute(`$ax("AXButton")`)
	if err == nil {
		t.Fatal("expected error when $ax called without app, got nil")
	}
	if !strings.Contains(err.Error(), "app") {
		t.Fatalf("error should mention 'app', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// $app — switch target application (function)
// ---------------------------------------------------------------------------

func TestDollarApp_IsFunction(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = typeof $app`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().String() != "function" {
		t.Fatalf("expected $app to be a function, got %v", rt.Output().Raw())
	}
}

// ---------------------------------------------------------------------------
// Globals survive Reset (re-injected)
// ---------------------------------------------------------------------------

func TestGlobals_SurviveReset(t *testing.T) {
	var logs []string
	rt := New(WithOnLog(func(level, msg string) {
		logs = append(logs, msg)
	}))
	err := rt.Execute(`$log("before reset")`)
	if err != nil {
		t.Fatalf("before reset: %v", err)
	}

	rt.Reset()

	err = rt.Execute(`$log("after reset")`)
	if err != nil {
		t.Fatalf("after reset: %v", err)
	}

	if len(logs) != 2 || logs[1] != "after reset" {
		t.Fatalf("expected globals to survive Reset, logs=%v", logs)
	}
}

func TestGlobals_ConsoleAfterReset(t *testing.T) {
	var logs []string
	rt := New(WithOnLog(func(level, msg string) {
		logs = append(logs, level+":"+msg)
	}))
	rt.Reset()
	err := rt.Execute(`console.log("post-reset")`)
	if err != nil {
		t.Fatalf("console after reset: %v", err)
	}
	if len(logs) != 1 || logs[0] != "log:post-reset" {
		t.Fatalf("expected console after Reset, logs=%v", logs)
	}
}

// ---------------------------------------------------------------------------
// $env re-injected after SetEnv
// ---------------------------------------------------------------------------

func TestSetEnv_UpdatesGlobal(t *testing.T) {
	rt := New()
	rt.SetEnv(map[string]string{"A": "1"})
	err := rt.Execute(`$output = $env.A`)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Output().String() != "1" {
		t.Fatalf("expected '1', got %v", rt.Output().Raw())
	}

	rt.SetEnv(map[string]string{"A": "2"})
	err = rt.Execute(`$output = $env.A`)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Output().String() != "2" {
		t.Fatalf("expected '2' after SetEnv, got %v", rt.Output().Raw())
	}
}

// ---------------------------------------------------------------------------
// parseModifier coverage
// ---------------------------------------------------------------------------

func TestParseModifier_AllCases(t *testing.T) {
	tests := []struct {
		input string
		want  ax.Modifier
	}{
		{"command", ax.ModCommand},
		{"cmd", ax.ModCommand},
		{"Command", ax.ModCommand},
		{"shift", ax.ModShift},
		{"Shift", ax.ModShift},
		{"option", ax.ModOption},
		{"alt", ax.ModOption},
		{"Option", ax.ModOption},
		{"control", ax.ModControl},
		{"ctrl", ax.ModControl},
		{"Control", ax.ModControl},
		{"unknown", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got := parseModifier(tt.input)
		if got != tt.want {
			t.Errorf("parseModifier(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Output — nil/undefined $output path
// ---------------------------------------------------------------------------

func TestOutput_UndefinedOutput(t *testing.T) {
	rt := New()
	// Overwrite $output to undefined
	err := rt.Execute(`$output = undefined`)
	if err != nil {
		t.Fatal(err)
	}
	out := rt.Output()
	if !out.IsNil() {
		t.Fatalf("expected nil result for undefined $output, got %v", out.Raw())
	}
}

func TestOutput_NullOutput(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = null`)
	if err != nil {
		t.Fatal(err)
	}
	out := rt.Output()
	if !out.IsNil() {
		t.Fatalf("expected nil result for null $output, got %v", out.Raw())
	}
}

func TestOutput_ReturnsResult(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = 42`)
	if err != nil {
		t.Fatal(err)
	}
	out := rt.Output()
	if out.Int() != 42 {
		t.Fatalf("expected 42, got %v", out.Raw())
	}
}

func TestOutput_ReturnsStringResult(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = "hello"`)
	if err != nil {
		t.Fatal(err)
	}
	out := rt.Output()
	if out.String() != "hello" {
		t.Fatalf("expected 'hello', got %v", out.Raw())
	}
}

// ---------------------------------------------------------------------------
// $ax with app set — wrapSelection path (needs AX permission)
// ---------------------------------------------------------------------------

func TestDollarAx_WithApp_ReturnsObject(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New()
	rt.SetApp(app)
	err = rt.Execute(`$output = typeof $ax("AXButton")`)
	if err != nil {
		t.Fatalf("$ax with app error: %v", err)
	}
	if rt.Output().String() != "object" {
		t.Fatalf("expected object, got %v", rt.Output().Raw())
	}
}

func TestDollarAx_WrapSelection_Count(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New()
	rt.SetApp(app)
	err = rt.Execute(`$output = $ax("AXButton").count()`)
	if err != nil {
		t.Fatalf("$ax count error: %v", err)
	}
	count := rt.Output().Int()
	t.Logf("Finder buttons via $ax: %d", count)
}

func TestDollarAx_WrapSelection_IsEmpty(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New()
	rt.SetApp(app)
	// Query something unlikely to exist
	err = rt.Execute(`$output = $ax("AXButton[title='__nonexistent_12345__']").isEmpty()`)
	if err != nil {
		t.Fatalf("$ax isEmpty error: %v", err)
	}
	if rt.Output().Bool() != true {
		t.Fatal("expected isEmpty() to be true for nonexistent selector")
	}
}

func TestDollarAx_WrapSelection_Err(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New()
	rt.SetApp(app)
	err = rt.Execute(`$output = $ax("AXButton").err()`)
	if err != nil {
		t.Fatalf("$ax err() error: %v", err)
	}
	// Successful query should return null (no error).
	if !rt.Output().IsNil() {
		t.Logf("err() returned: %v (may be expected for some selectors)", rt.Output().Raw())
	}
}

// ---------------------------------------------------------------------------
// $app — switch app (needs AX permission)
// ---------------------------------------------------------------------------

func TestDollarApp_SwitchToFinder(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	rt := New()
	err := rt.Execute(`$app("com.apple.finder")`)
	if err != nil {
		t.Fatalf("$app('com.apple.finder') error: %v", err)
	}
	// After $app, $ax should work.
	err = rt.Execute(`$output = typeof $ax("AXWindow")`)
	if err != nil {
		t.Fatalf("$ax after $app error: %v", err)
	}
	if rt.Output().String() != "object" {
		t.Fatalf("expected object from $ax, got %v", rt.Output().Raw())
	}
}

func TestDollarApp_EmptyArg_Error(t *testing.T) {
	rt := New()
	err := rt.Execute(`$app("")`)
	if err == nil {
		t.Fatal("expected error from $app(''), got nil")
	}
}

func TestDollarApp_InvalidApp_Error(t *testing.T) {
	rt := New()
	err := rt.Execute(`$app("com.nonexistent.app.12345")`)
	if err == nil {
		t.Fatal("expected error from $app with invalid bundle ID, got nil")
	}
}

// ---------------------------------------------------------------------------
// $ax.defaults — Task 17
// ---------------------------------------------------------------------------

func TestDollarAx_Defaults_Exists(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = typeof $ax.defaults`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().String() != "object" {
		t.Fatalf("expected $ax.defaults to be an object, got %v", rt.Output().Raw())
	}
}

func TestDollarAx_Defaults_HasTimeout(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = $ax.defaults.timeout`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	timeout := rt.Output().Int()
	if timeout != 5000 {
		t.Fatalf("expected default timeout=5000, got %d", timeout)
	}
}

func TestDollarAx_Defaults_HasPollInterval(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = $ax.defaults.pollInterval`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	interval := rt.Output().Int()
	if interval != 200 {
		t.Fatalf("expected default pollInterval=200, got %d", interval)
	}
}

func TestDollarAx_Defaults_Writable(t *testing.T) {
	rt := New()
	err := rt.Execute(`$ax.defaults.timeout = 10000`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	err = rt.Execute(`$output = $ax.defaults.timeout`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().Int() != 10000 {
		t.Fatalf("expected timeout=10000 after write, got %d", rt.Output().Int())
	}
}

func TestDollarAx_Defaults_SurvivesReset(t *testing.T) {
	rt := New()
	// Defaults should be re-injected after Reset.
	rt.Reset()
	err := rt.Execute(`$output = $ax.defaults.timeout`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().Int() != 5000 {
		t.Fatalf("expected default timeout=5000 after Reset, got %d", rt.Output().Int())
	}
}

func TestDollarAx_StillCallable(t *testing.T) {
	// $ax should still be callable as a function after adding .defaults property.
	rt := New()
	err := rt.Execute(`$output = typeof $ax`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// $ax might be a function or an object with [[Call]] — either is fine,
	// but it must be callable.
	typ := rt.Output().String()
	if typ != "function" && typ != "object" {
		t.Fatalf("expected $ax to be function or object, got %v", typ)
	}
	// Should still error when called without app.
	err = rt.Execute(`$ax("AXButton")`)
	if err == nil {
		t.Fatal("expected error from $ax without app")
	}
}
