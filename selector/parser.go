package selector

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Parse parses a CSS-like AX selector string into a SelectorGroup AST.
//
// Supported syntax:
//   - Role: AXButton, AXWindow, * (wildcard)
//   - Attributes: [name="value"], [name*="value"], [name^="value"], [name$="value"],
//     [name~="regex"], [name!="value"]
//   - Pseudos: :first, :last, :nth(N), :visible, :enabled, :focused, :selected
//   - Combinators: space (descendant), > (child)
//   - Groups: selector1, selector2
func Parse(input string) (*SelectorGroup, error) {
	p := &parser{input: input, pos: 0}
	return p.parseGroup()
}

type parser struct {
	input string
	pos   int
}

func (p *parser) parseGroup() (*SelectorGroup, error) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("selector: empty selector string")
	}

	var selectors []*CompoundSelector

	cs, err := p.parseCompound()
	if err != nil {
		return nil, err
	}
	selectors = append(selectors, cs)

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}
		if p.peek() != ',' {
			break
		}
		p.advance() // consume ','
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			return nil, fmt.Errorf("selector: trailing comma at position %d", p.pos)
		}
		cs, err := p.parseCompound()
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, cs)
	}

	p.skipWhitespace()
	if p.pos < len(p.input) {
		return nil, fmt.Errorf("selector: unexpected character %q at position %d", p.input[p.pos], p.pos)
	}

	return &SelectorGroup{Selectors: selectors}, nil
}

func (p *parser) parseCompound() (*CompoundSelector, error) {
	p.skipWhitespace()

	head, err := p.parseSimple()
	if err != nil {
		return nil, err
	}

	cs := &CompoundSelector{Head: head}

	for {
		// Save position to check if we have a combinator
		savedPos := p.pos
		p.skipWhitespace()

		if p.pos >= len(p.input) {
			break
		}

		ch := p.peek()

		// comma or end means end of this compound
		if ch == ',' {
			break
		}

		var comb Combinator

		if ch == '>' {
			p.advance() // consume '>'
			p.skipWhitespace()
			if p.pos >= len(p.input) {
				return nil, fmt.Errorf("selector: trailing '>' at position %d", savedPos)
			}
			comb = CombChild
		} else if savedPos < p.pos {
			// whitespace was consumed -> descendant combinator
			// but only if next char is a valid selector start
			if !p.isSimpleSelectorStart() {
				// not a selector start, restore and break
				p.pos = savedPos
				break
			}
			comb = CombDescendant
		} else {
			// no whitespace, no '>' -> end of compound
			break
		}

		sel, err := p.parseSimple()
		if err != nil {
			return nil, err
		}
		cs.Steps = append(cs.Steps, CompoundStep{Combinator: comb, Selector: sel})
	}

	return cs, nil
}

func (p *parser) parseSimple() (*SimpleSelector, error) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("selector: expected selector at position %d", p.pos)
	}

	// Parse role
	role, err := p.parseRole()
	if err != nil {
		return nil, err
	}

	sel := &SimpleSelector{Role: role}

	// Parse attributes and pseudos
	for p.pos < len(p.input) {
		ch := p.peek()
		if ch == '[' {
			attr, err := p.parseAttr()
			if err != nil {
				return nil, err
			}
			sel.Attrs = append(sel.Attrs, attr)
		} else if ch == ':' {
			pseudo, err := p.parsePseudo()
			if err != nil {
				return nil, err
			}
			sel.Pseudos = append(sel.Pseudos, pseudo)
		} else {
			break
		}
	}

	return sel, nil
}

func (p *parser) parseRole() (string, error) {
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("selector: expected role at position %d", p.pos)
	}

	if p.peek() == '*' {
		p.advance()
		return "*", nil
	}

	// Role must start with a letter or underscore (all AX roles start with "AX")
	if !isRoleStartChar(p.peek()) {
		return "", fmt.Errorf("selector: expected role name at position %d, got %q", p.pos, p.input[p.pos])
	}

	start := p.pos
	for p.pos < len(p.input) {
		ch := p.peek()
		if isRoleChar(ch) {
			p.advance()
		} else {
			break
		}
	}

	return p.input[start:p.pos], nil
}

func (p *parser) parseAttr() (AttrMatcher, error) {
	if p.peek() != '[' {
		return AttrMatcher{}, fmt.Errorf("selector: expected '[' at position %d", p.pos)
	}
	p.advance() // consume '['

	// Parse attribute name
	p.skipWhitespace()
	name, err := p.parseIdentifier()
	if err != nil {
		return AttrMatcher{}, fmt.Errorf("selector: expected attribute name: %w", err)
	}

	p.skipWhitespace()

	// Parse operator
	op, err := p.parseAttrOp()
	if err != nil {
		return AttrMatcher{}, err
	}

	p.skipWhitespace()

	// Parse value (quoted string)
	value, err := p.parseQuotedString()
	if err != nil {
		return AttrMatcher{}, err
	}

	p.skipWhitespace()

	// Expect ']'
	if p.pos >= len(p.input) || p.peek() != ']' {
		return AttrMatcher{}, fmt.Errorf("selector: expected ']' at position %d", p.pos)
	}
	p.advance() // consume ']'

	return AttrMatcher{Name: name, Op: op, Value: value}, nil
}

