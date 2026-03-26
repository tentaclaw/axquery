package selector

import "testing"

// mockElement implements Matchable for testing.
type mockElement struct {
	role     string
	attrs    map[string]string
	enabled  bool
	visible  bool
	focused  bool
	selected bool
}

func (m *mockElement) GetRole() string { return m.role }
func (m *mockElement) GetAttr(name string) string {
	if m.attrs == nil {
		return ""
	}
	return m.attrs[name]
}
func (m *mockElement) IsEnabled() bool  { return m.enabled }
func (m *mockElement) IsVisible() bool  { return m.visible }
func (m *mockElement) IsFocused() bool  { return m.focused }
func (m *mockElement) IsSelected() bool { return m.selected }

// --- Compile function tests ---

func TestCompile_ValidSelector(t *testing.T) {
	m, err := Compile("AXButton")
	if err != nil {
		t.Fatalf("Compile(AXButton) error: %v", err)
	}
	// Verify it actually matches AXButton elements.
	btn := &mockElement{role: "AXButton"}
	if !m.MatchSimple(btn) {
		t.Fatal("Compile(AXButton) should match an AXButton element")
	}
	// And does NOT match a different role.
	win := &mockElement{role: "AXWindow"}
	if m.MatchSimple(win) {
		t.Fatal("Compile(AXButton) should not match AXWindow")
	}
}

func TestCompile_InvalidSelector(t *testing.T) {
	_, err := Compile("")
	if err == nil {
		t.Fatal("Compile('') should return error")
	}

	_, err = Compile("[title=")
	if err == nil {
		t.Fatal("Compile('[title=') should return error")
	}
}

// --- MatchSimple: role matching ---

func TestMatchSimple_RoleExact(t *testing.T) {
	m, err := Compile("AXButton")
	if err != nil {
		t.Fatal(err)
	}

	btn := &mockElement{role: "AXButton"}
	txt := &mockElement{role: "AXStaticText"}

	if !m.MatchSimple(btn) {
		t.Fatal("AXButton should match AXButton")
	}
	if m.MatchSimple(txt) {
		t.Fatal("AXStaticText should NOT match AXButton")
	}
}

func TestMatchSimple_Wildcard(t *testing.T) {
	m, err := Compile("*")
	if err != nil {
		t.Fatal(err)
	}

	if !m.MatchSimple(&mockElement{role: "AXButton"}) {
		t.Fatal("wildcard should match AXButton")
	}
	if !m.MatchSimple(&mockElement{role: "AXWindow"}) {
		t.Fatal("wildcard should match AXWindow")
	}
	if !m.MatchSimple(&mockElement{role: "AXStaticText"}) {
		t.Fatal("wildcard should match AXStaticText")
	}
}

// --- MatchSimple: attribute operators ---

func TestMatchSimple_AttrEquals(t *testing.T) {
	m, err := Compile(`AXButton[title="OK"]`)
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXButton", attrs: map[string]string{"title": "OK"}}
	no := &mockElement{role: "AXButton", attrs: map[string]string{"title": "Cancel"}}
	wrongRole := &mockElement{role: "AXMenuItem", attrs: map[string]string{"title": "OK"}}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for title=OK")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no match for title=Cancel")
	}
	if m.MatchSimple(wrongRole) {
		t.Fatal("expected no match for wrong role")
	}
}

func TestMatchSimple_AttrContains(t *testing.T) {
	m, err := Compile(`AXButton[title*="end"]`)
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXButton", attrs: map[string]string{"title": "Send Message"}}
	no := &mockElement{role: "AXButton", attrs: map[string]string{"title": "Cancel"}}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for contains 'end'")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no match")
	}
}

func TestMatchSimple_AttrPrefix(t *testing.T) {
	m, err := Compile(`*[title^="Re:"]`)
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXWindow", attrs: map[string]string{"title": "Re: Hello"}}
	no := &mockElement{role: "AXWindow", attrs: map[string]string{"title": "Fwd: Hello"}}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for prefix")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no match for non-prefix")
	}
}

func TestMatchSimple_AttrSuffix(t *testing.T) {
	m, err := Compile(`*[title$=".txt"]`)
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXCell", attrs: map[string]string{"title": "readme.txt"}}
	no := &mockElement{role: "AXCell", attrs: map[string]string{"title": "readme.md"}}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for suffix")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no match for non-suffix")
	}
}

