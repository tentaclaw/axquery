package axquery

import (
	"errors"
	"testing"

	"github.com/tentaclaw/ax"
)

// === mockActionNode: supports action operations for unit testing ===

// mockActionNode extends mockTraversalNode with action capabilities.
// It implements the actionable interface so that action methods can
// be tested without real AX elements.
type mockActionNode struct {
	mockTraversalNode

	// Track which actions were performed
	pressCount    int
	setValueCalls []string
	performCalls  []string

	// Configurable errors
	pressErr    error
	setValueErr error
	performErr  error

	// Own children list so queryChildren returns *mockActionNode (actionable)
	actionChildren []*mockActionNode
}

func (m *mockActionNode) press() error {
	m.pressCount++
	return m.pressErr
}

func (m *mockActionNode) setValue(value string) error {
	m.setValueCalls = append(m.setValueCalls, value)
	return m.setValueErr
}

func (m *mockActionNode) performAction(action string) error {
	m.performCalls = append(m.performCalls, action)
	return m.performErr
}

// queryChildren overrides the embedded mockTraversalNode.queryChildren to
// return *mockActionNode values (which implement actionable), preserving
// action capabilities through BFS/Find.
func (m *mockActionNode) queryChildren() ([]queryNode, error) {
	result := make([]queryNode, len(m.actionChildren))
	for i, c := range m.actionChildren {
		result[i] = c
	}
	return result, nil
}

// addActionChild appends an action-capable child node.
func (m *mockActionNode) addActionChild(child *mockActionNode) {
	child.parent = &m.mockTraversalNode
	m.actionChildren = append(m.actionChildren, child)
}

// newMockActionNode creates a mockActionNode with the given role and optional attrs.
func newMockActionNode(role string, attrs map[string]string) *mockActionNode {
	return &mockActionNode{
		mockTraversalNode: mockTraversalNode{
			role:    role,
			attrs:   attrs,
			enabled: true,
			visible: true,
		},
	}
}

// buildActionTree creates a tree for action tests:
//
//	root (AXWindow)
//	├── btn (AXButton, title="OK", enabled)
//	├── input (AXTextField, title="Name")
//	└── disabled (AXButton, title="Nope", disabled)
func buildActionTree() (root, btn, input, disabledBtn *mockActionNode) {
	root = newMockActionNode("AXWindow", nil)
	btn = newMockActionNode("AXButton", map[string]string{"title": "OK"})
	input = newMockActionNode("AXTextField", map[string]string{"title": "Name"})
	disabledBtn = newMockActionNode("AXButton", map[string]string{"title": "Nope"})
	disabledBtn.enabled = false

	root.addActionChild(btn)
	root.addActionChild(input)
	root.addActionChild(disabledBtn)

	return
}

// === Click tests ===

func TestClick_Success(t *testing.T) {
	btn := newMockActionNode("AXButton", map[string]string{"title": "OK"})
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	result := sel.Click()

	if result != sel {
		t.Fatal("Click should return the same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if btn.pressCount != 1 {
		t.Fatalf("expected 1 press call, got %d", btn.pressCount)
	}
}

func TestClick_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXButton")
	result := sel.Click()

	if result != sel {
		t.Fatal("Click on error selection should return same selection")
	}
	if !errors.Is(result.Err(), errTest) {
		t.Fatalf("expected errTest, got %v", result.Err())
	}
}

