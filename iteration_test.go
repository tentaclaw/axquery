package axquery

import (
	"testing"
)

// === Each ===

func TestEach_Basic(t *testing.T) {
	_, btn1, btn2, group, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2, group}, "elements")

	var indices []int
	var roles []string
	result := sel.Each(func(i int, s *Selection) {
		indices = append(indices, i)
		roles = append(roles, s.Role())
	})

	if len(indices) != 3 {
		t.Fatalf("expected 3 iterations, got %d", len(indices))
	}
	if indices[0] != 0 || indices[1] != 1 || indices[2] != 2 {
		t.Fatalf("unexpected indices: %v", indices)
	}
	if roles[0] != "AXButton" || roles[1] != "AXButton" || roles[2] != "AXGroup" {
		t.Fatalf("unexpected roles: %v", roles)
	}
	// Each returns the original selection for chaining
	if result.Count() != 3 {
		t.Fatalf("expected chaining to return original selection, got count %d", result.Count())
	}
}

func TestEach_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	called := false
	result := sel.Each(func(i int, s *Selection) {
		called = true
	})
	if called {
		t.Fatal("callback should not be called on empty selection")
	}
	if result.Count() != 0 {
		t.Fatalf("expected 0 count, got %d", result.Count())
	}
}

func TestEach_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	called := false
	result := sel.Each(func(i int, s *Selection) {
		called = true
	})
	if called {
		t.Fatal("callback should not be called on error selection")
	}
	if result.Err() == nil {
		t.Fatal("expected error to be preserved")
	}
}

func TestEach_SingleElement(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")

	var count int
	sel.Each(func(i int, s *Selection) {
		count++
		if s.Count() != 1 {
			t.Fatalf("callback selection should have 1 element, got %d", s.Count())
		}
		if s.Title() != "OK" {
			t.Fatalf("expected 'OK', got %q", s.Title())
		}
	})
	if count != 1 {
		t.Fatalf("expected 1 iteration, got %d", count)
	}
}

func TestEach_CallbackReceivesSingleElementSelection(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2}, "buttons")

	var titles []string
	sel.Each(func(i int, s *Selection) {
		if s.Count() != 1 {
			t.Fatalf("callback selection should have exactly 1 element, got %d", s.Count())
		}
		titles = append(titles, s.Title())
	})
	if len(titles) != 2 {
		t.Fatalf("expected 2 titles, got %d", len(titles))
	}
	if titles[0] != "OK" || titles[1] != "Cancel" {
		t.Fatalf("unexpected titles: %v", titles)
	}
}

// === EachWithBreak ===

func TestEachWithBreak_Basic(t *testing.T) {
	_, btn1, btn2, group, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2, group}, "elements")

	var visited int
	result := sel.EachWithBreak(func(i int, s *Selection) bool {
		visited++
		return true // continue
	})

	if visited != 3 {
		t.Fatalf("expected 3 visits, got %d", visited)
	}
	if result.Count() != 3 {
		t.Fatalf("expected chaining, got count %d", result.Count())
	}
}

func TestEachWithBreak_StopsEarly(t *testing.T) {
	_, btn1, btn2, group, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2, group}, "elements")

	var visited int
	sel.EachWithBreak(func(i int, s *Selection) bool {
		visited++
		return i < 1 // stop after index 1
	})

	if visited != 2 {
		t.Fatalf("expected 2 visits (break at i=1), got %d", visited)
	}
}

func TestEachWithBreak_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	called := false
	result := sel.EachWithBreak(func(i int, s *Selection) bool {
		called = true
		return true
	})
	if called {
		t.Fatal("callback should not be called on empty selection")
	}
	if result.Count() != 0 {
		t.Fatalf("expected 0, got %d", result.Count())
	}
}

func TestEachWithBreak_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	called := false
	result := sel.EachWithBreak(func(i int, s *Selection) bool {
		called = true
		return true
	})
	if called {
		t.Fatal("callback should not be called on error selection")
	}
	if result.Err() == nil {
		t.Fatal("expected error preserved")
	}
}

func TestEachWithBreak_StopsAtFirst(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2}, "buttons")

	var visited int
	sel.EachWithBreak(func(i int, s *Selection) bool {
		visited++
		return false // stop immediately
	})

	if visited != 1 {
		t.Fatalf("expected 1 visit (break immediately), got %d", visited)
	}
}

