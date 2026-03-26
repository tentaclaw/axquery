package axquery

import (
	"testing"
)

// === Attr / AttrOr ===

func TestAttr_Basic(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	got := sel.Attr("title")
	if got != "OK" {
		t.Fatalf("expected 'OK', got %q", got)
	}
}

func TestAttr_Role(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	got := sel.Attr("role")
	if got != "AXButton" {
		t.Fatalf("expected 'AXButton', got %q", got)
	}
}

func TestAttr_Missing(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	got := sel.Attr("nonexistent")
	if got != "" {
		t.Fatalf("expected empty string for missing attr, got %q", got)
	}
}

func TestAttr_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	got := sel.Attr("title")
	if got != "" {
		t.Fatalf("expected empty string for empty selection, got %q", got)
	}
}

func TestAttr_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	got := sel.Attr("title")
	if got != "" {
		t.Fatalf("expected empty string for error selection, got %q", got)
	}
}

func TestAttr_FirstElementUsed(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2}, "buttons")
	got := sel.Attr("title")
	if got != "OK" {
		t.Fatalf("expected 'OK' (first element), got %q", got)
	}
}

func TestAttrOr_Found(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	got := sel.AttrOr("title", "fallback")
	if got != "OK" {
		t.Fatalf("expected 'OK', got %q", got)
	}
}

func TestAttrOr_Missing(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	got := sel.AttrOr("nonexistent", "fallback")
	if got != "fallback" {
		t.Fatalf("expected 'fallback', got %q", got)
	}
}

func TestAttrOr_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	got := sel.AttrOr("title", "fallback")
	if got != "fallback" {
		t.Fatalf("expected 'fallback' for empty selection, got %q", got)
	}
}

func TestAttrOr_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	got := sel.AttrOr("title", "fallback")
	if got != "fallback" {
		t.Fatalf("expected 'fallback' for error selection, got %q", got)
	}
}

// === Role ===

func TestRole_Basic(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	got := sel.Role()
	if got != "AXButton" {
		t.Fatalf("expected 'AXButton', got %q", got)
	}
}

func TestRole_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	got := sel.Role()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestRole_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	got := sel.Role()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

// === Title ===

func TestTitle_Basic(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	got := sel.Title()
	if got != "OK" {
		t.Fatalf("expected 'OK', got %q", got)
	}
}

func TestTitle_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	got := sel.Title()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestTitle_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	got := sel.Title()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

// === Description ===

func TestDescription_Basic(t *testing.T) {
	// Build a node with a description attr
	node := &mockTraversalNode{
		role:  "AXButton",
		attrs: map[string]string{"title": "OK", "description": "Confirm action"},
	}
	sel := newSelectionFromNodes([]queryNode{node}, "AXButton")
	got := sel.Description()
	if got != "Confirm action" {
		t.Fatalf("expected 'Confirm action', got %q", got)
	}
}

func TestDescription_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	got := sel.Description()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestDescription_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	got := sel.Description()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

// === Val ===

func TestVal_Basic(t *testing.T) {
	node := &mockTraversalNode{
		role:  "AXTextField",
		attrs: map[string]string{"value": "hello world"},
	}
	sel := newSelectionFromNodes([]queryNode{node}, "AXTextField")
	got := sel.Val()
	if got != "hello world" {
		t.Fatalf("expected 'hello world', got %q", got)
	}
}

func TestVal_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	got := sel.Val()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestVal_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	got := sel.Val()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

// === Text ===

func TestText_LeafNode(t *testing.T) {
	node := &mockTraversalNode{
		role:  "AXStaticText",
		attrs: map[string]string{"title": "Hello"},
	}
	sel := newSelectionFromNodes([]queryNode{node}, "AXStaticText")
	got := sel.Text()
	if got != "Hello" {
		t.Fatalf("expected 'Hello', got %q", got)
	}
}

func TestText_RecursiveChildren(t *testing.T) {
	// group has text1("Hello") and btn3("Submit") as children
	_, _, _, group, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{group}, "AXGroup")
	got := sel.Text()
	// Should concatenate all descendant titles
	if got != "Hello Submit" {
		t.Fatalf("expected 'Hello Submit', got %q", got)
	}
}

func TestText_DeepNesting(t *testing.T) {
	// root -> wrapper -> inner("Deep text")
	root := &mockTraversalNode{role: "AXGroup"}
	wrapper := &mockTraversalNode{role: "AXGroup"}
	inner := &mockTraversalNode{
		role:  "AXStaticText",
		attrs: map[string]string{"title": "Deep text"},
	}
	root.addChild(wrapper)
	wrapper.addChild(inner)

	sel := newSelectionFromNodes([]queryNode{root}, "AXGroup")
	got := sel.Text()
	if got != "Deep text" {
		t.Fatalf("expected 'Deep text', got %q", got)
	}
}

