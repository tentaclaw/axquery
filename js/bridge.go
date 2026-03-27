package js

import (
	"fmt"
	"reflect"
	"time"

	"github.com/dop251/goja"
	"github.com/tentaclaw/axquery"
)

// boolKind is cached to avoid repeated reflect.Bool allocation in hot paths.
var boolKind = reflect.Bool

// wrapSelection wraps a Go *axquery.Selection as a fully-featured JS object.
// Every Selection method is exposed as a camelCase JS method.
// Methods returning *Selection recursively wrap the result for chaining.
func (r *Runtime) wrapSelection(sel *axquery.Selection) goja.Value {
	obj := r.vm.NewObject()

	// -----------------------------------------------------------------------
	// Basic methods (selection.go)
	// -----------------------------------------------------------------------

	obj.Set("count", func() int { return sel.Count() })
	obj.Set("isEmpty", func() bool { return sel.IsEmpty() })
	obj.Set("err", func() interface{} {
		if err := sel.Err(); err != nil {
			return err.Error()
		}
		return nil
	})
	obj.Set("selector", func() string { return sel.Selector() })
	obj.Set("first", func() goja.Value { return r.wrapSelection(sel.First()) })
	obj.Set("last", func() goja.Value { return r.wrapSelection(sel.Last()) })
	obj.Set("eq", func(i int) goja.Value { return r.wrapSelection(sel.Eq(i)) })
	obj.Set("slice", func(start, end int) goja.Value { return r.wrapSelection(sel.Slice(start, end)) })

	// -----------------------------------------------------------------------
	// Traversal methods (traversal.go)
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
	// Filter methods (filter.go)
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
	// Property methods (property.go)
	// -----------------------------------------------------------------------

	obj.Set("attr", func(name string) string { return sel.Attr(name) })
	obj.Set("attrOr", func(name, def string) string { return sel.AttrOr(name, def) })
	obj.Set("role", func() string { return sel.Role() })
	obj.Set("title", func() string { return sel.Title() })
	obj.Set("description", func() string { return sel.Description() })
	obj.Set("val", func() string { return sel.Val() })
	obj.Set("text", func() string { return sel.Text() })
	obj.Set("isVisible", func() bool { return sel.IsVisible() })
	obj.Set("isEnabled", func() bool { return sel.IsEnabled() })
	obj.Set("isFocused", func() bool { return sel.IsFocused() })
	obj.Set("isSelected", func() bool { return sel.IsSelected() })

	// -----------------------------------------------------------------------
	// Iteration methods (iteration.go)
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
	// Action methods (action.go)
	// -----------------------------------------------------------------------

	obj.Set("click", func() goja.Value { return r.wrapSelection(sel.Click()) })
	obj.Set("setValue", func(v string) goja.Value { return r.wrapSelection(sel.SetValue(v)) })
	obj.Set("typeText", func(text string) goja.Value { return r.wrapSelection(sel.TypeText(text)) })
	obj.Set("press", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		var mods []string
		for i := 1; i < len(call.Arguments); i++ {
			mods = append(mods, call.Arguments[i].String())
		}
		return r.wrapSelection(sel.Press(key, mods...))
	})
	obj.Set("focus", func() goja.Value { return r.wrapSelection(sel.Focus()) })
	obj.Set("perform", func(action string) goja.Value { return r.wrapSelection(sel.Perform(action)) })

	// -----------------------------------------------------------------------
	// Wait methods (waiting.go)
	// -----------------------------------------------------------------------

	obj.Set("waitUntil", func(call goja.FunctionCall) goja.Value {
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
		return r.wrapSelection(sel.WaitVisible(time.Duration(ms) * time.Millisecond))
	})
	obj.Set("waitEnabled", func(ms int64) goja.Value {
		return r.wrapSelection(sel.WaitEnabled(time.Duration(ms) * time.Millisecond))
	})
	obj.Set("waitGone", func(ms int64) goja.Value {
		return r.wrapSelection(sel.WaitGone(time.Duration(ms) * time.Millisecond))
	})

	// -----------------------------------------------------------------------
	// Scroll methods (scroll.go)
	// -----------------------------------------------------------------------

	obj.Set("scrollDown", func(n int) goja.Value { return r.wrapSelection(sel.ScrollDown(n)) })
	obj.Set("scrollUp", func(n int) goja.Value { return r.wrapSelection(sel.ScrollUp(n)) })
	obj.Set("scrollIntoView", func() goja.Value { return r.wrapSelection(sel.ScrollIntoView()) })

	return obj
}

// errNotAFunction returns an error for when a JS callback argument is not a function.
func errNotAFunction(method string) error {
	return fmt.Errorf("%s: argument must be a function", method)
}
