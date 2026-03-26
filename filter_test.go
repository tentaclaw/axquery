package axquery

import (
	"errors"
	"testing"

	"github.com/tentaclaw/axquery/selector"
)

// === Filter tests ===

func TestFilter_BySelector(t *testing.T) {
	_, btn1, btn2, group, text1, btn3, disabled := buildTraversalTree()

	// Selection with all children of root
	nodes := []queryNode{btn1, btn2, group, text1, btn3, disabled}
	sel := newSelectionFromNodes(nodes, "")

	// Filter to AXButton only
	result := sel.Filter("AXButton")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 4 {
		t.Fatalf("expected 4 buttons, got %d", result.Count())
	}
}

func TestFilter_EnabledButtons(t *testing.T) {
	_, btn1, btn2, _, _, btn3, disabled := buildTraversalTree()

	nodes := []queryNode{btn1, btn2, btn3, disabled}
	sel := newSelectionFromNodes(nodes, "")

	// Filter to :enabled only
	result := sel.Filter("AXButton:enabled")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 3 {
		t.Fatalf("expected 3 enabled buttons, got %d", result.Count())
	}
}

func TestFilter_NoMatch(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Filter("AXStaticText")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 0 {
		t.Fatalf("expected 0, got %d", result.Count())
	}
}

func TestFilter_InvalidSelector(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Filter("[invalid")
	if result.Err() == nil {
		t.Fatal("expected error for invalid selector")
	}
	if !errors.Is(result.Err(), ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector, got %v", result.Err())
	}
}

func TestFilter_PropagatesError(t *testing.T) {
	errSel := newSelectionError(errTest, "test")
	result := errSel.Filter("AXButton")
	if result.Err() != errTest {
		t.Fatalf("expected propagated error, got %v", result.Err())
	}
}

func TestFilter_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "")
	result := sel.Filter("AXButton")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 0 {
		t.Fatalf("expected 0, got %d", result.Count())
	}
}

func TestFilter_PreservesNodes(t *testing.T) {
	_, btn1, btn2, _, _, btn3, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2, btn3}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Filter(`AXButton[title="OK"]`)
	if result.Count() != 1 {
		t.Fatalf("expected 1, got %d", result.Count())
	}
	// Filtered result should still support traversal (has nodes)
	parent := result.Parent()
	if parent.Err() != nil {
		t.Fatal(parent.Err())
	}
	if parent.Count() != 1 {
		t.Fatalf("expected 1 parent, got %d", parent.Count())
	}
}

// === FilterFunction tests ===

func TestFilterFunction_Basic(t *testing.T) {
	_, btn1, btn2, group, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2, group}
	sel := newSelectionFromNodes(nodes, "")

	// Keep only elements at even indices
	result := sel.FilterFunction(func(i int, s *Selection) bool {
		return i%2 == 0
	})
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 2 {
		t.Fatalf("expected 2, got %d", result.Count())
	}
}

func TestFilterFunction_All(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.FilterFunction(func(i int, s *Selection) bool {
		return true
	})
	if result.Count() != 2 {
		t.Fatalf("expected 2, got %d", result.Count())
	}
}

func TestFilterFunction_None(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.FilterFunction(func(i int, s *Selection) bool {
		return false
	})
	if result.Count() != 0 {
		t.Fatalf("expected 0, got %d", result.Count())
	}
}

func TestFilterFunction_PropagatesError(t *testing.T) {
	errSel := newSelectionError(errTest, "test")
	result := errSel.FilterFunction(func(i int, s *Selection) bool {
		return true
	})
	if result.Err() != errTest {
		t.Fatalf("expected propagated error, got %v", result.Err())
	}
}

func TestFilterFunction_SingleElement(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1}
	sel := newSelectionFromNodes(nodes, "")

	// Verify the callback receives a single-element Selection
	var received *Selection
	result := sel.FilterFunction(func(i int, s *Selection) bool {
		received = s
		return true
	})
	if result.Count() != 1 {
		t.Fatalf("expected 1, got %d", result.Count())
	}
	if received == nil || received.Count() != 1 {
		t.Fatal("callback should receive single-element Selection")
	}
}

// === Not tests ===