func TestText_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	got := sel.Text()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestText_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	got := sel.Text()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestText_NoTitle(t *testing.T) {
	// Node with no title and no children
	node := &mockTraversalNode{role: "AXGroup"}
	sel := newSelectionFromNodes([]queryNode{node}, "AXGroup")
	got := sel.Text()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestText_MixedWithAndWithoutTitle(t *testing.T) {
	// parent -> child1(no title) -> grandchild("Inner")
	//        -> child2("Visible")
	parent := &mockTraversalNode{role: "AXGroup"}
	child1 := &mockTraversalNode{role: "AXGroup"}
	grandchild := &mockTraversalNode{
		role:  "AXStaticText",
		attrs: map[string]string{"title": "Inner"},
	}
	child2 := &mockTraversalNode{
		role:  "AXStaticText",
		attrs: map[string]string{"title": "Visible"},
	}
	parent.addChild(child1)
	child1.addChild(grandchild)
	parent.addChild(child2)

	sel := newSelectionFromNodes([]queryNode{parent}, "AXGroup")
	got := sel.Text()
	if got != "Inner Visible" {
		t.Fatalf("expected 'Inner Visible', got %q", got)
	}
}

func TestText_ChildrenError(t *testing.T) {
	// Node whose children error — should still return self title
	errNode := &mockErrorTraversalNode{
		mockTraversalNode: mockTraversalNode{
			role:  "AXButton",
			attrs: map[string]string{"title": "Broken"},
		},
		childrenErr: errTest,
	}
	sel := newSelectionFromNodes([]queryNode{errNode}, "AXButton")
	got := sel.Text()
	if got != "Broken" {
		t.Fatalf("expected 'Broken', got %q", got)
	}
}

// === IsVisible ===

func TestIsVisible_True(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	if !sel.IsVisible() {
		t.Fatal("expected visible")
	}
}

func TestIsVisible_False(t *testing.T) {
	node := &mockTraversalNode{role: "AXButton", visible: false}
	sel := newSelectionFromNodes([]queryNode{node}, "AXButton")
	if sel.IsVisible() {
		t.Fatal("expected not visible")
	}
}

func TestIsVisible_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	if sel.IsVisible() {
		t.Fatal("expected false for empty selection")
	}
}

func TestIsVisible_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	if sel.IsVisible() {
		t.Fatal("expected false for error selection")
	}
}

// === IsEnabled ===

func TestIsEnabled_True(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	if !sel.IsEnabled() {
		t.Fatal("expected enabled")
	}
}

func TestIsEnabled_False(t *testing.T) {
	_, _, _, _, _, _, disabled := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{disabled}, "AXButton")
	if sel.IsEnabled() {
		t.Fatal("expected not enabled")
	}
}

func TestIsEnabled_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	if sel.IsEnabled() {
		t.Fatal("expected false for empty selection")
	}
}

func TestIsEnabled_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	if sel.IsEnabled() {
		t.Fatal("expected false for error selection")
	}
}

// === IsFocused ===

func TestIsFocused_True(t *testing.T) {
	_, _, _, _, _, btn3, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn3}, "AXButton")
	if !sel.IsFocused() {
		t.Fatal("expected focused")
	}
}

func TestIsFocused_False(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")
	if sel.IsFocused() {
		t.Fatal("expected not focused")
	}
}

func TestIsFocused_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	if sel.IsFocused() {
		t.Fatal("expected false for empty selection")
	}
}

func TestIsFocused_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	if sel.IsFocused() {
		t.Fatal("expected false for error selection")
	}
}

// === IsSelected ===

func TestIsSelected_True(t *testing.T) {
	node := &mockTraversalNode{role: "AXRow", selected: true}
	sel := newSelectionFromNodes([]queryNode{node}, "AXRow")
	if !sel.IsSelected() {
		t.Fatal("expected selected")
	}
}

func TestIsSelected_False(t *testing.T) {
	node := &mockTraversalNode{role: "AXRow", selected: false}
	sel := newSelectionFromNodes([]queryNode{node}, "AXRow")
	if sel.IsSelected() {
		t.Fatal("expected not selected")
	}
}

func TestIsSelected_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	if sel.IsSelected() {
		t.Fatal("expected false for empty selection")
	}
}

func TestIsSelected_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	if sel.IsSelected() {
		t.Fatal("expected false for error selection")
	}
}

// === Chaining: property methods after traversal ===

func TestProperty_AfterFind(t *testing.T) {
	root, _, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{root}, "AXWindow")
	found := sel.Find("AXStaticText")
	if found.Err() != nil {
		t.Fatal(found.Err())
	}
	title := found.Title()
	if title != "Hello" {
		t.Fatalf("expected 'Hello', got %q", title)
	}
}

func TestProperty_AfterFilter(t *testing.T) {
	root, _, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{root}, "AXWindow")
	buttons := sel.Find("AXButton").Filter(`AXButton[title="Submit"]`)
	if buttons.Err() != nil {
		t.Fatal(buttons.Err())
	}
	title := buttons.Title()
	if title != "Submit" {
		t.Fatalf("expected 'Submit', got %q", title)
	}
}

func TestProperty_AfterFirst(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2}, "buttons")
	first := sel.First()
	role := first.Role()
	if role != "AXButton" {
		t.Fatalf("expected 'AXButton', got %q", role)
	}
	title := first.Title()
	if title != "OK" {
		t.Fatalf("expected 'OK', got %q", title)
	}
}

func TestProperty_AfterLast(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2}, "buttons")
	last := sel.Last()
	title := last.Title()
	if title != "Cancel" {
		t.Fatalf("expected 'Cancel', got %q", title)
	}
}
