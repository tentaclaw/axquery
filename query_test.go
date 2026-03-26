package axquery

import (
	"errors"
	"testing"

	"github.com/tentaclaw/ax"
	"github.com/tentaclaw/axquery/selector"
)

// === Mock tree node for testing BFS/DFS without real AX elements ===

type mockNode struct {
	role        string
	attrs       map[string]string
	enabled     bool
	visible     bool
	focused     bool
	selected    bool
	childNodes  []*mockNode
	childrenErr error // simulate Children() failure
}

func (m *mockNode) GetRole() string { return m.role }
func (m *mockNode) GetAttr(name string) string {
	if m.attrs != nil {
		return m.attrs[name]
	}
	return ""
}
func (m *mockNode) IsEnabled() bool      { return m.enabled }
func (m *mockNode) IsVisible() bool      { return m.visible }
func (m *mockNode) IsFocused() bool      { return m.focused }
func (m *mockNode) IsSelected() bool     { return m.selected }
func (m *mockNode) element() *ax.Element { return nil }

func (m *mockNode) queryChildren() ([]queryNode, error) {
	if m.childrenErr != nil {
		return nil, m.childrenErr
	}
	result := make([]queryNode, len(m.childNodes))
	for i, c := range m.childNodes {
		result[i] = c
	}
	return result, nil
}

// helper: build a simple mock tree
//
//	root (AXWindow)
//	├── btn1 (AXButton, title="OK", enabled)
//	├── btn2 (AXButton, title="Cancel", enabled)
//	├── group (AXGroup)
//	│   ├── text1 (AXStaticText, title="Hello")
//	│   └── btn3 (AXButton, title="Submit", enabled, focused)
//	└── disabled (AXButton, title="Nope", disabled)
func buildMockTree() *mockNode {
	btn1 := &mockNode{role: "AXButton", attrs: map[string]string{"title": "OK"}, enabled: true, visible: true}
	btn2 := &mockNode{role: "AXButton", attrs: map[string]string{"title": "Cancel"}, enabled: true, visible: true}
	text1 := &mockNode{role: "AXStaticText", attrs: map[string]string{"title": "Hello"}, visible: true}
	btn3 := &mockNode{role: "AXButton", attrs: map[string]string{"title": "Submit"}, enabled: true, visible: true, focused: true}
	group := &mockNode{role: "AXGroup", visible: true, childNodes: []*mockNode{text1, btn3}}
	disabled := &mockNode{role: "AXButton", attrs: map[string]string{"title": "Nope"}, enabled: false, visible: true}
	root := &mockNode{role: "AXWindow", visible: true, childNodes: []*mockNode{btn1, btn2, group, disabled}}
	return root
}

// --- searchBFS tests ---

func TestSearchBFS_MatchRole(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	// Should find 4 buttons: btn1, btn2, btn3, disabled
	if len(results) != 4 {
		t.Fatalf("expected 4 buttons, got %d", len(results))
	}
}

func TestSearchBFS_MatchAttr(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile(`AXButton[title="OK"]`)
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 button with title OK, got %d", len(results))
	}
}

func TestSearchBFS_MatchPseudo(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXButton:enabled")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	// btn1, btn2, btn3 are enabled; disabled is not
	if len(results) != 3 {
		t.Fatalf("expected 3 enabled buttons, got %d", len(results))
	}
}

func TestSearchBFS_MatchFocused(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXButton:focused")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 focused button, got %d", len(results))
	}
}

func TestSearchBFS_Wildcard(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("*")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	// root + btn1 + btn2 + group + text1 + btn3 + disabled = 7
	if len(results) != 7 {
		t.Fatalf("expected 7 elements for wildcard, got %d", len(results))
	}
}

func TestSearchBFS_MaxResults(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	opts := defaultQueryOptions()
	opts.MaxResults = 2
	results, err := searchBFS(root, m, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results with MaxResults=2, got %d", len(results))
	}
}