func (p *parser) parseAttrOp() (AttrOp, error) {
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("selector: expected operator at position %d", p.pos)
	}

	ch := p.peek()
	switch ch {
	case '=':
		p.advance()
		return OpEquals, nil
	case '*':
		p.advance()
		if p.pos < len(p.input) && p.peek() == '=' {
			p.advance()
			return OpContains, nil
		}
		return 0, fmt.Errorf("selector: expected '=' after '*' at position %d", p.pos)
	case '^':
		p.advance()
		if p.pos < len(p.input) && p.peek() == '=' {
			p.advance()
			return OpPrefix, nil
		}
		return 0, fmt.Errorf("selector: expected '=' after '^' at position %d", p.pos)
	case '$':
		p.advance()
		if p.pos < len(p.input) && p.peek() == '=' {
			p.advance()
			return OpSuffix, nil
		}
		return 0, fmt.Errorf("selector: expected '=' after '$' at position %d", p.pos)
	case '~':
		p.advance()
		if p.pos < len(p.input) && p.peek() == '=' {
			p.advance()
			return OpRegex, nil
		}
		return 0, fmt.Errorf("selector: expected '=' after '~' at position %d", p.pos)
	case '!':
		p.advance()
		if p.pos < len(p.input) && p.peek() == '=' {
			p.advance()
			return OpNotEquals, nil
		}
		return 0, fmt.Errorf("selector: expected '=' after '!' at position %d", p.pos)
	default:
		return 0, fmt.Errorf("selector: expected operator at position %d, got %q", p.pos, ch)
	}
}

func (p *parser) parseQuotedString() (string, error) {
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("selector: expected quoted string at position %d", p.pos)
	}

	quote := p.peek()
	if quote != '"' && quote != '\'' {
		return "", fmt.Errorf("selector: expected quoted string at position %d, got %q", p.pos, quote)
	}
	p.advance() // consume opening quote

	var b strings.Builder
	for p.pos < len(p.input) {
		ch := p.peek()
		if ch == '\\' && p.pos+1 < len(p.input) {
			next := p.input[p.pos+1]
			if next == quote || next == '\\' {
				b.WriteByte(next)
				p.pos += 2
				continue
			}
			// other escapes: pass through as-is (e.g., \d -> \d)
			b.WriteByte(ch)
			p.advance()
			continue
		}
		if ch == quote {
			p.advance() // consume closing quote
			return b.String(), nil
		}
		b.WriteByte(ch)
		p.advance()
	}

	return "", fmt.Errorf("selector: unclosed string starting at position %d", p.pos)
}

func (p *parser) parsePseudo() (Pseudo, error) {
	if p.peek() != ':' {
		return Pseudo{}, fmt.Errorf("selector: expected ':' at position %d", p.pos)
	}
	p.advance() // consume ':'

	name, err := p.parseIdentifier()
	if err != nil {
		return Pseudo{}, fmt.Errorf("selector: expected pseudo name: %w", err)
	}

	switch name {
	case "first":
		return Pseudo{Type: PseudoFirst}, nil
	case "last":
		return Pseudo{Type: PseudoLast}, nil
	case "visible":
		return Pseudo{Type: PseudoVisible}, nil
	case "enabled":
		return Pseudo{Type: PseudoEnabled}, nil
	case "focused":
		return Pseudo{Type: PseudoFocused}, nil
	case "selected":
		return Pseudo{Type: PseudoSelected}, nil
	case "nth":
		return p.parseNth()
	default:
		return Pseudo{}, fmt.Errorf("selector: unknown pseudo-selector %q", name)
	}
}

func (p *parser) parseNth() (Pseudo, error) {
	if p.pos >= len(p.input) || p.peek() != '(' {
		return Pseudo{}, fmt.Errorf("selector: expected '(' after :nth at position %d", p.pos)
	}
	p.advance() // consume '('

	p.skipWhitespace()

	// Parse integer
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
		p.advance()
	}

	if p.pos == start {
		return Pseudo{}, fmt.Errorf("selector: expected integer in :nth() at position %d", p.pos)
	}

	n, err := strconv.Atoi(p.input[start:p.pos])
	if err != nil {
		return Pseudo{}, fmt.Errorf("selector: invalid integer in :nth(): %w", err)
	}

	p.skipWhitespace()

	if p.pos >= len(p.input) || p.peek() != ')' {
		return Pseudo{}, fmt.Errorf("selector: expected ')' in :nth() at position %d", p.pos)
	}
	p.advance() // consume ')'

	return Pseudo{Type: PseudoNth, N: n}, nil
}

func (p *parser) parseIdentifier() (string, error) {
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.peek()
		if isIdentChar(ch) {
			p.advance()
		} else {
			break
		}
	}
	if p.pos == start {
		if p.pos < len(p.input) {
			return "", fmt.Errorf("selector: expected identifier at position %d, got %q", p.pos, p.input[p.pos])
		}
		return "", fmt.Errorf("selector: expected identifier at position %d", p.pos)
	}
	return p.input[start:p.pos], nil
}

// Helper methods

func (p *parser) peek() byte {
	return p.input[p.pos]
}

func (p *parser) advance() {
	p.pos++
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t' || p.input[p.pos] == '\n' || p.input[p.pos] == '\r') {
		p.pos++
	}
}

func (p *parser) isSimpleSelectorStart() bool {
	if p.pos >= len(p.input) {
		return false
	}
	ch := p.peek()
	return ch == '*' || isRoleStartChar(ch)
}

func isRoleChar(ch byte) bool {
	return isRoleStartChar(ch) || (ch >= '0' && ch <= '9')
}

func isRoleStartChar(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_'
}

func isIdentChar(ch byte) bool {
	r := rune(ch)
	return unicode.IsLetter(r) || unicode.IsDigit(r) || ch == '_' || ch == '-'
}
