package axquery

import (
	"github.com/tentaclaw/ax"
	"github.com/tentaclaw/axquery/selector"
)

// queryNode is the internal interface for tree traversal in the query engine.
// It extends selector.Matchable with child access, enabling unit-testable BFS/DFS
// without real AX elements (which require CGo + macOS permissions).
type queryNode interface {
	selector.Matchable
	queryChildren() ([]queryNode, error)
	// element returns the underlying *ax.Element, or nil for mock nodes.
	element() *ax.Element
}

// axElementReader abstracts the read-only methods of *ax.Element for
// testability. *ax.Element satisfies this interface implicitly.
type axElementReader interface {
	Role() (string, error)
	Title() (string, error)
	Description() (string, error)
	Subrole() (string, error)
	RoleDescription() (string, error)
	Attribute(name string) (*ax.Value, error)
	IsEnabled() (bool, error)
	IsHidden() (bool, error)
	IsFocused() (bool, error)
	IsSelected() (bool, error)
}

// searchBFS performs a breadth-first search of the accessibility tree starting
// from root, collecting nodes that match the compiled selector.
func searchBFS(root queryNode, matcher selector.Matcher, opts QueryOptions) ([]queryNode, error) {
	var results []queryNode

	type queueItem struct {
		node  queryNode
		depth int
	}
	queue := []queueItem{{node: root, depth: 0}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		// Check match
		if matcher.MatchSimple(item.node) {
			results = append(results, item.node)
			if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
				return results, nil
			}
		}

		// Expand children if within depth limit
		if opts.MaxDepth > 0 && item.depth >= opts.MaxDepth {
			continue
		}
		children, err := item.node.queryChildren()
		if err != nil {
			// Skip nodes whose children can't be read (AX errors are common)
			continue
		}
		for _, child := range children {
			queue = append(queue, queueItem{node: child, depth: item.depth + 1})
		}
	}

	return results, nil
}

// searchDFS performs a depth-first search of the accessibility tree.
func searchDFS(root queryNode, matcher selector.Matcher, opts QueryOptions) ([]queryNode, error) {
	var results []queryNode
	var done bool

	var walk func(node queryNode, depth int)
	walk = func(node queryNode, depth int) {
		// Check match
		if matcher.MatchSimple(node) {
			results = append(results, node)
			if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
				done = true
				return
			}
		}

		// Expand children if within depth limit
		if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
			return
		}
		children, err := node.queryChildren()
		if err != nil {
			return
		}
		for _, child := range children {
			if done {
				return
			}
			walk(child, depth+1)
		}
	}

	walk(root, 0)
	return results, nil
}

// queryFromRoot compiles the selector and runs a BFS or DFS search starting
// from the given root queryNode. Returns a Selection. This is the testable
// core of the query engine.
func queryFromRoot(root queryNode, sel string, opts QueryOptions) *Selection {
	matcher, err := selector.Compile(sel)
	if err != nil {
		return newSelectionError(
			&InvalidSelectorError{Selector: sel, Reason: err.Error()},
			sel,
		)
	}

	var nodes []queryNode
	switch opts.Strategy {
	case StrategyDFS:
		nodes, _ = searchDFS(root, matcher, opts)
	default:
		nodes, _ = searchBFS(root, matcher, opts)
	}

	// Convert queryNode results to []*ax.Element.
	// Mock nodes return nil from element(), which is fine for Selection
	// operations that don't dereference elements.
	elems := make([]*ax.Element, len(nodes))
	for i, n := range nodes {
		elems[i] = n.element()
	}

	return newSelection(elems, sel)
}

// === elementAdapter: wraps *ax.Element to implement queryNode ===

// elementAdapter adapts an *ax.Element to the queryNode interface,
// mapping ax's (value, error) methods to the simpler selector.Matchable interface.
// It uses axElementReader for testability: the matching logic works against the
// interface, while real *ax.Element is only needed for queryChildren and element().
type elementAdapter struct {
	reader  axElementReader               // for Matchable methods; nil means nil element
	el      *ax.Element                   // real element for queryChildren + element()
	childFn func() ([]*ax.Element, error) // injectable for testing; nil falls back to el.Children()
}

// newElementAdapter creates an elementAdapter from a real *ax.Element.
// Nil-safe: a nil element produces an adapter that reports empty/false for all fields.
func newElementAdapter(el *ax.Element) *elementAdapter {
	if el == nil {
		return &elementAdapter{}
	}
	return &elementAdapter{reader: el, el: el}
}

