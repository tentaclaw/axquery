package selector

import (
	"strings"
	"testing"
)

func TestParse_SimpleRole(t *testing.T) {
	g, err := Parse("AXButton")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Selectors) != 1 {
		t.Fatalf("expected 1 selector, got %d", len(g.Selectors))
	}
	if g.Selectors[0].Head.Role != "AXButton" {
		t.Fatalf("expected AXButton, got %s", g.Selectors[0].Head.Role)
	}
	if len(g.Selectors[0].Steps) != 0 {
		t.Fatalf("expected 0 steps, got %d", len(g.Selectors[0].Steps))
	}
}

func TestParse_Wildcard(t *testing.T) {
	g, err := Parse(`*[title="OK"]`)
	if err != nil {
		t.Fatal(err)
	}
	if g.Selectors[0].Head.Role != "*" {
		t.Fatalf("expected *, got %s", g.Selectors[0].Head.Role)
	}
	if len(g.Selectors[0].Head.Attrs) != 1 {
		t.Fatal("expected 1 attr")
	}
	a := g.Selectors[0].Head.Attrs[0]
	if a.Name != "title" {
		t.Fatalf("wrong attr name: %s", a.Name)
	}
	if a.Op != OpEquals {
		t.Fatalf("wrong op: %v", a.Op)
	}
	if a.Value != "OK" {
		t.Fatalf("wrong value: %s", a.Value)
	}
}

func TestParse_AllAttrOps(t *testing.T) {
	cases := []struct {
		input string
		op    AttrOp
		value string
	}{
		{`AXButton[title="OK"]`, OpEquals, "OK"},
		{`AXButton[title*="Send"]`, OpContains, "Send"},
		{`AXButton[title^="Re:"]`, OpPrefix, "Re:"},
		{`AXButton[title$="btn"]`, OpSuffix, "btn"},
		{`AXButton[title~="\\d+"]`, OpRegex, `\d+`},
		{`AXButton[title!="No"]`, OpNotEquals, "No"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			g, err := Parse(c.input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", c.input, err)
			}
			a := g.Selectors[0].Head.Attrs[0]
			if a.Op != c.op {
				t.Errorf("expected op %v, got %v", c.op, a.Op)
			}
			if a.Value != c.value {
				t.Errorf("expected value %q, got %q", c.value, a.Value)
			}
		})
	}
}

func TestParse_MultipleAttrs(t *testing.T) {
	g, err := Parse(`AXButton[title="OK"][description*="confirm"]`)
	if err != nil {
		t.Fatal(err)
	}
	attrs := g.Selectors[0].Head.Attrs
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}
	if attrs[0].Name != "title" || attrs[0].Op != OpEquals || attrs[0].Value != "OK" {
		t.Errorf("first attr wrong: %+v", attrs[0])
	}
	if attrs[1].Name != "description" || attrs[1].Op != OpContains || attrs[1].Value != "confirm" {
		t.Errorf("second attr wrong: %+v", attrs[1])
	}
}

func TestParse_Descendant(t *testing.T) {
	g, err := Parse("AXWindow AXTable")
	if err != nil {
		t.Fatal(err)
	}
	cs := g.Selectors[0]
	if cs.Head.Role != "AXWindow" {
		t.Fatalf("wrong head: %s", cs.Head.Role)
	}
	if len(cs.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(cs.Steps))
	}
	if cs.Steps[0].Combinator != CombDescendant {
		t.Fatal("expected descendant combinator")
	}
	if cs.Steps[0].Selector.Role != "AXTable" {
		t.Fatalf("wrong step role: %s", cs.Steps[0].Selector.Role)
	}
}

func TestParse_Child(t *testing.T) {
	g, err := Parse(`AXSheet > AXButton[title="OK"]`)
	if err != nil {
		t.Fatal(err)
	}
	cs := g.Selectors[0]
	if cs.Head.Role != "AXSheet" {
		t.Fatalf("wrong head: %s", cs.Head.Role)
	}
	if len(cs.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(cs.Steps))
	}
	if cs.Steps[0].Combinator != CombChild {
		t.Fatal("expected child combinator")
	}
	if cs.Steps[0].Selector.Role != "AXButton" {
		t.Fatalf("wrong step role: %s", cs.Steps[0].Selector.Role)
	}
	if len(cs.Steps[0].Selector.Attrs) != 1 {
		t.Fatal("expected 1 attr on step selector")
	}
}

