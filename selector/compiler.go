package selector

import (
	"regexp"
	"strings"
)

// Compile parses a selector string and compiles it into a Matcher.
// The returned Matcher can be reused for matching multiple elements.
func Compile(sel string) (Matcher, error) {
	group, err := Parse(sel)
	if err != nil {
		return nil, err
	}
	return newCompiledSelector(group), nil
}

// MustCompile is like Compile but panics on error.
func MustCompile(sel string) Matcher {
	m, err := Compile(sel)
	if err != nil {
		panic("selector: Compile(" + sel + "): " + err.Error())
	}
	return m
}

// compiledSelector is the concrete Matcher implementation.
type compiledSelector struct {
	group      *SelectorGroup
	regexCache map[string]*regexp.Regexp
}

func newCompiledSelector(group *SelectorGroup) *compiledSelector {
	return &compiledSelector{
		group:      group,
		regexCache: make(map[string]*regexp.Regexp),
	}
}

// MatchSimple tests whether a single element matches any selector in the group.
// For compound selectors (with combinators), only the LEAF (last) simple selector
// is tested. Combinator validation (parent/ancestor) is handled at the query/selection layer.
//
// Collection-level pseudos (:first, :last, :nth) are ignored here — they require
// knowledge of the result set and are applied by the query engine.
func (cs *compiledSelector) MatchSimple(el Matchable) bool {
	for _, compound := range cs.group.Selectors {
		leaf := compound.Head
		if len(compound.Steps) > 0 {
			leaf = compound.Steps[len(compound.Steps)-1].Selector
		}
		if cs.matchSimpleSelector(leaf, el) {
			return true
		}
	}
	return false
}

// Group returns the parsed AST for use by the query engine.
func (cs *compiledSelector) Group() *SelectorGroup {
	return cs.group
}

// matchSimpleSelector checks role, attributes, and boolean pseudos.
func (cs *compiledSelector) matchSimpleSelector(s *SimpleSelector, el Matchable) bool {
	// Role match
	if s.Role != "*" && el.GetRole() != s.Role {
		return false
	}

	// Attribute matches — all must pass
	for _, attr := range s.Attrs {
		val := el.GetAttr(attr.Name)
		if !cs.matchAttr(attr, val) {
			return false
		}
	}

	// Boolean pseudo matches
	for _, p := range s.Pseudos {
		switch p.Type {
		case PseudoEnabled:
			if !el.IsEnabled() {
				return false
			}
		case PseudoVisible:
			if !el.IsVisible() {
				return false
			}
		case PseudoFocused:
			if !el.IsFocused() {
				return false
			}
		case PseudoSelected:
			if !el.IsSelected() {
				return false
			}
		case PseudoFirst, PseudoLast, PseudoNth:
			// Collection-level pseudos: skip, handled by query engine
		}
	}

	return true
}

// matchAttr evaluates a single attribute match condition.
func (cs *compiledSelector) matchAttr(am AttrMatcher, val string) bool {
	switch am.Op {
	case OpEquals:
		return val == am.Value
	case OpContains:
		return strings.Contains(val, am.Value)
	case OpPrefix:
		return strings.HasPrefix(val, am.Value)
	case OpSuffix:
		return strings.HasSuffix(val, am.Value)
	case OpNotEquals:
		return val != am.Value
	case OpRegex:
		return cs.matchRegex(am.Value, val)
	default:
		return false
	}
}

// matchRegex compiles (and caches) a regex pattern, then tests val.
func (cs *compiledSelector) matchRegex(pattern, val string) bool {
	re, ok := cs.regexCache[pattern]
	if !ok {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			// Cache nil to avoid recompiling bad patterns
			cs.regexCache[pattern] = nil
			return false
		}
		cs.regexCache[pattern] = re
	}
	if re == nil {
		return false // cached compile failure
	}
	return re.MatchString(val)
}
