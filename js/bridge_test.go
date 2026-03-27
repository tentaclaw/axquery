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
		t.Fatal("expected error object, got null")
	}
	// .err() now returns a structured object, not a string.
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map from err(), got %T: %v", got, got)
	}
	if m["code"] != "ERROR" {
		t.Fatalf("expected code 'ERROR', got %v", m["code"])
	}
	msg, _ := m["message"].(string)
	if !strings.Contains(msg, "test error") {
		t.Fatalf("expected 'test error' in message %q", msg)
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
	// Non-throwing methods on error selection should still return wrapped Selections.
	// Terminal methods (click, focus, scroll*, etc.) are now expected to throw,
	// so they are tested separately in TestBridge_Terminal_*_ThrowsOnError.
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

// ===========================================================================
// Task 17: Structured errors — terminal methods throw on error selections
// ===========================================================================

// ---------------------------------------------------------------------------
// Terminal property methods should throw structured JS exceptions on error sel
// ---------------------------------------------------------------------------

func TestBridge_Terminal_Text_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.text()`)
	if err == nil {
		t.Fatal("expected text() to throw on error selection")
	}
	// The thrown object is a structured JS object; its string representation
	// via Go error is "[object Object]". Shape is tested in StructuredError tests.
}

func TestBridge_Terminal_Title_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.title()`)
	if err == nil {
		t.Fatal("expected title() to throw on error selection")
	}
}

func TestBridge_Terminal_Role_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.role()`)
	if err == nil {
		t.Fatal("expected role() to throw on error selection")
	}
}

func TestBridge_Terminal_Description_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.description()`)
	if err == nil {
		t.Fatal("expected description() to throw on error selection")
	}
}

func TestBridge_Terminal_Val_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.val()`)
	if err == nil {
		t.Fatal("expected val() to throw on error selection")
	}
}

func TestBridge_Terminal_Attr_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.attr("title")`)
	if err == nil {
		t.Fatal("expected attr() to throw on error selection")
	}
}

func TestBridge_Terminal_AttrOr_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.attrOr("title", "def")`)
	if err == nil {
		t.Fatal("expected attrOr() to throw on error selection")
	}
}

func TestBridge_Terminal_IsVisible_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.isVisible()`)
	if err == nil {
		t.Fatal("expected isVisible() to throw on error selection")
	}
}

func TestBridge_Terminal_IsEnabled_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.isEnabled()`)
	if err == nil {
		t.Fatal("expected isEnabled() to throw on error selection")
	}
}

func TestBridge_Terminal_IsFocused_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.isFocused()`)
	if err == nil {
		t.Fatal("expected isFocused() to throw on error selection")
	}
}

func TestBridge_Terminal_IsSelected_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.isSelected()`)
	if err == nil {
		t.Fatal("expected isSelected() to throw on error selection")
	}
}

// ---------------------------------------------------------------------------
// Terminal action methods should throw structured JS exceptions on error sel
// ---------------------------------------------------------------------------

func TestBridge_Terminal_Click_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.click()`)
	if err == nil {
		t.Fatal("expected click() to throw on error selection")
	}
}

func TestBridge_Terminal_SetValue_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXTextField"}, "AXTextField")
	err := evalWithSelectionErr(t, sel, `sel.setValue("x")`)
	if err == nil {
		t.Fatal("expected setValue() to throw on error selection")
	}
}

func TestBridge_Terminal_TypeText_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXTextField"}, "AXTextField")
	err := evalWithSelectionErr(t, sel, `sel.typeText("x")`)
	if err == nil {
		t.Fatal("expected typeText() to throw on error selection")
	}
}

func TestBridge_Terminal_Press_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.press("return")`)
	if err == nil {
		t.Fatal("expected press() to throw on error selection")
	}
}

func TestBridge_Terminal_Focus_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.focus()`)
	if err == nil {
		t.Fatal("expected focus() to throw on error selection")
	}
}

func TestBridge_Terminal_Perform_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.perform("AXPress")`)
	if err == nil {
		t.Fatal("expected perform() to throw on error selection")
	}
}

// ---------------------------------------------------------------------------
// Terminal wait methods should throw
// ---------------------------------------------------------------------------

func TestBridge_Terminal_WaitVisible_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.waitVisible(100)`)
	if err == nil {
		t.Fatal("expected waitVisible() to throw on error selection")
	}
}