func TestMatchSimple_AttrNotEquals(t *testing.T) {
	m, err := Compile(`AXButton[title!="Cancel"]`)
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXButton", attrs: map[string]string{"title": "OK"}}
	no := &mockElement{role: "AXButton", attrs: map[string]string{"title": "Cancel"}}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for != Cancel")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no match for == Cancel")
	}
}

func TestMatchSimple_AttrRegex(t *testing.T) {
	m, err := Compile(`AXStaticText[title~="\\d+ unread"]`)
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXStaticText", attrs: map[string]string{"title": "42 unread messages"}}
	no := &mockElement{role: "AXStaticText", attrs: map[string]string{"title": "no unread"}}

	if !m.MatchSimple(ok) {
		t.Fatal("expected regex match")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no regex match")
	}
}

func TestMatchSimple_AttrRegexInvalid(t *testing.T) {
	// Invalid regex pattern — should compile the selector but not match anything
	m, err := Compile(`AXButton[title~="[invalid"]`)
	if err != nil {
		t.Fatal(err)
	}

	el := &mockElement{role: "AXButton", attrs: map[string]string{"title": "[invalid"}}
	if m.MatchSimple(el) {
		t.Fatal("invalid regex should not match")
	}
}

func TestMatchSimple_AttrRegexCached(t *testing.T) {
	// Compile once, match twice — regex should be cached internally
	m, err := Compile(`AXButton[title~="^OK$"]`)
	if err != nil {
		t.Fatal(err)
	}

	el := &mockElement{role: "AXButton", attrs: map[string]string{"title": "OK"}}
	if !m.MatchSimple(el) {
		t.Fatal("first match should succeed")
	}
	// Second call should use cached regex
	if !m.MatchSimple(el) {
		t.Fatal("second match (cached) should succeed")
	}
}

func TestMatchSimple_MultipleAttrs(t *testing.T) {
	m, err := Compile(`AXButton[title="OK"][description*="confirm"]`)
	if err != nil {
		t.Fatal(err)
	}

	both := &mockElement{role: "AXButton", attrs: map[string]string{"title": "OK", "description": "Please confirm"}}
	onlyTitle := &mockElement{role: "AXButton", attrs: map[string]string{"title": "OK", "description": "nope"}}
	neither := &mockElement{role: "AXButton", attrs: map[string]string{"title": "No", "description": "nope"}}

	if !m.MatchSimple(both) {
		t.Fatal("expected match for both attrs")
	}
	if m.MatchSimple(onlyTitle) {
		t.Fatal("expected no match when only title matches")
	}
	if m.MatchSimple(neither) {
		t.Fatal("expected no match when neither matches")
	}
}

func TestMatchSimple_NilAttrs(t *testing.T) {
	m, err := Compile(`AXButton[title="OK"]`)
	if err != nil {
		t.Fatal(err)
	}

	el := &mockElement{role: "AXButton"} // attrs is nil
	if m.MatchSimple(el) {
		t.Fatal("nil attrs map should return empty string, not match")
	}
}

// --- MatchSimple: pseudo-selectors (boolean) ---

func TestMatchSimple_PseudoEnabled(t *testing.T) {
	m, err := Compile("AXButton:enabled")
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXButton", enabled: true}
	no := &mockElement{role: "AXButton", enabled: false}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for enabled")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no match for disabled")
	}
}

func TestMatchSimple_PseudoVisible(t *testing.T) {
	m, err := Compile("AXButton:visible")
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXButton", visible: true}
	no := &mockElement{role: "AXButton", visible: false}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for visible")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no match for invisible")
	}
}

func TestMatchSimple_PseudoFocused(t *testing.T) {
	m, err := Compile("AXTextField:focused")
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXTextField", focused: true}
	no := &mockElement{role: "AXTextField", focused: false}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for focused")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no match for unfocused")
	}
}

func TestMatchSimple_PseudoSelected(t *testing.T) {
	m, err := Compile("AXRow:selected")
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXRow", selected: true}
	no := &mockElement{role: "AXRow", selected: false}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for selected")
	}
	if m.MatchSimple(no) {
		t.Fatal("expected no match for unselected")
	}
}

