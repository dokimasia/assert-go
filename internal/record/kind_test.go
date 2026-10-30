// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package record_test

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/record"
)

// kindWriteAllocs is the ceiling of a write of a record into a test's seat: its
// key, and the growth of the attributes that the seat keeps.
const kindWriteAllocs = 3

// testSeat is a test's seat that is a TB too, so that a wrapper can wrap it.
type testSeat struct {
	*attrs
}

func (testSeat) Helper()               {}
func (testSeat) Fatalf(string, ...any) {}
func (testSeat) Errorf(string, ...any) {}

// keeperOfTest keeps call records and wraps a test's seat, as a recorder
// that states its test does.
type keeperOfTest struct {
	*keeping

	test assert.TB
}

func (*keeperOfTest) Helper()               {}
func (*keeperOfTest) Fatalf(string, ...any) {}
func (*keeperOfTest) Errorf(string, ...any) {}

// Unwrap returns the test's seat.
func (k *keeperOfTest) Unwrap() assert.TB { return k.test }

func TestKind(t *testing.T) {
	t.Parallel()

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		t.Run("numbers the records of each kind of a test from 1", func(t *testing.T) {
			t.Parallel()

			seat := &attrs{name: "TestSuite"}
			record.Write(seat, "dokimi.suite", `{"a":1}`)
			record.Write(seat, "dokimi.trace", `{"b":2}`)
			record.Write(seat, "dokimi.suite", `{"c":3}`)
			want := [][2]string{
				{"dokimi.suite.1", `{"a":1}`},
				{"dokimi.trace.1", `{"b":2}`},
				{"dokimi.suite.2", `{"c":3}`},
			}
			assert.Equal(t, seat.attributes(), want, "each kind counts its records from 1")
		})

		t.Run("numbers the records of each test on its own", func(t *testing.T) {
			t.Parallel()

			first := &attrs{name: "TestFirst"}
			second := &attrs{name: "TestSecond"}
			record.Write(first, "dokimi.suite", `{}`)
			record.Write(second, "dokimi.suite", `{}`)
			assert.Equal(t, [][][2]string{first.attributes(), second.attributes()},
				[][][2]string{{{"dokimi.suite.1", `{}`}}, {{"dokimi.suite.1", `{}`}}},
				"the first record of each test is number 1")
		})

		t.Run("writes a record with line breaks as one line", func(t *testing.T) {
			t.Parallel()

			seat := &attrs{name: "TestSuite"}
			record.Write(seat, "dokimi.suite", "{\r\n\"a\": 1\n}")
			assert.Equal(t, seat.attributes(), [][2]string{{"dokimi.suite.1", `{  "a": 1 }`}},
				"each line break is a space")
		})

		t.Run("writes into the test's seat that a wrapper wraps", func(t *testing.T) {
			t.Parallel()

			seat := &attrs{name: "TestSuite"}
			record.Write(wrapper{wrapper{testSeat{seat}}}, "dokimi.suite", `{}`)
			assert.Equal(t, seat.attributes(), [][2]string{{"dokimi.suite.1", `{}`}},
				"the wrapped test's seat takes the record")
		})

		t.Run("writes nothing through a seat that keeps call records", func(t *testing.T) {
			t.Parallel()

			seat := &attrs{name: "TestSuite"}
			record.Write(&keeperOfTest{keeping: newKeeping(), test: testSeat{seat}}, "dokimi.suite", `{}`)
			assert.Empty(t, seat.attributes(), "the keeper ends the walk")
		})

		t.Run("returns at the end of a chain without a seat of a test", func(t *testing.T) {
			t.Parallel()

			assert.CompletesWithin(t, 10*time.Second, func(context.Context) error {
				record.Write(wrapper{}, "dokimi.suite", `{}`)
				return nil
			}, "the walk ends after the last seat of the chain")
		})

		t.Run("splits a record whose line does not fit test2json's buffer at UTF-8 boundaries", func(t *testing.T) {
			t.Parallel()

			seat := &attrs{name: "TestSuite"}
			text := `{"a":"` + strings.Repeat("é", 2500) + `"}`
			record.Write(seat, "dokimi.suite", text)
			written := seat.attributes()
			var joined strings.Builder
			for _, a := range written {
				assert.Equal(t, a[0], "dokimi.suite.1", "each part has the key of the record")
				assert.True(t, utf8.ValidString(a[1]), "each part is UTF-8")
				fits := len(attrLine)+len(seat.name)+1+len(a[0])+1+len(a[1])+1 <= lineBuffer
				assert.True(t, fits, "each line fits test2json's buffer")
				joined.WriteString(a[1])
			}
			assert.Equal(t, []any{len(written), joined.String()}, []any{2, text},
				"two parts that join into the record")
		})
	})
}

// TestKindAllocs checks the ceiling of Write.
func TestKindAllocs(t *testing.T) {
	seat := &attrs{name: "TestSuite"}
	keeper := newKeeping()
	assert.MaxAllocs(t, func() { record.Write(seat, "dokimi.suite", `{}`) }, kindWriteAllocs,
		"Write allocates the key and the growth of the seat's attributes")
	assert.MaxAllocs(t, func() { record.Write(keeper, "dokimi.suite", `{}`) }, 0,
		"Write through a keeper allocates nothing")
}

// BenchmarkKind measures Write into a test's seat.
func BenchmarkKind(b *testing.B) {
	b.Run("Write", func(b *testing.B) {
		seat := &attrs{name: "TestSuite"}
		bc := bench.Start(b).MaxAllocs(kindWriteAllocs)
		defer bc.End()
		for bc.Loop() {
			record.Write(seat, "dokimi.suite", `{}`)
		}
	})
}
