// Package js provides a goja-powered JavaScript runtime for axquery.
//
// It bridges the Go Selection API to JavaScript, exposing $ax(), $app(),
// and other global functions for automation scripting.
package js

import (
	"fmt"
	"os"
	"time"

	"github.com/dop251/goja"
	"github.com/tentaclaw/ax"
)

// ---------------------------------------------------------------------------
// ScriptError — structured error from JS execution
// ---------------------------------------------------------------------------

// ScriptError wraps a JavaScript execution error with optional source location.
type ScriptError struct {
	Message  string // Human-readable error message
	Filename string // Source filename (empty for inline Execute)
	Wrapped  error  // Original goja error
}

func (e *ScriptError) Error() string {
	if e.Filename != "" {
		return fmt.Sprintf("%s: %s", e.Filename, e.Message)
	}
	return e.Message
}

func (e *ScriptError) Unwrap() error { return e.Wrapped }

// ---------------------------------------------------------------------------
// RuntimeOption — functional options
// ---------------------------------------------------------------------------

// SystemBridge abstracts OS-level operations (clipboard, keyboard) so that
// tests can inject fakes without touching the real system.
type SystemBridge interface {
	ClipboardRead() (string, error)
	ClipboardWrite(text string) error
	KeyPress(key string, mods ...ax.Modifier) error
	TypeText(text string) error
}

// defaultBridge delegates to the ax package functions.
type defaultBridge struct{}

func (defaultBridge) ClipboardRead() (string, error)   { return ax.ClipboardRead() }
func (defaultBridge) ClipboardWrite(text string) error { return ax.ClipboardWrite(text) }
func (defaultBridge) KeyPress(key string, mods ...ax.Modifier) error {
	return ax.KeyPress(key, mods...)
}
func (defaultBridge) TypeText(text string) error { return ax.TypeText(text) }

// RuntimeOption configures a Runtime.
type RuntimeOption func(*runtimeConfig)

type runtimeConfig struct {
	timeout time.Duration
	onLog   func(level, msg string)
	onError func(err error)
	bridge  SystemBridge
}

// WithTimeout sets the maximum execution time for each Execute/ExecuteFile call.
// Zero means no timeout (default).
func WithTimeout(d time.Duration) RuntimeOption {
	return func(c *runtimeConfig) { c.timeout = d }
}

// WithOnLog sets a callback for log output (console.log, $log, etc.).
// The callback receives a level ("log", "warn", "error") and the message.
func WithOnLog(fn func(level, msg string)) RuntimeOption {
	return func(c *runtimeConfig) { c.onLog = fn }
}

// WithOnError sets a callback for unhandled runtime errors.
func WithOnError(fn func(err error)) RuntimeOption {
	return func(c *runtimeConfig) { c.onError = fn }
}

// WithBridge sets the system bridge for clipboard/keyboard operations.
// If not set, a default bridge that delegates to the ax package is used.
func WithBridge(b SystemBridge) RuntimeOption {
	return func(c *runtimeConfig) { c.bridge = b }
}

// ---------------------------------------------------------------------------
// Runtime
// ---------------------------------------------------------------------------

// Runtime manages a goja JavaScript VM with axquery bindings.
type Runtime struct {
	vm        *goja.Runtime
	app       *ax.Application
	conf      runtimeConfig
	bridge    SystemBridge
	env       map[string]string      // $env
	input     map[string]interface{} // $input
	outputObj *goja.Object           // $output (JS-side writable)
}

// New creates a new JavaScript runtime with the given options.
func New(opts ...RuntimeOption) *Runtime {
	var conf runtimeConfig
	for _, o := range opts {
		o(&conf)
	}
	bridge := conf.bridge
	if bridge == nil {
		bridge = defaultBridge{}
	}
	rt := &Runtime{
		vm:     goja.New(),
		conf:   conf,
		bridge: bridge,
	}
	rt.injectGlobals()
	return rt
}

// SetApp sets the target macOS application for $ax queries.
func (r *Runtime) SetApp(app *ax.Application) {
	r.app = app
}

// SetEnv sets environment variables accessible as $env in JS.
func (r *Runtime) SetEnv(env map[string]string) {
	r.env = env
	r.injectEnv()
}

// SetInput sets the input parameters accessible as $input in JS.
func (r *Runtime) SetInput(input map[string]interface{}) {
	r.input = input
	r.injectInput()
}

// Output returns the current $output object contents as a Go map.
func (r *Runtime) Output() map[string]interface{} {
	if r.outputObj == nil {
		return nil
	}
	exported := r.outputObj.Export()
	if m, ok := exported.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// Reset discards the current VM state and creates a fresh goja runtime.
// Options (timeout, callbacks) are preserved. Globals are re-injected.
func (r *Runtime) Reset() {
	r.vm = goja.New()
	r.injectGlobals()
}

// Execute runs a JavaScript string and returns the result.
func (r *Runtime) Execute(script string) (goja.Value, error) {
	return r.execute(script, "")
}

// ExecuteFile reads a .js file and executes its contents.
func (r *Runtime) ExecuteFile(path string) (goja.Value, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read script file: %w", err)
	}
	return r.execute(string(data), path)
}

// execute is the shared implementation for Execute and ExecuteFile.
func (r *Runtime) execute(script, filename string) (goja.Value, error) {
	// Set up timeout if configured.
	if r.conf.timeout > 0 {
		timer := time.AfterFunc(r.conf.timeout, func() {
			r.vm.Interrupt("execution timeout")
		})
		defer func() {
			timer.Stop()
			r.vm.ClearInterrupt()
		}()
	}

	val, err := r.vm.RunString(script)
	if err != nil {
		return nil, r.wrapError(err, filename)
	}
	return val, nil
}

// wrapError converts a goja error into a ScriptError.
func (r *Runtime) wrapError(err error, filename string) *ScriptError {
	msg := err.Error()

	// Try to extract a cleaner message from goja exception types.
	if exc, ok := err.(*goja.Exception); ok {
		msg = exc.Value().String()
	} else if interrupted, ok := err.(*goja.InterruptedError); ok {
		msg = fmt.Sprintf("interrupted: %v", interrupted.Value())
	}

	return &ScriptError{
		Message:  msg,
		Filename: filename,
		Wrapped:  err,
	}
}