func TestMatchSimple_PseudoFirstLastNthIgnored(t *testing.T) {
	// :first, :last, :nth are collection-level pseudos — MatchSimple should
	// NOT filter on them (they're handled by the query/selection layer).
	m, err := Compile("AXButton:first")
	if err != nil {
		t.Fatal(err)
	}

	el := &mockElement{role: "AXButton"}
	if !m.MatchSimple(el) {
		t.Fatal(":first should be ignored in MatchSimple — element should match")
	}

	m2, err := Compile("AXButton:last")
	if err != nil {
		t.Fatal(err)
	}
	if !m2.MatchSimple(el) {
		t.Fatal(":last should be ignored in MatchSimple")
	}

	m3, err := Compile("AXButton:nth(2)")
	if err != nil {
		t.Fatal(err)
	}
	if !m3.MatchSimple(el) {
		t.Fatal(":nth should be ignored in MatchSimple")
	}
}

func TestMatchSimple_MultiplePseudos(t *testing.T) {
	m, err := Compile("AXButton:enabled:visible")
	if err != nil {
		t.Fatal(err)
	}

	both := &mockElement{role: "AXButton", enabled: true, visible: true}
	onlyEnabled := &mockElement{role: "AXButton", enabled: true, visible: false}
	onlyVisible := &mockElement{role: "AXButton", enabled: false, visible: true}

	if !m.MatchSimple(both) {
		t.Fatal("expected match for both enabled+visible")
	}
	if m.MatchSimple(onlyEnabled) {
		t.Fatal("expected no match: only enabled")
	}
	if m.MatchSimple(onlyVisible) {
		t.Fatal("expected no match: only visible")
	}
}

// --- MatchSimple: combined attrs + pseudos ---

func TestMatchSimple_AttrAndPseudo(t *testing.T) {
	m, err := Compile(`AXButton[title="Save"]:enabled`)
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXButton", attrs: map[string]string{"title": "Save"}, enabled: true}
	noAttr := &mockElement{role: "AXButton", attrs: map[string]string{"title": "Cancel"}, enabled: true}
	noPseudo := &mockElement{role: "AXButton", attrs: map[string]string{"title": "Save"}, enabled: false}

	if !m.MatchSimple(ok) {
		t.Fatal("expected match for attr+pseudo")
	}
	if m.MatchSimple(noAttr) {
		t.Fatal("expected no match: wrong title")
	}
	if m.MatchSimple(noPseudo) {
		t.Fatal("expected no match: not enabled")
	}
}

// --- MatchSimple: compound selectors (leaf matching) ---

func TestMatchSimple_CompoundUsesLeaf(t *testing.T) {
	// "AXWindow AXTable > AXRow" — MatchSimple should match the LEAF (AXRow)
	m, err := Compile("AXWindow AXTable > AXRow")
	if err != nil {
		t.Fatal(err)
	}

	row := &mockElement{role: "AXRow"}
	table := &mockElement{role: "AXTable"}
	window := &mockElement{role: "AXWindow"}

	if !m.MatchSimple(row) {
		t.Fatal("compound selector should match leaf role AXRow")
	}
	if m.MatchSimple(table) {
		t.Fatal("should not match non-leaf AXTable")
	}
	if m.MatchSimple(window) {
		t.Fatal("should not match non-leaf AXWindow")
	}
}

// --- MatchSimple: group selectors (any match) ---

func TestMatchSimple_GroupAnyMatch(t *testing.T) {
	m, err := Compile("AXButton, AXMenuItem")
	if err != nil {
		t.Fatal(err)
	}

	btn := &mockElement{role: "AXButton"}
	menu := &mockElement{role: "AXMenuItem"}
	txt := &mockElement{role: "AXStaticText"}

	if !m.MatchSimple(btn) {
		t.Fatal("group should match AXButton")
	}
	if !m.MatchSimple(menu) {
		t.Fatal("group should match AXMenuItem")
	}
	if m.MatchSimple(txt) {
		t.Fatal("group should not match AXStaticText")
	}
}

