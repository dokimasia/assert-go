// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"reflect"
	"regexp"
	"strings"

	"go.dokimi.dev/assert/internal/prop/pattern"
)

// lineTerminatorFree is the RE2 class of the portable subset's ., which
// matches every character but the five line terminators. RE2's own .
// leaves out \n alone.
const lineTerminatorFree = `[^\n\r\x{85}\x{2028}\x{2029}]`

// HasPrefix reports when got does not start with prefix.
//
// got is a string, a []byte, or any type defined over either.
//
// # Allocation contract
//
// A passing call on a string allocates nothing.
func HasPrefix(seat Seat, mode Mode, got any, prefix, msg string) {
	seat.Helper()

	text, ok := textOf(got)
	if !ok {
		Fail(seat, mode, "has-prefix", msg, map[string]any{"got": got, "prefix": prefix})
		return
	}
	if !strings.HasPrefix(text, prefix) {
		Fail(seat, mode, "has-prefix", msg, map[string]any{"got": text, "prefix": prefix})
		return
	}
	Pass(seat, mode, "has-prefix", msg)
}

// HasSuffix reports when got does not end with suffix.
//
// got is a string, a []byte, or any type defined over either.
//
// # Allocation contract
//
// A passing call on a string allocates nothing.
func HasSuffix(seat Seat, mode Mode, got any, suffix, msg string) {
	seat.Helper()

	text, ok := textOf(got)
	if !ok {
		Fail(seat, mode, "has-suffix", msg, map[string]any{"got": got, "suffix": suffix})
		return
	}
	if !strings.HasSuffix(text, suffix) {
		Fail(seat, mode, "has-suffix", msg, map[string]any{"got": text, "suffix": suffix})
		return
	}
	Pass(seat, mode, "has-suffix", msg)
}

// Matches reports when got does not match expr, a regular expression of
// the portable subset that every target language reads the same way.
// expr matches anywhere in got. Anchor expr to require the whole value.
//
// $ matches at the end of the text only, \d, \w and \s are their ASCII
// classes, and . matches no line terminator: \n, \r, U+0085, U+2028 or
// U+2029. A pattern outside the subset, such as one with a backreference,
// a lookaround, a flag or \b, fails the assertion and does not panic,
// because a test with such a pattern has established nothing.
//
// # Allocation contract
//
// Matches compiles expr on every call: a passing call of a pattern of a
// literal, a class and an anchor at each end allocates 62 times.
func Matches(seat Seat, mode Mode, got any, expr, msg string) {
	seat.Helper()

	text, ok := textOf(got)
	if !ok {
		Fail(seat, mode, "matches", msg, map[string]any{"got": got, "pattern": expr})
		return
	}

	re, err := portable(expr)
	if err != nil || !re.MatchString(text) {
		Fail(seat, mode, "matches", msg, map[string]any{"got": text, "pattern": expr})
		return
	}
	Pass(seat, mode, "matches", msg)
}

// portable compiles expr, a pattern of the portable subset, into the RE2
// expression that reads it as the subset does. RE2 already reads $ at the
// end of the text only and \d, \w and \s as ASCII classes, so only a .
// outside a class changes. It returns the error of [pattern.Parse] for
// expr outside the subset.
//
// The subset escapes only ASCII characters, so the byte after a backslash
// is the whole escaped character, and it is copied unchanged.
func portable(expr string) (*regexp.Regexp, error) {
	if _, err := pattern.Parse(expr); err != nil {
		return nil, err
	}

	var b strings.Builder
	escaped, inClass := false, false
	for i := range len(expr) {
		c := expr[i]
		if escaped {
			escaped = false
		} else if c == '\\' {
			escaped = true
		} else if !inClass && c == '.' {
			b.WriteString(lineTerminatorFree)
			continue
		} else if !inClass && c == '[' {
			inClass = true
		} else if inClass && c == ']' {
			inClass = false
		}
		b.WriteByte(c)
	}
	return regexp.Compile(b.String())
}

// textOf reads a value as text, accepting a string, a []byte, or any
// type defined over either. A nil value has the kind [reflect.Invalid],
// and is no text.
func textOf(v any) (string, bool) {
	switch s := v.(type) {
	case string:
		return s, true
	case []byte:
		return string(s), true
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.String {
		return rv.String(), true
	}
	if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
		return string(rv.Bytes()), true
	}
	return "", false
}
