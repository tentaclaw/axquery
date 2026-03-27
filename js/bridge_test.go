package js

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tentaclaw/ax"
	"github.com/tentaclaw/axquery"
)

// ---------------------------------------------------------------------------
// Helper: execute JS against a known Selection
// ---------------------------------------------------------------------------

// evalWithSelection creates a Runtime, injects a Selection via Go, then runs JS against it.
func evalWithSelection(t *testing.T, sel *axquery.Selection, js string) interface{} {
	t.Helper()
	rt := New(WithBridge(&fakeBridge{}))
	// Inject the selection as a global variable for testing.
	rt.vm.Set("sel", rt.wrapSelection(sel))
	val, err := rt.Execute(js)
	if err != nil {
		t.Fatalf("Execute(%q): %v", js, err)
	}
	return val.Export()
}

// evalWithSelectionErr is like evalWithSelection but expects an error.
func evalWithSelectionErr(t *testing.T, sel *axquery.Selection, js string) error {
	t.Helper()
	rt := New(WithBridge(&fakeBridge{}))
	rt.vm.Set("sel", rt.wrapSelection(sel))
	_, err := rt.Execute(js)
	return err
}

// ---------------------------------------------------------------------------
// count / isEmpty / err / selector — basic methods
// ---------------------------------------------------------------------------

func TestBridge_Count_Empty(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v (%T)", got, got)
	}
}

func TestBridge_Count_NonEmpty(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXButton").count()`)
	if err != nil {
		t.Fatal(err)
	}
	count := val.ToInteger()
	if count <= 0 {
		t.Logf("Finder returned 0 buttons (may happen with no window), count=%d", count)
	}
}

func TestBridge_IsEmpty_True(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.isEmpty()`)
	if got != true {
		t.Fatalf("expected true, got %v", got)
	}
}

func TestBridge_IsEmpty_False(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXWindow").isEmpty()`)
	if err != nil {
		t.Fatal(err)
	}
	// Finder should have at least one window; but CI might not.
	t.Logf("AXWindow isEmpty: %v", val.Export())
}

func TestBridge_Err_NoError(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.err()`)
	if got != nil {
		t.Fatalf("expected null, got %v", got)
	}
}

func TestBridge_Err_WithError(t *testing.T) {
	sel := axquery.NewSelectionError(fmt.Errorf("test error"), "AXButton")
	got := evalWithSelection(t, sel, `sel.err()`)
	if got == nil {
		t.Fatal("expected error string, got null")
	}
	if !strings.Contains(got.(string), "test error") {
		t.Fatalf("expected 'test error' in %q", got)
	}
}

func TestBridge_Selector(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton[title='OK']")
	got := evalWithSelection(t, sel, `sel.selector()`)
	if got != "AXButton[title='OK']" {
		t.Fatalf("expected selector string, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// first / last / eq / slice — collection narrowing
// ---------------------------------------------------------------------------

func TestBridge_First_Empty_ReturnsErrorSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.first().err()`)
	if got == nil {
		t.Fatal("expected error from first() on empty, got null")
	}
}

func TestBridge_Last_Empty_ReturnsErrorSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.last().err()`)
	if got == nil {
		t.Fatal("expected error from last() on empty, got null")
	}
}

func TestBridge_Eq_OutOfRange(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.eq(5).err()`)
	if got == nil {
		t.Fatal("expected error from eq(5) on empty, got null")
	}
}