func TestBridge_Terminal_WaitEnabled_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.waitEnabled(100)`)
	if err == nil {
		t.Fatal("expected waitEnabled() to throw on error selection")
	}
}

func TestBridge_Terminal_WaitGone_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.waitGone(100)`)
	if err == nil {
		t.Fatal("expected waitGone() to throw on error selection")
	}
}

func TestBridge_Terminal_WaitUntil_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.waitUntil(function(s) { return true; }, 100)`)
	if err == nil {
		t.Fatal("expected waitUntil() to throw on error selection")
	}
}

// ---------------------------------------------------------------------------
// Terminal scroll methods should throw
// ---------------------------------------------------------------------------

func TestBridge_Terminal_ScrollDown_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXScrollArea"}, "AXScrollArea")
	err := evalWithSelectionErr(t, sel, `sel.scrollDown(1)`)
	if err == nil {
		t.Fatal("expected scrollDown() to throw on error selection")
	}
}

func TestBridge_Terminal_ScrollUp_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXScrollArea"}, "AXScrollArea")
	err := evalWithSelectionErr(t, sel, `sel.scrollUp(1)`)
	if err == nil {
		t.Fatal("expected scrollUp() to throw on error selection")
	}
}

func TestBridge_Terminal_ScrollIntoView_ThrowsOnError(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.scrollIntoView()`)
	if err == nil {
		t.Fatal("expected scrollIntoView() to throw on error selection")
	}
}

// ---------------------------------------------------------------------------
// Structured error object shape — catch and inspect thrown exception
// ---------------------------------------------------------------------------

func TestBridge_StructuredError_NotFound_Shape(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	// Catch the exception and inspect its shape.
	got := evalWithSelection(t, sel, `
		try {
			sel.click();
			"no_error";
		} catch(e) {
			JSON.stringify({code: e.code, message: e.message, selector: e.selector});
		}
	`)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string from JSON.stringify, got %T: %v", got, got)
	}
	if !strings.Contains(s, `"code":"NOT_FOUND"`) {
		t.Fatalf("expected code NOT_FOUND in %s", s)
	}
	if !strings.Contains(s, `"selector":"AXButton"`) {
		t.Fatalf("expected selector in %s", s)
	}
	if !strings.Contains(s, `"message"`) {
		t.Fatalf("expected message in %s", s)
	}
}

func TestBridge_StructuredError_Timeout_Shape(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.TimeoutError{Selector: "AXButton", Duration: "5s"}, "AXButton")
	got := evalWithSelection(t, sel, `
		try {
			sel.text();
			"no_error";
		} catch(e) {
			JSON.stringify({code: e.code, selector: e.selector});
		}
	`)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T: %v", got, got)
	}
	if !strings.Contains(s, `"code":"TIMEOUT"`) {
		t.Fatalf("expected code TIMEOUT in %s", s)
	}
}

func TestBridge_StructuredError_Ambiguous_Shape(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.AmbiguousError{Selector: "AXButton", Count: 3}, "AXButton")
	got := evalWithSelection(t, sel, `
		try {
			sel.click();
			"no_error";
		} catch(e) {
			JSON.stringify({code: e.code, count: e.count});
		}
	`)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T: %v", got, got)
	}
	if !strings.Contains(s, `"code":"AMBIGUOUS"`) {
		t.Fatalf("expected code AMBIGUOUS in %s", s)
	}
	if !strings.Contains(s, `"count":3`) {
		t.Fatalf("expected count 3 in %s", s)
	}
}

func TestBridge_StructuredError_InvalidSelector_Shape(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.InvalidSelectorError{Selector: "???", Reason: "bad syntax"}, "???")
	got := evalWithSelection(t, sel, `
		try {
			sel.text();
			"no_error";
		} catch(e) {
			JSON.stringify({code: e.code, selector: e.selector});
		}
	`)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T: %v", got, got)
	}
	if !strings.Contains(s, `"code":"INVALID_SELECTOR"`) {
		t.Fatalf("expected code INVALID_SELECTOR in %s", s)
	}
}

