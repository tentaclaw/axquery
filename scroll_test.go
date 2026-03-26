package axquery

import (
	"errors"
	"testing"

	"github.com/tentaclaw/ax"
)

// === ScrollDown tests ===

func TestScrollDown_Success(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	result := sel.ScrollDown(3)

	if result != sel {
		t.Fatal("ScrollDown should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	// Should call performAction("AXScrollDownByPage") 3 times
	if len(area.performCalls) != 3 {
		t.Fatalf("expected 3 performAction calls, got %d", len(area.performCalls))
	}
	for _, action := range area.performCalls {
		if action != "AXScrollDownByPage" {
			t.Fatalf("expected AXScrollDownByPage, got %q", action)
		}
	}
}

func TestScrollDown_ZeroPages(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	result := sel.ScrollDown(0)

	if result != sel {
		t.Fatal("ScrollDown(0) should return same Selection")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if len(area.performCalls) != 0 {
		t.Fatalf("expected no performAction calls for 0 pages, got %d", len(area.performCalls))
	}
}

func TestScrollDown_NegativePages(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	result := sel.ScrollDown(-1)

	if result != sel {
		t.Fatal("ScrollDown(-1) should return same Selection")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if len(area.performCalls) != 0 {
		t.Fatalf("expected no performAction calls for negative pages, got %d", len(area.performCalls))
	}
}

func TestScrollDown_DefaultOnePage(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	result := sel.ScrollDown(1)

	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if len(area.performCalls) != 1 {
		t.Fatalf("expected 1 performAction call, got %d", len(area.performCalls))
	}
	if area.performCalls[0] != "AXScrollDownByPage" {
		t.Fatalf("expected AXScrollDownByPage, got %q", area.performCalls[0])
	}
}

func TestScrollDown_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXScrollArea")
	result := sel.ScrollDown(1)

	if result != sel {
		t.Fatal("ScrollDown on error selection should return same selection")
	}
	if !errors.Is(result.Err(), errTest) {
		t.Fatalf("expected errTest, got %v", result.Err())
	}
}

func TestScrollDown_EmptySelection(t *testing.T) {
	sel := newSelection([]*ax.Element{}, "AXScrollArea")
	result := sel.ScrollDown(1)

	if result.Err() == nil {
		t.Fatal("ScrollDown on empty selection should set error")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

func TestScrollDown_ActionError(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	area.performErr = errors.New("scroll not supported")
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	result := sel.ScrollDown(2)

	if result.Err() == nil {
		t.Fatal("expected error from scroll failure")
	}
	// Should fail on the first call and not continue
	if len(area.performCalls) != 1 {
		t.Fatalf("expected 1 performAction call (fail fast), got %d", len(area.performCalls))
	}
}

// === ScrollUp tests ===

func TestScrollUp_Success(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	result := sel.ScrollUp(2)

	if result != sel {
		t.Fatal("ScrollUp should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if len(area.performCalls) != 2 {
		t.Fatalf("expected 2 performAction calls, got %d", len(area.performCalls))
	}
	for _, action := range area.performCalls {
		if action != "AXScrollUpByPage" {
			t.Fatalf("expected AXScrollUpByPage, got %q", action)
		}
	}
}

func TestScrollUp_ZeroPages(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	result := sel.ScrollUp(0)

	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if len(area.performCalls) != 0 {
		t.Fatalf("expected no performAction calls for 0 pages, got %d", len(area.performCalls))
	}
}

func TestScrollUp_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXScrollArea")
	result := sel.ScrollUp(1)

	if result != sel {
		t.Fatal("ScrollUp on error selection should return same selection")
	}
}

func TestScrollUp_EmptySelection(t *testing.T) {
	sel := newSelection([]*ax.Element{}, "AXScrollArea")
	result := sel.ScrollUp(1)

	if result.Err() == nil {
		t.Fatal("ScrollUp on empty selection should set error")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

func TestScrollUp_ActionError(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	area.performErr = errors.New("scroll not supported")
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	result := sel.ScrollUp(3)

	if result.Err() == nil {
		t.Fatal("expected error from scroll failure")
	}
	// Should fail on the first call and not continue
	if len(area.performCalls) != 1 {
		t.Fatalf("expected 1 performAction call (fail fast), got %d", len(area.performCalls))
	}
}

// === ScrollIntoView tests ===

func TestScrollIntoView_Success(t *testing.T) {
	// ScrollIntoView calls AXScrollToVisible on the first element
	btn := newMockActionNode("AXButton", map[string]string{"title": "OK"})
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	result := sel.ScrollIntoView()

	if result != sel {
		t.Fatal("ScrollIntoView should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if len(btn.performCalls) != 1 || btn.performCalls[0] != "AXScrollToVisible" {
		t.Fatalf("expected performAction('AXScrollToVisible'), got %v", btn.performCalls)
	}
}

func TestScrollIntoView_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXButton")
	result := sel.ScrollIntoView()

	if result != sel {
		t.Fatal("ScrollIntoView on error selection should return same selection")
	}
	if !errors.Is(result.Err(), errTest) {
		t.Fatalf("expected errTest, got %v", result.Err())
	}
}

func TestScrollIntoView_EmptySelection(t *testing.T) {
	sel := newSelection([]*ax.Element{}, "AXButton")
	result := sel.ScrollIntoView()

	if result.Err() == nil {
		t.Fatal("ScrollIntoView on empty selection should set error")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

func TestScrollIntoView_ActionError(t *testing.T) {
	btn := newMockActionNode("AXButton", nil)
	btn.performErr = errors.New("cannot scroll to visible")
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	result := sel.ScrollIntoView()

	if result.Err() == nil {
		t.Fatal("expected error from ScrollIntoView failure")
	}
}

// === Chaining tests ===

func TestScroll_Chaining(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	// Chain: ScrollDown -> ScrollUp -> ScrollIntoView
	result := sel.ScrollDown(1).ScrollUp(1).ScrollIntoView()

	if result.Err() != nil {
		t.Fatalf("unexpected error in chain: %v", result.Err())
	}
	// Should have 3 calls: AXScrollDownByPage, AXScrollUpByPage, AXScrollToVisible
	if len(area.performCalls) != 3 {
		t.Fatalf("expected 3 performAction calls, got %d: %v", len(area.performCalls), area.performCalls)
	}
	expected := []string{"AXScrollDownByPage", "AXScrollUpByPage", "AXScrollToVisible"}
	for i, exp := range expected {
		if area.performCalls[i] != exp {
			t.Errorf("call %d: expected %q, got %q", i, exp, area.performCalls[i])
		}
	}
}

func TestScroll_ChainStopsOnError(t *testing.T) {
	area := newMockActionNode("AXScrollArea", nil)
	area.performErr = errors.New("scroll failed")
	sel := newSelectionFromNodes([]queryNode{area}, "AXScrollArea")

	// ScrollDown should fail, subsequent ScrollUp should not execute
	result := sel.ScrollDown(1).ScrollUp(1)

	if result.Err() == nil {
		t.Fatal("expected error to propagate through chain")
	}
	// Only 1 call (the failing ScrollDown), not the ScrollUp
	if len(area.performCalls) != 1 {
		t.Fatalf("expected 1 performAction call, got %d", len(area.performCalls))
	}
}

// === Integration with Find ===

func TestScroll_AfterFind(t *testing.T) {
	root := newMockActionNode("AXWindow", nil)
	area := newMockActionNode("AXScrollArea", nil)
	root.addActionChild(area)

	sel := newSelectionFromNodes([]queryNode{root}, "AXWindow")
	result := sel.Find("AXScrollArea").ScrollDown(2)

	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if len(area.performCalls) != 2 {
		t.Fatalf("expected 2 scroll calls after Find, got %d", len(area.performCalls))
	}
}
