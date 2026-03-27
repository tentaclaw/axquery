package axquery

import (
	"errors"
	"testing"

	"github.com/tentaclaw/ax"
)

// We use nil *ax.Element pointers for testing basic Selection operations
// (Count, IsEmpty, First, Last, Eq, Slice, Err).
// These tests never call any Element methods, so nil is safe.

func makeTestSelection(n int) *Selection {
	elems := make([]*ax.Element, n)
	// All nil — safe for index/slice operations that don't dereference
	return newSelection(elems, "test")
}

func makeErrSelection(err error) *Selection {
	return newSelectionError(err, "test")
}

// --- newSelection / newSelectionError ---

func TestNewSelection_Empty(t *testing.T) {
	s := newSelection(nil, "AXButton")
	if s == nil {
		t.Fatal("newSelection should not return nil")
	}
	if s.Count() != 0 {
		t.Fatalf("expected 0, got %d", s.Count())
	}
	if s.Err() != nil {
		t.Fatalf("expected no error, got %v", s.Err())
	}
	if s.Selector() != "AXButton" {
		t.Fatalf("expected selector AXButton, got %s", s.Selector())
	}
}

func TestNewSelection_WithElements(t *testing.T) {
	s := makeTestSelection(3)
	if s.Count() != 3 {
		t.Fatalf("expected 3, got %d", s.Count())
	}
}

func TestNewSelectionError(t *testing.T) {
	err := errors.New("something broke")
	s := makeErrSelection(err)
	if s.Err() != err {
		t.Fatalf("expected error, got %v", s.Err())
	}
	if s.Count() != 0 {
		t.Fatalf("error selection should have 0 elements, got %d", s.Count())
	}
}

// --- Exported constructors (NewSelection / NewSelectionError) ---

func TestExportedNewSelection(t *testing.T) {
	s := NewSelection(nil, "AXWindow")
	if s == nil {
		t.Fatal("NewSelection returned nil")
	}
	if s.Count() != 0 {
		t.Fatalf("expected 0, got %d", s.Count())
	}
	if s.Selector() != "AXWindow" {
		t.Fatalf("expected AXWindow, got %s", s.Selector())
	}
}

func TestExportedNewSelectionError(t *testing.T) {
	err := errors.New("export err")
	s := NewSelectionError(err, "AXMenuItem")
	if s.Err() != err {
		t.Fatalf("expected err, got %v", s.Err())
	}
	if s.Selector() != "AXMenuItem" {
		t.Fatalf("expected AXMenuItem, got %s", s.Selector())
	}
}

// --- Count ---

func TestSelection_Count(t *testing.T) {
	cases := []struct {
		name string
		sel  *Selection
		want int
	}{
		{"empty", makeTestSelection(0), 0},
		{"one", makeTestSelection(1), 1},
		{"many", makeTestSelection(10), 10},
		{"error", makeErrSelection(errors.New("err")), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.sel.Count() != c.want {
				t.Fatalf("Count: expected %d, got %d", c.want, c.sel.Count())
			}
		})
	}
}

// --- IsEmpty ---

func TestSelection_IsEmpty(t *testing.T) {
	if !makeTestSelection(0).IsEmpty() {
		t.Fatal("0 elements should be empty")
	}
	if makeTestSelection(1).IsEmpty() {
		t.Fatal("1 element should not be empty")
	}
	if !makeErrSelection(errors.New("err")).IsEmpty() {
		t.Fatal("error selection should be empty")
	}
}

// --- Err ---

func TestSelection_Err_NoError(t *testing.T) {
	s := makeTestSelection(1)
	if s.Err() != nil {
		t.Fatalf("expected nil error, got %v", s.Err())
	}
}

func TestSelection_Err_WithError(t *testing.T) {
	want := errors.New("fail")
	s := makeErrSelection(want)
	if s.Err() != want {
		t.Fatal("expected stored error")
	}
}

// --- Selector ---

func TestSelection_Selector(t *testing.T) {
	s := newSelection(nil, "AXButton:enabled")
	if s.Selector() != "AXButton:enabled" {
		t.Fatalf("expected AXButton:enabled, got %s", s.Selector())
	}
}

// --- First ---

func TestSelection_First(t *testing.T) {
	s := makeTestSelection(3)
	first := s.First()
	if first.Count() != 1 {
		t.Fatalf("First should return 1 element, got %d", first.Count())
	}
	if first.Err() != nil {
		t.Fatalf("First should not error, got %v", first.Err())
	}
}

func TestSelection_First_Empty(t *testing.T) {
	s := makeTestSelection(0)
	first := s.First()
	if first.Count() != 0 {
		t.Fatalf("First of empty should have 0 elements, got %d", first.Count())
	}
	if first.Err() == nil {
		t.Fatal("First of empty should return an error")
	}
	if !errors.Is(first.Err(), ErrNotFound) {
		t.Fatal("First of empty should be ErrNotFound")
	}
}

func TestSelection_First_PropagatesError(t *testing.T) {
	original := errors.New("upstream")
	s := makeErrSelection(original)
	first := s.First()
	if first.Err() != original {
		t.Fatal("First should propagate existing error")
	}
}

// --- Last ---