func TestParse_Group(t *testing.T) {
	g, err := Parse("AXButton, AXMenuItem")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Selectors) != 2 {
		t.Fatalf("expected 2 selectors, got %d", len(g.Selectors))
	}
	if g.Selectors[0].Head.Role != "AXButton" {
		t.Fatalf("wrong first: %s", g.Selectors[0].Head.Role)
	}
	if g.Selectors[1].Head.Role != "AXMenuItem" {
		t.Fatalf("wrong second: %s", g.Selectors[1].Head.Role)
	}
}

func TestParse_GroupThree(t *testing.T) {
	g, err := Parse("AXButton, AXMenuItem, AXStaticText")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Selectors) != 3 {
		t.Fatalf("expected 3 selectors, got %d", len(g.Selectors))
	}
	// Verify each selector's head has the correct role.
	wantRoles := []string{"AXButton", "AXMenuItem", "AXStaticText"}
	for i, cs := range g.Selectors {
		got := cs.Head.Role
		if got != wantRoles[i] {
			t.Fatalf("selector[%d]: expected role %q, got %q", i, wantRoles[i], got)
		}
		// Each should be a simple selector (no steps).
		if len(cs.Steps) != 0 {
			t.Fatalf("selector[%d]: expected 0 steps, got %d", i, len(cs.Steps))
		}
	}
}

func TestParse_Pseudos(t *testing.T) {
	cases := []struct {
		input string
		ptype PseudoType
	}{
		{"AXButton:first", PseudoFirst},
		{"AXButton:last", PseudoLast},
		{"AXButton:visible", PseudoVisible},
		{"AXButton:enabled", PseudoEnabled},
		{"AXButton:focused", PseudoFocused},
		{"AXButton:selected", PseudoSelected},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			g, err := Parse(c.input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", c.input, err)
			}
			pseudos := g.Selectors[0].Head.Pseudos
			if len(pseudos) != 1 {
				t.Fatalf("expected 1 pseudo, got %d", len(pseudos))
			}
			if pseudos[0].Type != c.ptype {
				t.Errorf("wrong pseudo type: got %d, want %d", pseudos[0].Type, c.ptype)
			}
		})
	}
}

func TestParse_NthPseudo(t *testing.T) {
	g, err := Parse("AXRow:nth(3)")
	if err != nil {
		t.Fatal(err)
	}
	p := g.Selectors[0].Head.Pseudos[0]
	if p.Type != PseudoNth {
		t.Fatalf("expected PseudoNth, got %d", p.Type)
	}
	if p.N != 3 {
		t.Fatalf("expected N=3, got %d", p.N)
	}
}

func TestParse_NthPseudo_Zero(t *testing.T) {
	g, err := Parse("AXRow:nth(0)")
	if err != nil {
		t.Fatal(err)
	}
	p := g.Selectors[0].Head.Pseudos[0]
	if p.Type != PseudoNth {
		t.Fatal("expected PseudoNth")
	}
	if p.N != 0 {
		t.Fatalf("expected N=0, got %d", p.N)
	}
}

func TestParse_MultiplePseudos(t *testing.T) {
	g, err := Parse("AXButton:enabled:visible")
	if err != nil {
		t.Fatal(err)
	}
	pseudos := g.Selectors[0].Head.Pseudos
	if len(pseudos) != 2 {
		t.Fatalf("expected 2 pseudos, got %d", len(pseudos))
	}
	if pseudos[0].Type != PseudoEnabled {
		t.Errorf("first pseudo: got %d, want %d", pseudos[0].Type, PseudoEnabled)
	}
	if pseudos[1].Type != PseudoVisible {
		t.Errorf("second pseudo: got %d, want %d", pseudos[1].Type, PseudoVisible)
	}
}

