package axquery

import "github.com/tentaclaw/ax"

// ---------------------------------------------------------------------------
// Executor — engine-agnostic script execution interface
// ---------------------------------------------------------------------------

// Executor runs JavaScript automation scripts against macOS applications.
// Implementations hide the underlying JS engine (e.g. goja, v8go), so that
// callers depend only on this interface and never on engine-specific types.
//
// Usage:
//
//	exec := js.New(js.WithTimeout(10 * time.Second))
//	exec.SetApp(app)
//	exec.SetInput(map[string]any{"query": "AXButton"})
//	if err := exec.Execute(script); err != nil { ... }
//	result := exec.Output()
//	count := result.Int()
type Executor interface {
	// Execute runs a JS script string. The script can set $output to
	// communicate results back to Go (readable via Output()).
	Execute(script string) error

	// ExecuteFile reads a .js file and executes its contents.
	ExecuteFile(path string) error

	// SetApp sets the target macOS application for $ax queries.
	SetApp(app *ax.Application)

	// SetInput sets input parameters accessible as $input in JS.
	SetInput(input map[string]any)

	// SetEnv sets environment variables accessible as $env in JS.
	SetEnv(env map[string]string)

	// Output returns the value of the $output variable after script execution.
	// Returns a nil Result if the script did not set $output.
	Output() *Result

	// Reset discards the current VM state and creates a fresh runtime.
	// Configuration (timeout, callbacks) is preserved.
	Reset()
}

// ---------------------------------------------------------------------------
// SystemBridge — OS-level operation abstraction
// ---------------------------------------------------------------------------

// SystemBridge abstracts OS-level operations (clipboard, keyboard) so that
// tests can inject fakes without touching the real system.
type SystemBridge interface {
	ClipboardRead() (string, error)
	ClipboardWrite(text string) error
	KeyPress(key string, mods ...ax.Modifier) error
	TypeText(text string) error
}
