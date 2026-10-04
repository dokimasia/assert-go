// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package record

import (
	"strconv"
	"sync"
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
// output: testing.T, testing.B and testing.F.
type attrSeat interface {
	// Attr writes the attribute key with value.
	Attr(key, value string)
	// Name returns the test's name.
	Name() string
}

// tests are the Calls of every test that recorded a call, by the test's
// seat. The map keeps each entry for the life of the process.
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
	if c, ok := tests.Load(seat); ok {
		return c.(*Calls)
	}
	c, _ := tests.LoadOrStore(seat, &Calls{state: writing, sink: t})
	return c.(*Calls)
}

// write writes the record of call seq through the Attr of seat, under the
// key dokimi.assert.<seq>. It splits a record whose line would not fit
// test2json's buffer over consecutive attributes of the same key, at UTF-8
// boundaries. A reader joins their values in order.
func write(seat attrSeat, seq int, text string) {
	key := keyPrefix + strconv.Itoa(seq)
	room := max(lineBuffer-attrFraming-attrSeparators-len(seat.Name())-len(key), utf8.UTFMax)
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
