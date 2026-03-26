package selector

// Matchable is the interface that a UI element must satisfy for selector matching.
// ax.Element will implement this through an adapter in the axquery package.
type Matchable interface {
	// GetRole returns the accessibility role (e.g., "AXButton").
	GetRole() string
	// GetAttr returns the value of a named attribute.
	// Known attributes: "title", "description", "value", "role", "subrole",
	// "roleDescription", "identifier", "label", "help", "placeholder".
	GetAttr(name string) string
	// IsEnabled reports whether the element is enabled.
	IsEnabled() bool
	// IsVisible reports whether the element is visible (not hidden).
	IsVisible() bool
	// IsFocused reports whether the element has keyboard focus.
	IsFocused() bool
	// IsSelected reports whether the element is selected.
	IsSelected() bool
}

// Matcher is a compiled selector that can match elements.
type Matcher interface {
	// MatchSimple tests whether a single element matches the leaf selector.
	// It does NOT evaluate combinators (descendant/child); those require tree context
	// and are handled at the Selection/query layer.
	MatchSimple(el Matchable) bool

	// Group returns the parsed AST for use by the query engine (combinators, etc.).
	Group() *SelectorGroup
}
