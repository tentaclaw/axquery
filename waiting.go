package axquery

import "time"

// DefaultPollInterval is the default duration between condition checks in
// Wait methods. 200ms provides responsive polling without excessive CPU usage.
const DefaultPollInterval = 200 * time.Millisecond

// Package-level function variables for time operations, enabling pure unit
// tests without real time.Sleep delays.
var (
	sleepFn = time.Sleep
	nowFn   = time.Now
)

// WaitUntil polls the given condition function until it returns true or the
// timeout elapses. The condition receives the Selection on each poll, allowing
// callers to check any property (visibility, enabled state, text content, etc.).
//
// Returns the same *Selection for chaining. On timeout, sets s.err to a
// TimeoutError wrapping ErrTimeout.
//
// If the selection already carries an error, returns immediately.
func (s *Selection) WaitUntil(fn func(*Selection) bool, timeout time.Duration) *Selection {
	if s.err != nil {
		return s
	}

	deadline := nowFn().Add(timeout)

	for {
		if fn(s) {
			return s
		}
		if nowFn().After(deadline) {
			s.err = &TimeoutError{
				Selector: s.selector,
				Duration: timeout.String(),
			}
			return s
		}
		sleepFn(DefaultPollInterval)
	}
}

// WaitVisible polls until the first element in the selection reports as visible
// (not hidden). This is a convenience wrapper around WaitUntil.
//
// On real AX elements, IsHidden() is queried live each poll — no re-query needed.
// Returns the same *Selection for chaining. On timeout, sets s.err.
func (s *Selection) WaitVisible(timeout time.Duration) *Selection {
	return s.WaitUntil(func(sel *Selection) bool {
		return sel.IsVisible()
	}, timeout)
}

// WaitEnabled polls until the first element in the selection reports as enabled.
// This is a convenience wrapper around WaitUntil.
//
// On real AX elements, IsEnabled() is queried live each poll.
// Returns the same *Selection for chaining. On timeout, sets s.err.
func (s *Selection) WaitEnabled(timeout time.Duration) *Selection {
	return s.WaitUntil(func(sel *Selection) bool {
		return sel.IsEnabled()
	}, timeout)
}

// WaitGone polls until the selection appears to be "gone" — either the selection
// is empty, has an existing error, or the first element's role returns empty
// (indicating the AX element has been destroyed).
//
// For error selections, WaitGone returns immediately (element is already
// unreachable). Returns the same *Selection for chaining. On timeout, sets s.err.
func (s *Selection) WaitGone(timeout time.Duration) *Selection {
	if s.err != nil {
		return s // already errored = effectively gone
	}
	return s.WaitUntil(func(sel *Selection) bool {
		if sel.IsEmpty() {
			return true
		}
		// If the first node's role is empty, the element is likely destroyed
		node := sel.firstNode()
		if node == nil {
			return true
		}
		return node.GetRole() == ""
	}, timeout)
}
