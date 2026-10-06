// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package record_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/record"
)

// on is the environment of a child whose switch is on.
var on = record.Variable + "=1"

// attrLine starts the line that testing writes for an attribute under
// -json: the framing byte and the prefix.
const attrLine = "\x16=== ATTR  "

// lineBuffer is the size of test2json's buffer, which a whole line of an
// attribute fits.
const lineBuffer = 4096

// TestAttr checks the records that a test's seat writes through its Attr,
// in a child process, whose switch the parent sets.
func TestAttr(t *testing.T) {
	t.Parallel()

	t.Run("Of", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for a test's seat while the switch is off", func(t *testing.T) {
			t.Parallel()
			if !childtest.InChild(t) {
				runChild(t, record.Variable+"=0")
				return
			}
			assert.Nil(t, record.Of(&attrs{name: "TestOff"}), "no Calls")
			assert.Nil(t, record.Of(t), "no Calls for the child's own test")
		})
		t.Run("returns one Calls for each test while the switch is on", func(t *testing.T) {
			t.Parallel()
			if !childtest.InChild(t) {
				runChild(t, on)
				return
			}
			first, second := &attrs{name: "TestFirst"}, &attrs{name: "TestSecond"}
			again := record.Of(first)
			assert.True(t, record.Of(first) == again, "one Calls for one test")
			record.Add(record.Of(first), call("true", "a"))
			record.Add(record.Of(second), call("true", "b"))
			assert.Equal(t, first.attributes()[0][0], "dokimi.assert.1", "the first test's first number")
			assert.Equal(t, second.attributes()[0][0], "dokimi.assert.1", "the second test's first number")
		})
	})

	t.Run("Add", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the record through the test's Attr under its number", func(t *testing.T) {
			t.Parallel()
			if !childtest.InChild(t) {
				out := runChild(t, on)
				want := attrLine + t.Name() + ` dokimi.assert.1 {"definition":"7.0.0","seq":1,"assertion":"true"`
				assert.Contains(t, out, want, "the attribute of the record in the test's output")
				return
			}
			record.Add(record.Of(t), call("true", "the child's call"))
		})
		t.Run("splits a record whose line does not fit test2json's buffer at UTF-8 boundaries", func(t *testing.T) {
			t.Parallel()
			if !childtest.InChild(t) {
				out := runChild(t, on)
				var joined strings.Builder
				for line := range strings.Lines(out) {
					value, ok := strings.CutPrefix(line, attrLine+t.Name()+" dokimi.assert.1 ")
					if !ok {
						continue
					}
					assert.True(t, len(line) <= lineBuffer, "each line fits test2json's buffer")
					joined.WriteString(strings.TrimSuffix(value, "\n"))
				}
				got := decoded(t, joined.String())
				assert.Equal(t, got["contract"], any(strings.Repeat("é", 3000)), "the joined record")
				return
			}
			record.Add(record.Of(t), call("true", strings.Repeat("é", 3000)))
			seat := &attrs{name: "TestSplit"}
			record.Add(record.Of(seat), call("true", strings.Repeat("é", 3000)))
			written := seat.attributes()
			assert.Length(t, written, 2, "two attributes")
			for _, a := range written {
				assert.True(t, utf8.ValidString(a[1]), "each value is UTF-8")
				fits := len(attrLine)+len(seat.name)+1+len(a[0])+1+len(a[1])+1 <= lineBuffer
				assert.True(t, fits, "each line fits test2json's buffer")
			}
		})
		t.Run("writes a rune per attribute for a test whose name fills the buffer", func(t *testing.T) {
			t.Parallel()
			if !childtest.InChild(t) {
				runChild(t, on)
				return
			}
			seat := &attrs{name: strings.Repeat("n", lineBuffer)}
			record.Add(record.Of(seat), call("true", "c"))
			written := seat.attributes()
			assert.True(t, len(written) > 1, "more than one attribute")
			for _, a := range written[:len(written)-1] {
				assert.Length(t, a[1], utf8.UTFMax, "the room for the least runes")
			}
		})
	})

	t.Run("Slot.Write", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the record of a call that ran a body under the number it took first", func(t *testing.T) {
			t.Parallel()
			if !childtest.InChild(t) {
				runChild(t, on)
				return
			}
			seat := &attrs{name: "TestBody"}
			slot := record.Begin(seat)
			b := &body{}
			record.Run(&b.calls, slot, nil)
			record.Add(&b.calls, call("true", "in the body"))
			slot.Take(&b.calls, record.NoPhase)
			slot.Write(call("eventually", "it converges"))
			written := seat.attributes()
			assert.Length(t, written, 2, "two records")
			assert.Equal(t, written[0][0], "dokimi.assert.2", "the body's call is written first")
			assert.Equal(t, written[1][0], "dokimi.assert.1", "the call that ran the body is written at its end")
		})
	})
}
