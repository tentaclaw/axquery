package js

import (
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/tentaclaw/ax"
	"github.com/tentaclaw/axquery"
)

// ---------------------------------------------------------------------------
// injectGlobals — called by New() and Reset() to populate JS globals
// ---------------------------------------------------------------------------

// injectGlobals registers all built-in global functions and objects into the VM.
func (r *Runtime) injectGlobals() {
	r.vm.Set("$log", r.jsLog)
	r.vm.Set("$delay", r.jsDelay)
	r.vm.Set("$app", r.jsApp)
	r.vm.Set("$ax", r.jsAx)
	r.vm.Set("$clipboard", r.jsClipboardObj())
	r.vm.Set("$keyboard", r.jsKeyboardObj())
	r.injectEnv()
	r.injectInput()
	r.injectOutput()
	r.injectConsole()
}

// ---------------------------------------------------------------------------
// $log(args...) — log output
// ---------------------------------------------------------------------------

func (r *Runtime) jsLog(call goja.FunctionCall) goja.Value {
	r.emitLog("log", call.Arguments)
	return goja.Undefined()
}

// emitLog formats arguments and invokes the onLog callback.
func (r *Runtime) emitLog(level string, args []goja.Value) {
	if r.conf.onLog == nil {
		return
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.String()
	}
	r.conf.onLog(level, strings.Join(parts, " "))
}

// ---------------------------------------------------------------------------
// $delay(ms) — synchronous sleep
// ---------------------------------------------------------------------------

func (r *Runtime) jsDelay(call goja.FunctionCall) goja.Value {
	ms := call.Argument(0).ToInteger()
	if ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	return goja.Undefined()
}

// ---------------------------------------------------------------------------
// $app(nameOrBundleID) — switch target application
// ---------------------------------------------------------------------------

func (r *Runtime) jsApp(call goja.FunctionCall) goja.Value {
	id := call.Argument(0).String()
	if id == "" || id == "undefined" {
		panic(r.vm.NewGoError(fmt.Errorf("$app: name or bundle ID required")))
	}

	// Try bundle ID first, then name.
	app, err := ax.ApplicationFromBundleID(id)
	if err != nil {
		app, err = ax.ApplicationFromName(id)
	}
	if err != nil {
		panic(r.vm.NewGoError(fmt.Errorf("$app: cannot find application %q: %w", id, err)))
	}

	// Close previous app if we opened it.
	if r.app != nil {
		r.app.Close()
	}
	r.app = app
	return goja.Undefined()
}

// ---------------------------------------------------------------------------
// $ax(selector) — query the current app (stub: returns object, full bridge in Task 16)
// ---------------------------------------------------------------------------

func (r *Runtime) jsAx(call goja.FunctionCall) goja.Value {
	if r.app == nil {
		panic(r.vm.NewGoError(fmt.Errorf("$ax: no app set (call $app or SetApp first)")))
	}
	selector := call.Argument(0).String()
	sel := axquery.Query(r.app, selector)
	return r.wrapSelection(sel)
}

// wrapSelection is defined in bridge.go (Task 16).

// ---------------------------------------------------------------------------
// $clipboard — {read(), write(text)}
// ---------------------------------------------------------------------------

func (r *Runtime) jsClipboardObj() *goja.Object {
	obj := r.vm.NewObject()
	obj.Set("read", func() (string, error) {
		return r.bridge.ClipboardRead()
	})
	obj.Set("write", func(text string) error {
		return r.bridge.ClipboardWrite(text)
	})
	return obj
}

// ---------------------------------------------------------------------------
// $keyboard — {press(key, ...modifiers), type(text)}
// ---------------------------------------------------------------------------

func (r *Runtime) jsKeyboardObj() *goja.Object {
	obj := r.vm.NewObject()
	obj.Set("press", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		var mods []ax.Modifier
		for i := 1; i < len(call.Arguments); i++ {
			mod := parseModifier(call.Arguments[i].String())
			if mod != 0 {
				mods = append(mods, mod)
			}
		}
		if err := r.bridge.KeyPress(key, mods...); err != nil {
			panic(r.vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	obj.Set("type", func(text string) error {
		return r.bridge.TypeText(text)
	})
	return obj
}

// parseModifier converts a string modifier name to ax.Modifier.
func parseModifier(s string) ax.Modifier {
	switch strings.ToLower(s) {
	case "command", "cmd":
		return ax.ModCommand
	case "shift":
		return ax.ModShift
	case "option", "alt":
		return ax.ModOption
	case "control", "ctrl":
		return ax.ModControl
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// $env — environment variables (read-only map)
// ---------------------------------------------------------------------------

func (r *Runtime) injectEnv() {
	if r.env == nil {
		r.vm.Set("$env", r.vm.NewObject())
		return
	}
	r.vm.Set("$env", r.env)
}

// ---------------------------------------------------------------------------
// $input — input parameters object
// ---------------------------------------------------------------------------

func (r *Runtime) injectInput() {
	if r.input == nil {
		r.vm.Set("$input", r.vm.NewObject())
		return
	}
	r.vm.Set("$input", r.input)
}

// ---------------------------------------------------------------------------
// $output — output object (writable by scripts)
// ---------------------------------------------------------------------------

func (r *Runtime) injectOutput() {
	r.outputObj = r.vm.NewObject()
	r.vm.Set("$output", r.outputObj)
}

// ---------------------------------------------------------------------------
// console.log / console.warn / console.error
// ---------------------------------------------------------------------------

func (r *Runtime) injectConsole() {
	console := r.vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		r.emitLog("log", call.Arguments)
		return goja.Undefined()
	})
	console.Set("warn", func(call goja.FunctionCall) goja.Value {
		r.emitLog("warn", call.Arguments)
		return goja.Undefined()
	})
	console.Set("error", func(call goja.FunctionCall) goja.Value {
		r.emitLog("error", call.Arguments)
		return goja.Undefined()
	})
	r.vm.Set("console", console)
}