func TestBridge_Slice_Empty(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.slice(0, 0).count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Chaining — methods returning Selection should be wrapped as JS Selection
// ---------------------------------------------------------------------------

func TestBridge_Chaining_FirstCount(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXButton").first().count()`)
	if err != nil {
		// May error if no buttons found
		t.Logf("first().count() error (may be OK): %v", err)
		return
	}
	count := val.ToInteger()
	if count != 1 && count != 0 {
		t.Fatalf("first().count() should be 0 or 1, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// find — traversal returning wrapped Selection
// ---------------------------------------------------------------------------

func TestBridge_Find_ReturnsSelection(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXWindow").find("AXButton").count()`)
	if err != nil {
		t.Logf("find error (may be OK if no window): %v", err)
		return
	}
	t.Logf("AXWindow > find(AXButton) count: %d", val.ToInteger())
}

func TestBridge_Children_ReturnsSelection(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXWindow").children().count()`)
	if err != nil {
		t.Logf("children error: %v", err)
		return
	}
	t.Logf("AXWindow children count: %d", val.ToInteger())
}

// ---------------------------------------------------------------------------
// filter / not / has / is / contains — filter methods
// ---------------------------------------------------------------------------

func TestBridge_Filter_ReturnsSelection(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	// filter("AXButton") on a set of buttons should return all of them
	val, err := rt.Execute(`$ax("AXButton").filter("AXButton").count()`)
	if err != nil {
		t.Logf("filter error: %v", err)
		return
	}
	t.Logf("filter result count: %d", val.ToInteger())
}

func TestBridge_Is_ReturnsBool(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXButton").is("AXButton")`)
	if err != nil {
		t.Logf("is error: %v", err)
		return
	}
	// Should be true since we queried AXButton and asked is("AXButton")
	t.Logf("is(AXButton) = %v", val.Export())
}

func TestBridge_Contains_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.contains("test").count()`)
	if got != int64(0) {
		t.Fatalf("expected 0 for contains on empty, got %v", got)
	}
}

func TestBridge_Not_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.not("AXWindow").count()`)
	if got != int64(0) {
		t.Fatalf("expected 0 for not on empty, got %v", got)
	}
}

func TestBridge_Has_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.has("AXStaticText").count()`)
	if got != int64(0) {
		t.Fatalf("expected 0 for has on empty, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// filterFunction — JS callback filtering
// ---------------------------------------------------------------------------

func TestBridge_FilterFunction_Callback(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	// filterFunction: keep only first element
	val, err := rt.Execute(`$ax("AXButton").filterFunction(function(i, s) { return i === 0; }).count()`)
	if err != nil {
		t.Logf("filterFunction error: %v", err)
		return
	}
	count := val.ToInteger()
	t.Logf("filterFunction result count: %d", count)
	if count > 1 {
		t.Fatalf("filterFunction(i===0) should return at most 1, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// attr / attrOr / role / title / description / val / text — property methods
// ---------------------------------------------------------------------------

func TestBridge_Title_WithFinder(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXButton").first().title()`)
	if err != nil {
		t.Logf("title error (may be OK): %v", err)
		return
	}
	t.Logf("first button title: %q", val.Export())
}

func TestBridge_Role_ReturnsString(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXButton").first().role()`)
	if err != nil {
		t.Logf("role error: %v", err)
		return
	}
	role := val.Export().(string)
	if role != "AXButton" {
		t.Fatalf("expected role AXButton, got %q", role)
	}
}

func TestBridge_Attr_ReturnsString(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXWindow").first().attr("title")`)
	if err != nil {
		t.Logf("attr error: %v", err)
		return
	}
	t.Logf("window attr(title): %q", val.Export())
}

func TestBridge_AttrOr_FallbackDefault(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.attrOr("title", "default_val")`)
	if got != "default_val" {
		t.Fatalf("expected 'default_val', got %v", got)
	}
}

func TestBridge_Text_ReturnsString(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.text()`)
	if got != "" {
		t.Fatalf("expected empty text for empty selection, got %v", got)
	}
}

func TestBridge_Val_ReturnsString(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.val()`)
	if got != "" {
		t.Fatalf("expected empty val for empty selection, got %v", got)
	}
}

func TestBridge_Description_ReturnsString(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.description()`)
	if got != "" {
		t.Fatalf("expected empty description for empty selection, got %v", got)
	}
}

func TestBridge_IsVisible_ReturnsBool(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.isVisible()`)
	if got != false {
		t.Fatalf("expected false for empty selection, got %v", got)
	}
}

func TestBridge_IsEnabled_ReturnsBool(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.isEnabled()`)
	if got != false {
		t.Fatalf("expected false for empty selection, got %v", got)
	}
}

