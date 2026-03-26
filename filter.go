package axquery

import (
	"strings"

	"github.com/tentaclaw/axquery/selector"
)

// Filter returns a new Selection containing only the elements from the current
// selection that match the given CSS-like selector.
func (s *Selection) Filter(sel string) *Selection {
	if s.err != nil {
		return s
	}
	matcher, err := selector.Compile(sel)
	if err != nil {
		return newSelectionError(
			&InvalidSelectorError{Selector: sel, Reason: err.Error()},
			sel,
		)
	}
	return s.FilterMatcher(matcher)
}

// FilterMatcher returns a new Selection containing only the elements that
// match the pre-compiled Matcher.
func (s *Selection) FilterMatcher(m selector.Matcher) *Selection {
	if s.err != nil {
		return s
	}
	nodes := s.getNodes()
	if len(nodes) == 0 {
		return newSelectionFromNodes(nil, s.selector)
	}
	var kept []queryNode
	for _, n := range nodes {
		if m.MatchSimple(n) {
			kept = append(kept, n)
		}
	}
	return newSelectionFromNodes(kept, s.selector)
}

// FilterFunction returns a new Selection containing only the elements for
// which fn returns true. The callback receives the 0-based index and a
// single-element Selection wrapping that element.
func (s *Selection) FilterFunction(fn func(int, *Selection) bool) *Selection {
	if s.err != nil {
		return s
	}
	nodes := s.getNodes()
	if len(nodes) == 0 {
		return newSelectionFromNodes(nil, s.selector)
	}
	var kept []queryNode
	for i, n := range nodes {
		single := newSelectionFromNodes([]queryNode{n}, s.selector)
		if fn(i, single) {
			kept = append(kept, n)
		}
	}
	return newSelectionFromNodes(kept, s.selector)
}

// Not returns a new Selection containing the elements from the current
// selection that do NOT match the given selector. It is the inverse of Filter.
func (s *Selection) Not(sel string) *Selection {
	if s.err != nil {
		return s
	}
	matcher, err := selector.Compile(sel)
	if err != nil {
		return newSelectionError(
			&InvalidSelectorError{Selector: sel, Reason: err.Error()},
			sel,
		)
	}
	return s.NotMatcher(matcher)
}

// NotMatcher returns a new Selection excluding elements that match the
// pre-compiled Matcher.
func (s *Selection) NotMatcher(m selector.Matcher) *Selection {
	if s.err != nil {
		return s
	}
	nodes := s.getNodes()
	if len(nodes) == 0 {
		return newSelectionFromNodes(nil, s.selector)
	}
	var kept []queryNode
	for _, n := range nodes {
		if !m.MatchSimple(n) {
			kept = append(kept, n)
		}
	}
	return newSelectionFromNodes(kept, s.selector)
}

// Has returns a new Selection containing only the elements that have at least
// one descendant matching the given selector. The element itself is not tested.
func (s *Selection) Has(sel string) *Selection {
	if s.err != nil {
		return s
	}
	matcher, err := selector.Compile(sel)
	if err != nil {
		return newSelectionError(
			&InvalidSelectorError{Selector: sel, Reason: err.Error()},
			sel,
		)
	}
	nodes := s.getNodes()
	if len(nodes) == 0 {
		return newSelectionFromNodes(nil, s.selector)
	}
	var kept []queryNode
	for _, n := range nodes {
		if hasMatchingDescendant(n, matcher) {
			kept = append(kept, n)
		}
	}
	return newSelectionFromNodes(kept, s.selector)
}

// hasMatchingDescendant returns true if any descendant of node matches the
// given matcher. The node itself is not tested (matching goquery's Has semantics).
func hasMatchingDescendant(node queryNode, matcher selector.Matcher) bool {
	children, err := node.queryChildren()
	if err != nil {
		return false
	}
	for _, child := range children {
		if matcher.MatchSimple(child) {
			return true
		}
		if hasMatchingDescendant(child, matcher) {
			return true
		}
	}
	return false
}

// Is reports whether any element in the selection matches the given selector.
func (s *Selection) Is(sel string) bool {
	if s.err != nil {
		return false
	}
	matcher, err := selector.Compile(sel)
	if err != nil {
		return false
	}
	nodes := s.getNodes()
	for _, n := range nodes {
		if matcher.MatchSimple(n) {
			return true
		}
	}
	return false
}

// Contains returns a new Selection containing only the elements whose "title"
// attribute contains the given text (case-sensitive substring match).
// This is a convenience method similar to jQuery's :contains pseudo-selector.
func (s *Selection) Contains(text string) *Selection {
	if s.err != nil {
		return s
	}
	nodes := s.getNodes()
	if len(nodes) == 0 {
		return newSelectionFromNodes(nil, s.selector)
	}
	var kept []queryNode
	for _, n := range nodes {
		title := n.GetAttr("title")
		if strings.Contains(title, text) {
			kept = append(kept, n)
		}
	}
	return newSelectionFromNodes(kept, s.selector)
}
