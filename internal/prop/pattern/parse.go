// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package pattern

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/prop/alphabet"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// The vocabulary of the portable subset.
const (
	// metacharacters are the characters that are literal only after a
	// backslash.
	metacharacters = `\.^$|?*+()[]{}`
	// classEscapes are the characters that a backslash makes literal inside
	// a class: the metacharacters and the hyphen.
	classEscapes = `\.^$|?*+()[]{}-`
	// quantifiers are the characters that start a quantifier.
	quantifiers = "*+?{"
	// doubled are the characters that a class may not contain twice in a
	// row. Java reads && as an intersection, and other engines reserve --,
	// || and ~~ for set operations.
	doubled = "&-|~"
	// maxCount is the largest count that a quantifier may state, and the
	// largest product of the counts of nested quantifiers, the limits RE2
	// sets.
	maxCount = 1000
	// maxDepth is the deepest that groups may nest. Python's engine refuses
	// a pattern whose groups nest 495 deep.
	maxDepth = 100
	// none is what peek returns past the end of the pattern, and the
	// previous member of a class that has none to pair with: one past
	// utf8.MaxRune, which no character of a pattern equals.
	none rune = 0x110000
)

// parser is a recursive-descent parser over one pattern of the portable
// subset.
type parser struct {
	// text is the pattern, for the errors.
	text string
	// runes are the pattern's characters.
	runes []rune
	// at is the index of the current character in runes.
	at int
	// depth is the number of groups open at the current character.
	depth int
}

// newParser returns a parser at the first character of text. It returns an
// error that wraps [ErrOutside] for text that is not UTF-8.
func newParser(text string) (*parser, error) {
	if !utf8.ValidString(text) {
		return nil, fmt.Errorf("%w: %q is not UTF-8", ErrOutside, text)
	}
	return &parser{text: text, runes: []rune(text)}, nil
}

// pattern parses the whole pattern, with its optional anchors.
func (p *parser) pattern() (Node, error) {
	if p.peek(0) == '^' {
		p.at++
	}
	root, err := p.alternation()
	if err != nil {
		return nil, err
	}
	if p.peek(0) == '$' && p.at == len(p.runes)-1 {
		p.at++
	}
	if p.at != len(p.runes) {
		return nil, p.fail("%q is not expected", p.peek(0))
	}
	return root, nil
}

// alternation parses branches separated by |.
func (p *parser) alternation() (Node, error) {
	first, err := p.sequence()
	if err != nil {
		return nil, err
	}
	branches := Alternation{first}
	for p.peek(0) == '|' {
		p.at++
		branch, err := p.sequence()
		if err != nil {
			return nil, err
		}
		branches = append(branches, branch)
	}
	if len(branches) == 1 {
		return first, nil
	}
	return branches, nil
}