func TestBridge_IsFocused_ReturnsBool(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.isFocused()`)
	if got != false {
		t.Fatalf("expected false for empty selection, got %v", got)
	}
}

func TestBridge_IsSelected_ReturnsBool(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.isSelected()`)
	if got != false {
		t.Fatalf("expected false for empty selection, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// each — iteration with JS callback
// ---------------------------------------------------------------------------

func TestBridge_Each_CallbackInvoked(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`
		var count = 0;
		$ax("AXButton").each(function(i, s) {
			count++;
			if (i >= 2) return false; // break after 3
		});
		count;
	`)
	if err != nil {
		t.Logf("each error: %v", err)
		return
	}
	count := val.ToInteger()
	t.Logf("each callback invoked %d times", count)
	if count > 3 {
		t.Fatalf("expected break at 3, got %d", count)
	}
}

func TestBridge_Each_EmptySelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `
		var count = 0;
		sel.each(function(i, s) { count++; });
		count;
	`)
	if got != int64(0) {
		t.Fatalf("expected 0 iterations on empty, got %v", got)
	}
}

func TestBridge_Each_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.each(function(){}).count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// map — returns string array
// ---------------------------------------------------------------------------

func TestBridge_Map_ReturnsArray(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`$ax("AXButton").map(function(i, s) { return s.role(); })`)
	if err != nil {
		t.Logf("map error: %v", err)
		return
	}
	exported := val.Export()
	t.Logf("map result: %v (type=%T)", exported, exported)
}

func TestBridge_Map_EmptySelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.map(function(i, s) { return "x"; }).length`)
	if got != int64(0) {
		t.Fatalf("expected empty array length 0, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// click / setValue / typeText / press / focus / perform — action methods
// (Need AX permission + real elements to actually test actions)
// ---------------------------------------------------------------------------

func TestBridge_Click_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	// Click on empty selection should just return the selection (with error)
	got := evalWithSelection(t, sel, `typeof sel.click()`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_SetValue_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXTextField")
	got := evalWithSelection(t, sel, `typeof sel.setValue("test")`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_TypeText_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXTextField")
	got := evalWithSelection(t, sel, `typeof sel.typeText("hello")`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_Press_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `typeof sel.press("return")`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_Focus_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `typeof sel.focus()`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_Perform_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `typeof sel.perform("AXPress")`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// waitUntil / waitVisible / waitEnabled / waitGone — wait methods
// ---------------------------------------------------------------------------

func TestBridge_WaitUntil_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	// waitUntil on empty with always-true fn: should return immediately
	got := evalWithSelection(t, sel, `typeof sel.waitUntil(function(s) { return true; }, 100)`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_WaitVisible_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `typeof sel.waitVisible(100)`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_WaitEnabled_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `typeof sel.waitEnabled(100)`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_WaitGone_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `typeof sel.waitGone(100)`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// scrollDown / scrollUp / scrollIntoView — scroll methods
// ---------------------------------------------------------------------------

func TestBridge_ScrollDown_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXScrollArea")
	got := evalWithSelection(t, sel, `typeof sel.scrollDown(1)`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_ScrollUp_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXScrollArea")
	got := evalWithSelection(t, sel, `typeof sel.scrollUp(1)`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

func TestBridge_ScrollIntoView_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `typeof sel.scrollIntoView()`)
	if got != "object" {
		t.Fatalf("expected object, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Traversal methods — parent/parents/parentsUntil/closest/siblings/next/prev
// ---------------------------------------------------------------------------

func TestBridge_Parent_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.parent().count()`)
	if got != int64(0) {
		t.Fatalf("expected 0 for parent of empty, got %v", got)
	}
}