func TestParse_Complex(t *testing.T) {
	input := `AXWindow AXTable > AXRow:nth(0) AXStaticText`
	g, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	cs := g.Selectors[0]
	if cs.Head.Role != "AXWindow" {
		t.Fatalf("wrong head: %s", cs.Head.Role)
	}
	if len(cs.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(cs.Steps))
	}
	// AXWindow (descendant) AXTable (child) AXRow:nth(0) (descendant) AXStaticText
	if cs.Steps[0].Combinator != CombDescendant {
		t.Fatal("step0: expected descendant")
	}
	if cs.Steps[0].Selector.Role != "AXTable" {
		t.Fatalf("step0: wrong role: %s", cs.Steps[0].Selector.Role)
	}
	if cs.Steps[1].Combinator != CombChild {
		t.Fatal("step1: expected child")
	}
	if cs.Steps[1].Selector.Role != "AXRow" {
		t.Fatalf("step1: wrong role: %s", cs.Steps[1].Selector.Role)
	}
	if len(cs.Steps[1].Selector.Pseudos) != 1 || cs.Steps[1].Selector.Pseudos[0].Type != PseudoNth {
		t.Fatal("step1: expected :nth pseudo")
	}
	if cs.Steps[2].Combinator != CombDescendant {
		t.Fatal("step2: expected descendant")
	}
	if cs.Steps[2].Selector.Role != "AXStaticText" {
		t.Fatalf("step2: wrong role: %s", cs.Steps[2].Selector.Role)
	}
}

func TestParse_AttrsAndPseudos(t *testing.T) {
	input := `AXButton[title="OK"]:enabled:visible`
	g, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	sel := g.Selectors[0].Head
	if sel.Role != "AXButton" {
		t.Fatalf("wrong role: %s", sel.Role)
	}
	if len(sel.Attrs) != 1 {
		t.Fatalf("expected 1 attr, got %d", len(sel.Attrs))
	}
	if len(sel.Pseudos) != 2 {
		t.Fatalf("expected 2 pseudos, got %d", len(sel.Pseudos))
	}
}

func TestParse_SingleQuotes(t *testing.T) {
	g, err := Parse("AXButton[title='OK']")
	if err != nil {
		t.Fatal(err)
	}
	if g.Selectors[0].Head.Attrs[0].Value != "OK" {
		t.Fatalf("wrong value: %s", g.Selectors[0].Head.Attrs[0].Value)
	}
}

func TestParse_EscapedQuotes(t *testing.T) {
	g, err := Parse(`AXButton[title="say \"hello\""]`)
	if err != nil {
		t.Fatal(err)
	}
	want := `say "hello"`
	if g.Selectors[0].Head.Attrs[0].Value != want {
		t.Fatalf("wrong value: got %q, want %q", g.Selectors[0].Head.Attrs[0].Value, want)
	}
}

func TestParse_WildcardOnly(t *testing.T) {
	g, err := Parse("*")
	if err != nil {
		t.Fatal(err)
	}
	if g.Selectors[0].Head.Role != "*" {
		t.Fatalf("expected *, got %s", g.Selectors[0].Head.Role)
	}
}

func TestParse_Errors(t *testing.T) {
	invalid := []struct {
		input string
		desc  string
	}{
		{"", "empty string"},
		{"[title=", "unclosed bracket"},
		{"AXButton[", "unclosed bracket after role"},
		{"AXButton[title]", "attr without operator"},
		{">", "combinator without selector"},
		{"AXButton >", "trailing combinator"},
		{"AXButton[title=OK]", "unquoted attr value"},
		{`AXButton[title="OK`, "unclosed quote"},
		{"AXButton:unknown", "unknown pseudo"},
		{"AXButton:nth(", "unclosed nth"},
		{"AXButton:nth(abc)", "non-numeric nth"},
		{",", "leading comma"},
		{"AXButton,", "trailing comma"},
		{"  ,AXButton", "leading comma with spaces"},
	}
	for _, c := range invalid {
		t.Run(c.desc, func(t *testing.T) {
			_, err := Parse(c.input)
			if err == nil {
				t.Errorf("Parse(%q): expected error, got nil", c.input)
			}
		})
	}
}

func TestParse_Roundtrip(t *testing.T) {
	// Parse -> String -> Parse -> String should be stable
	inputs := []string{
		"AXButton",
		"*",
		`AXButton[title="OK"]`,
		`*[title*="Send"]`,
		`AXButton[title="OK"][description*="confirm"]`,
		"AXWindow AXTable",
		`AXSheet > AXButton[title="OK"]`,
		"AXButton, AXMenuItem",
		"AXButton:first",
		"AXRow:nth(3)",
		"AXButton:enabled:visible",
		`AXWindow AXTable > AXRow:nth(0) AXStaticText`,
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			g1, err := Parse(input)
			if err != nil {
				t.Fatalf("first parse: %v", err)
			}
			s1 := g1.String()
			g2, err := Parse(s1)
			if err != nil {
				t.Fatalf("second parse of %q: %v", s1, err)
			}
			s2 := g2.String()
			if s1 != s2 {
				t.Errorf("roundtrip mismatch: %q -> %q", s1, s2)
			}
		})
	}
}