func TestBridge_StructuredError_NotActionable_Shape(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotActionableError{Action: "click", Reason: "element disabled"}, "AXButton")
	got := evalWithSelection(t, sel, `
		try {
			sel.click();
			"no_error";
		} catch(e) {
			JSON.stringify({code: e.code, action: e.action});
		}
	`)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T: %v", got, got)
	}
	if !strings.Contains(s, `"code":"NOT_ACTIONABLE"`) {
		t.Fatalf("expected code NOT_ACTIONABLE in %s", s)
	}
	if !strings.Contains(s, `"action":"click"`) {
		t.Fatalf("expected action click in %s", s)
	}
}

func TestBridge_StructuredError_GenericError_Shape(t *testing.T) {
	sel := axquery.NewSelectionError(fmt.Errorf("something unknown"), "AXButton")
	got := evalWithSelection(t, sel, `
		try {
			sel.click();
			"no_error";
		} catch(e) {
			JSON.stringify({code: e.code, message: e.message});
		}
	`)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T: %v", got, got)
	}
	if !strings.Contains(s, `"code":"ERROR"`) {
		t.Fatalf("expected code ERROR in %s", s)
	}
	if !strings.Contains(s, `something unknown`) {
		t.Fatalf("expected error message in %s", s)
	}
}

// ---------------------------------------------------------------------------
// Terminal methods should NOT throw on healthy selections (no error)
// ---------------------------------------------------------------------------

func TestBridge_Terminal_Text_NoThrowOnHealthy(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	// Empty selection with no error — terminal methods should NOT throw.
	err := evalWithSelectionErr(t, sel, `sel.text()`)
	if err != nil {
		t.Fatalf("text() on healthy empty selection should not throw: %v", err)
	}
}

func TestBridge_Terminal_Click_NoThrowOnHealthy(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	err := evalWithSelectionErr(t, sel, `sel.click()`)
	if err != nil {
		t.Fatalf("click() on healthy empty selection should not throw: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Chained query → terminal: error propagates to terminal which throws
// ---------------------------------------------------------------------------

func TestBridge_Chain_FindThenClick_ThrowsNotFound(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXWindow"}, "AXWindow")
	// find on error selection propagates error; click at end should throw.
	err := evalWithSelectionErr(t, sel, `sel.find("AXButton").click()`)
	if err == nil {
		t.Fatal("expected click() at end of chain to throw when selection has error")
	}
}

// ---------------------------------------------------------------------------
// .err() should return structured object instead of plain string
// ---------------------------------------------------------------------------

func TestBridge_Err_ReturnsStructuredObject_NotFound(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.NotFoundError{Selector: "AXButton"}, "AXButton")
	got := evalWithSelection(t, sel, `
		var e = sel.err();
		JSON.stringify({code: e.code, message: e.message, selector: e.selector});
	`)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string from JSON.stringify, got %T: %v", got, got)
	}
	if !strings.Contains(s, `"code":"NOT_FOUND"`) {
		t.Fatalf("expected code NOT_FOUND in err() result: %s", s)
	}
	if !strings.Contains(s, `"selector":"AXButton"`) {
		t.Fatalf("expected selector in err() result: %s", s)
	}
}

func TestBridge_Err_ReturnsStructuredObject_Timeout(t *testing.T) {
	sel := axquery.NewSelectionError(&axquery.TimeoutError{Selector: "AXButton", Duration: "3s"}, "AXButton")
	got := evalWithSelection(t, sel, `
		var e = sel.err();
		JSON.stringify({code: e.code, selector: e.selector});
	`)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T: %v", got, got)
	}
	if !strings.Contains(s, `"code":"TIMEOUT"`) {
		t.Fatalf("expected code TIMEOUT in %s", s)
	}
}

func TestBridge_Err_ReturnsStructuredObject_Generic(t *testing.T) {
	sel := axquery.NewSelectionError(fmt.Errorf("some generic error"), "AXButton")
	got := evalWithSelection(t, sel, `
		var e = sel.err();
		JSON.stringify({code: e.code, message: e.message});
	`)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T: %v", got, got)
	}
	if !strings.Contains(s, `"code":"ERROR"`) {
		t.Fatalf("expected code ERROR in %s", s)
	}
}

func TestBridge_Err_ReturnsNull_WhenNoError(t *testing.T) {
	sel := axquery.NewSelection(nil, "AXButton")
	got := evalWithSelection(t, sel, `sel.err()`)
	if got != nil {
		t.Fatalf("expected null from err() on healthy selection, got %v", got)
	}
}