func TestClick_EmptySelection(t *testing.T) {
	sel := newSelection([]*ax.Element{}, "AXButton")
	result := sel.Click()

	if result.Err() == nil {
		t.Fatal("Click on empty selection should set error")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

func TestClick_PressError(t *testing.T) {
	btn := newMockActionNode("AXButton", map[string]string{"title": "OK"})
	btn.pressErr = errors.New("press failed")
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	result := sel.Click()

	if result.Err() == nil {
		t.Fatal("expected error from press failure")
	}
	if btn.pressCount != 1 {
		t.Fatalf("expected 1 press call, got %d", btn.pressCount)
	}
}

func TestClick_MultiElement_OnlyFirst(t *testing.T) {
	btn1 := newMockActionNode("AXButton", map[string]string{"title": "A"})
	btn2 := newMockActionNode("AXButton", map[string]string{"title": "B"})
	sel := newSelectionFromNodes([]queryNode{btn1, btn2}, "AXButton")

	sel.Click()

	if btn1.pressCount != 1 {
		t.Fatal("Click should call press on first element")
	}
	if btn2.pressCount != 0 {
		t.Fatal("Click should NOT call press on second element")
	}
}

// === SetValue tests ===

func TestSetValue_Success(t *testing.T) {
	input := newMockActionNode("AXTextField", map[string]string{"title": "Name"})
	sel := newSelectionFromNodes([]queryNode{input}, "AXTextField")

	result := sel.SetValue("hello")

	if result != sel {
		t.Fatal("SetValue should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if len(input.setValueCalls) != 1 || input.setValueCalls[0] != "hello" {
		t.Fatalf("expected SetValue('hello'), got %v", input.setValueCalls)
	}
}

func TestSetValue_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXTextField")
	result := sel.SetValue("hello")

	if result != sel {
		t.Fatal("SetValue on error selection should return same selection")
	}
}

func TestSetValue_EmptySelection(t *testing.T) {
	sel := newSelection([]*ax.Element{}, "AXTextField")
	result := sel.SetValue("hello")

	if result.Err() == nil {
		t.Fatal("SetValue on empty selection should set error")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

func TestSetValue_Error(t *testing.T) {
	input := newMockActionNode("AXTextField", nil)
	input.setValueErr = errors.New("cannot set value")
	sel := newSelectionFromNodes([]queryNode{input}, "AXTextField")

	result := sel.SetValue("hello")

	if result.Err() == nil {
		t.Fatal("expected error from SetValue failure")
	}
}

// === Perform tests ===

func TestPerform_Success(t *testing.T) {
	btn := newMockActionNode("AXButton", map[string]string{"title": "OK"})
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	result := sel.Perform("AXShowMenu")

	if result != sel {
		t.Fatal("Perform should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if len(btn.performCalls) != 1 || btn.performCalls[0] != "AXShowMenu" {
		t.Fatalf("expected Perform('AXShowMenu'), got %v", btn.performCalls)
	}
}

func TestPerform_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXButton")
	result := sel.Perform("AXPress")

	if result != sel {
		t.Fatal("Perform on error selection should return same selection")
	}
}

func TestPerform_EmptySelection(t *testing.T) {
	sel := newSelection([]*ax.Element{}, "AXButton")
	result := sel.Perform("AXPress")

	if result.Err() == nil {
		t.Fatal("Perform on empty selection should set error")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

func TestPerform_ActionError(t *testing.T) {
	btn := newMockActionNode("AXButton", nil)
	btn.performErr = errors.New("action unsupported")
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	result := sel.Perform("AXInvalidAction")

	if result.Err() == nil {
		t.Fatal("expected error from perform failure")
	}
}

// === Focus tests ===

func TestFocus_Success(t *testing.T) {
	btn := newMockActionNode("AXButton", map[string]string{"title": "OK"})
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	result := sel.Focus()

	if result != sel {
		t.Fatal("Focus should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	// Focus delegates to performAction("AXRaise")
	if len(btn.performCalls) != 1 || btn.performCalls[0] != "AXRaise" {
		t.Fatalf("expected Focus to call performAction('AXRaise'), got %v", btn.performCalls)
	}
}

func TestFocus_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXButton")
	result := sel.Focus()

	if result != sel {
		t.Fatal("Focus on error selection should return same selection")
	}
}

func TestFocus_EmptySelection(t *testing.T) {
	sel := newSelection([]*ax.Element{}, "AXButton")
	result := sel.Focus()

	if result.Err() == nil {
		t.Fatal("Focus on empty selection should set error")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

// === TypeText tests ===

// TypeText needs a special approach: it calls a package-level function (ax.TypeText)
// which requires CGo. For pure unit testing we inject a typeTextFn.

func TestTypeText_Success(t *testing.T) {
	input := newMockActionNode("AXTextField", map[string]string{"title": "Name"})
	sel := newSelectionFromNodes([]queryNode{input}, "AXTextField")

	var typed string
	origTypeText := typeTextFn
	typeTextFn = func(text string) error {
		typed = text
		return nil
	}
	defer func() { typeTextFn = origTypeText }()

	result := sel.TypeText("hello world")

	if result != sel {
		t.Fatal("TypeText should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	// TypeText should first focus (AXRaise), then type
	if len(input.performCalls) != 1 || input.performCalls[0] != "AXRaise" {
		t.Fatalf("TypeText should call Focus first, got %v", input.performCalls)
	}
	if typed != "hello world" {
		t.Fatalf("expected typed 'hello world', got %q", typed)
	}
}

func TestTypeText_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXTextField")
	result := sel.TypeText("hello")

	if result != sel {
		t.Fatal("TypeText on error selection should return same selection")
	}
}

func TestTypeText_EmptySelection(t *testing.T) {
	sel := newSelection([]*ax.Element{}, "AXTextField")
	result := sel.TypeText("hello")

	if result.Err() == nil {
		t.Fatal("TypeText on empty selection should set error")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

func TestTypeText_FocusError(t *testing.T) {
	input := newMockActionNode("AXTextField", nil)
	input.performErr = errors.New("cannot raise")
	sel := newSelectionFromNodes([]queryNode{input}, "AXTextField")

	origTypeText := typeTextFn
	typeTextFn = func(text string) error { return nil }
	defer func() { typeTextFn = origTypeText }()

	result := sel.TypeText("hello")

	// Should still set error from focus failure
	if result.Err() == nil {
		t.Fatal("expected error from focus failure")
	}
}

func TestTypeText_TypeError(t *testing.T) {
	input := newMockActionNode("AXTextField", nil)
	sel := newSelectionFromNodes([]queryNode{input}, "AXTextField")

	origTypeText := typeTextFn
	typeTextFn = func(text string) error {
		return errors.New("type failed")
	}
	defer func() { typeTextFn = origTypeText }()

	result := sel.TypeText("hello")

	if result.Err() == nil {
		t.Fatal("expected error from type failure")
	}
}

// === Press tests ===

func TestPress_Success(t *testing.T) {
	btn := newMockActionNode("AXButton", map[string]string{"title": "OK"})
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	var pressedKey string
	var pressedMods []ax.Modifier
	origKeyPress := keyPressFn
	keyPressFn = func(key string, mods ...ax.Modifier) error {
		pressedKey = key
		pressedMods = mods
		return nil
	}
	defer func() { keyPressFn = origKeyPress }()

	result := sel.Press("return", "command", "shift")

	if result != sel {
		t.Fatal("Press should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if pressedKey != "return" {
		t.Fatalf("expected key 'return', got %q", pressedKey)
	}
	if len(pressedMods) != 2 {
		t.Fatalf("expected 2 modifiers, got %d", len(pressedMods))
	}
}

func TestPress_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXButton")
	result := sel.Press("return")

	if result != sel {
		t.Fatal("Press on error selection should return same selection")
	}
}

func TestPress_EmptySelection(t *testing.T) {
	sel := newSelection([]*ax.Element{}, "AXButton")

	origKeyPress := keyPressFn
	keyPressFn = func(key string, mods ...ax.Modifier) error { return nil }
	defer func() { keyPressFn = origKeyPress }()

	result := sel.Press("return")

	if result.Err() == nil {
		t.Fatal("Press on empty selection should set error")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

func TestPress_InvalidModifier(t *testing.T) {
	btn := newMockActionNode("AXButton", nil)
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origKeyPress := keyPressFn
	keyPressFn = func(key string, mods ...ax.Modifier) error { return nil }
	defer func() { keyPressFn = origKeyPress }()

	result := sel.Press("return", "invalid_modifier")

	if result.Err() == nil {
		t.Fatal("expected error for invalid modifier")
	}
	if !errors.Is(result.Err(), ErrNotActionable) {
		t.Fatalf("expected ErrNotActionable, got %v", result.Err())
	}
}

func TestPress_KeyPressError(t *testing.T) {
	btn := newMockActionNode("AXButton", nil)
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origKeyPress := keyPressFn
	keyPressFn = func(key string, mods ...ax.Modifier) error {
		return errors.New("key press failed")
	}
	defer func() { keyPressFn = origKeyPress }()

	result := sel.Press("return")

	if result.Err() == nil {
		t.Fatal("expected error from key press failure")
	}
}

func TestPress_ModifierMapping(t *testing.T) {
	btn := newMockActionNode("AXButton", nil)
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	var pressedMods []ax.Modifier
	origKeyPress := keyPressFn
	keyPressFn = func(key string, mods ...ax.Modifier) error {
		pressedMods = mods
		return nil
	}
	defer func() { keyPressFn = origKeyPress }()

	cases := []struct {
		modStr   string
		expected ax.Modifier
	}{
		{"command", ax.ModCommand},
		{"cmd", ax.ModCommand},
		{"shift", ax.ModShift},
		{"option", ax.ModOption},
		{"alt", ax.ModOption},
		{"control", ax.ModControl},
		{"ctrl", ax.ModControl},
	}
	for _, c := range cases {
		pressedMods = nil
		sel.err = nil // reset error for each case
		result := sel.Press("a", c.modStr)
		if result.Err() != nil {
			t.Fatalf("Press(a, %q): unexpected error: %v", c.modStr, result.Err())
		}
		if len(pressedMods) != 1 || pressedMods[0] != c.expected {
			t.Errorf("Press(a, %q): expected modifier %v, got %v", c.modStr, c.expected, pressedMods)
		}
	}
}

// === Chaining tests ===

func TestAction_Chaining(t *testing.T) {
	btn := newMockActionNode("AXButton", map[string]string{"title": "OK"})
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origTypeText := typeTextFn
	typeTextFn = func(text string) error { return nil }
	defer func() { typeTextFn = origTypeText }()

	origKeyPress := keyPressFn
	keyPressFn = func(key string, mods ...ax.Modifier) error { return nil }
	defer func() { keyPressFn = origKeyPress }()

	// Chain multiple actions
	result := sel.Click().SetValue("test").Focus()

	if result.Err() != nil {
		t.Fatalf("unexpected error in chain: %v", result.Err())
	}
	if btn.pressCount != 1 {
		t.Fatal("Click not called in chain")
	}
	if len(btn.setValueCalls) != 1 || btn.setValueCalls[0] != "test" {
		t.Fatal("SetValue not called in chain")
	}
	// Focus calls performAction("AXRaise")
	if len(btn.performCalls) != 1 || btn.performCalls[0] != "AXRaise" {
		t.Fatalf("Focus not called in chain, got %v", btn.performCalls)
	}
}

func TestAction_ChainStopsOnError(t *testing.T) {
	btn := newMockActionNode("AXButton", map[string]string{"title": "OK"})
	btn.pressErr = errors.New("click failed")
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	// Click fails, subsequent SetValue should not execute
	result := sel.Click().SetValue("test")

	if result.Err() == nil {
		t.Fatal("expected error to propagate")
	}
	if len(btn.setValueCalls) != 0 {
		t.Fatal("SetValue should not execute after Click failure")
	}
}

// === Integration with Find/Filter ===

func TestAction_AfterFind(t *testing.T) {
	root := newMockActionNode("AXWindow", nil)
	btn := newMockActionNode("AXButton", map[string]string{"title": "OK"})
	root.addActionChild(btn)

	sel := newSelectionFromNodes([]queryNode{root}, "AXWindow")
	result := sel.Find("AXButton").Click()

	// Find should locate btn, Click should invoke press on it
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if btn.pressCount != 1 {
		t.Fatal("expected Click to call press on found button")
	}
}