func (a *elementAdapter) element() *ax.Element {
	return a.el
}

func (a *elementAdapter) GetRole() string {
	if a.reader == nil {
		return ""
	}
	role, _ := a.reader.Role()
	return role
}

func (a *elementAdapter) GetAttr(name string) string {
	if a.reader == nil {
		return ""
	}
	switch name {
	case "title":
		v, _ := a.reader.Title()
		return v
	case "description":
		v, _ := a.reader.Description()
		return v
	case "role":
		v, _ := a.reader.Role()
		return v
	case "subrole":
		v, _ := a.reader.Subrole()
		return v
	case "roleDescription":
		v, _ := a.reader.RoleDescription()
		return v
	default:
		// Try generic Attribute for unknown names
		val, err := a.reader.Attribute(name)
		if err != nil || val == nil {
			return ""
		}
		return val.Str
	}
}

func (a *elementAdapter) IsEnabled() bool {
	if a.reader == nil {
		return false
	}
	v, _ := a.reader.IsEnabled()
	return v
}

func (a *elementAdapter) IsVisible() bool {
	if a.reader == nil {
		return false
	}
	// AX uses "hidden" semantics: IsHidden=true means not visible
	hidden, _ := a.reader.IsHidden()
	return !hidden
}

func (a *elementAdapter) IsFocused() bool {
	if a.reader == nil {
		return false
	}
	v, _ := a.reader.IsFocused()
	return v
}

func (a *elementAdapter) IsSelected() bool {
	if a.reader == nil {
		return false
	}
	v, _ := a.reader.IsSelected()
	return v
}

func (a *elementAdapter) queryParent() (queryNode, error) {
	if a.el == nil {
		return nil, nil
	}
	parent, err := a.el.Parent()
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, nil
	}
	return newElementAdapter(parent), nil
}

func (a *elementAdapter) queryChildren() ([]queryNode, error) {
	if a.reader == nil {
		return nil, nil
	}
	// Use injectable childFn if provided (for testing), else real element
	var children []*ax.Element
	var err error
	if a.childFn != nil {
		children, err = a.childFn()
	} else if a.el != nil {
		children, err = a.el.Children()
	} else {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	nodes := make([]queryNode, len(children))
	for i, child := range children {
		nodes[i] = newElementAdapter(child)
	}
	return nodes, nil
}

// === Root resolution ===

// rootResolver abstracts how the query engine obtains the root element for
// tree traversal. This enables unit-testing Query logic without real AX.
type rootResolver interface {
	resolveRoot() (queryNode, error)
}

// appRootResolver resolves the root from a real *ax.Application by trying
// FocusedWindow first, then MainWindow.
type appRootResolver struct {
	app *ax.Application
}

func (r *appRootResolver) resolveRoot() (queryNode, error) {
	root, err := r.app.FocusedWindow()
	if err != nil || root == nil {
		root, err = r.app.MainWindow()
		if err != nil {
			return nil, err
		}
	}
	if root == nil {
		return nil, nil
	}
	return newElementAdapter(root), nil
}

// queryWithResolver is the testable core of Query. It resolves the root via
// the resolver, then delegates to queryFromRoot.
func queryWithResolver(resolver rootResolver, sel string, opts QueryOptions) *Selection {
	root, err := resolver.resolveRoot()
	if err != nil {
		return newSelectionError(err, sel)
	}
	if root == nil {
		return newSelectionError(
			&NotFoundError{Selector: sel},
			sel,
		)
	}
	return queryFromRoot(root, sel, opts)
}

// === Public Query API ===

// Query searches an application's UI tree for elements matching the given
// CSS-like selector. It returns a Selection that can be chained with methods
// like First(), Last(), Eq(), Find(), etc.
//
// The search starts from the application's focused window, falling back to
// the main window if no focused window is available.
//
// Example:
//
//	sel := axquery.Query(app, "AXButton[title=\"OK\"]:enabled")
//	if sel.Err() != nil { ... }
//	sel.First().Click()
func Query(app *ax.Application, sel string, optFns ...QueryOption) *Selection {
	opts := applyOptions(optFns...)
	return queryWithResolver(&appRootResolver{app: app}, sel, opts)
}
