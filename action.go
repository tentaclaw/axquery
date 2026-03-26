package axquery

import (
	"fmt"
	"strings"

	"github.com/tentaclaw/ax"
)

// actionable is the internal interface for nodes that support interaction
// actions. Mock nodes in tests implement this directly; real *ax.Element
// nodes are accessed through their element() method instead.
type actionable interface {
	press() error
	setValue(value string) error
	performAction(action string) error
}

// Package-level function variables for keyboard operations, enabling
// pure unit tests without CGo. Defaults point to the real ax functions.
var (
	typeTextFn = ax.TypeText
	keyPressFn = ax.KeyPress
)

// firstActionable returns the actionable interface for the first node in the
// selection. It checks:
//  1. Does the first queryNode implement actionable? (mock path)
//  2. Does node.element() return a non-nil *ax.Element? (real path via elementActionAdapter)
//
// Returns nil if neither applies.
func (s *Selection) firstActionable() actionable {
	nodes := s.getNodes()
	if len(nodes) == 0 {
		return nil
	}
	first := nodes[0]
	if a, ok := first.(actionable); ok {
		return a
	}
	if el := first.element(); el != nil {
		return &elementActionAdapter{el: el}
	}
	return nil
}

// elementActionAdapter wraps a real *ax.Element to satisfy the actionable interface.
type elementActionAdapter struct {
	el *ax.Element
}

func (a *elementActionAdapter) press() error {
	return a.el.Press()
}

func (a *elementActionAdapter) setValue(value string) error {
	return a.el.SetValue(value)
}

func (a *elementActionAdapter) performAction(action string) error {
	return a.el.PerformAction(action)
}

// Click performs AXPress on the first element in the selection.
// Returns the same Selection for chaining. Sets s.err on failure.
func (s *Selection) Click() *Selection {
	if s.err != nil {
		return s
	}
	a := s.firstActionable()
	if a == nil {
		s.err = &NotActionableError{Action: "Click", Reason: "empty selection or no actionable element"}
		return s
	}
	if err := a.press(); err != nil {
		s.err = fmt.Errorf("axquery: Click failed: %w", err)
	}
	return s
}

// SetValue sets the value of the first element in the selection.
// Returns the same Selection for chaining. Sets s.err on failure.
func (s *Selection) SetValue(value string) *Selection {
	if s.err != nil {
		return s
	}
	a := s.firstActionable()
	if a == nil {
		s.err = &NotActionableError{Action: "SetValue", Reason: "empty selection or no actionable element"}
		return s
	}
	if err := a.setValue(value); err != nil {
		s.err = fmt.Errorf("axquery: SetValue failed: %w", err)
	}
	return s
}

// Perform executes an arbitrary accessibility action on the first element.
// Returns the same Selection for chaining. Sets s.err on failure.
func (s *Selection) Perform(action string) *Selection {
	if s.err != nil {
		return s
	}
	a := s.firstActionable()
	if a == nil {
		s.err = &NotActionableError{Action: action, Reason: "empty selection or no actionable element"}
		return s
	}
	if err := a.performAction(action); err != nil {
		s.err = fmt.Errorf("axquery: Perform(%q) failed: %w", action, err)
	}
	return s
}

// Focus raises/focuses the first element by performing AXRaise.
// Returns the same Selection for chaining. Sets s.err on failure.
func (s *Selection) Focus() *Selection {
	if s.err != nil {
		return s
	}
	a := s.firstActionable()
	if a == nil {
		s.err = &NotActionableError{Action: "Focus", Reason: "empty selection or no actionable element"}
		return s
	}
	if err := a.performAction("AXRaise"); err != nil {
		s.err = fmt.Errorf("axquery: Focus failed: %w", err)
	}
	return s
}

// TypeText focuses the first element then types the given text using
// synthetic keyboard events. Returns the same Selection for chaining.
func (s *Selection) TypeText(text string) *Selection {
	if s.err != nil {
		return s
	}
	a := s.firstActionable()
	if a == nil {
		s.err = &NotActionableError{Action: "TypeText", Reason: "empty selection or no actionable element"}
		return s
	}
	// Focus first
	if err := a.performAction("AXRaise"); err != nil {
		s.err = fmt.Errorf("axquery: TypeText focus failed: %w", err)
		return s
	}
	if err := typeTextFn(text); err != nil {
		s.err = fmt.Errorf("axquery: TypeText failed: %w", err)
	}
	return s
}

// modifierMap maps human-readable modifier strings to ax.Modifier values.
var modifierMap = map[string]ax.Modifier{
	"command": ax.ModCommand,
	"cmd":     ax.ModCommand,
	"shift":   ax.ModShift,
	"option":  ax.ModOption,
	"alt":     ax.ModOption,
	"control": ax.ModControl,
	"ctrl":    ax.ModControl,
}

// Press sends a key press with optional modifier strings to the first element.
// Modifier strings: "command"/"cmd", "shift", "option"/"alt", "control"/"ctrl".
// Focuses the element first, then sends the key event.
// Returns the same Selection for chaining.
func (s *Selection) Press(key string, modifiers ...string) *Selection {
	if s.err != nil {
		return s
	}
	a := s.firstActionable()
	if a == nil {
		s.err = &NotActionableError{Action: "Press", Reason: "empty selection or no actionable element"}
		return s
	}

	// Parse modifier strings
	var mods []ax.Modifier
	for _, ms := range modifiers {
		m, ok := modifierMap[strings.ToLower(ms)]
		if !ok {
			s.err = &NotActionableError{
				Action: "Press",
				Reason: fmt.Sprintf("unknown modifier %q", ms),
			}
			return s
		}
		mods = append(mods, m)
	}

	// Focus first
	if err := a.performAction("AXRaise"); err != nil {
		s.err = fmt.Errorf("axquery: Press focus failed: %w", err)
		return s
	}

	if err := keyPressFn(key, mods...); err != nil {
		s.err = fmt.Errorf("axquery: Press failed: %w", err)
	}
	return s
}