func TestBridge_ParentFiltered_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.parentFiltered("AXWindow").count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestBridge_Parents_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.parents().count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestBridge_ParentsUntil_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.parentsUntil("AXWindow").count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestBridge_Closest_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.closest("AXWindow").count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestBridge_Siblings_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.siblings().count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestBridge_Next_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.next().count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestBridge_Prev_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.prev().count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestBridge_ChildrenFiltered_ReturnsSelection(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXWindow")
	got := evalWithSelection(t, sel, `sel.childrenFiltered("AXButton").count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Integration: full chain test
// ---------------------------------------------------------------------------

func TestBridge_FullChain_Integration(t *testing.T) {
	if !ax.IsTrusted(false) {
		t.Skip("no AX permission")
	}
	app, err := ax.ApplicationFromBundleID("com.apple.finder")
	if err != nil {
		t.Skip("Finder not available:", err)
	}
	defer app.Close()

	rt := New(WithBridge(&fakeBridge{}))
	rt.SetApp(app)
	val, err := rt.Execute(`
		var btns = $ax("AXButton");
		var count = btns.count();
		var first = btns.first();
		var title = first.title();
		var role = first.role();
		$output.count = count;
		$output.title = title;
		$output.role = role;
		count;
	`)
	if err != nil {
		t.Logf("full chain error (may be OK): %v", err)
		return
	}
	t.Logf("Full chain: count=%d", val.ToInteger())
	out := rt.Output()
	t.Logf("Output: %v", out)
}

// ---------------------------------------------------------------------------
// Error propagation — error selection methods still work
// ---------------------------------------------------------------------------

func TestBridge_ErrorSelection_Chaining(t *testing.T) {
	sel := axquery.NewSelectionError(fmt.Errorf("broken"), "AXButton")
	// All methods should still work on error selection
	got := evalWithSelection(t, sel, `sel.first().last().count()`)
	if got != int64(0) {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestBridge_ErrorSelection_MethodsReturnSelection(t *testing.T) {
	sel := axquery.NewSelectionError(fmt.Errorf("broken"), "AXButton")
	methods := []string{
		"sel.first()",
		"sel.last()",
		"sel.filter('AXButton')",
		"sel.not('AXWindow')",
		"sel.has('AXButton')",
		"sel.contains('x')",
		"sel.children()",
		"sel.parent()",
		"sel.parents()",
		"sel.siblings()",
		"sel.next()",
		"sel.prev()",
		"sel.click()",
		"sel.focus()",
		"sel.scrollDown(1)",
		"sel.scrollUp(1)",
		"sel.scrollIntoView()",
	}
	for _, m := range methods {
		rt := New(WithBridge(&fakeBridge{}))
		rt.vm.Set("sel", rt.wrapSelection(sel))
		val, err := rt.Execute(fmt.Sprintf(`typeof %s`, m))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", m, err)
		}
		if val.Export().(string) != "object" {
			t.Fatalf("%s: expected object, got %v", m, val.Export())
		}
	}
}

// ---------------------------------------------------------------------------
// errNotAFunction — passing non-function to callback-accepting methods
// ---------------------------------------------------------------------------

func TestBridge_Each_NonFunction_Panics(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.each("not a function")`)
	if err == nil {
		t.Fatal("expected error when passing non-function to each")
	}
	if !strings.Contains(err.Error(), "each") || !strings.Contains(err.Error(), "function") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestBridge_Map_NonFunction_Panics(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.map(42)`)
	if err == nil {
		t.Fatal("expected error when passing non-function to map")
	}
	if !strings.Contains(err.Error(), "map") || !strings.Contains(err.Error(), "function") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestBridge_FilterFunction_NonFunction_Panics(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.filterFunction(true)`)
	if err == nil {
		t.Fatal("expected error when passing non-function to filterFunction")
	}
	if !strings.Contains(err.Error(), "filterFunction") || !strings.Contains(err.Error(), "function") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestBridge_WaitUntil_NonFunction_Panics(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.waitUntil("nope", 100)`)
	if err == nil {
		t.Fatal("expected error when passing non-function to waitUntil")
	}
	if !strings.Contains(err.Error(), "waitUntil") || !strings.Contains(err.Error(), "function") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
