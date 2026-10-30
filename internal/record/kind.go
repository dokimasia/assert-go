// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package record

import (
	"io"
	"strconv"
	"strings"
	"sync"

	"go.dokimi.dev/assert/internal/wrap"
)

// kinds maps the writer of the output of each test to the numbers of the
// records of each kind that the test wrote. The map keeps each entry for the
// life of the process, as tests does.
var kinds sync.Map

// lineBreaks replaces each line break of a record with a space. A line break
// of JSON text is whitespace between two tokens, so the record keeps its
// value.
var lineBreaks = strings.NewReplacer("\r", " ", "\n", " ")

// numbers are the numbers of the last record of each kind of one test.
type numbers struct {
	mu   sync.Mutex
	last map[string]int
}

// Write writes text into the output of the test of seat. text is the JSON
// text of one record of kind. The test receives an attribute with the key
// <kind>.<n>, and Write numbers the records of each kind of a test from 1. An
// attribute is one line, so Write replaces each line break of text with a
// space. It splits a text whose line does not fit test2json's buffer as it
// splits the text of a call record.
//
// Write walks the chain of seat as [Of] walks it, and writes into the first
// seat of a test. It does not write when a seat that keeps call records comes
// first, or when the chain does not have a seat of a test. Write ignores [On],
// and it does not check kind or text.
func Write(seat any, kind, text string) {
	for ; seat != nil; seat = wrap.Unwrap(seat) {
		if _, ok := seat.(keeper); ok {
			return
		}
		if t, ok := seat.(attrSeat); ok {
			write(t, kind+"."+strconv.Itoa(next(t.Output(), kind)), lineBreaks.Replace(text))
			return
		}
	}
}

// next returns the number of the next record of kind of the test whose
// output is output.
func next(output io.Writer, kind string) int {
	v, ok := kinds.Load(output)
	if !ok {
		v, _ = kinds.LoadOrStore(output, &numbers{last: map[string]int{}})
	}
	n := v.(*numbers)
	n.mu.Lock()
	defer n.mu.Unlock()
	n.last[kind]++
	return n.last[kind]
}
