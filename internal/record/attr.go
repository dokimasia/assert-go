// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package record

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
	"unicode/utf8"
)

// The line that test2json reads for an attribute: under -json, the framing
// byte ^V, then "=== ATTR  ", the test's name, a space, the key, a space,
// the value and a newline. test2json turns the line into an attr event only
// when the whole line fits its buffer of 4,096 bytes.
const (
	// lineBuffer is the size of test2json's buffer.
	lineBuffer = 4096
	// attrFraming is the framing byte and the prefix of the line.
	attrFraming = len("\x16=== ATTR  ")
	// attrSeparators are the two spaces and the newline of the line.
	attrSeparators = 3
)

// keyPrefix starts the key of every attribute of a call record, which ends
// with the call's number.
const keyPrefix = "dokimi.assert."

// attrSeat is a test's seat, which writes attributes into the test's
// output: testing.T, testing.B and testing.F, and a type that embeds one of
// them.
type attrSeat interface {
	// Attr writes the attribute key with value.
	Attr(key, value string)
	// Name returns the test's name.
	Name() string
	// Output returns the writer of the test's output.
	Output() io.Writer
}

// tests are the Calls of every test that recorded a call, by the writer of
// the test's output. Every seat of one run of a test returns that writer,
// and so does a type that embeds the test's seat, so their calls take their
// numbers in one sequence. A benchmark keeps its writer over its rounds. The
// map keeps each entry for the life of the process.
var tests sync.Map

// testCalls returns the Calls of a test's seat while [On] reports true, and
// nil for any other seat.
func testCalls(seat any) *Calls {
	t, ok := seat.(attrSeat)
	if !ok {
		return nil
	}
	if on, _ := On(); !on {
		return nil
	}
	output := t.Output()
	if c, ok := tests.Load(output); ok {
		return c.(*Calls)
	}
	c, _ := tests.LoadOrStore(output, &Calls{state: writing, sink: t})
	return c.(*Calls)
}

// write writes the record of call seq through the Attr of seat, under the
// key dokimi.assert.<seq>. It splits a record whose line would not fit
// test2json's buffer over consecutive attributes of the same key, and a
// reader joins their values in order. Each value is valid UTF-8, because
// every split falls between two runes:
//
//   - A line with room for a rune of any length gets the runes that fit.
//   - A line with room for fewer bytes than a rune can take gets the record
//     with every rune beyond ASCII as a JSON escape, so that every byte is a
//     rune.
//   - A line without room for one byte gets the whole record, which
//     test2json reports as output and not as an attribute.
func write(seat attrSeat, seq int, text string) {
	key := keyPrefix + strconv.Itoa(seq)
	room := lineBuffer - attrFraming - attrSeparators - len(seat.Name()) - len(key)
	switch {
	case room < 1:
		room = len(text)
	case room < utf8.UTFMax:
		text = escaped(text)
	}
	for len(text) > room {
		cut := room
		for !utf8.RuneStart(text[cut]) {
			cut--
		}
		seat.Attr(key, text[:cut])
		text = text[cut:]
	}
	seat.Attr(key, text)
}

// escaped returns the JSON text with every rune beyond ASCII as a JSON
// escape: one \uXXXX, or a surrogate pair of two for a rune beyond the Basic
// Multilingual Plane. The text states the same JSON value in ASCII alone,
// because a rune beyond ASCII occurs only inside a JSON string.
func escaped(text string) string {
	var b strings.Builder
	for _, r := range text {
		if r < utf8.RuneSelf {
			b.WriteRune(r)
			continue
		}
		for _, unit := range utf16.AppendRune(nil, r) {
			fmt.Fprintf(&b, `\u%04x`, unit)
		}
	}
	return b.String()
}
