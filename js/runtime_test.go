package js

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// New + basic Execute
// ---------------------------------------------------------------------------

func TestNew_CanExecuteJS(t *testing.T) {
	rt := New()
	err := rt.Execute("$output = 2 + 2")
	if err != nil {
		t.Fatalf("New() runtime cannot execute JS: %v", err)
	}
	if rt.Output().Int() != 4 {
		t.Fatalf("expected 4, got %v", rt.Output().Raw())
	}
}

func TestExecute_BasicArithmetic(t *testing.T) {
	rt := New()
	err := rt.Execute("$output = 1 + 2")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().Int() != 3 {
		t.Fatalf("expected 3, got %v", rt.Output().Raw())
	}
}

func TestExecute_StringResult(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = "hello" + " " + "world"`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().String() != "hello world" {
		t.Fatalf("expected 'hello world', got %v", rt.Output().Raw())
	}
}

func TestExecute_BoolResult(t *testing.T) {
	rt := New()
	err := rt.Execute("$output = true && false")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if rt.Output().Bool() != false {
		t.Fatalf("expected false, got %v", rt.Output().Raw())
	}
}

func TestExecute_UndefinedResult(t *testing.T) {
	rt := New()
	err := rt.Execute("var x = 42;")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// $output was not assigned, so it stays as the initial empty object.
}

func TestExecute_ObjectResult(t *testing.T) {
	rt := New()
	err := rt.Execute(`$output = {name: "test", value: 42}`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	m := rt.Output().Map()
	if m == nil {
		t.Fatalf("expected map, got nil")
	}
	if m["name"] != "test" {
		t.Fatalf("expected name=test, got %v", m["name"])
	}
}

func TestExecute_ArrayResult(t *testing.T) {
	rt := New()
	err := rt.Execute("$output = [1, 2, 3]")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	arr := rt.Output().Slice()
	if arr == nil {
		t.Fatalf("expected slice, got nil")
	}
	if len(arr) != 3 {
		t.Fatalf("expected length 3, got %d", len(arr))
	}
	// Verify actual contents.
	for i, want := range []int64{1, 2, 3} {
		got, ok := arr[i].(int64)
		if !ok {
			t.Fatalf("arr[%d]: expected int64, got %T", i, arr[i])
		}
		if got != want {
			t.Fatalf("arr[%d] = %d, want %d", i, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Execute errors
// ---------------------------------------------------------------------------

func TestExecute_SyntaxError(t *testing.T) {
	rt := New()
	err := rt.Execute("function {")
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}
}

func TestExecute_ThrowError(t *testing.T) {
	rt := New()
	err := rt.Execute(`throw new Error("boom")`)
	if err == nil {
		t.Fatal("expected error from throw, got nil")
	}
}

func TestExecute_ReferenceError(t *testing.T) {
	rt := New()
	err := rt.Execute("nonExistentVar.foo")
	if err == nil {
		t.Fatal("expected ReferenceError, got nil")
	}
}

// ---------------------------------------------------------------------------
// Execute preserves VM state across calls
// ---------------------------------------------------------------------------

func TestExecute_PreservesState(t *testing.T) {
	rt := New()
	err := rt.Execute("var counter = 10;")
	if err != nil {
		t.Fatalf("first Execute error: %v", err)
	}
	err = rt.Execute("$output = counter + 5")
	if err != nil {
		t.Fatalf("second Execute error: %v", err)
	}
	if rt.Output().Int() != 15 {
		t.Fatalf("expected 15, got %v", rt.Output().Raw())
	}
}

// ---------------------------------------------------------------------------
// ExecuteFile
// ---------------------------------------------------------------------------

func TestExecuteFile_BasicScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.js")
	if err := os.WriteFile(path, []byte("$output = 40 + 2"), 0644); err != nil {
		t.Fatal(err)
	}

	rt := New()
	err := rt.ExecuteFile(path)
	if err != nil {
		t.Fatalf("ExecuteFile error: %v", err)
	}
	if rt.Output().Int() != 42 {
		t.Fatalf("expected 42, got %v", rt.Output().Raw())
	}
}

func TestExecuteFile_NonExistentFile(t *testing.T) {
	rt := New()
	err := rt.ExecuteFile("/no/such/file.js")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestExecuteFile_SyntaxError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.js")
	if err := os.WriteFile(path, []byte("function {"), 0644); err != nil {
		t.Fatal(err)
	}

	rt := New()
	err := rt.ExecuteFile(path)
	if err == nil {
		t.Fatal("expected syntax error from file, got nil")
	}
}

// ---------------------------------------------------------------------------
// Timeout via WithTimeout option
// ---------------------------------------------------------------------------

func TestExecute_Timeout(t *testing.T) {
	rt := New(WithTimeout(50 * time.Millisecond))
	err := rt.Execute("while(true) {}")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// The error should be an interrupt error — we just need it to be non-nil.
}

func TestExecute_NoTimeoutForFastScript(t *testing.T) {
	rt := New(WithTimeout(1 * time.Second))
	err := rt.Execute("$output = 1 + 1")
	if err != nil {
		t.Fatalf("fast script should not timeout: %v", err)
	}
	if rt.Output().Int() != 2 {
		t.Fatalf("expected 2, got %v", rt.Output().Raw())
	}
}

// ---------------------------------------------------------------------------
// WithOnLog callback
// ---------------------------------------------------------------------------

func TestWithOnLog_ReceivesLogs(t *testing.T) {
	var logs []string
	rt := New(WithOnLog(func(level, msg string) {
		logs = append(logs, level+":"+msg)
	}))
	// Actually trigger the callback via $log (injected global).
	err := rt.Execute(`$log("hello"); $log("world")`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 log entries, got %d: %v", len(logs), logs)
	}
	if logs[0] != "log:hello" {
		t.Fatalf("expected 'log:hello', got %q", logs[0])
	}
	if logs[1] != "log:world" {
		t.Fatalf("expected 'log:world', got %q", logs[1])
	}
}

// ---------------------------------------------------------------------------
// WithOnError callback
// ---------------------------------------------------------------------------

func TestWithOnError_CallbackStored(t *testing.T) {
	var errs []error
	rt := New(WithOnError(func(err error) {
		errs = append(errs, err)
	}))
	// Verify the callback is stored by checking the config field directly.
	if rt.conf.onError == nil {
		t.Fatal("WithOnError: callback not stored in config")
	}
	// Invoke it to verify it works.
	rt.conf.onError(fmt.Errorf("test error"))
	if len(errs) != 1 || errs[0].Error() != "test error" {
		t.Fatalf("expected callback to receive 'test error', got %v", errs)
	}
}

// ---------------------------------------------------------------------------
// Multiple options
// ---------------------------------------------------------------------------

func TestNew_MultipleOptions_AllApplied(t *testing.T) {
	var logs []string
	var errs []error
	rt := New(
		WithTimeout(5*time.Second),
		WithOnLog(func(level, msg string) {
			logs = append(logs, level+":"+msg)
		}),
		WithOnError(func(err error) {
			errs = append(errs, err)
		}),
	)
	// Verify timeout is set.
	if rt.conf.timeout != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", rt.conf.timeout)
	}
	// Verify onLog works.
	err := rt.Execute(`$log("multi-opt")`)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0] != "log:multi-opt" {
		t.Fatalf("expected onLog to capture 'log:multi-opt', got %v", logs)
	}
	// Verify onError is stored.
	if rt.conf.onError == nil {
		t.Fatal("expected onError to be set")
	}
}

// ---------------------------------------------------------------------------
// Reset
// ---------------------------------------------------------------------------

func TestReset_ClearsState(t *testing.T) {
	rt := New()
	err := rt.Execute("var foo = 42;")
	if err != nil {
		t.Fatal(err)
	}

	rt.Reset()

	// After reset, foo should no longer exist.
	err = rt.Execute("foo")
	if err == nil {
		// If foo still exists (returns 42 without error), reset didn't work.
		t.Fatal("expected ReferenceError after Reset, got nil error")
	}
}

// ---------------------------------------------------------------------------
// SetApp (basic — just verify it doesn't panic)
// ---------------------------------------------------------------------------

func TestSetApp_NilApp_AxStillFails(t *testing.T) {
	rt := New()
	rt.SetApp(nil)
	// With nil app, $ax should error since it requires an app.
	err := rt.Execute(`$ax("AXButton")`)
	if err == nil {
		t.Fatal("expected error from $ax with nil app, got nil")
	}
}

// ---------------------------------------------------------------------------
// Execute empty script
// ---------------------------------------------------------------------------

func TestExecute_EmptyScript(t *testing.T) {
	rt := New()
	err := rt.Execute("")
	if err != nil {
		t.Fatalf("empty script should not error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ExecuteFile preserves state
// ---------------------------------------------------------------------------

func TestExecuteFile_PreservesState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "setup.js")
	if err := os.WriteFile(path, []byte("var setupDone = true;"), 0644); err != nil {
		t.Fatal(err)
	}

	rt := New()
	err := rt.ExecuteFile(path)
	if err != nil {
		t.Fatal(err)
	}

	err = rt.Execute("$output = setupDone")
	if err != nil {
		t.Fatalf("expected setupDone to be defined: %v", err)
	}
	if rt.Output().Bool() != true {
		t.Fatalf("expected true, got %v", rt.Output().Raw())
	}
}

// ---------------------------------------------------------------------------
// Timeout cleans up properly — subsequent Execute still works
// ---------------------------------------------------------------------------

func TestExecute_TimeoutRecovery(t *testing.T) {
	rt := New(WithTimeout(50 * time.Millisecond))

	// First: timeout
	err := rt.Execute("while(true) {}")
	if err == nil {
		t.Fatal("expected timeout error")
	}

	// After timeout, we should be able to Reset and run again.
	rt.Reset()
	err = rt.Execute("$output = 1 + 1")
	if err != nil {
		t.Fatalf("after Reset, expected no error: %v", err)
	}
	if rt.Output().Int() != 2 {
		t.Fatalf("expected 2, got %v", rt.Output().Raw())
	}
}

// ---------------------------------------------------------------------------
// ScriptError type
// ---------------------------------------------------------------------------

func TestExecute_ScriptError_Type(t *testing.T) {
	rt := New()
	err := rt.Execute(`throw new Error("test error")`)
	if err == nil {
		t.Fatal("expected error")
	}

	var se *ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ScriptError, got %T: %v", err, err)
	}
	// Verify the message contains the thrown text.
	if !strings.Contains(se.Message, "test error") {
		t.Fatalf("ScriptError.Message should contain 'test error', got %q", se.Message)
	}
	// Filename should be empty for inline Execute.
	if se.Filename != "" {
		t.Fatalf("expected empty Filename for inline Execute, got %q", se.Filename)
	}
	// Error() should equal Message for inline scripts (no filename prefix).
	if se.Error() != se.Message {
		t.Fatalf("expected Error()=%q to equal Message=%q", se.Error(), se.Message)
	}
	// Unwrap() should return the original goja error.
	if se.Unwrap() == nil {
		t.Fatal("ScriptError.Unwrap() should not be nil")
	}
}

func TestExecuteFile_ScriptError_HasFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "error.js")
	if err := os.WriteFile(path, []byte(`throw new Error("file error")`), 0644); err != nil {
		t.Fatal(err)
	}

	rt := New()
	err := rt.ExecuteFile(path)
	if err == nil {
		t.Fatal("expected error")
	}

	var se *ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ScriptError, got %T: %v", err, err)
	}
	// Verify filename matches the actual path.
	if se.Filename != path {
		t.Fatalf("expected Filename=%q, got %q", path, se.Filename)
	}
	// Verify message contains the thrown text.
	if !strings.Contains(se.Message, "file error") {
		t.Fatalf("ScriptError.Message should contain 'file error', got %q", se.Message)
	}
	// Error() should include the filename prefix.
	expectedPrefix := path + ": "
	if !strings.HasPrefix(se.Error(), expectedPrefix) {
		t.Fatalf("expected Error() to start with %q, got %q", expectedPrefix, se.Error())
	}
}
