package selector

import (
	"fmt"
	"testing"
)

func TestAttrOp_String(t *testing.T) {
	cases := []struct {
		op   AttrOp
		want string
	}{
		{OpEquals, "="},
		{OpContains, "*="},
		{OpPrefix, "^="},
		{OpSuffix, "$="},
		{OpRegex, "~="},
		{OpNotEquals, "!="},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if got := c.op.String(); got != c.want {
				t.Errorf("AttrOp(%d).String() = %q, want %q", c.op, got, c.want)
			}
		})
	}
}

func TestAttrOp_String_Unknown(t *testing.T) {
	op := AttrOp(99)
	got := op.String()
	want := "unknown_op(99)"
	if got != want {
		t.Errorf("unknown AttrOp.String() = %q, want %q", got, want)
	}
}

func TestCombinator_String(t *testing.T) {
	cases := []struct {
		c    Combinator
		want string
	}{
		{CombDescendant, " "},
		{CombChild, " > "},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("comb_%d", c.c), func(t *testing.T) {
			if got := c.c.String(); got != c.want {
				t.Errorf("Combinator(%d).String() = %q, want %q", c.c, got, c.want)
			}
		})
	}
}

func TestCombinator_String_Unknown(t *testing.T) {
	c := Combinator(99)
	got := c.String()
	want := "unknown_comb(99)"
	if got != want {
		t.Errorf("unknown Combinator.String() = %q, want %q", got, want)
	}
}