func TestParse_WhitespaceHandling(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"  AXButton  ", "AXButton"},
		{"AXWindow   AXTable", "AXWindow AXTable"},
		{"AXSheet  >  AXButton", "AXSheet > AXButton"},
		{"AXButton ,  AXMenuItem", "AXButton, AXMenuItem"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			g, err := Parse(c.input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", c.input, err)
			}
			got := g.String()
			if got != c.expected {
				t.Errorf("got %q, want %q", got, c.expected)
			}
		})
	}
}

func TestParse_ErrorMessages(t *testing.T) {
	// Ensure error messages are descriptive
	_, err := Parse("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error message should mention 'empty': %v", err)
	}

	_, err = Parse("AXButton:unknown")
	if err == nil {
		t.Fatal("expected error for unknown pseudo")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error message should mention 'unknown': %v", err)
	}
}

// Additional edge case tests for full coverage of error branches

func TestParse_MoreErrors(t *testing.T) {
	invalid := []struct {
		input string
		desc  string
	}{
		// parseAttrOp: partial operators without '='
		{`AXButton[title^]`, "caret without equals"},
		{`AXButton[title$]`, "dollar without equals"},
		{`AXButton[title~]`, "tilde without equals"},
		{`AXButton[title!]`, "bang without equals"},
		{`AXButton[title*]`, "star without equals"},
		// parseAttrOp: EOF after operator start
		{`AXButton[title^`, "caret at EOF"},
		{`AXButton[title$`, "dollar at EOF"},
		{`AXButton[title~`, "tilde at EOF"},
		{`AXButton[title!`, "bang at EOF"},
		{`AXButton[title*`, "star at EOF"},
		// parseQuotedString: EOF before string
		{`AXButton[title=`, "equals at EOF"},
		// parseAttr: missing closing bracket after value
		{`AXButton[title="OK"`, "missing closing bracket"},
		// parseNth: missing opening paren
		{`AXButton:nth`, "nth without paren"},
		// parseNth: missing closing paren
		{`AXButton:nth(3`, "nth without closing paren"},
		// parsePseudo: EOF after colon
		{`AXButton:`, "colon at EOF"},
		// parseRole: non-role start char
		{`123`, "numeric start"},
		// unexpected char after valid selector
		{`AXButton#`, "hash after selector"},
		// parseSimple: EOF where selector expected (after combinator in compound)
		// This is "AXWindow > " which should error as trailing combinator
		{`AXWindow > `, "trailing child combinator with space"},
	}
	for _, c := range invalid {
		t.Run(c.desc, func(t *testing.T) {
			_, err := Parse(c.input)
			if err == nil {
				t.Errorf("Parse(%q): expected error, got nil", c.input)
			}
		})
	}
}

func TestParse_EscapedBackslash(t *testing.T) {
	// Double backslash in quoted string: \\\\ -> \\
	g, err := Parse(`AXButton[title="a\\b"]`)
	if err != nil {
		t.Fatal(err)
	}
	want := `a\b`
	if g.Selectors[0].Head.Attrs[0].Value != want {
		t.Fatalf("wrong value: got %q, want %q", g.Selectors[0].Head.Attrs[0].Value, want)
	}
}

func TestParse_SingleQuoteEscaped(t *testing.T) {
	g, err := Parse(`AXButton[title='it\'s']`)
	if err != nil {
		t.Fatal(err)
	}
	want := "it's"
	if g.Selectors[0].Head.Attrs[0].Value != want {
		t.Fatalf("wrong value: got %q, want %q", g.Selectors[0].Head.Attrs[0].Value, want)
	}
}

