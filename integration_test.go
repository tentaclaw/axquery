package axquery_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tentaclaw/ax"
	"github.com/tentaclaw/axquery"
	"github.com/tentaclaw/axquery/js"
)

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

// newExec creates an Executor typed as the interface, verifying that
// js.New returns something assignable to axquery.Executor.
func newExec(opts ...js.RuntimeOption) axquery.Executor {
	return js.New(opts...)
}

// requireMail returns an Executor wired to Mail.app.
// Skips the test if AX permission is missing or Mail is not running.
func requireMail(t *testing.T) axquery.Executor {
	t.Helper()
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.mail")
	if err != nil {
		t.Skip("Mail.app not available:", err)
	}
	t.Cleanup(func() { app.Close() })

	exec := js.New(js.WithTimeout(30 * time.Second))
	exec.SetApp(app)
	return exec
}

// ═══════════════════════════════════════════════════════════════════════════
// Part 1 — Pure-logic tests (no AX permission required)
// ═══════════════════════════════════════════════════════════════════════════

func TestE2E_InputOutputRoundTrip(t *testing.T) {
	exec := newExec()
	exec.SetInput(map[string]any{"name": "axquery", "version": 1})

	if err := exec.Execute(`$output = $input.name + " v" + $input.version`); err != nil {
		t.Fatal(err)
	}
	if got := exec.Output().String(); got != "axquery v1" {
		t.Fatalf("expected 'axquery v1', got %q", got)
	}
}

func TestE2E_OutputScalar(t *testing.T) {
	exec := newExec()

	// integer
	if err := exec.Execute(`$output = 42`); err != nil {
		t.Fatal(err)
	}
	if got := exec.Output().Int(); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	exec.Reset()

	// boolean
	if err := exec.Execute(`$output = true`); err != nil {
		t.Fatal(err)
	}
	if got := exec.Output().Bool(); got != true {
		t.Fatalf("expected true, got %v", got)
	}

	exec.Reset()

	// string
	if err := exec.Execute(`$output = "hello"`); err != nil {
		t.Fatal(err)
	}
	if got := exec.Output().String(); got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
}

func TestE2E_OutputObject(t *testing.T) {
	exec := newExec()
	if err := exec.Execute(`$output.name = "test"; $output.count = 3`); err != nil {
		t.Fatal(err)
	}
	m := exec.Output().Map()
	if m == nil {
		t.Fatal("expected map output, got nil")
	}
	if m["name"] != "test" {
		t.Fatalf("expected name='test', got %v", m["name"])
	}
	if v, ok := m["count"].(int64); !ok || v != 3 {
		t.Fatalf("expected count=3, got %v (%T)", m["count"], m["count"])
	}
}

func TestE2E_OutputArray(t *testing.T) {
	exec := newExec()
	if err := exec.Execute(`$output = [1, "two", true]`); err != nil {
		t.Fatal(err)
	}
	s := exec.Output().Slice()
	if len(s) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(s))
	}
}

func TestE2E_EnvVariables(t *testing.T) {
	exec := newExec()
	exec.SetEnv(map[string]string{"MODE": "test", "LANG": "go"})

	if err := exec.Execute(`$output = $env.MODE + "-" + $env.LANG`); err != nil {
		t.Fatal(err)
	}
	if got := exec.Output().String(); got != "test-go" {
		t.Fatalf("expected 'test-go', got %q", got)
	}
}

func TestE2E_SyntaxError(t *testing.T) {
	exec := newExec()
	err := exec.Execute(`var x = {{{`)
	if err == nil {
		t.Fatal("expected error for syntax error")
	}
	var se *js.ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("expected ScriptError, got %T: %v", err, err)
	}
}

func TestE2E_RuntimeThrow(t *testing.T) {
	exec := newExec()
	err := exec.Execute(`throw new Error("boom")`)
	if err == nil {
		t.Fatal("expected error for throw")
	}
	var se *js.ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("expected ScriptError, got %T: %v", err, err)
	}
	if !strings.Contains(se.Message, "boom") {
		t.Fatalf("expected 'boom' in message, got %q", se.Message)
	}
}

func TestE2E_Timeout(t *testing.T) {
	exec := newExec(js.WithTimeout(200 * time.Millisecond))
	err := exec.Execute(`while(true) {}`)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected 'timeout' in error, got: %v", err)
	}
}

