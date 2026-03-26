package axquery

import "strings"

// property.go implements property-reading methods on Selection.
// All property methods operate on the FIRST element in the selection
// (matching goquery/jQuery semantics). Empty or error selections return
// zero values without panicking.

// firstNode returns the first queryNode in the selection, or nil if the
// selection is empty, errored, or has no nodes. This is a shared guard
// used by all property methods.
func (s *Selection) firstNode() queryNode {
	if s.err != nil || len(s.elems) == 0 {
		return nil
	}
	nodes := s.getNodes()
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

// Attr returns the value of the named attribute from the first element.
// Returns "" for empty/error selections.
func (s *Selection) Attr(name string) string {
	node := s.firstNode()
	if node == nil {
		return ""
	}
	if name == "role" {
		return node.GetRole()
	}
	return node.GetAttr(name)
}

// AttrOr returns the value of the named attribute from the first element,
// or defaultVal if the attribute is empty or the selection is empty/errored.
func (s *Selection) AttrOr(name, defaultVal string) string {
	v := s.Attr(name)
	if v == "" {
		return defaultVal
	}
	return v
}

// Role returns the AX role of the first element (e.g. "AXButton").
// Returns "" for empty/error selections.
func (s *Selection) Role() string {
	node := s.firstNode()
	if node == nil {
		return ""
	}
	return node.GetRole()
}

// Title returns the "title" attribute of the first element.
// Returns "" for empty/error selections.
func (s *Selection) Title() string {
	return s.Attr("title")
}

// Description returns the "description" attribute of the first element.
// Returns "" for empty/error selections.
func (s *Selection) Description() string {
	return s.Attr("description")
}

// Val returns the "value" attribute of the first element.
// This corresponds to AXValue in the accessibility tree.
// Returns "" for empty/error selections.
func (s *Selection) Val() string {
	return s.Attr("value")
}

// Text returns the combined text content of the first element and all its
// descendants. It collects "title" attributes from the element and walks the
// subtree depth-first, concatenating non-empty titles with spaces.
// Returns "" for empty/error selections.
func (s *Selection) Text() string {
	node := s.firstNode()
	if node == nil {
		return ""
	}
	var parts []string
	collectText(node, &parts)
	return strings.Join(parts, " ")
}

// collectText recursively gathers title text from a node and its descendants.
func collectText(node queryNode, parts *[]string) {
	title := node.GetAttr("title")
	if title != "" {
		*parts = append(*parts, title)
	}
	children, err := node.queryChildren()
	if err != nil {
		return
	}
	for _, child := range children {
		collectText(child, parts)
	}
}

// IsVisible returns whether the first element is visible (not hidden).
// Returns false for empty/error selections.
func (s *Selection) IsVisible() bool {
	node := s.firstNode()
	if node == nil {
		return false
	}
	return node.IsVisible()
}

// IsEnabled returns whether the first element is enabled.
// Returns false for empty/error selections.
func (s *Selection) IsEnabled() bool {
	node := s.firstNode()
	if node == nil {
		return false
	}
	return node.IsEnabled()
}

// IsFocused returns whether the first element has keyboard focus.
// Returns false for empty/error selections.
func (s *Selection) IsFocused() bool {
	node := s.firstNode()
	if node == nil {
		return false
	}
	return node.IsFocused()
}

// IsSelected returns whether the first element is selected.
// Returns false for empty/error selections.
func (s *Selection) IsSelected() bool {
	node := s.firstNode()
	if node == nil {
		return false
	}
	return node.IsSelected()
}
