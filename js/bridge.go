package js

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/dop251/goja"
	"github.com/tentaclaw/axquery"
)

// boolKind is cached to avoid repeated reflect.Bool allocation in hot paths.
var boolKind = reflect.Bool

// ---------------------------------------------------------------------------
// Structured error conversion: Go error → JS object
// ---------------------------------------------------------------------------

// errorToJSObject converts a Go error into a structured JS object with code, message,
// and type-specific fields (selector, count, action, timeout).
func (r *Runtime) errorToJSObject(err error) *goja.Object {
	obj := r.vm.NewObject()

	// Default code and message.
	code := "ERROR"
	obj.Set("message", err.Error())

	var nf *axquery.NotFoundError
	var te *axquery.TimeoutError
	var ae *axquery.AmbiguousError
	var ise *axquery.InvalidSelectorError
	var nae *axquery.NotActionableError

	switch {
	case errors.As(err, &nf):
		code = "NOT_FOUND"
		obj.Set("selector", nf.Selector)
	case errors.As(err, &te):
		code = "TIMEOUT"
		obj.Set("selector", te.Selector)
		obj.Set("timeout", te.Duration)
	case errors.As(err, &ae):
		code = "AMBIGUOUS"
		obj.Set("selector", ae.Selector)
		obj.Set("count", ae.Count)
	case errors.As(err, &ise):
		code = "INVALID_SELECTOR"
		obj.Set("selector", ise.Selector)
	case errors.As(err, &nae):
		code = "NOT_ACTIONABLE"
		obj.Set("action", nae.Action)
	}

	obj.Set("code", code)
	return obj
}

// throwIfErr panics with a structured JS exception if sel carries an error.
// This is used by terminal methods (property reads, actions, waits, scrolls)
// to enforce "query silently, fail loudly on use" semantics.
func (r *Runtime) throwIfErr(sel *axquery.Selection) {
	if err := sel.Err(); err != nil {
		panic(r.vm.ToValue(r.errorToJSObject(err)))
	}
}

