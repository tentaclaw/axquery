package axquery

import (
	"github.com/tentaclaw/ax"
	"github.com/tentaclaw/axquery/selector"
)

// traversableNode extends queryNode with parent access for bidirectional
// tree traversal. Nodes that support parent-based operations (Parent, Siblings,
// Next, Prev, Closest, Parents) implement this interface.
type traversableNode interface {
	queryNode
	queryParent() (queryNode, error)
}

// === Find ===

// findInSubtrees searches the subtrees of each root for nodes matching the
// compiled matcher. Root nodes themselves are excluded from results (matching
// goquery's Find semantics). Results are deduplicated by pointer identity.
func findInSubtrees(roots []queryNode, matcher selector.Matcher) []queryNode {
	if len(roots) == 0 {
		return nil
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, root := range roots {
		children, err := root.queryChildren()
		if err != nil {
			continue
		}
		for _, child := range children {
			collectMatches(child, matcher, seen, &results)
		}
	}

	return results
}

// collectMatches recursively collects matching nodes via DFS, deduplicating
// by pointer identity.
func collectMatches(node queryNode, matcher selector.Matcher, seen map[queryNode]bool, results *[]queryNode) {
	if seen[node] {
		return
	}
	if matcher.MatchSimple(node) {
		seen[node] = true
		*results = append(*results, node)
	}
	children, err := node.queryChildren()
	if err != nil {
		return
	}
	for _, child := range children {
		collectMatches(child, matcher, seen, results)
	}
}

// === Children ===

// getChildren returns the direct children of each root node, deduplicated.
func getChildren(roots []queryNode) []queryNode {
	if len(roots) == 0 {
		return nil
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, root := range roots {
		children, err := root.queryChildren()
		if err != nil {
			continue
		}
		for _, child := range children {
			if !seen[child] {
				seen[child] = true
				results = append(results, child)
			}
		}
	}

	return results
}

// getChildrenFiltered returns direct children of each root that match the selector.
func getChildrenFiltered(roots []queryNode, matcher selector.Matcher) []queryNode {
	if len(roots) == 0 {
		return nil
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, root := range roots {
		children, err := root.queryChildren()
		if err != nil {
			continue
		}
		for _, child := range children {
			if !seen[child] && matcher.MatchSimple(child) {
				seen[child] = true
				results = append(results, child)
			}
		}
	}

	return results
}

// === Parent ===

// getParents returns the immediate parent of each node, deduplicated.
// Nodes without a parent (roots) are skipped.
func getParents(nodes []traversableNode) []queryNode {
	if len(nodes) == 0 {
		return nil
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, node := range nodes {
		parent, err := node.queryParent()
		if err != nil || parent == nil {
			continue
		}
		if !seen[parent] {
			seen[parent] = true
			results = append(results, parent)
		}
	}

	return results
}

// getParentsFiltered returns the immediate parent of each node that matches the selector.
func getParentsFiltered(nodes []traversableNode, matcher selector.Matcher) []queryNode {
	if len(nodes) == 0 {
		return nil
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, node := range nodes {
		parent, err := node.queryParent()
		if err != nil || parent == nil {
			continue
		}
		if !seen[parent] && matcher.MatchSimple(parent) {
			seen[parent] = true
			results = append(results, parent)
		}
	}

	return results
}

// === Ancestors (Parents/ParentsUntil) ===

// getAncestors walks up from each node collecting all ancestor nodes.
// If untilMatcher is non-nil, the walk stops when an ancestor matches the
// until selector (that ancestor is excluded). Results are deduplicated.
func getAncestors(nodes []traversableNode, untilMatcher selector.Matcher) []queryNode {
	if len(nodes) == 0 {
		return nil
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, node := range nodes {
		cur := queryNode(node)
		for {
			tn, ok := cur.(traversableNode)
			if !ok {
				break
			}
			parent, err := tn.queryParent()
			if err != nil || parent == nil {
				break
			}
			// Stop if parent matches the until selector
			if untilMatcher != nil && untilMatcher.MatchSimple(parent) {
				break
			}
			if !seen[parent] {
				seen[parent] = true
				results = append(results, parent)
			}
			cur = parent
		}
	}

	return results
}

// === Closest ===

// getClosest finds the closest ancestor (or self) matching the selector for each node.
// Walks from self upward. Deduplicated.
func getClosest(nodes []traversableNode, matcher selector.Matcher) []queryNode {
	if len(nodes) == 0 {
		return nil
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, node := range nodes {
		cur := queryNode(node)
		for cur != nil {
			if matcher.MatchSimple(cur) {
				if !seen[cur] {
					seen[cur] = true
					results = append(results, cur)
				}
				break
			}
			tn, ok := cur.(traversableNode)
			if !ok {
				break
			}
			parent, err := tn.queryParent()
			if err != nil {
				break
			}
			cur = parent
		}
	}

	return results
}

// === Siblings ===

// getSiblings returns all siblings of each node (excluding the node itself).
// A sibling is another child of the same parent. Deduplicated.
func getSiblings(nodes []traversableNode) []queryNode {
	if len(nodes) == 0 {
		return nil
	}

	// Track source nodes to exclude them from results
	sourceSet := make(map[queryNode]bool, len(nodes))
	for _, n := range nodes {
		sourceSet[n] = true
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, node := range nodes {
		parent, err := node.queryParent()
		if err != nil || parent == nil {
			continue
		}
		children, err := parent.queryChildren()
		if err != nil {
			continue
		}
		for _, child := range children {
			if sourceSet[child] {
				continue
			}
			if !seen[child] {
				seen[child] = true
				results = append(results, child)
			}
		}
	}

	return results
}

// === Next / Prev ===

// getNextSiblings returns the immediately following sibling of each node. Deduplicated.
func getNextSiblings(nodes []traversableNode) []queryNode {
	if len(nodes) == 0 {
		return nil
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, node := range nodes {
		parent, err := node.queryParent()
		if err != nil || parent == nil {
			continue
		}
		children, err := parent.queryChildren()
		if err != nil {
			continue
		}
		// Find node's index among siblings
		idx := -1
		for i, child := range children {
			if child == node {
				idx = i
				break
			}
		}
		if idx < 0 || idx+1 >= len(children) {
			continue
		}
		next := children[idx+1]
		if !seen[next] {
			seen[next] = true
			results = append(results, next)
		}
	}

	return results
}

// getPrevSiblings returns the immediately preceding sibling of each node. Deduplicated.
func getPrevSiblings(nodes []traversableNode) []queryNode {
	if len(nodes) == 0 {
		return nil
	}

	seen := make(map[queryNode]bool)
	var results []queryNode

	for _, node := range nodes {
		parent, err := node.queryParent()
		if err != nil || parent == nil {
			continue
		}
		children, err := parent.queryChildren()
		if err != nil {
			continue
		}
		// Find node's index among siblings
		idx := -1
		for i, child := range children {
			if child == node {
				idx = i
				break
			}
		}
		if idx <= 0 {
			continue
		}
		prev := children[idx-1]
		if !seen[prev] {
			seen[prev] = true
			results = append(results, prev)
		}
	}

	return results
}

// === Selection construction helpers ===

// newSelectionFromNodes creates a Selection backed by queryNode references.
// This is the internal constructor used by traversal methods to preserve
// node identity across chained operations. Elements are extracted from nodes.
func newSelectionFromNodes(nodes []queryNode, sel string) *Selection {
	if len(nodes) == 0 {
		return &Selection{
			elems:    []*ax.Element{},
			selector: sel,
		}
	}
	elems := make([]*ax.Element, len(nodes))
	for i, n := range nodes {
		elems[i] = n.element()
	}
	return &Selection{
		elems:    elems,
		nodes:    nodes,
		selector: sel,
	}
}

// getNodes returns the internal queryNode slice. If nodes is nil (Selection
// created without node tracking), wraps each *ax.Element in an elementAdapter.
func (s *Selection) getNodes() []queryNode {
	if s.nodes != nil {
		return s.nodes
	}
	if len(s.elems) == 0 {
		return nil
	}
	nodes := make([]queryNode, len(s.elems))
	for i, el := range s.elems {
		nodes[i] = newElementAdapter(el)
	}
	return nodes
}

// getTraversableNodes returns traversableNode slice for parent-aware operations.
// Nodes that don't implement traversableNode are silently skipped.
func (s *Selection) getTraversableNodes() []traversableNode {
	raw := s.getNodes()
	if len(raw) == 0 {
		return nil
	}
	var result []traversableNode
	for _, n := range raw {
		if tn, ok := n.(traversableNode); ok {
			result = append(result, tn)
		}
	}
	return result
}

// === Selection traversal methods ===

// Find searches within the subtrees of each element in the selection for
// elements matching the given selector. The elements in the current selection
// are not included in results (matching goquery's Find semantics).
func (s *Selection) Find(sel string) *Selection {
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
	results := findInSubtrees(nodes, matcher)
	return newSelectionFromNodes(results, sel)
}

// Children returns a new Selection containing the direct children of each
// element in the current selection.
func (s *Selection) Children() *Selection {
	if s.err != nil {
		return s
	}
	nodes := s.getNodes()
	results := getChildren(nodes)
	return newSelectionFromNodes(results, s.selector)
}

// ChildrenFiltered returns a new Selection containing the direct children of
// each element that match the given selector.
func (s *Selection) ChildrenFiltered(sel string) *Selection {
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
	results := getChildrenFiltered(nodes, matcher)
	return newSelectionFromNodes(results, sel)
}

// Parent returns a new Selection containing the immediate parent of each
// element in the current selection. Deduplicated.
func (s *Selection) Parent() *Selection {
	if s.err != nil {
		return s
	}
	tnodes := s.getTraversableNodes()
	results := getParents(tnodes)
	return newSelectionFromNodes(results, s.selector)
}

// ParentFiltered returns a new Selection containing the immediate parent of
// each element, filtered by the given selector. Deduplicated.
func (s *Selection) ParentFiltered(sel string) *Selection {
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
	tnodes := s.getTraversableNodes()
	results := getParentsFiltered(tnodes, matcher)
	return newSelectionFromNodes(results, sel)
}

// Parents returns a new Selection containing all ancestors of each element,
// ordered from nearest to farthest. Deduplicated.
func (s *Selection) Parents() *Selection {
	if s.err != nil {
		return s
	}
	tnodes := s.getTraversableNodes()
	results := getAncestors(tnodes, nil)
	return newSelectionFromNodes(results, s.selector)
}

// ParentsUntil returns a new Selection containing all ancestors of each element
// up to (but not including) the ancestor matching the until selector.
func (s *Selection) ParentsUntil(sel string) *Selection {
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
	tnodes := s.getTraversableNodes()
	results := getAncestors(tnodes, matcher)
	return newSelectionFromNodes(results, sel)
}

// Closest returns a new Selection containing the closest ancestor (or self)
// matching the given selector, for each element. Deduplicated.
func (s *Selection) Closest(sel string) *Selection {
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
	tnodes := s.getTraversableNodes()
	results := getClosest(tnodes, matcher)
	return newSelectionFromNodes(results, sel)
}

// Siblings returns a new Selection containing all siblings of each element
// (excluding the elements themselves). Deduplicated.
func (s *Selection) Siblings() *Selection {
	if s.err != nil {
		return s
	}
	tnodes := s.getTraversableNodes()
	results := getSiblings(tnodes)
	return newSelectionFromNodes(results, s.selector)
}

// Next returns a new Selection containing the immediately following sibling
// of each element. Deduplicated.
func (s *Selection) Next() *Selection {
	if s.err != nil {
		return s
	}
	tnodes := s.getTraversableNodes()
	results := getNextSiblings(tnodes)
	return newSelectionFromNodes(results, s.selector)
}

// Prev returns a new Selection containing the immediately preceding sibling
// of each element. Deduplicated.
func (s *Selection) Prev() *Selection {
	if s.err != nil {
		return s
	}
	tnodes := s.getTraversableNodes()
	results := getPrevSiblings(tnodes)
	return newSelectionFromNodes(results, s.selector)
}