func TestNot_BySelector(t *testing.T) {
	_, btn1, btn2, group, text1, btn3, disabled := buildTraversalTree()

	nodes := []queryNode{btn1, btn2, group, text1, btn3, disabled}
	sel := newSelectionFromNodes(nodes, "")

	// Remove all buttons
	result := sel.Not("AXButton")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 2 {
		t.Fatalf("expected 2 non-buttons (group + text1), got %d", result.Count())
	}
}

func TestNot_NoMatch(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2}
	sel := newSelectionFromNodes(nodes, "")

	// Not matching anything means all kept
	result := sel.Not("AXStaticText")
	if result.Count() != 2 {
		t.Fatalf("expected 2, got %d", result.Count())
	}
}

func TestNot_InvalidSelector(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Not("[invalid")
	if result.Err() == nil {
		t.Fatal("expected error for invalid selector")
	}
	if !errors.Is(result.Err(), ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector, got %v", result.Err())
	}
}

func TestNot_PropagatesError(t *testing.T) {
	errSel := newSelectionError(errTest, "test")
	result := errSel.Not("AXButton")
	if result.Err() != errTest {
		t.Fatalf("expected propagated error, got %v", result.Err())
	}
}

func TestNot_PreservesNodes(t *testing.T) {
	_, btn1, _, group, text1, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, group, text1}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Not("AXButton")
	if result.Count() != 2 {
		t.Fatalf("expected 2, got %d", result.Count())
	}
	// Filtered result should still support traversal
	parent := result.Parent()
	if parent.Err() != nil {
		t.Fatal(parent.Err())
	}
}

// === Has tests ===

func TestHas_MatchingDescendants(t *testing.T) {
	root, _, _, group, _, _, _ := buildTraversalTree()

	// Selection of root and group
	nodes := []queryNode{root, group}
	sel := newSelectionFromNodes(nodes, "")

	// Has AXStaticText descendant — both root (via group->text1) and group (directly) qualify
	result := sel.Has("AXStaticText")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 2 {
		t.Fatalf("expected 2, got %d", result.Count())
	}
}

func TestHas_NoMatchingDescendant(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()

	// Buttons have no children in our tree
	nodes := []queryNode{btn1, btn2}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Has("AXStaticText")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 0 {
		t.Fatalf("expected 0, got %d", result.Count())
	}
}

func TestHas_InvalidSelector(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Has("[invalid")
	if result.Err() == nil {
		t.Fatal("expected error for invalid selector")
	}
	if !errors.Is(result.Err(), ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector, got %v", result.Err())
	}
}

func TestHas_PropagatesError(t *testing.T) {
	errSel := newSelectionError(errTest, "test")
	result := errSel.Has("AXButton")
	if result.Err() != errTest {
		t.Fatalf("expected propagated error, got %v", result.Err())
	}
}

func TestHas_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "")
	result := sel.Has("AXButton")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 0 {
		t.Fatalf("expected 0, got %d", result.Count())
	}
}

func TestHas_PreservesNodes(t *testing.T) {
	_, _, _, group, _, _, _ := buildTraversalTree()

	nodes := []queryNode{group}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Has("AXButton")
	if result.Count() != 1 {
		t.Fatalf("expected 1, got %d", result.Count())
	}
	// Should still support traversal
	parent := result.Parent()
	if parent.Err() != nil {
		t.Fatal(parent.Err())
	}
}

// === Is tests ===

func TestIs_MatchExists(t *testing.T) {
	_, btn1, btn2, group, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2, group}
	sel := newSelectionFromNodes(nodes, "")

	if !sel.Is("AXButton") {
		t.Fatal("expected Is('AXButton') to be true")
	}
}

func TestIs_NoMatch(t *testing.T) {
	_, _, _, group, text1, _, _ := buildTraversalTree()

	nodes := []queryNode{group, text1}
	sel := newSelectionFromNodes(nodes, "")

	if sel.Is("AXButton") {
		t.Fatal("expected Is('AXButton') to be false")
	}
}

func TestIs_InvalidSelector(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1}
	sel := newSelectionFromNodes(nodes, "")

	// Invalid selector should return false
	if sel.Is("[invalid") {
		t.Fatal("expected false for invalid selector")
	}
}

func TestIs_PropagatesError(t *testing.T) {
	errSel := newSelectionError(errTest, "test")
	if errSel.Is("AXButton") {
		t.Fatal("expected false when selection has error")
	}
}

