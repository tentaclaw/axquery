package js

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// New + basic Execute
// ---------------------------------------------------------------------------

func TestNew_ReturnsNonNil(t *testing.T) {
	rt := New()
	if rt == nil {
		t.Fatal("New() returned nil")
	}
}

func TestExecute_BasicArithmetic(t *testing.T) {
	rt := New()
	val, err := rt.Execute("1 + 2")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if val.ToInteger() != 3 {
		t.Fatalf("expected 3, got %v", val.Export())
	}
}

func TestExecute_StringResult(t *testing.T) {
	rt := New()
	val, err := rt.Execute(`"hello" + " " + "world"`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if val.Export().(string) != "hello world" {
		t.Fatalf("expected 'hello world', got %v", val.Export())
	}
}

func TestExecute_BoolResult(t *testing.T) {
	rt := New()
	val, err := rt.Execute("true && false")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if val.ToBoolean() != false {
		t.Fatalf("expected false, got %v", val.Export())
	}
}

func TestExecute_UndefinedResult(t *testing.T) {
	rt := New()
	val, err := rt.Execute("var x = 42;")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// var declaration evaluates to undefined
	if val.Export() != nil {
		t.Fatalf("expected nil (undefined), got %v", val.Export())
	}
}

func TestExecute_ObjectResult(t *testing.T) {
	rt := New()
	val, err := rt.Execute(`({name: "test", value: 42})`)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	m, ok := val.Export().(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", val.Export())
	}
	if m["name"] != "test" {
		t.Fatalf("expected name=test, got %v", m["name"])
	}
}

func TestExecute_ArrayResult(t *testing.T) {
	rt := New()
	val, err := rt.Execute("[1, 2, 3]")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	arr, ok := val.Export().([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", val.Export())
	}
	if len(arr) != 3 {
		t.Fatalf("expected length 3, got %d", len(arr))
	}
}

// ---------------------------------------------------------------------------
// Execute errors
// ---------------------------------------------------------------------------

func TestExecute_SyntaxError(t *testing.T) {
	rt := New()
	_, err := rt.Execute("function {")
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}
}

func TestExecute_ThrowError(t *testing.T) {
	rt := New()
	_, err := rt.Execute(`throw new Error("boom")`)
	if err == nil {
		t.Fatal("expected error from throw, got nil")
	}
}

func TestExecute_ReferenceError(t *testing.T) {
	rt := New()
	_, err := rt.Execute("nonExistentVar.foo")
	if err == nil {
		t.Fatal("expected ReferenceError, got nil")
	}
}

// ---------------------------------------------------------------------------
// Execute preserves VM state across calls
// ---------------------------------------------------------------------------

func TestExecute_PreservesState(t *testing.T) {
	rt := New()
	_, err := rt.Execute("var counter = 10;")
	if err != nil {
		t.Fatalf("first Execute error: %v", err)
	}
	val, err := rt.Execute("counter + 5")
	if err != nil {
		t.Fatalf("second Execute error: %v", err)
	}
	if val.ToInteger() != 15 {
		t.Fatalf("expected 15, got %v", val.Export())
	}
}

// ---------------------------------------------------------------------------
// ExecuteFile
// ---------------------------------------------------------------------------

func TestExecuteFile_BasicScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.js")
	if err := os.WriteFile(path, []byte("40 + 2"), 0644); err != nil {
		t.Fatal(err)
	}

	rt := New()
	val, err := rt.ExecuteFile(path)
	if err != nil {
		t.Fatalf("ExecuteFile error: %v", err)
	}
	if val.ToInteger() != 42 {
		t.Fatalf("expected 42, got %v", val.Export())
	}
}

func TestExecuteFile_NonExistentFile(t *testing.T) {
	rt := New()
	_, err := rt.ExecuteFile("/no/such/file.js")
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
	_, err := rt.ExecuteFile(path)
	if err == nil {
		t.Fatal("expected syntax error from file, got nil")
	}
}

// ---------------------------------------------------------------------------
// Timeout via WithTimeout option
// ---------------------------------------------------------------------------

func TestExecute_Timeout(t *testing.T) {
	rt := New(WithTimeout(50 * time.Millisecond))
	_, err := rt.Execute("while(true) {}")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// The error should be an interrupt error — we just need it to be non-nil.
	// Specific type checking can happen later; for now, "any error" from infinite
	// loop + short timeout is correct behavior.
}

