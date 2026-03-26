package axquery

import "iter"

// iteration.go implements iteration/callback methods on Selection.
// These methods follow goquery/jQuery semantics: Each iterates all elements,
// EachWithBreak allows early termination, Map collects transformed values,
// and EachIter provides a Go 1.23+ range iterator.

// Each calls fn for every element in the selection, passing the 0-based index
// and a single-element Selection. Returns the original selection for chaining.
// No-op for empty or error selections.
func (s *Selection) Each(fn func(int, *Selection)) *Selection {
	if s.err != nil || len(s.elems) == 0 {
		return s
	}
	nodes := s.getNodes()
	for i, node := range nodes {
		single := newSelectionFromNodes([]queryNode{node}, s.selector)
		fn(i, single)
	}
	return s
}

// EachWithBreak calls fn for every element in the selection. If fn returns
// false, iteration stops immediately. Returns the original selection for chaining.
// No-op for empty or error selections.
func (s *Selection) EachWithBreak(fn func(int, *Selection) bool) *Selection {
	if s.err != nil || len(s.elems) == 0 {
		return s
	}
	nodes := s.getNodes()
	for i, node := range nodes {
		single := newSelectionFromNodes([]queryNode{node}, s.selector)
		if !fn(i, single) {
			break
		}
	}
	return s
}

// Map calls fn for every element in the selection and collects the returned
// strings into a slice. Returns nil for empty or error selections.
func (s *Selection) Map(fn func(int, *Selection) string) []string {
	if s.err != nil || len(s.elems) == 0 {
		return nil
	}
	nodes := s.getNodes()
	result := make([]string, len(nodes))
	for i, node := range nodes {
		single := newSelectionFromNodes([]queryNode{node}, s.selector)
		result[i] = fn(i, single)
	}
	return result
}

// EachIter returns a Go 1.23+ range iterator over the selection elements.
// Each iteration yields the 0-based index and a single-element Selection.
// Empty or error selections yield no values.
func (s *Selection) EachIter() iter.Seq2[int, *Selection] {
	return func(yield func(int, *Selection) bool) {
		if s.err != nil || len(s.elems) == 0 {
			return
		}
		nodes := s.getNodes()
		for i, node := range nodes {
			single := newSelectionFromNodes([]queryNode{node}, s.selector)
			if !yield(i, single) {
				return
			}
		}
	}
}
