// Package selector implements a CSS-like selector parser for AX elements.
// It is internal to axquery and not published as a separate package.
package selector

import (
	"fmt"
	"strings"
)

// AttrOp is an attribute matching operator.
type AttrOp int

const (
	OpEquals    AttrOp = iota // =  exact match
	OpContains                // *= contains
	OpPrefix                  // ^= prefix
	OpSuffix                  // $= suffix
	OpRegex                   // ~= regex
	OpNotEquals               // != not equals
)

var attrOpStrings = map[AttrOp]string{
	OpEquals:    "=",
	OpContains:  "*=",
	OpPrefix:    "^=",
	OpSuffix:    "$=",
	OpRegex:     "~=",
	OpNotEquals: "!=",
}

func (op AttrOp) String() string {
	if s, ok := attrOpStrings[op]; ok {
		return s
	}
	return fmt.Sprintf("unknown_op(%d)", int(op))
}

// Combinator is a selector combinator.
type Combinator int

const (
	CombDescendant Combinator = iota // space - descendant
	CombChild                        // >     - direct child
)

func (c Combinator) String() string {
	switch c {
	case CombDescendant:
		return " "
	case CombChild:
		return " > "
	default:
		return fmt.Sprintf("unknown_comb(%d)", int(c))
	}
}

// PseudoType is a pseudo-selector type.
type PseudoType int

const (
	PseudoFirst PseudoType = iota
	PseudoLast
	PseudoNth
	PseudoVisible
	PseudoEnabled
	PseudoFocused
	PseudoSelected
)

// Pseudo represents a pseudo-selector (e.g., :first, :nth(3), :visible).
type Pseudo struct {
	Type PseudoType
	N    int // used for :nth(N)
}

// AttrMatcher is a single attribute match condition.
type AttrMatcher struct {
	Name  string // attribute name (e.g., "title", "description")
	Op    AttrOp // matching operator
	Value string // match value
}

func (am AttrMatcher) String() string {
	return fmt.Sprintf(`[%s%s"%s"]`, am.Name, am.Op, am.Value)
}

// SimpleSelector represents a single selector (role + attributes + pseudos).
type SimpleSelector struct {
	Role    string        // AX role name, "*" for wildcard
	Attrs   []AttrMatcher // attribute match conditions
	Pseudos []Pseudo      // pseudo-selectors
}

func (s *SimpleSelector) String() string {
	var b strings.Builder
	b.WriteString(s.Role)
	for _, a := range s.Attrs {
		b.WriteString(a.String())
	}
	for _, p := range s.Pseudos {
		switch p.Type {
		case PseudoFirst:
			b.WriteString(":first")
		case PseudoLast:
			b.WriteString(":last")
		case PseudoNth:
			fmt.Fprintf(&b, ":nth(%d)", p.N)
		case PseudoVisible:
			b.WriteString(":visible")
		case PseudoEnabled:
			b.WriteString(":enabled")
		case PseudoFocused:
			b.WriteString(":focused")
		case PseudoSelected:
			b.WriteString(":selected")
		}
	}
	return b.String()
}

// CompoundStep is a step in a compound selector (combinator + simple selector).
type CompoundStep struct {
	Combinator Combinator
	Selector   *SimpleSelector
}

// CompoundSelector is a compound selector (e.g., "AXWindow AXTable > AXRow").
type CompoundSelector struct {
	Head  *SimpleSelector // first selector
	Steps []CompoundStep  // subsequent steps
}

func (cs *CompoundSelector) String() string {
	var b strings.Builder
	b.WriteString(cs.Head.String())
	for _, step := range cs.Steps {
		b.WriteString(step.Combinator.String())
		b.WriteString(step.Selector.String())
	}
	return b.String()
}

// SelectorGroup is a comma-separated group of selectors (e.g., "AXButton, AXMenuItem").
type SelectorGroup struct {
	Selectors []*CompoundSelector
}

func (sg *SelectorGroup) String() string {
	parts := make([]string, len(sg.Selectors))
	for i, s := range sg.Selectors {
		parts[i] = s.String()
	}
	return strings.Join(parts, ", ")
}