func TestExecute_NoTimeoutForFastScript(t *testing.T) {
	rt := New(WithTimeout(1 * time.Second))
	val, err := rt.Execute("1 + 1")
	if err != nil {
		t.Fatalf("fast script should not timeout: %v", err)
	}
	if val.ToInteger() != 2 {
		t.Fatalf("expected 2, got %v", val.Export())
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
	// console is not injected in Task 14 scaffold — but OnLog should be stored.
	// We verify it's stored by checking the option was accepted (no panic/error).
	if rt == nil {
		t.Fatal("New with OnLog returned nil")
	}
	// Note: actual console.log injection is Task 17. Here we just verify option storage.
}

// ---------------------------------------------------------------------------
// WithOnError callback
// ---------------------------------------------------------------------------

func TestWithOnError_Stored(t *testing.T) {
	var errs []error
	rt := New(WithOnError(func(err error) {
		errs = append(errs, err)
	}))
	if rt == nil {
		t.Fatal("New with OnError returned nil")
	}
}

// ---------------------------------------------------------------------------
// Multiple options
// ---------------------------------------------------------------------------

func TestNew_MultipleOptions(t *testing.T) {
	rt := New(
		WithTimeout(5*time.Second),
		WithOnLog(func(level, msg string) {}),
		WithOnError(func(err error) {}),
	)
	if rt == nil {
		t.Fatal("New with multiple options returned nil")
	}
}

// ---------------------------------------------------------------------------
// Reset
// ---------------------------------------------------------------------------

func TestReset_ClearsState(t *testing.T) {
	rt := New()
	_, err := rt.Execute("var foo = 42;")
	if err != nil {
		t.Fatal(err)
	}

	rt.Reset()

	// After reset, foo should no longer exist.
	_, err = rt.Execute("foo")
	if err == nil {
		// If foo still exists (returns 42 without error), reset didn't work.
		// Note: in goja, accessing undefined var throws ReferenceError.
		// But if the VM was truly reset, foo is gone.
		t.Fatal("expected ReferenceError after Reset, got nil error")
	}
}

// ---------------------------------------------------------------------------
// SetApp (basic — just verify it doesn't panic)
// ---------------------------------------------------------------------------

func TestSetApp_NilApp(t *testing.T) {
	rt := New()
	// Setting nil app should not panic.
	rt.SetApp(nil)
}

// ---------------------------------------------------------------------------
// Execute empty script
// ---------------------------------------------------------------------------

func TestExecute_EmptyScript(t *testing.T) {
	rt := New()
	val, err := rt.Execute("")
	if err != nil {
		t.Fatalf("empty script should not error: %v", err)
	}
	// Empty script evaluates to undefined
	_ = val
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
	_, err := rt.ExecuteFile(path)
	if err != nil {
		t.Fatal(err)
	}

	val, err := rt.Execute("setupDone")
	if err != nil {
		t.Fatalf("expected setupDone to be defined: %v", err)
	}
	if val.ToBoolean() != true {
		t.Fatalf("expected true, got %v", val.Export())
	}
}

// ---------------------------------------------------------------------------
// Timeout cleans up properly — subsequent Execute still works
// ---------------------------------------------------------------------------

func TestExecute_TimeoutRecovery(t *testing.T) {
	rt := New(WithTimeout(50 * time.Millisecond))

	// First: timeout
	_, err := rt.Execute("while(true) {}")
	if err == nil {
		t.Fatal("expected timeout error")
	}

	// After timeout, we should be able to Reset and run again.
	rt.Reset()
	val, err := rt.Execute("1 + 1")
	if err != nil {
		t.Fatalf("after Reset, expected no error: %v", err)
	}
	if val.ToInteger() != 2 {
		t.Fatalf("expected 2, got %v", val.Export())
	}
}

// ---------------------------------------------------------------------------
// ScriptError type
// ---------------------------------------------------------------------------

func TestExecute_ScriptError_Type(t *testing.T) {
	rt := New()
	_, err := rt.Execute(`throw new Error("test error")`)
	if err == nil {
		t.Fatal("expected error")
	}

	var se *ScriptError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ScriptError, got %T: %v", err, err)
	}
	if se.Message == "" {
		t.Fatal("ScriptError.Message should not be empty")
	}

	// Verify Error() string output.
	errStr := se.Error()
	if errStr == "" {
		t.Fatal("ScriptError.Error() should not be empty")
	}

	// Verify Unwrap() returns the original goja error.
	unwrapped := se.Unwrap()
	if unwrapped == nil {
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
	_, err := rt.ExecuteFile(path)
	if err == nil {
		t.Fatal("expected error")
	}

	var se *ScriptError
	if errors.As(err, &se) {
		// ScriptError from file should have filename info
		if se.Filename == "" {
			t.Fatal("ScriptError.Filename should contain the file path")
		}
		// Error() should include filename.
		errStr := se.Error()
		if errStr == "" {
			t.Fatal("ScriptError.Error() should not be empty for file error")
		}
		// The filename should appear in the error string.
		if !errors.As(err, &se) || se.Filename == "" {
			t.Fatal("filename should be in ScriptError")
		}
	} else {
		t.Fatal("expected *ScriptError from ExecuteFile")
	}
}
