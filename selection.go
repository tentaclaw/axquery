package axquery

import (
	"fmt"

	"github.com/tentaclaw/ax"
)

// Selection holds a set of AX elements matched by a selector, supporting
// chainable narrowing operations (First, Last, Eq, Slice).
// A Selection may carry an error instead of elements; all chaining methods
// propagate that error without panicking.
type Selection struct {
	elems    []*ax.Element
	nodes    []queryNode // internal: parallel to elems for traversal; may be nil
	err      error
	selector string
}

// newSelection creates a Selection holding the given elements.
func newSelection(elems []*ax.Element, selector string) *Selection {
	if elems == nil {
		elems = []*ax.Element{}
	}
	return &Selection{
		elems:    elems,
		selector: selector,
	}
}

// newSelectionError creates a Selection that carries an error and no elements.
func newSelectionError(err error, selector string) *Selection {
	return &Selection{
		err:      err,
		selector: selector,
	}
}

// Count returns the number of elements in the selection.
func (s *Selection) Count() int {
	return len(s.elems)
}

// IsEmpty returns true if the selection contains no elements.
func (s *Selection) IsEmpty() bool {
	return len(s.elems) == 0
}

// Err returns the error associated with this selection, or nil.
func (s *Selection) Err() error {
	return s.err
}

// Selector returns the selector string that produced this selection.
func (s *Selection) Selector() string {
	return s.selector
}

// Elements returns the underlying element slice, or nil if the selection
// carries an error.
func (s *Selection) Elements() []*ax.Element {
	if s.err != nil {
		return nil
	}
	return s.elems
}

// First returns a new Selection containing only the first element.
// If the selection is empty, returns an error Selection with ErrNotFound.
// If the selection already carries an error, that error is propagated.
func (s *Selection) First() *Selection {
	if s.err != nil {
		return s
	}
	if len(s.elems) == 0 {
		return &Selection{
			err:      &NotFoundError{Selector: s.selector},
			selector: s.selector,
		}
	}
	result := &Selection{
		elems:    s.elems[:1],
		selector: s.selector,
	}
	if s.nodes != nil {
		result.nodes = s.nodes[:1]
	}
	return result
}

// Last returns a new Selection containing only the last element.
// If the selection is empty, returns an error Selection with ErrNotFound.
// If the selection already carries an error, that error is propagated.
func (s *Selection) Last() *Selection {
	if s.err != nil {
		return s
	}
	if len(s.elems) == 0 {
		return &Selection{
			err:      &NotFoundError{Selector: s.selector},
			selector: s.selector,
		}
	}
	n := len(s.elems)
	result := &Selection{
		elems:    s.elems[n-1:],
		selector: s.selector,
	}
	if s.nodes != nil {
		result.nodes = s.nodes[n-1:]
	}
	return result
}

// Eq returns a new Selection containing only the element at index i (0-based).
// If the index is out of bounds, returns an error Selection.
// If the selection already carries an error, that error is propagated.
func (s *Selection) Eq(i int) *Selection {
	if s.err != nil {
		return s
	}
	if i < 0 || i >= len(s.elems) {
		return &Selection{
			err:      fmt.Errorf("axquery: index %d out of range [0, %d)", i, len(s.elems)),
			selector: s.selector,
		}
	}
	result := &Selection{
		elems:    s.elems[i : i+1],
		selector: s.selector,
	}
	if s.nodes != nil {
		result.nodes = s.nodes[i : i+1]
	}
	return result
}

// Slice returns a new Selection containing elements in [start, end).
// end is clamped to the selection length. start must be >= 0 and <= end.
// If the selection already carries an error, that error is propagated.
func (s *Selection) Slice(start, end int) *Selection {
	if s.err != nil {
		return s
	}
	if start < 0 {
		return &Selection{
			err:      fmt.Errorf("axquery: slice start %d must be >= 0", start),
			selector: s.selector,
		}
	}
	if end > len(s.elems) {
		end = len(s.elems)
	}
	if start > end {
		return &Selection{
			err:      fmt.Errorf("axquery: slice start %d > end %d", start, end),
			selector: s.selector,
		}
	}
	result := &Selection{
		elems:    s.elems[start:end],
		selector: s.selector,
	}
	if s.nodes != nil {
		result.nodes = s.nodes[start:end]
	}
	return result
}