func TestMatchSimple_GroupWithAttrs(t *testing.T) {
	m, err := Compile(`AXButton[title="OK"], AXMenuItem[title="Paste"]`)
	if err != nil {
		t.Fatal(err)
	}

	btn := &mockElement{role: "AXButton", attrs: map[string]string{"title": "OK"}}
	menu := &mockElement{role: "AXMenuItem", attrs: map[string]string{"title": "Paste"}}
	wrongBtn := &mockElement{role: "AXButton", attrs: map[string]string{"title": "Cancel"}}

	if !m.MatchSimple(btn) {
		t.Fatal("should match OK button")
	}
	if !m.MatchSimple(menu) {
		t.Fatal("should match Paste menu item")
	}
	if m.MatchSimple(wrongBtn) {
		t.Fatal("should not match Cancel button")
	}
}

// --- Matcher.Group() ---

func TestMatcher_Group(t *testing.T) {
	m, err := Compile("AXButton, AXWindow > AXSheet")
	if err != nil {
		t.Fatal(err)
	}

	g := m.Group()
	if g == nil {
		t.Fatal("Group() should not return nil")
	}
	if len(g.Selectors) != 2 {
		t.Fatalf("expected 2 selectors in group, got %d", len(g.Selectors))
	}

	// Verify the first selector matches AXButton.
	btn := &mockElement{role: "AXButton"}
	if !m.MatchSimple(btn) {
		t.Fatal("group matcher should match AXButton (first alternative)")
	}

	// Verify AXSheet alone doesn't match (needs AXWindow parent context,
	// but MatchSimple only checks simple/compound, not hierarchy).
	// AXSheet should still match the second branch's last compound.
	sheet := &mockElement{role: "AXSheet"}
	if !m.MatchSimple(sheet) {
		t.Fatal("group matcher should match AXSheet (second alternative's last compound)")
	}

	// Verify non-matching role.
	text := &mockElement{role: "AXStaticText"}
	if m.MatchSimple(text) {
		t.Fatal("group matcher should not match AXStaticText")
	}
}

// --- Edge cases ---

func TestMatchSimple_WildcardWithAttr(t *testing.T) {
	m, err := Compile(`*[description="hello"]`)
	if err != nil {
		t.Fatal(err)
	}

	ok := &mockElement{role: "AXAnything", attrs: map[string]string{"description": "hello"}}
	no := &mockElement{role: "AXAnything", attrs: map[string]string{"description": "world"}}

	if !m.MatchSimple(ok) {
		t.Fatal("wildcard+attr should match")
	}
	if m.MatchSimple(no) {
		t.Fatal("wildcard+attr should not match wrong value")
	}
}

func TestMatchSimple_EmptyAttrValue(t *testing.T) {
	m, err := Compile(`AXButton[title=""]`)
	if err != nil {
		t.Fatal(err)
	}

	empty := &mockElement{role: "AXButton", attrs: map[string]string{"title": ""}}
	nonempty := &mockElement{role: "AXButton", attrs: map[string]string{"title": "X"}}
	missing := &mockElement{role: "AXButton"} // nil attrs -> GetAttr returns ""

	if !m.MatchSimple(empty) {
		t.Fatal("should match empty title")
	}
	if m.MatchSimple(nonempty) {
		t.Fatal("should not match non-empty title")
	}
	if !m.MatchSimple(missing) {
		t.Fatal("missing attr (empty string) should match empty value")
	}
}

// --- MustCompile ---

func TestMustCompile_Valid(t *testing.T) {
	m := MustCompile("AXButton")
	if m == nil {
		t.Fatal("MustCompile returned nil for valid selector")
	}
	if !m.MatchSimple(&mockElement{role: "AXButton"}) {
		t.Fatal("MustCompile result should match")
	}
}

func TestMustCompile_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustCompile should panic on invalid selector")
		}
	}()
	MustCompile("")
}

// --- matchAttr: unknown op (default branch) ---

func TestMatchSimple_UnknownAttrOp(t *testing.T) {
	// Manually construct a compiled selector with an invalid AttrOp to hit the default branch
	group := &SelectorGroup{
		Selectors: []*CompoundSelector{
			{
				Head: &SimpleSelector{
					Role: "AXButton",
					Attrs: []AttrMatcher{
						{Name: "title", Op: AttrOp(99), Value: "anything"},
					},
				},
			},
		},
	}
	cs := newCompiledSelector(group)
	el := &mockElement{role: "AXButton", attrs: map[string]string{"title": "anything"}}
	if cs.MatchSimple(el) {
		t.Fatal("unknown attr op should not match")
	}
}