// sequence parses pieces up to a |, a ), the closing anchor or the end. A $
// that ends the pattern inside a group stops the sequence, and the group
// then finds no ) to close it.
func (p *parser) sequence() (Node, error) {
	var items Sequence
	for r := p.peek(0); r != none && r != '|' && r != ')'; r = p.peek(0) {
		if r == '$' && p.at == len(p.runes)-1 {
			break
		}
		item, err := p.quantified()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return items, nil
}

// quantified parses an atom and the quantifier after it, if any. A second
// quantifier is then parsed as an atom, and refused as a metacharacter. It
// refuses a quantifier whose count and the counts nested in its atom
// multiply past maxCount.
func (p *parser) quantified() (Node, error) {
	item, err := p.atom()
	if err != nil {
		return nil, err
	}
	if !strings.ContainsRune(quantifiers, p.peek(0)) {
		return item, nil
	}
	sizes, err := p.quantifier()
	if err != nil {
		return nil, err
	}
	repeat := Repeat{Item: item, Sizes: sizes}
	if w := weight(repeat); w > maxCount {
		return nil, p.fail("nested counts multiply to %d, above %d", w, maxCount)
	}
	return repeat, nil
}

// weight returns the largest product of the counts of the quantifiers along
// one path through n. The atom of a quantifier passed the same check, so
// the product is at most maxCount times maxCount.
func weight(n Node) int {
	switch n := n.(type) {
	case Repeat:
		return count(n.Sizes) * weight(n.Item)
	case Sequence:
		return heaviest(n)
	case Alternation:
		return heaviest(n)
	}
	return 1
}

// heaviest returns the largest weight of nodes, and 1 for no node.
func heaviest(nodes []Node) int {
	w := 1
	for _, n := range nodes {
		w = max(w, weight(n))
	}
	return w
}

// count returns the count of a quantifier that a product of nested counts
// multiplies by: its upper count, or its lower count when it has none, and
// 1 for a count of 0.
func count(s choice.Sizes) int {
	n, bounded := s.Max()
	if !bounded {
		n = s.Min()
	}
	return max(n, 1)
}

// quantifier parses *, +, ?, {m}, {m,} or {m,n} into the numbers of
// repetitions that it allows.
func (p *parser) quantifier() (choice.Sizes, error) {
	r := p.take()
	if r == '*' {
		return choice.NewUnboundedSizes(0)
	}
	if r == '+' {
		return choice.NewUnboundedSizes(1)
	}
	if r == '?' {
		return choice.NewSizes(0, 1)
	}
	low, err := p.count()
	if err != nil {
		return choice.Sizes{}, err
	}
	high, bounded, err := p.upper(low)
	if err != nil {
		return choice.Sizes{}, err
	}
	closing, err := p.next("a count is not closed")
	if err != nil {
		return choice.Sizes{}, err
	}
	if closing != '}' {
		return choice.Sizes{}, p.fail("a count is not closed by }")
	}
	if !bounded {
		return choice.NewUnboundedSizes(low)
	}
	if high < low {
		return choice.Sizes{}, p.fail("the count {%d,%d} runs backwards", low, high)
	}
	return choice.NewSizes(low, high)
}

// upper parses what follows the minimum of a count: nothing for {m}, a
// comma for {m,}, and a comma and the maximum for {m,n}. It returns low for
// {m}, and reports false for {m,}, which has no maximum.
func (p *parser) upper(low int) (int, bool, error) {
	if p.peek(0) != ',' {
		return low, true, nil
	}
	p.at++
	if p.peek(0) == '}' {
		return low, false, nil
	}
	high, err := p.count()
	return high, true, err
}

// count parses the ASCII digits of a count, at most maxCount. A count of
// two or more digits does not start with 0, because RE2 reads a count with
// a leading zero, such as {007}, as literal text.
func (p *parser) count() (int, error) {
	start := p.at
	for isDigit(p.peek(0)) {
		p.at++
	}
	digits := p.runes[start:p.at]
	if len(digits) == 0 {
		return 0, p.fail("a count has no digits")
	}
	if len(digits) > 1 && digits[0] == '0' {
		return 0, p.fail("the count %s has a leading zero", string(digits))
	}
	n := 0
	for _, d := range digits {
		n = min(10*n+int(d-'0'), maxCount+1)
	}
	if n > maxCount {
		return 0, p.fail("the count %s is above %d", string(digits), maxCount)
	}
	return n, nil
}

// atom parses a literal, a dot, an escape, a class or a group. The caller
// has seen through peek that a character is left.
func (p *parser) atom() (Node, error) {
	r := p.take()
	if r == '(' {
		return p.group()
	}
	if r == '[' {
		return p.class()
	}
	if r == '.' {
		return dot, nil
	}
	if r == '\\' {
		return p.escape()
	}
	if strings.ContainsRune(metacharacters, r) {
		return nil, p.fail("%q must be escaped here", r)
	}
	return Literal(r), nil
}

// escape parses the character after a backslash: \d, \w, \s or an escaped
// metacharacter.
func (p *parser) escape() (Node, error) {
	r, err := p.next("the pattern ends with a backslash")
	if err != nil {
		return nil, err
	}
	if cl, ok := shorthand(r); ok {
		return cl, nil
	}
	if !strings.ContainsRune(metacharacters, r) {
		return nil, p.fail(`\%c is not in the portable subset`, r)
	}
	return Literal(r), nil
}

// group parses a group after its (, at most maxDepth deep.
func (p *parser) group() (Node, error) {
	p.depth++
	if p.depth > maxDepth {
		return nil, p.fail("groups nest deeper than %d", maxDepth)
	}
	if p.peek(0) == '?' {
		if p.peek(1) != ':' {
			return nil, p.fail("only the (?: group is in the portable subset")
		}
		p.at += 2
	}
	inner, err := p.alternation()
	if err != nil {
		return nil, err
	}
	closing, err := p.next("a group is not closed")
	if err != nil {
		return nil, err
	}
	if closing != ')' {
		return nil, p.fail("a group is not closed by )")
	}
	p.depth--
	return inner, nil
}

// class parses a class after its [.
func (p *parser) class() (Node, error) {
	negated := p.peek(0) == '^'
	if negated {
		p.at++
	}
	var members []alphabet.Interval
	previous, first := none, true
	for {
		r, err := p.next("a class is not closed")
		if err != nil {
			return nil, err
		}
		if r == ']' {
			break
		}
		if named, ok := shorthand(p.peek(0)); ok && r == '\\' {
			p.at++
			members = append(members, named.Members...)
			previous, first = none, false
			continue
		}
		low, err := p.classChar(r, previous, first)
		if err != nil {
			return nil, err
		}
		previous, first = r, false
		if r == '\\' {
			previous = none
		}
		if p.peek(0) != '-' || p.peek(1) == none || p.peek(1) == ']' {
			members = alphabet.AppendIndices(members, low, low)
			continue
		}
		if previous == '-' {
			return nil, p.fail("%q is reserved inside a class", "--")
		}
		p.at++
		high, err := p.classChar(p.take(), '-', false)
		if err != nil {
			return nil, err
		}
		if high < low {
			return nil, p.fail("the range %c-%c runs backwards", low, high)
		}
		members = alphabet.AppendIndices(members, low, high)
		previous = none
	}
	if first {
		return nil, p.fail("a class is empty")
	}
	if negated {
		members = complement(members)
	}
	cl, ok := newClass(members)
	if !ok {
		return nil, p.fail("a class has no member")
	}
	return cl, nil
}

// classChar returns the character that a class member states, unescaped.
// previous is the member before it, and first reports whether it is the
// class's first member. It refuses a [, a reserved pair, a misplaced hyphen
// and an escape that the subset does not have.
func (p *parser) classChar(r, previous rune, first bool) (rune, error) {
	if r == '\\' {
		escaped, err := p.next("a class ends inside an escape")
		if err != nil {
			return none, err
		}
		if !strings.ContainsRune(classEscapes, escaped) {
			return none, p.fail(`\%c is not in the portable subset`, escaped)
		}
		return escaped, nil
	}
	if r == '[' {
		return none, p.fail("[ must be escaped inside a class")
	}
	if r == previous && strings.ContainsRune(doubled, r) {
		return none, p.fail("%q is reserved inside a class", string([]rune{r, r}))
	}
	if r == '-' && !first && p.peek(0) != ']' {
		return none, p.fail("a hyphen inside a class must be escaped")
	}
	return r, nil
}

// peek returns the character ahead characters past the current one, and
// none past the end of the pattern.
func (p *parser) peek(ahead int) rune {
	if at := p.at + ahead; at < len(p.runes) {
		return p.runes[at]
	}
	return none
}

// take returns the current character and moves past it. The caller has
// seen through peek that the character exists.
func (p *parser) take() rune {
	r := p.runes[p.at]
	p.at++
	return r
}

// next returns the current character and moves past it. At the end of the
// pattern it returns an error that states what is missing.
func (p *parser) next(missing string) (rune, error) {
	if p.at == len(p.runes) {
		return none, p.fail("%s", missing)
	}
	return p.take(), nil
}

// fail returns the error for what is wrong at the current position, which
// wraps [ErrOutside].
func (p *parser) fail(format string, args ...any) error {
	return fmt.Errorf("%w: %q at %d: %s", ErrOutside, p.text, p.at, fmt.Sprintf(format, args...))
}

// isDigit reports whether r is an ASCII digit, the only digits that a count
// may state.
func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}
