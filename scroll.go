package axquery

import "fmt"

// scroll.go implements scroll-related methods on Selection.
// All scroll methods operate on the FIRST element in the selection
// via the actionable interface, matching the pattern used by action.go.

// ScrollDown scrolls the first element down by n pages via the
// AXScrollDownByPage accessibility action. If n <= 0, no action is taken.
// Returns the same *Selection for chaining. Sets s.err on failure.
func (s *Selection) ScrollDown(n int) *Selection {
	if s.err != nil {
		return s
	}
	if n <= 0 {
		return s
	}
	a := s.firstActionable()
	if a == nil {
		s.err = &NotActionableError{Action: "ScrollDown", Reason: "empty selection or no actionable element"}
		return s
	}
	for i := 0; i < n; i++ {
		if err := a.performAction("AXScrollDownByPage"); err != nil {
			s.err = fmt.Errorf("axquery: ScrollDown failed on page %d: %w", i+1, err)
			return s
		}
	}
	return s
}

// ScrollUp scrolls the first element up by n pages via the
// AXScrollUpByPage accessibility action. If n <= 0, no action is taken.
// Returns the same *Selection for chaining. Sets s.err on failure.
func (s *Selection) ScrollUp(n int) *Selection {
	if s.err != nil {
		return s
	}
	if n <= 0 {
		return s
	}
	a := s.firstActionable()
	if a == nil {
		s.err = &NotActionableError{Action: "ScrollUp", Reason: "empty selection or no actionable element"}
		return s
	}
	for i := 0; i < n; i++ {
		if err := a.performAction("AXScrollUpByPage"); err != nil {
			s.err = fmt.Errorf("axquery: ScrollUp failed on page %d: %w", i+1, err)
			return s
		}
	}
	return s
}

// ScrollIntoView scrolls the nearest scrollable ancestor so that the first
// element in the selection becomes visible, via the AXScrollToVisible action.
// Returns the same *Selection for chaining. Sets s.err on failure.
func (s *Selection) ScrollIntoView() *Selection {
	if s.err != nil {
		return s
	}
	a := s.firstActionable()
	if a == nil {
		s.err = &NotActionableError{Action: "ScrollIntoView", Reason: "empty selection or no actionable element"}
		return s
	}
	if err := a.performAction("AXScrollToVisible"); err != nil {
		s.err = fmt.Errorf("axquery: ScrollIntoView failed: %w", err)
	}
	return s
}