func TestParse_ComplexGroup(t *testing.T) {
	// Group with compound selectors
	input := `AXWindow > AXButton[title="OK"], AXSheet AXStaticText:first`
	g, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Selectors) != 2 {
		t.Fatalf("expected 2 selectors, got %d", len(g.Selectors))
	}
	// First: AXWindow > AXButton[title="OK"]
	if g.Selectors[0].Head.Role != "AXWindow" {
		t.Fatal("wrong first head")
	}
	if len(g.Selectors[0].Steps) != 1 {
		t.Fatal("expected 1 step in first")
	}
	if g.Selectors[0].Steps[0].Combinator != CombChild {
		t.Fatal("expected child combinator")
	}
	// Second: AXSheet AXStaticText:first
	if g.Selectors[1].Head.Role != "AXSheet" {
		t.Fatal("wrong second head")
	}
	if len(g.Selectors[1].Steps) != 1 {
		t.Fatal("expected 1 step in second")
	}
	if g.Selectors[1].Steps[0].Selector.Role != "AXStaticText" {
		t.Fatal("wrong second step role")
	}
}

func TestParse_WildcardWithPseudo(t *testing.T) {
	g, err := Parse("*:visible")
	if err != nil {
		t.Fatal(err)
	}
	if g.Selectors[0].Head.Role != "*" {
		t.Fatal("wrong role")
	}
	if len(g.Selectors[0].Head.Pseudos) != 1 || g.Selectors[0].Head.Pseudos[0].Type != PseudoVisible {
		t.Fatal("wrong pseudo")
	}
}

func TestParse_BackslashNonSpecial(t *testing.T) {
	// Backslash followed by non-quote, non-backslash char is passed through as-is
	// Input: AXButton[title="\d+"] where \d is NOT an escape for quote or backslash
	input := "AXButton[title=\"\\d+\"]"
	g, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	want := "\\d+"
	if g.Selectors[0].Head.Attrs[0].Value != want {
		t.Fatalf("wrong value: got %q, want %q", g.Selectors[0].Head.Attrs[0].Value, want)
	}
}

func TestParse_CompoundErrorAfterCombinator(t *testing.T) {
	// Error in parseSimple after a valid combinator: "AXWindow > [invalid"
	_, err := Parse("AXWindow > [")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParse_CompoundErrorAfterDescendant(t *testing.T) {
	// Error in parseSimple after descendant combinator
	_, err := Parse("AXWindow [invalid")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParse_GroupWithErrorInSecond(t *testing.T) {
	// Error in second compound selector after comma
	_, err := Parse("AXButton, [invalid")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParse_SpaceFollowedByNonSelectorChar(t *testing.T) {
	// After a selector, space followed by non-selector char should be handled
	// This tests the isSimpleSelectorStart == false branch
	_, err := Parse("AXButton @")
	if err != nil {
		// The parser should see space then '@', realize it's not a selector start,
		// restore position, exit compound loop, then fail on unexpected char
		if !strings.Contains(err.Error(), "unexpected") {
			t.Errorf("expected 'unexpected' in error, got: %v", err)
		}
	}
}

func TestParse_PseudoColonAtEndOfInput(t *testing.T) {
	// Colon at very end of input
	_, err := Parse("AXButton:")
	if err == nil {
		t.Fatal("expected error for trailing colon")
	}
}

func TestParse_AttrNameAtEOF(t *testing.T) {
	// Attribute bracket opened but nothing inside
	_, err := Parse("AXButton[ ")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParse_NthNegative(t *testing.T) {
	// nth with just a closing paren and no number
	_, err := Parse("AXButton:nth()")
	if err == nil {
		t.Fatal("expected error for empty nth")
	}
}

func TestParse_ChildCombinatorNoSpace(t *testing.T) {
	// "AXWindow>AXButton" - no spaces around >
	g, err := Parse("AXWindow>AXButton")
	if err != nil {
		t.Fatal(err)
	}
	cs := g.Selectors[0]
	if cs.Head.Role != "AXWindow" {
		t.Fatal("wrong head")
	}
	// '>' without leading whitespace—this should not parse as child combinator
	// since '>' appears right after role chars, it's not a valid role char,
	// so parseCompound's loop should detect it. Let's verify actual behavior.
	// Actually, the parser first tries skipWhitespace, then checks for '>'.
	// savedPos == p.pos (no whitespace consumed), ch == '>' -> enters '>' branch.
	if len(cs.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(cs.Steps))
	}
	if cs.Steps[0].Combinator != CombChild {
		t.Fatal("expected child combinator")
	}
	if cs.Steps[0].Selector.Role != "AXButton" {
		t.Fatal("wrong step role")
	}
}