func TestE2E_ExecuteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_script.js")
	if err := os.WriteFile(path, []byte(`$output = { sum: $input.a + $input.b }`), 0644); err != nil {
		t.Fatal(err)
	}

	exec := newExec()
	exec.SetInput(map[string]any{"a": 10, "b": 32})
	if err := exec.ExecuteFile(path); err != nil {
		t.Fatal(err)
	}

	m := exec.Output().Map()
	if m == nil {
		t.Fatal("expected map output")
	}
	if v, ok := m["sum"].(int64); !ok || v != 42 {
		t.Fatalf("expected sum=42, got %v (%T)", m["sum"], m["sum"])
	}
}

func TestE2E_ExecuteFile_NotFound(t *testing.T) {
	exec := newExec()
	err := exec.ExecuteFile("/nonexistent/path/script.js")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestE2E_ResetIsolation(t *testing.T) {
	exec := newExec()

	if err := exec.Execute(`$output = 42`); err != nil {
		t.Fatal(err)
	}
	if exec.Output().Int() != 42 {
		t.Fatal("expected 42 before reset")
	}

	exec.Reset()

	// After reset, $output is a fresh object, not the old scalar.
	if err := exec.Execute(`$output = typeof $output`); err != nil {
		t.Fatal(err)
	}
	if got := exec.Output().String(); got != "object" {
		t.Fatalf("after reset, expected typeof $output = 'object', got %q", got)
	}
}

func TestE2E_MultipleScriptsShareState(t *testing.T) {
	exec := newExec()

	if err := exec.Execute(`var counter = 10`); err != nil {
		t.Fatal(err)
	}
	if err := exec.Execute(`$output = counter + 5`); err != nil {
		t.Fatal(err)
	}
	if got := exec.Output().Int(); got != 15 {
		t.Fatalf("expected 15, got %d", got)
	}
}

func TestE2E_ConsoleLog(t *testing.T) {
	var logs []string
	exec := newExec(js.WithOnLog(func(level, msg string) {
		logs = append(logs, level+":"+msg)
	}))

	if err := exec.Execute(`
		console.log("info msg");
		console.warn("warn msg");
		console.error("err msg");
	`); err != nil {
		t.Fatal(err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 log entries, got %d: %v", len(logs), logs)
	}
	if logs[0] != "log:info msg" {
		t.Fatalf("logs[0] = %q, want 'log:info msg'", logs[0])
	}
	if logs[1] != "warn:warn msg" {
		t.Fatalf("logs[1] = %q, want 'warn:warn msg'", logs[1])
	}
	if logs[2] != "error:err msg" {
		t.Fatalf("logs[2] = %q, want 'error:err msg'", logs[2])
	}
}

func TestE2E_OutputResultType(t *testing.T) {
	exec := newExec()

	cases := []struct {
		script string
		want   axquery.ResultType
	}{
		{`$output = null`, axquery.ResultNil},
		{`$output = "text"`, axquery.ResultString},
		{`$output = 99`, axquery.ResultInt},
		{`$output = 3.14`, axquery.ResultFloat},
		{`$output = true`, axquery.ResultBool},
		{`$output = [1,2]`, axquery.ResultSlice},
		{`$output = {a: 1}`, axquery.ResultMap},
	}
	for _, tc := range cases {
		exec.Reset()
		if err := exec.Execute(tc.script); err != nil {
			t.Fatalf("script %q: %v", tc.script, err)
		}
		if got := exec.Output().Type(); got != tc.want {
			t.Errorf("script %q: type = %v, want %v", tc.script, got, tc.want)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Part 2 — Mail.app E2E (need AX permission + Mail.app running)
// ═══════════════════════════════════════════════════════════════════════════

// TestE2E_MailQueryToolbar verifies basic $ax queries against Mail's
// toolbar — a shallow query that doesn't require deep tree traversal.
func TestE2E_MailQueryToolbar(t *testing.T) {
	exec := requireMail(t)

	// AXToolbar is a direct child of the focused window.
	// Use maxDepth:2 to avoid deep traversal into WebKit/split views.
	script := `
		var toolbar = $ax("AXToolbar", {maxDepth: 2});
		$output = {
			found:    !toolbar.isEmpty(),
			count:    toolbar.count(),
			role:     toolbar.isEmpty() ? "" : toolbar.first().role()
		};
	`
	if err := exec.Execute(script); err != nil {
		t.Fatal(err)
	}
	m := exec.Output().Map()
	if m == nil {
		t.Fatal("expected map output")
	}
	if m["found"] != true {
		t.Fatal("expected to find AXToolbar in Mail")
	}
	t.Logf("Mail toolbar — count: %v, role: %v", m["count"], m["role"])
}

// TestE2E_MailQueryButtons queries buttons in the Mail window.
// AXButton elements are direct children of the window (close/minimize/fullscreen).
func TestE2E_MailQueryButtons(t *testing.T) {
	exec := requireMail(t)

	// Use maxDepth:2 to find buttons that are direct children or in toolbar.
	script := `
		var buttons = $ax("AXButton", {maxDepth: 2});
		var titles = [];
		buttons.each(function(i, btn) {
			if (i >= 10) return false;
			var desc = btn.description();
			titles.push(desc || "(no desc)");
		});
		$output = {
			buttonCount: buttons.count(),
			descriptions: titles
		};
	`
	if err := exec.Execute(script); err != nil {
		t.Fatal(err)
	}
	m := exec.Output().Map()
	if m == nil {
		t.Fatal("expected map output")
	}
	t.Logf("Mail buttons: count=%v", m["buttonCount"])
	if descs, ok := m["descriptions"].([]interface{}); ok {
		t.Logf("Descriptions (%d): %v", len(descs), descs)
	}
}

// TestE2E_MailReadStaticTexts reads AXStaticText elements from the window.
func TestE2E_MailReadStaticTexts(t *testing.T) {
	exec := requireMail(t)

	// Use maxDepth:2 to find top-level static texts.
	script := `
		var texts = $ax("AXStaticText", {maxDepth: 2});
		if (texts.isEmpty()) {
			$output = { empty: true };
		} else {
			var titles = texts.map(function(i, el) {
				return el.title();
			});
			$output = {
				empty:  false,
				count:  texts.count(),
				titles: titles
			};
		}
	`
	if err := exec.Execute(script); err != nil {
		t.Fatal(err)
	}
	m := exec.Output().Map()
	if m == nil {
		t.Fatal("expected map output")
	}
	if m["empty"] == true {
		t.Log("No AXStaticText found in Mail window")
		return
	}
	t.Logf("Static texts — count: %v, titles: %v", m["count"], m["titles"])
}

// TestE2E_MailComposeAndSend opens a compose window, fills in fields, and sends.
//
// Requires environment variables:
//
//	TENTACLAW_TEST_EMAIL — recipient address (sends to self)
//
// The test is skipped if the env var is not set.
func TestE2E_MailComposeAndSend(t *testing.T) {
	email := os.Getenv("TENTACLAW_TEST_EMAIL")
	if email == "" {
		t.Skip("TENTACLAW_TEST_EMAIL not set")
	}

	exec := requireMail(t)
	exec.SetInput(map[string]any{
		"to":      email,
		"subject": "axquery E2E " + time.Now().Format("15:04:05"),
		"body":    "Automated test from axquery integration suite.",
	})

	script := `
		// 1. Open new compose window
		$keyboard.press("n", "command");
		$delay(2000);

		// 2. Cursor should be in the To field — type address
		$keyboard.type($input.to);
		$delay(500);

		// 3. Tab past To field (confirm address) → Subject
		$keyboard.press("tab");
		$delay(500);
		$keyboard.press("tab");
		$delay(500);

		// 4. Type subject
		$keyboard.type($input.subject);
		$delay(300);

		// 5. Tab to body
		$keyboard.press("tab");
		$delay(300);

		// 6. Type body
		$keyboard.type($input.body);
		$delay(300);

		// 7. Send (Cmd+Shift+D)
		$keyboard.press("d", "command", "shift");
		$delay(1000);

		$output = "sent";
	`

	if err := exec.Execute(script); err != nil {
		t.Fatal(err)
	}
	if got := exec.Output().String(); got != "sent" {
		t.Fatalf("expected 'sent', got %q", got)
	}
	t.Log("Compose & send completed")
}

// TestE2E_MailInvalidSelector tests structured error handling with real AX.
func TestE2E_MailInvalidSelector(t *testing.T) {
	exec := requireMail(t)

	// Invalid selector should produce a ScriptError when we try to use
	// a terminal method on the result.
	script := `
		try {
			var sel = $ax("[invalid");
			sel.title(); // terminal method on error selection → throw
			$output = "should not reach here";
		} catch (e) {
			$output = {
				code:    e.code    || "unknown",
				message: e.message || String(e)
			};
		}
	`
	if err := exec.Execute(script); err != nil {
		t.Fatal(err)
	}
	m := exec.Output().Map()
	if m == nil {
		t.Fatal("expected map output")
	}
	t.Logf("Caught error: code=%v message=%v", m["code"], m["message"])
	if m["code"] != "INVALID_SELECTOR" {
		t.Fatalf("expected INVALID_SELECTOR, got %v", m["code"])
	}
}