func TestIs_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "")
	if sel.Is("AXButton") {
		t.Fatal("expected false for empty selection")
	}
}

func TestIs_WithAttrMatch(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2}
	sel := newSelectionFromNodes(nodes, "")

	if !sel.Is(`AXButton[title="OK"]`) {
		t.Fatal("expected Is to match btn1 with title OK")
	}
	if sel.Is(`AXButton[title="Submit"]`) {
		t.Fatal("expected Is to be false — no btn with title Submit in selection")
	}
}

// === Contains tests (text-based filtering) ===

func TestContains_MatchingText(t *testing.T) {
	_, btn1, btn2, _, text1, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2, text1}
	sel := newSelectionFromNodes(nodes, "")

	// Contains should keep elements whose "title" attr contains the given text
	result := sel.Contains("OK")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 1 {
		t.Fatalf("expected 1, got %d", result.Count())
	}
}

func TestContains_NoMatch(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Contains("nonexistent")
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 0 {
		t.Fatalf("expected 0, got %d", result.Count())
	}
}

func TestContains_PropagatesError(t *testing.T) {
	errSel := newSelectionError(errTest, "test")
	result := errSel.Contains("text")
	if result.Err() != errTest {
		t.Fatalf("expected propagated error, got %v", result.Err())
	}
}

// === FilterMatcher / NotMatcher tests (pre-compiled matcher variants) ===

func TestFilterMatcher_Basic(t *testing.T) {
	_, btn1, btn2, group, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2, group}
	sel := newSelectionFromNodes(nodes, "")

	matcher := selector.MustCompile("AXButton")
	result := sel.FilterMatcher(matcher)
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 2 {
		t.Fatalf("expected 2, got %d", result.Count())
	}
}

func TestNotMatcher_Basic(t *testing.T) {
	_, btn1, btn2, group, _, _, _ := buildTraversalTree()

	nodes := []queryNode{btn1, btn2, group}
	sel := newSelectionFromNodes(nodes, "")

	matcher := selector.MustCompile("AXButton")
	result := sel.NotMatcher(matcher)
	if result.Err() != nil {
		t.Fatal(result.Err())
	}
	if result.Count() != 1 {
		t.Fatalf("expected 1 (group), got %d", result.Count())
	}
}

func TestFilterMatcher_PropagatesError(t *testing.T) {
	errSel := newSelectionError(errTest, "test")
	matcher := selector.MustCompile("AXButton")
	result := errSel.FilterMatcher(matcher)
	if result.Err() != errTest {
		t.Fatalf("expected propagated error, got %v", result.Err())
	}
}

func TestNotMatcher_PropagatesError(t *testing.T) {
	errSel := newSelectionError(errTest, "test")
	matcher := selector.MustCompile("AXButton")
	result := errSel.NotMatcher(matcher)
	if result.Err() != errTest {
		t.Fatalf("expected propagated error, got %v", result.Err())
	}
}

// === Chaining filter with other methods ===

func TestFilter_ChainingWithTraversal(t *testing.T) {
	root, _, _, group, _, btn3, _ := buildTraversalTree()
	_ = root

	// Start from group's children, filter to AXButton, then get parent
	nodes := []queryNode{group}
	sel := newSelectionFromNodes(nodes, "")

	result := sel.Children().Filter("AXButton")
	if result.Count() != 1 {
		t.Fatalf("expected 1 button child (btn3), got %d", result.Count())
	}
	_ = btn3

	// Chaining: filter result should support Parent()
	parent := result.Parent()
	if parent.Err() != nil {
		t.Fatal(parent.Err())
	}
	if parent.Count() != 1 {
		t.Fatalf("expected 1 parent (group), got %d", parent.Count())
	}
}

func TestNot_ChainingWithTraversal(t *testing.T) {
	_, _, _, group, _, _, _ := buildTraversalTree()

	nodes := []queryNode{group}
	sel := newSelectionFromNodes(nodes, "")

	// Children: text1 + btn3, Not AXButton -> text1
	result := sel.Children().Not("AXButton")
	if result.Count() != 1 {
		t.Fatalf("expected 1 (text1), got %d", result.Count())
	}
}