func TestSearchBFS_MaxDepth(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	opts := defaultQueryOptions()
	opts.MaxDepth = 1 // only root's direct children
	results, err := searchBFS(root, m, opts)
	if err != nil {
		t.Fatal(err)
	}
	// At depth 1: btn1, btn2, disabled (not btn3 which is at depth 2)
	if len(results) != 3 {
		t.Fatalf("expected 3 buttons at depth <= 1, got %d", len(results))
	}
}

func TestSearchBFS_NoMatch(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXTable")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchBFS_EmptyTree(t *testing.T) {
	root := &mockNode{role: "AXWindow"}
	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results in empty tree, got %d", len(results))
	}
}

func TestSearchBFS_ChildrenError_Continues(t *testing.T) {
	// A node with children error should be skipped, not abort the search
	badGroup := &mockNode{role: "AXGroup", childrenErr: errors.New("AX error")}
	btn := &mockNode{role: "AXButton", enabled: true, visible: true}
	root := &mockNode{role: "AXWindow", childNodes: []*mockNode{badGroup, btn}}

	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 button despite error, got %d", len(results))
	}
}

func TestSearchBFS_BFSOrder(t *testing.T) {
	// Verify BFS order: breadth-first means shallow nodes come first
	root := buildMockTree()
	m, err := selector.Compile("*")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	// BFS order: root, btn1, btn2, group, disabled, text1, btn3
	roles := make([]string, len(results))
	for i, n := range results {
		roles[i] = n.GetRole()
	}
	expected := []string{"AXWindow", "AXButton", "AXButton", "AXGroup", "AXButton", "AXStaticText", "AXButton"}
	if len(roles) != len(expected) {
		t.Fatalf("expected %d elements, got %d: %v", len(expected), len(roles), roles)
	}
	for i := range expected {
		if roles[i] != expected[i] {
			t.Fatalf("BFS order mismatch at index %d: expected %s, got %s\nfull: %v", i, expected[i], roles[i], roles)
		}
	}
}

func TestSearchBFS_RootMatchesSelector(t *testing.T) {
	// Root itself should be included if it matches
	root := &mockNode{role: "AXButton", enabled: true}
	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected root to match, got %d", len(results))
	}
}