func TestSelection_Last(t *testing.T) {
	s := makeTestSelection(3)
	last := s.Last()
	if last.Count() != 1 {
		t.Fatalf("Last should return 1 element, got %d", last.Count())
	}
}

func TestSelection_Last_Empty(t *testing.T) {
	s := makeTestSelection(0)
	last := s.Last()
	if last.Count() != 0 {
		t.Fatal("Last of empty should have 0 elements")
	}
	if !errors.Is(last.Err(), ErrNotFound) {
		t.Fatal("Last of empty should be ErrNotFound")
	}
}

func TestSelection_Last_PropagatesError(t *testing.T) {
	original := errors.New("upstream")
	s := makeErrSelection(original)
	last := s.Last()
	if last.Err() != original {
		t.Fatal("Last should propagate existing error")
	}
}

// --- Eq ---

func TestSelection_Eq(t *testing.T) {
	s := makeTestSelection(5)

	eq0 := s.Eq(0)
	if eq0.Count() != 1 {
		t.Fatal("Eq(0) should return 1 element")
	}
	if eq0.Err() != nil {
		t.Fatal("Eq(0) should not error")
	}

	eq4 := s.Eq(4)
	if eq4.Count() != 1 {
		t.Fatal("Eq(4) should return 1 element")
	}
}

func TestSelection_Eq_OutOfBounds(t *testing.T) {
	s := makeTestSelection(3)

	neg := s.Eq(-1)
	if neg.Err() == nil {
		t.Fatal("Eq(-1) should error")
	}

	over := s.Eq(3)
	if over.Err() == nil {
		t.Fatal("Eq(3) on 3-element selection should error")
	}

	over2 := s.Eq(100)
	if over2.Err() == nil {
		t.Fatal("Eq(100) should error")
	}
}

func TestSelection_Eq_PropagatesError(t *testing.T) {
	s := makeErrSelection(errors.New("upstream"))
	eq := s.Eq(0)
	if eq.Err() == nil {
		t.Fatal("Eq should propagate error")
	}
}

// --- Slice ---

func TestSelection_Slice(t *testing.T) {
	s := makeTestSelection(5)

	sl := s.Slice(1, 3)
	if sl.Count() != 2 {
		t.Fatalf("Slice(1,3) should return 2 elements, got %d", sl.Count())
	}
	if sl.Err() != nil {
		t.Fatalf("Slice should not error, got %v", sl.Err())
	}
}

func TestSelection_Slice_Full(t *testing.T) {
	s := makeTestSelection(3)
	sl := s.Slice(0, 3)
	if sl.Count() != 3 {
		t.Fatalf("Slice(0,3) should return all 3 elements, got %d", sl.Count())
	}
}

func TestSelection_Slice_Empty(t *testing.T) {
	s := makeTestSelection(3)
	sl := s.Slice(2, 2)
	if sl.Count() != 0 {
		t.Fatalf("Slice(2,2) should return 0 elements, got %d", sl.Count())
	}
}

func TestSelection_Slice_Clamped(t *testing.T) {
	s := makeTestSelection(3)
	// end > len should be clamped
	sl := s.Slice(1, 100)
	if sl.Count() != 2 {
		t.Fatalf("Slice(1,100) should clamp to 2 elements, got %d", sl.Count())
	}
}

func TestSelection_Slice_InvalidRange(t *testing.T) {
	s := makeTestSelection(3)

	// start < 0
	sl := s.Slice(-1, 2)
	if sl.Err() == nil {
		t.Fatal("Slice(-1,2) should error")
	}

	// start > end
	sl2 := s.Slice(3, 1)
	if sl2.Err() == nil {
		t.Fatal("Slice(3,1) should error")
	}
}

func TestSelection_Slice_PropagatesError(t *testing.T) {
	s := makeErrSelection(errors.New("upstream"))
	sl := s.Slice(0, 1)
	if sl.Err() == nil {
		t.Fatal("Slice should propagate error")
	}
}

// --- Elements ---

func TestSelection_Elements(t *testing.T) {
	s := makeTestSelection(3)
	elems := s.Elements()
	if len(elems) != 3 {
		t.Fatalf("Elements() should return 3, got %d", len(elems))
	}
}

func TestSelection_Elements_Empty(t *testing.T) {
	s := makeTestSelection(0)
	elems := s.Elements()
	if len(elems) != 0 {
		t.Fatalf("Elements() on empty should return 0, got %d", len(elems))
	}
}

func TestSelection_Elements_Error(t *testing.T) {
	s := makeErrSelection(errors.New("err"))
	elems := s.Elements()
	if elems != nil {
		t.Fatal("Elements() on error selection should return nil")
	}
}

// --- Chaining preserves selector ---

func TestSelection_ChainingPreservesSelector(t *testing.T) {
	s := newSelection(make([]*ax.Element, 3), "AXButton")

	if s.First().Selector() != "AXButton" {
		t.Fatal("First should preserve selector")
	}
	if s.Last().Selector() != "AXButton" {
		t.Fatal("Last should preserve selector")
	}
	if s.Eq(1).Selector() != "AXButton" {
		t.Fatal("Eq should preserve selector")
	}
	if s.Slice(0, 2).Selector() != "AXButton" {
		t.Fatal("Slice should preserve selector")
	}
}
