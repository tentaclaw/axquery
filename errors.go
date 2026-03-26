package axquery

import (
	"errors"
	"fmt"
)

// Sentinel errors for use with errors.Is().
var (
	ErrNotFound        = errors.New("not found")
	ErrTimeout         = errors.New("timeout")
	ErrAmbiguous       = errors.New("ambiguous")
	ErrInvalidSelector = errors.New("invalid selector")
	ErrNotActionable   = errors.New("not actionable")
)

// NotFoundError indicates that a selector matched zero elements.
type NotFoundError struct {
	Selector string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("axquery: no elements matching %q", e.Selector)
}

func (e *NotFoundError) Unwrap() error {
	return ErrNotFound
}

// TimeoutError indicates that a wait/poll operation exceeded its deadline.
type TimeoutError struct {
	Selector string
	Duration string // human-readable duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("axquery: timeout after %s waiting for %q", e.Duration, e.Selector)
}

func (e *TimeoutError) Unwrap() error {
	return ErrTimeout
}

// AmbiguousError indicates that a selector expecting a single element matched multiple.
type AmbiguousError struct {
	Selector string
	Count    int
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("axquery: selector %q matched %d elements, expected 1", e.Selector, e.Count)
}

func (e *AmbiguousError) Unwrap() error {
	return ErrAmbiguous
}

// InvalidSelectorError indicates a malformed selector string.
type InvalidSelectorError struct {
	Selector string
	Reason   string
}

func (e *InvalidSelectorError) Error() string {
	return fmt.Sprintf("axquery: invalid selector %q: %s", e.Selector, e.Reason)
}

func (e *InvalidSelectorError) Unwrap() error {
	return ErrInvalidSelector
}

// NotActionableError indicates that an action cannot be performed on an element.
type NotActionableError struct {
	Action string
	Reason string
}

func (e *NotActionableError) Error() string {
	return fmt.Sprintf("axquery: cannot perform %q: %s", e.Action, e.Reason)
}

func (e *NotActionableError) Unwrap() error {
	return ErrNotActionable
}