func TestSearchBFS_GroupSelector(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXStaticText, AXGroup")
	if err != nil {
		t.Fatal(err)
	}
	results, err := searchBFS(root, m, defaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	// 1 AXGroup + 1 AXStaticText = 2
	if len(results) != 2 {
		t.Fatalf("expected 2 results for group selector, got %d", len(results))
	}
}

// --- searchDFS tests ---

func TestSearchDFS_MatchRole(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	opts := defaultQueryOptions()
	opts.Strategy = StrategyDFS
	results, err := searchDFS(root, m, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 buttons, got %d", len(results))
	}
}

func TestSearchDFS_DFSOrder(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("*")
	if err != nil {
		t.Fatal(err)
	}
	opts := defaultQueryOptions()
	opts.Strategy = StrategyDFS
	results, err := searchDFS(root, m, opts)
	if err != nil {
		t.Fatal(err)
	}
	// DFS order: root, btn1, btn2, group, text1, btn3, disabled
	roles := make([]string, len(results))
	for i, n := range results {
		roles[i] = n.GetRole()
	}
	expected := []string{"AXWindow", "AXButton", "AXButton", "AXGroup", "AXStaticText", "AXButton", "AXButton"}
	if len(roles) != len(expected) {
		t.Fatalf("expected %d elements, got %d: %v", len(expected), len(roles), roles)
	}
	for i := range expected {
		if roles[i] != expected[i] {
			t.Fatalf("DFS order mismatch at index %d: expected %s, got %s\nfull: %v", i, expected[i], roles[i], roles)
		}
	}
}

func TestSearchDFS_MaxResults(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	opts := defaultQueryOptions()
	opts.Strategy = StrategyDFS
	opts.MaxResults = 2
	results, err := searchDFS(root, m, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results with MaxResults=2, got %d", len(results))
	}
}

func TestSearchDFS_MaxDepth(t *testing.T) {
	root := buildMockTree()
	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	opts := defaultQueryOptions()
	opts.Strategy = StrategyDFS
	opts.MaxDepth = 1
	results, err := searchDFS(root, m, opts)
	if err != nil {
		t.Fatal(err)
	}
	// Only depth 0 (root) and depth 1 (btn1, btn2, disabled)
	if len(results) != 3 {
		t.Fatalf("expected 3 buttons at depth <= 1, got %d", len(results))
	}
}

func TestSearchDFS_ChildrenError_Continues(t *testing.T) {
	badGroup := &mockNode{role: "AXGroup", childrenErr: errors.New("AX error")}
	btn := &mockNode{role: "AXButton", enabled: true}
	root := &mockNode{role: "AXWindow", childNodes: []*mockNode{badGroup, btn}}

	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	opts := defaultQueryOptions()
	opts.Strategy = StrategyDFS
	results, err := searchDFS(root, m, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 button despite error, got %d", len(results))
	}
}

// --- elementAdapter tests ---

func TestElementAdapter_NilElement(t *testing.T) {
	a := newElementAdapter(nil)
	if a.GetRole() != "" {
		t.Fatal("nil element should return empty role")
	}
	if a.GetAttr("title") != "" {
		t.Fatal("nil element should return empty attr")
	}
	if a.IsEnabled() {
		t.Fatal("nil element should not be enabled")
	}
	if a.IsVisible() {
		t.Fatal("nil element should not be visible")
	}
	if a.IsFocused() {
		t.Fatal("nil element should not be focused")
	}
	if a.IsSelected() {
		t.Fatal("nil element should not be selected")
	}
}

// --- queryFromRoot tests ---

func TestQueryFromRoot_InvalidSelector(t *testing.T) {
	root := buildMockTree()
	sel := queryFromRoot(root, "[invalid", defaultQueryOptions())
	if sel.Err() == nil {
		t.Fatal("expected error for invalid selector")
	}
	if !errors.Is(sel.Err(), ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector, got %v", sel.Err())
	}
}

func TestQueryFromRoot_BFS(t *testing.T) {
	root := buildMockTree()
	sel := queryFromRoot(root, "AXButton", defaultQueryOptions())
	if sel.Err() != nil {
		t.Fatal(sel.Err())
	}
	if sel.Count() != 4 {
		t.Fatalf("expected 4 buttons, got %d", sel.Count())
	}
	if sel.Selector() != "AXButton" {
		t.Fatalf("expected selector AXButton, got %s", sel.Selector())
	}
}

func TestQueryFromRoot_DFS(t *testing.T) {
	root := buildMockTree()
	opts := defaultQueryOptions()
	opts.Strategy = StrategyDFS
	sel := queryFromRoot(root, "AXButton:enabled", opts)
	if sel.Err() != nil {
		t.Fatal(sel.Err())
	}
	if sel.Count() != 3 {
		t.Fatalf("expected 3 enabled buttons, got %d", sel.Count())
	}
}

func TestQueryFromRoot_EmptyResult(t *testing.T) {
	root := buildMockTree()
	sel := queryFromRoot(root, "AXTable", defaultQueryOptions())
	if sel.Err() != nil {
		t.Fatal(sel.Err())
	}
	if sel.Count() != 0 {
		t.Fatalf("expected 0 results, got %d", sel.Count())
	}
	if !sel.IsEmpty() {
		t.Fatal("expected empty selection")
	}
}

func TestQueryFromRoot_WithOptions(t *testing.T) {
	root := buildMockTree()
	opts := applyOptions(WithMaxResults(1))
	sel := queryFromRoot(root, "AXButton", opts)
	if sel.Err() != nil {
		t.Fatal(sel.Err())
	}
	if sel.Count() != 1 {
		t.Fatalf("expected 1 result with MaxResults=1, got %d", sel.Count())
	}
}

// === rootResolver tests ===

// mockResolver implements rootResolver for testing queryWithResolver.
type mockResolver struct {
	root queryNode
	err  error
}

func (m *mockResolver) resolveRoot() (queryNode, error) {
	return m.root, m.err
}

func TestQueryWithResolver_Success(t *testing.T) {
	tree := buildMockTree()
	r := &mockResolver{root: tree}
	sel := queryWithResolver(r, "AXButton", defaultQueryOptions())
	if sel.Err() != nil {
		t.Fatal(sel.Err())
	}
	if sel.Count() != 4 {
		t.Fatalf("expected 4 buttons, got %d", sel.Count())
	}
}

func TestQueryWithResolver_ResolverError(t *testing.T) {
	r := &mockResolver{err: errors.New("no window")}
	sel := queryWithResolver(r, "AXButton", defaultQueryOptions())
	if sel.Err() == nil {
		t.Fatal("expected error when resolver fails")
	}
}

func TestQueryWithResolver_NilRoot(t *testing.T) {
	r := &mockResolver{root: nil, err: nil}
	sel := queryWithResolver(r, "AXButton", defaultQueryOptions())
	if sel.Err() == nil {
		t.Fatal("expected error when root is nil")
	}
	if !errors.Is(sel.Err(), ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", sel.Err())
	}
}

func TestQueryWithResolver_InvalidSelector(t *testing.T) {
	tree := buildMockTree()
	r := &mockResolver{root: tree}
	sel := queryWithResolver(r, "[bad", defaultQueryOptions())
	if sel.Err() == nil {
		t.Fatal("expected error for invalid selector")
	}
	if !errors.Is(sel.Err(), ErrInvalidSelector) {
		t.Fatalf("expected ErrInvalidSelector, got %v", sel.Err())
	}
}

func TestQueryWithResolver_WithOptions(t *testing.T) {
	tree := buildMockTree()
	r := &mockResolver{root: tree}
	opts := applyOptions(WithMaxResults(2), WithStrategy(StrategyDFS))
	sel := queryWithResolver(r, "AXButton", opts)
	if sel.Err() != nil {
		t.Fatal(sel.Err())
	}
	if sel.Count() != 2 {
		t.Fatalf("expected 2 results with MaxResults=2, got %d", sel.Count())
	}
}

// === elementAdapter.queryChildren nil path ===

func TestElementAdapter_QueryChildren_NilElement(t *testing.T) {
	a := newElementAdapter(nil)
	children, err := a.queryChildren()
	if err != nil {
		t.Fatalf("expected no error for nil element, got %v", err)
	}
	if children != nil {
		t.Fatalf("expected nil children for nil element, got %v", children)
	}
}

// === mockAXElement: implements axElementReader for unit testing elementAdapter ===

type mockAXElement struct {
	role            string
	roleErr         error
	title           string
	titleErr        error
	description     string
	descriptionErr  error
	subrole         string
	subroleErr      error
	roleDescription string
	roleDescErr     error
	attrVal         *ax.Value
	attrErr         error
	enabled         bool
	enabledErr      error
	hidden          bool
	hiddenErr       error
	focused         bool
	focusedErr      error
	selected        bool
	selectedErr     error
}

func (m *mockAXElement) Role() (string, error)            { return m.role, m.roleErr }
func (m *mockAXElement) Title() (string, error)           { return m.title, m.titleErr }
func (m *mockAXElement) Description() (string, error)     { return m.description, m.descriptionErr }
func (m *mockAXElement) Subrole() (string, error)         { return m.subrole, m.subroleErr }
func (m *mockAXElement) RoleDescription() (string, error) { return m.roleDescription, m.roleDescErr }
func (m *mockAXElement) Attribute(name string) (*ax.Value, error) {
	return m.attrVal, m.attrErr
}
func (m *mockAXElement) IsEnabled() (bool, error)  { return m.enabled, m.enabledErr }
func (m *mockAXElement) IsHidden() (bool, error)   { return m.hidden, m.hiddenErr }
func (m *mockAXElement) IsFocused() (bool, error)  { return m.focused, m.focusedErr }
func (m *mockAXElement) IsSelected() (bool, error) { return m.selected, m.selectedErr }

// === elementAdapter non-nil path tests ===

func TestElementAdapter_GetRole_NonNil(t *testing.T) {
	mock := &mockAXElement{role: "AXButton"}
	a := &elementAdapter{reader: mock}
	if got := a.GetRole(); got != "AXButton" {
		t.Fatalf("expected AXButton, got %s", got)
	}
}

func TestElementAdapter_GetRole_Error(t *testing.T) {
	mock := &mockAXElement{roleErr: errors.New("ax error")}
	a := &elementAdapter{reader: mock}
	if got := a.GetRole(); got != "" {
		t.Fatalf("expected empty on error, got %s", got)
	}
}

func TestElementAdapter_GetAttr_Title(t *testing.T) {
	mock := &mockAXElement{title: "OK"}
	a := &elementAdapter{reader: mock}
	if got := a.GetAttr("title"); got != "OK" {
		t.Fatalf("expected OK, got %s", got)
	}
}

func TestElementAdapter_GetAttr_Description(t *testing.T) {
	mock := &mockAXElement{description: "Close button"}
	a := &elementAdapter{reader: mock}
	if got := a.GetAttr("description"); got != "Close button" {
		t.Fatalf("expected 'Close button', got %s", got)
	}
}

func TestElementAdapter_GetAttr_Role(t *testing.T) {
	mock := &mockAXElement{role: "AXTextField"}
	a := &elementAdapter{reader: mock}
	if got := a.GetAttr("role"); got != "AXTextField" {
		t.Fatalf("expected AXTextField, got %s", got)
	}
}

func TestElementAdapter_GetAttr_Subrole(t *testing.T) {
	mock := &mockAXElement{subrole: "AXSearchField"}
	a := &elementAdapter{reader: mock}
	if got := a.GetAttr("subrole"); got != "AXSearchField" {
		t.Fatalf("expected AXSearchField, got %s", got)
	}
}

func TestElementAdapter_GetAttr_RoleDescription(t *testing.T) {
	mock := &mockAXElement{roleDescription: "button"}
	a := &elementAdapter{reader: mock}
	if got := a.GetAttr("roleDescription"); got != "button" {
		t.Fatalf("expected button, got %s", got)
	}
}

func TestElementAdapter_GetAttr_GenericAttribute(t *testing.T) {
	mock := &mockAXElement{attrVal: &ax.Value{Str: "hello"}}
	a := &elementAdapter{reader: mock}
	if got := a.GetAttr("customAttr"); got != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
}

func TestElementAdapter_GetAttr_GenericAttribute_Error(t *testing.T) {
	mock := &mockAXElement{attrErr: errors.New("not found")}
	a := &elementAdapter{reader: mock}
	if got := a.GetAttr("missing"); got != "" {
		t.Fatalf("expected empty on error, got %s", got)
	}
}

func TestElementAdapter_GetAttr_GenericAttribute_NilValue(t *testing.T) {
	mock := &mockAXElement{attrVal: nil}
	a := &elementAdapter{reader: mock}
	if got := a.GetAttr("nilAttr"); got != "" {
		t.Fatalf("expected empty for nil value, got %s", got)
	}
}

func TestElementAdapter_IsEnabled_NonNil(t *testing.T) {
	mock := &mockAXElement{enabled: true}
	a := &elementAdapter{reader: mock}
	if !a.IsEnabled() {
		t.Fatal("expected enabled")
	}
}

func TestElementAdapter_IsVisible_NonNil(t *testing.T) {
	mock := &mockAXElement{hidden: false}
	a := &elementAdapter{reader: mock}
	if !a.IsVisible() {
		t.Fatal("expected visible (not hidden)")
	}
}

func TestElementAdapter_IsVisible_Hidden(t *testing.T) {
	mock := &mockAXElement{hidden: true}
	a := &elementAdapter{reader: mock}
	if a.IsVisible() {
		t.Fatal("expected not visible (hidden)")
	}
}

func TestElementAdapter_IsFocused_NonNil(t *testing.T) {
	mock := &mockAXElement{focused: true}
	a := &elementAdapter{reader: mock}
	if !a.IsFocused() {
		t.Fatal("expected focused")
	}
}

func TestElementAdapter_IsSelected_NonNil(t *testing.T) {
	mock := &mockAXElement{selected: true}
	a := &elementAdapter{reader: mock}
	if !a.IsSelected() {
		t.Fatal("expected selected")
	}
}

func TestElementAdapter_Element_Nil(t *testing.T) {
	a := newElementAdapter(nil)
	if a.element() != nil {
		t.Fatal("expected nil element for nil adapter")
	}
}

// === elementAdapter.queryChildren with childFn ===

func TestElementAdapter_QueryChildren_WithChildFn(t *testing.T) {
	mock := &mockAXElement{role: "AXWindow"}
	a := &elementAdapter{
		reader: mock,
		childFn: func() ([]*ax.Element, error) {
			// Return two nil elements — they get wrapped in adapters with nil reader
			return []*ax.Element{nil, nil}, nil
		},
	}
	children, err := a.queryChildren()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
}

func TestElementAdapter_QueryChildren_WithChildFn_Error(t *testing.T) {
	mock := &mockAXElement{role: "AXWindow"}
	a := &elementAdapter{
		reader: mock,
		childFn: func() ([]*ax.Element, error) {
			return nil, errors.New("children error")
		},
	}
	children, err := a.queryChildren()
	if err == nil {
		t.Fatal("expected error")
	}
	if children != nil {
		t.Fatalf("expected nil children on error, got %v", children)
	}
}

func TestElementAdapter_QueryChildren_NilReaderNilEl(t *testing.T) {
	// reader is nil but not created via newElementAdapter
	a := &elementAdapter{}
	children, err := a.queryChildren()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if children != nil {
		t.Fatal("expected nil children for nil reader")
	}
}

func TestElementAdapter_QueryChildren_ReaderSetButNoElNoChildFn(t *testing.T) {
	// reader is set (non-nil path taken), but el is nil and childFn is nil
	mock := &mockAXElement{role: "AXGroup"}
	a := &elementAdapter{reader: mock}
	children, err := a.queryChildren()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if children != nil {
		t.Fatal("expected nil children when el is nil and childFn is nil")
	}
}

// === searchDFS early-done-in-sibling-loop ===

func TestSearchDFS_MaxResults_DoneInSiblingLoop(t *testing.T) {
	// Tree: root has 3 children, each is AXButton.
	// With MaxResults=1, first child matches and sets done=true.
	// The sibling loop should exit early for remaining children.
	c1 := &mockNode{role: "AXButton", enabled: true}
	c2 := &mockNode{role: "AXButton", enabled: true}
	c3 := &mockNode{role: "AXButton", enabled: true}
	root := &mockNode{role: "AXWindow", childNodes: []*mockNode{c1, c2, c3}}

	m, err := selector.Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	opts := defaultQueryOptions()
	opts.Strategy = StrategyDFS
	opts.MaxResults = 1
	results, err := searchDFS(root, m, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

// === newElementAdapter non-nil path ===

// === elementAdapter implements traversableNode ===

func TestElementAdapter_ImplementsTraversableNode(t *testing.T) {
	// Compile-time assertion: elementAdapter must satisfy traversableNode.
	var _ traversableNode = (*elementAdapter)(nil)
}

func TestElementAdapter_QueryParent_NilElement(t *testing.T) {
	a := newElementAdapter(nil)
	parent, err := a.queryParent()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if parent != nil {
		t.Fatal("expected nil parent for nil element")
	}
}

func TestNewElementAdapter_NonNil(t *testing.T) {
	// Construct a zero-valued *ax.Element to exercise the non-nil branch.
	// We don't call AX methods on it; just verify adapter fields are set.
	el := &ax.Element{}
	a := newElementAdapter(el)
	if a.reader == nil {
		t.Fatal("expected non-nil reader for non-nil element")
	}
	if a.el == nil {
		t.Fatal("expected non-nil el for non-nil element")
	}
	if a.element() != el {
		t.Fatal("expected element() to return the same element")
	}
}