func TestAttrMatcher_String(t *testing.T) {
	cases := []struct {
		am   AttrMatcher
		want string
	}{
		{AttrMatcher{Name: "title", Op: OpEquals, Value: "OK"}, `[title="OK"]`},
		{AttrMatcher{Name: "description", Op: OpContains, Value: "Send"}, `[description*="Send"]`},
		{AttrMatcher{Name: "value", Op: OpPrefix, Value: "Re:"}, `[value^="Re:"]`},
		{AttrMatcher{Name: "title", Op: OpSuffix, Value: "btn"}, `[title$="btn"]`},
		{AttrMatcher{Name: "title", Op: OpRegex, Value: `\d+`}, `[title~="\d+"]`},
		{AttrMatcher{Name: "title", Op: OpNotEquals, Value: "No"}, `[title!="No"]`},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if got := c.am.String(); got != c.want {
				t.Errorf("AttrMatcher.String() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestSimpleSelector_String(t *testing.T) {
	cases := []struct {
		name string
		sel  SimpleSelector
		want string
	}{
		{
			name: "role_only",
			sel:  SimpleSelector{Role: "AXButton"},
			want: "AXButton",
		},
		{
			name: "wildcard",
			sel:  SimpleSelector{Role: "*"},
			want: "*",
		},
		{
			name: "role_with_attr",
			sel: SimpleSelector{
				Role:  "AXButton",
				Attrs: []AttrMatcher{{Name: "title", Op: OpEquals, Value: "OK"}},
			},
			want: `AXButton[title="OK"]`,
		},
		{
			name: "wildcard_with_attr",
			sel: SimpleSelector{
				Role:  "*",
				Attrs: []AttrMatcher{{Name: "title", Op: OpContains, Value: "Send"}},
			},
			want: `*[title*="Send"]`,
		},
		{
			name: "multiple_attrs",
			sel: SimpleSelector{
				Role: "AXButton",
				Attrs: []AttrMatcher{
					{Name: "title", Op: OpEquals, Value: "OK"},
					{Name: "description", Op: OpContains, Value: "confirm"},
				},
			},
			want: `AXButton[title="OK"][description*="confirm"]`,
		},
		{
			name: "pseudo_first",
			sel: SimpleSelector{
				Role:    "AXButton",
				Pseudos: []Pseudo{{Type: PseudoFirst}},
			},
			want: "AXButton:first",
		},
		{
			name: "pseudo_last",
			sel: SimpleSelector{
				Role:    "AXRow",
				Pseudos: []Pseudo{{Type: PseudoLast}},
			},
			want: "AXRow:last",
		},
		{
			name: "pseudo_nth",
			sel: SimpleSelector{
				Role:    "AXRow",
				Pseudos: []Pseudo{{Type: PseudoNth, N: 3}},
			},
			want: "AXRow:nth(3)",
		},
		{
			name: "pseudo_visible",
			sel: SimpleSelector{
				Role:    "AXButton",
				Pseudos: []Pseudo{{Type: PseudoVisible}},
			},
			want: "AXButton:visible",
		},
		{
			name: "pseudo_enabled",
			sel: SimpleSelector{
				Role:    "AXButton",
				Pseudos: []Pseudo{{Type: PseudoEnabled}},
			},
			want: "AXButton:enabled",
		},
		{
			name: "pseudo_focused",
			sel: SimpleSelector{
				Role:    "AXTextField",
				Pseudos: []Pseudo{{Type: PseudoFocused}},
			},
			want: "AXTextField:focused",
		},
		{
			name: "pseudo_selected",
			sel: SimpleSelector{
				Role:    "AXCheckBox",
				Pseudos: []Pseudo{{Type: PseudoSelected}},
			},
			want: "AXCheckBox:selected",
		},
		{
			name: "role_attrs_pseudos",
			sel: SimpleSelector{
				Role:    "AXButton",
				Attrs:   []AttrMatcher{{Name: "title", Op: OpEquals, Value: "OK"}},
				Pseudos: []Pseudo{{Type: PseudoEnabled}, {Type: PseudoVisible}},
			},
			want: `AXButton[title="OK"]:enabled:visible`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.sel.String(); got != c.want {
				t.Errorf("SimpleSelector.String() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestCompoundSelector_String(t *testing.T) {
	// AXWindow AXTable > AXRow:nth(0) AXStaticText
	cs := CompoundSelector{
		Head: &SimpleSelector{Role: "AXWindow"},
		Steps: []CompoundStep{
			{Combinator: CombDescendant, Selector: &SimpleSelector{Role: "AXTable"}},
			{Combinator: CombChild, Selector: &SimpleSelector{
				Role:    "AXRow",
				Pseudos: []Pseudo{{Type: PseudoNth, N: 0}},
			}},
			{Combinator: CombDescendant, Selector: &SimpleSelector{Role: "AXStaticText"}},
		},
	}
	want := "AXWindow AXTable > AXRow:nth(0) AXStaticText"
	if got := cs.String(); got != want {
		t.Errorf("CompoundSelector.String() = %q, want %q", got, want)
	}
}

func TestCompoundSelector_String_HeadOnly(t *testing.T) {
	cs := CompoundSelector{
		Head: &SimpleSelector{Role: "AXButton"},
	}
	want := "AXButton"
	if got := cs.String(); got != want {
		t.Errorf("CompoundSelector.String() = %q, want %q", got, want)
	}
}

func TestSelectorGroup_String(t *testing.T) {
	sg := SelectorGroup{
		Selectors: []*CompoundSelector{
			{Head: &SimpleSelector{Role: "AXButton"}},
			{Head: &SimpleSelector{Role: "AXMenuItem"}},
		},
	}
	want := "AXButton, AXMenuItem"
	if got := sg.String(); got != want {
		t.Errorf("SelectorGroup.String() = %q, want %q", got, want)
	}
}

func TestSelectorGroup_String_Single(t *testing.T) {
	sg := SelectorGroup{
		Selectors: []*CompoundSelector{
			{Head: &SimpleSelector{Role: "AXButton"}},
		},
	}
	want := "AXButton"
	if got := sg.String(); got != want {
		t.Errorf("SelectorGroup.String() = %q, want %q", got, want)
	}
}

func TestPseudoType_Constants(t *testing.T) {
	// Ensure pseudo constants are distinct
	seen := make(map[PseudoType]bool)
	for _, pt := range []PseudoType{PseudoFirst, PseudoLast, PseudoNth, PseudoVisible, PseudoEnabled, PseudoFocused, PseudoSelected} {
		if seen[pt] {
			t.Errorf("duplicate PseudoType value: %d", pt)
		}
		seen[pt] = true
	}
}

func TestAttrOp_Constants(t *testing.T) {
	// Ensure attr op constants are distinct
	seen := make(map[AttrOp]bool)
	for _, op := range []AttrOp{OpEquals, OpContains, OpPrefix, OpSuffix, OpRegex, OpNotEquals} {
		if seen[op] {
			t.Errorf("duplicate AttrOp value: %d", op)
		}
		seen[op] = true
	}
}

func TestCombinator_Constants(t *testing.T) {
	seen := make(map[Combinator]bool)
	for _, c := range []Combinator{CombDescendant, CombChild} {
		if seen[c] {
			t.Errorf("duplicate Combinator value: %d", c)
		}
		seen[c] = true
	}
}