// wrapSelection wraps a Go *axquery.Selection as a fully-featured JS object.
// Every Selection method is exposed as a camelCase JS method.
// Methods returning *Selection recursively wrap the result for chaining.
func (r *Runtime) wrapSelection(sel *axquery.Selection) goja.Value {
	obj := r.vm.NewObject()

	// -----------------------------------------------------------------------
	// Basic methods (selection.go) — non-throwing
	// -----------------------------------------------------------------------

	obj.Set("count", func() int { return sel.Count() })
	obj.Set("isEmpty", func() bool { return sel.IsEmpty() })
	obj.Set("err", func() interface{} {
		if err := sel.Err(); err != nil {
			return r.errorToJSObject(err)
		}
		return nil
	})
	obj.Set("selector", func() string { return sel.Selector() })
	obj.Set("first", func() goja.Value { return r.wrapSelection(sel.First()) })
	obj.Set("last", func() goja.Value { return r.wrapSelection(sel.Last()) })
	obj.Set("eq", func(i int) goja.Value { return r.wrapSelection(sel.Eq(i)) })
	obj.Set("slice", func(start, end int) goja.Value { return r.wrapSelection(sel.Slice(start, end)) })

	// -----------------------------------------------------------------------
	// Traversal methods (traversal.go) — non-throwing
	// -----------------------------------------------------------------------

	obj.Set("find", func(s string) goja.Value { return r.wrapSelection(sel.Find(s)) })
	obj.Set("children", func() goja.Value { return r.wrapSelection(sel.Children()) })
	obj.Set("childrenFiltered", func(s string) goja.Value { return r.wrapSelection(sel.ChildrenFiltered(s)) })
	obj.Set("parent", func() goja.Value { return r.wrapSelection(sel.Parent()) })
	obj.Set("parentFiltered", func(s string) goja.Value { return r.wrapSelection(sel.ParentFiltered(s)) })
	obj.Set("parents", func() goja.Value { return r.wrapSelection(sel.Parents()) })
	obj.Set("parentsUntil", func(s string) goja.Value { return r.wrapSelection(sel.ParentsUntil(s)) })
	obj.Set("closest", func(s string) goja.Value { return r.wrapSelection(sel.Closest(s)) })
	obj.Set("siblings", func() goja.Value { return r.wrapSelection(sel.Siblings()) })
	obj.Set("next", func() goja.Value { return r.wrapSelection(sel.Next()) })
	obj.Set("prev", func() goja.Value { return r.wrapSelection(sel.Prev()) })

	// -----------------------------------------------------------------------
	// Filter methods (filter.go) — non-throwing
	// -----------------------------------------------------------------------

	obj.Set("filter", func(s string) goja.Value { return r.wrapSelection(sel.Filter(s)) })
	obj.Set("filterFunction", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(r.vm.NewGoError(errNotAFunction("filterFunction")))
		}
		result := sel.FilterFunction(func(i int, s *axquery.Selection) bool {
			ret, err := fn(goja.Undefined(), r.vm.ToValue(i), r.wrapSelection(s))
			if err != nil {
				return false
			}
			return ret.ToBoolean()
		})
		return r.wrapSelection(result)
	})
	obj.Set("not", func(s string) goja.Value { return r.wrapSelection(sel.Not(s)) })
	obj.Set("has", func(s string) goja.Value { return r.wrapSelection(sel.Has(s)) })
	obj.Set("is", func(s string) bool { return sel.Is(s) })
	obj.Set("contains", func(text string) goja.Value { return r.wrapSelection(sel.Contains(text)) })

	// -----------------------------------------------------------------------
	// Property methods (property.go) — TERMINAL: throw on error
	// -----------------------------------------------------------------------

	obj.Set("attr", func(name string) string { r.throwIfErr(sel); return sel.Attr(name) })
	obj.Set("attrOr", func(name, def string) string { r.throwIfErr(sel); return sel.AttrOr(name, def) })
	obj.Set("role", func() string { r.throwIfErr(sel); return sel.Role() })
	obj.Set("title", func() string { r.throwIfErr(sel); return sel.Title() })
	obj.Set("description", func() string { r.throwIfErr(sel); return sel.Description() })
	obj.Set("val", func() string { r.throwIfErr(sel); return sel.Val() })
	obj.Set("text", func() string { r.throwIfErr(sel); return sel.Text() })
	obj.Set("isVisible", func() bool { r.throwIfErr(sel); return sel.IsVisible() })
	obj.Set("isEnabled", func() bool { r.throwIfErr(sel); return sel.IsEnabled() })
	obj.Set("isFocused", func() bool { r.throwIfErr(sel); return sel.IsFocused() })
	obj.Set("isSelected", func() bool { r.throwIfErr(sel); return sel.IsSelected() })

	// -----------------------------------------------------------------------
	// Iteration methods (iteration.go) — non-throwing
	// -----------------------------------------------------------------------

	obj.Set("each", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(r.vm.NewGoError(errNotAFunction("each")))
		}
		result := sel.EachWithBreak(func(i int, s *axquery.Selection) bool {
			ret, err := fn(goja.Undefined(), r.vm.ToValue(i), r.wrapSelection(s))
			if err != nil {
				return false // stop on error
			}
			// If callback explicitly returns false, break.
			// Guard against undefined/null which have nil ExportType().
			if ret != nil && ret != goja.Undefined() && ret != goja.Null() {
				if et := ret.ExportType(); et != nil && et.Kind() == boolKind && !ret.ToBoolean() {
					return false
				}
			}
			return true
		})
		return r.wrapSelection(result)
	})
	obj.Set("map", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(r.vm.NewGoError(errNotAFunction("map")))
		}
		result := sel.Map(func(i int, s *axquery.Selection) string {
			ret, err := fn(goja.Undefined(), r.vm.ToValue(i), r.wrapSelection(s))
			if err != nil {
				return ""
			}
			return ret.String()
		})
		return r.vm.ToValue(result)
	})

	// -----------------------------------------------------------------------
	// Action methods (action.go) — TERMINAL: throw on error
	// -----------------------------------------------------------------------

	obj.Set("click", func() goja.Value { r.throwIfErr(sel); return r.wrapSelection(sel.Click()) })
	obj.Set("setValue", func(v string) goja.Value { r.throwIfErr(sel); return r.wrapSelection(sel.SetValue(v)) })
	obj.Set("typeText", func(text string) goja.Value { r.throwIfErr(sel); return r.wrapSelection(sel.TypeText(text)) })
	obj.Set("press", func(call goja.FunctionCall) goja.Value {
		r.throwIfErr(sel)
		key := call.Argument(0).String()
		var mods []string
		for i := 1; i < len(call.Arguments); i++ {
			mods = append(mods, call.Arguments[i].String())
		}
		return r.wrapSelection(sel.Press(key, mods...))
	})
	obj.Set("focus", func() goja.Value { r.throwIfErr(sel); return r.wrapSelection(sel.Focus()) })
	obj.Set("perform", func(action string) goja.Value { r.throwIfErr(sel); return r.wrapSelection(sel.Perform(action)) })

	// -----------------------------------------------------------------------
	// Wait methods (waiting.go) — TERMINAL: throw on error
	// -----------------------------------------------------------------------

	obj.Set("waitUntil", func(call goja.FunctionCall) goja.Value {
		r.throwIfErr(sel)
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(r.vm.NewGoError(errNotAFunction("waitUntil")))
		}
		ms := call.Argument(1).ToInteger()
		timeout := time.Duration(ms) * time.Millisecond
		result := sel.WaitUntil(func(s *axquery.Selection) bool {
			ret, err := fn(goja.Undefined(), r.wrapSelection(s))
			if err != nil {
				return false
			}
			return ret.ToBoolean()
		}, timeout)
		return r.wrapSelection(result)
	})
	obj.Set("waitVisible", func(ms int64) goja.Value {
		r.throwIfErr(sel)
		return r.wrapSelection(sel.WaitVisible(time.Duration(ms) * time.Millisecond))
	})
	obj.Set("waitEnabled", func(ms int64) goja.Value {
		r.throwIfErr(sel)
		return r.wrapSelection(sel.WaitEnabled(time.Duration(ms) * time.Millisecond))
	})
	obj.Set("waitGone", func(ms int64) goja.Value {
		r.throwIfErr(sel)
		return r.wrapSelection(sel.WaitGone(time.Duration(ms) * time.Millisecond))
	})

	// -----------------------------------------------------------------------
	// Scroll methods (scroll.go) — TERMINAL: throw on error
	// -----------------------------------------------------------------------

	obj.Set("scrollDown", func(n int) goja.Value { r.throwIfErr(sel); return r.wrapSelection(sel.ScrollDown(n)) })
	obj.Set("scrollUp", func(n int) goja.Value { r.throwIfErr(sel); return r.wrapSelection(sel.ScrollUp(n)) })
	obj.Set("scrollIntoView", func() goja.Value { r.throwIfErr(sel); return r.wrapSelection(sel.ScrollIntoView()) })

	return obj
}

// errNotAFunction returns an error for when a JS callback argument is not a function.
func errNotAFunction(method string) error {
	return fmt.Errorf("%s: argument must be a function", method)
}