// === Map ===

func TestMap_Basic(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2}, "buttons")

	result := sel.Map(func(i int, s *Selection) string {
		return s.Title()
	})

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result[0] != "OK" || result[1] != "Cancel" {
		t.Fatalf("unexpected results: %v", result)
	}
}

func TestMap_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")
	result := sel.Map(func(i int, s *Selection) string {
		return "should not happen"
	})
	if len(result) != 0 {
		t.Fatalf("expected 0 results, got %d", len(result))
	}
}

func TestMap_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")
	result := sel.Map(func(i int, s *Selection) string {
		return "should not happen"
	})
	if len(result) != 0 {
		t.Fatalf("expected 0 results for error selection, got %d", len(result))
	}
}

func TestMap_IndexIsCorrect(t *testing.T) {
	_, btn1, btn2, group, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2, group}, "elements")

	var indices []int
	sel.Map(func(i int, s *Selection) string {
		indices = append(indices, i)
		return ""
	})
	if len(indices) != 3 {
		t.Fatalf("expected 3, got %d", len(indices))
	}
	if indices[0] != 0 || indices[1] != 1 || indices[2] != 2 {
		t.Fatalf("unexpected indices: %v", indices)
	}
}

func TestMap_TransformValues(t *testing.T) {
	node1 := &mockTraversalNode{
		role:  "AXStaticText",
		attrs: map[string]string{"title": "Hello"},
	}
	node2 := &mockTraversalNode{
		role:  "AXStaticText",
		attrs: map[string]string{"title": "World"},
	}
	sel := newSelectionFromNodes([]queryNode{node1, node2}, "text")

	result := sel.Map(func(i int, s *Selection) string {
		return s.Role() + ":" + s.Title()
	})
	if result[0] != "AXStaticText:Hello" || result[1] != "AXStaticText:World" {
		t.Fatalf("unexpected: %v", result)
	}
}

// === EachIter ===

func TestEachIter_Basic(t *testing.T) {
	_, btn1, btn2, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2}, "buttons")

	var indices []int
	var titles []string
	for i, s := range sel.EachIter() {
		indices = append(indices, i)
		titles = append(titles, s.Title())
	}

	if len(indices) != 2 {
		t.Fatalf("expected 2 iterations, got %d", len(indices))
	}
	if indices[0] != 0 || indices[1] != 1 {
		t.Fatalf("unexpected indices: %v", indices)
	}
	if titles[0] != "OK" || titles[1] != "Cancel" {
		t.Fatalf("unexpected titles: %v", titles)
	}
}

func TestEachIter_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "empty")

	count := 0
	for range sel.EachIter() {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 iterations, got %d", count)
	}
}

func TestEachIter_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "test")

	count := 0
	for range sel.EachIter() {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 iterations for error selection, got %d", count)
	}
}

func TestEachIter_BreakEarly(t *testing.T) {
	_, btn1, btn2, group, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1, btn2, group}, "elements")

	var visited int
	for range sel.EachIter() {
		visited++
		if visited == 2 {
			break
		}
	}
	if visited != 2 {
		t.Fatalf("expected 2 visits before break, got %d", visited)
	}
}

func TestEachIter_SingleElement(t *testing.T) {
	_, btn1, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{btn1}, "AXButton")

	var count int
	for _, s := range sel.EachIter() {
		count++
		if s.Role() != "AXButton" {
			t.Fatalf("expected AXButton, got %q", s.Role())
		}
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}
}

// === Chaining: iteration after traversal ===

func TestEach_AfterFind(t *testing.T) {
	root, _, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{root}, "AXWindow")
	buttons := sel.Find("AXButton")

	var titles []string
	buttons.Each(func(i int, s *Selection) {
		titles = append(titles, s.Title())
	})
	if len(titles) == 0 {
		t.Fatal("expected at least one button title")
	}
}

func TestMap_AfterFilter(t *testing.T) {
	root, _, _, _, _, _, _ := buildTraversalTree()
	sel := newSelectionFromNodes([]queryNode{root}, "AXWindow")
	buttons := sel.Find("AXButton").Filter(`AXButton[title="OK"]`)

	result := buttons.Map(func(i int, s *Selection) string {
		return s.Title()
	})
	if len(result) != 1 || result[0] != "OK" {
		t.Fatalf("expected ['OK'], got %v", result)
	}
}
