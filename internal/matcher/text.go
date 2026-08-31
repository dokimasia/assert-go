// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"reflect"
	"regexp"
	"strings"
)

// HasPrefix reports when got does not start with prefix.
//
// got is a string, a []byte, or any type defined over either.
func HasPrefix(seat Seat, mode Mode, got any, prefix, msg string) {
	seat.Helper()

	text, ok := textOf(got)
	if !ok {
		Fail(seat, mode, "has-prefix", msg, map[string]any{"got": got, "prefix": prefix})
		return
	}
	if !strings.HasPrefix(text, prefix) {
		Fail(seat, mode, "has-prefix", msg, map[string]any{"got": text, "prefix": prefix})
	}
}

// HasSuffix reports when got does not end with suffix. See [HasPrefix]
// for the types that answer.
func HasSuffix(seat Seat, mode Mode, got any, suffix, msg string) {
	seat.Helper()

	text, ok := textOf(got)
	if !ok {
		Fail(seat, mode, "has-suffix", msg, map[string]any{"got": got, "suffix": suffix})
		return
	}
	if !strings.HasSuffix(text, suffix) {
		Fail(seat, mode, "has-suffix", msg, map[string]any{"got": text, "suffix": suffix})
	}
}

// Matches reports when got does not match the regular expression
// pattern. It matches anywhere in got; anchor the pattern to require
// the whole value.
//
// A pattern that does not compile is a failure rather than a panic,
// because a test with a broken pattern has not established anything
// and should say so on the seat like every other failure.
func Matches(seat Seat, mode Mode, got any, pattern, msg string) {
	seat.Helper()

	text, ok := textOf(got)
	if !ok {
		Fail(seat, mode, "matches", msg, map[string]any{"got": got, "pattern": pattern})
		return
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		Fail(seat, mode, "matches", msg, map[string]any{"got": text, "pattern": pattern})
		return
	}
	if !re.MatchString(text) {
		Fail(seat, mode, "matches", msg, map[string]any{"got": text, "pattern": pattern})
	}
}

// textOf reads a value as text, accepting a string, a []byte, or any
// type defined over either.
func textOf(v any) (string, bool) {
	switch s := v.(type) {
	case string:
		return s, true
	case []byte:
		return string(s), true
	}

	rv := reflect.ValueOf(v)
	switch {
	case !rv.IsValid():
		return "", false
	case rv.Kind() == reflect.String:
		return rv.String(), true
	case rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8:
		return string(rv.Bytes()), true
	default:
		return "", false
	}
}
