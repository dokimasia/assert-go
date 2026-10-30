// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// recordOp is the operation that the cases of WriteRecord name.
const recordOp = "assert.WriteRecord"

// attrSeat is a test's seat that keeps every attribute written to it, in
// order.
type attrSeat struct {
	matchertest.Seat

	output  strings.Builder
	written [][2]string
}

// Attr keeps the attribute key with value.
func (s *attrSeat) Attr(key, value string) { s.written = append(s.written, [2]string{key, value}) }

// Name returns the test's name.
func (*attrSeat) Name() string { return "TestSuite" }

// Output returns the test's output.
func (s *attrSeat) Output() io.Writer { return &s.output }

// recordFaultOf returns the one fault that seat received, and fails the test
// for any other number of faults or for an error that is not a fault.
func recordFaultOf(t *testing.T, seat *matchertest.Seat) *fault.Error {
	t.Helper()
	faults := seat.Faults()
	var got *fault.Error
	if len(faults) != 1 || !errors.As(faults[0], &got) {
		t.Fatalf("the seat received the faults %v, want one fault", faults)
	}
	return got
}

func TestRecord(t *testing.T) {
	t.Parallel()

	t.Run("WriteRecord", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the record into the artifact of a test's seat", func(t *testing.T) {
			t.Parallel()

			seat := &attrSeat{}
			matcher.WriteRecord(seat, recordOp, "dokimi.suite", `{"seed":"7"}`)
			matcher.WriteRecord(seat, recordOp, "dokimi.suite", " \n{}")
			want := [][2]string{{"dokimi.suite.1", `{"seed":"7"}`}, {"dokimi.suite.2", "  {}"}}
			if !slices.Equal(seat.written, want) {
				t.Fatalf("the seat received %v, want %v", seat.written, want)
			}
		})

		t.Run("writes a record of a kind with each boundary character of the pattern", func(t *testing.T) {
			t.Parallel()

			seat := &attrSeat{}
			matcher.WriteRecord(seat, recordOp, "dokimi.zaz-09", `{}`)
			want := [][2]string{{"dokimi.zaz-09.1", `{}`}}
			if !slices.Equal(seat.written, want) || len(seat.Faults()) != 0 {
				t.Fatalf("the seat received %v and the faults %v, want %v", seat.written, seat.Faults(), want)
			}
		})

		tests := []struct {
			name       string
			giveKind   string
			giveRecord string
			wantField  string
			wantReason string
		}{
			{
				name:       "ends the call with a fault for a kind without the prefix dokimi.",
				giveKind:   "suite",
				wantField:  "kind",
				wantReason: `"suite" does not match dokimi\.[a-z][a-z0-9-]*`,
			},
			{
				name:       "ends the call with a fault for a kind that ends after the prefix",
				giveKind:   "dokimi.",
				wantField:  "kind",
				wantReason: `"dokimi." does not match dokimi\.[a-z][a-z0-9-]*`,
			},
			{
				name:       "ends the call with a fault for a kind that starts with a digit",
				giveKind:   "dokimi.1suite",
				wantField:  "kind",
				wantReason: `"dokimi.1suite" does not match dokimi\.[a-z][a-z0-9-]*`,
			},
			{
				name:       "ends the call with a fault for a kind that starts with a character after z",
				giveKind:   "dokimi.{suite",
				wantField:  "kind",
				wantReason: `"dokimi.{suite" does not match dokimi\.[a-z][a-z0-9-]*`,
			},
			{
				name:       "ends the call with a fault for a kind with a character after z",
				giveKind:   "dokimi.su{te",
				wantField:  "kind",
				wantReason: `"dokimi.su{te" does not match dokimi\.[a-z][a-z0-9-]*`,
			},
			{
				name:       "ends the call with a fault for a kind with a capital letter",
				giveKind:   "dokimi.suIte",
				wantField:  "kind",
				wantReason: `"dokimi.suIte" does not match dokimi\.[a-z][a-z0-9-]*`,
			},
			{
				name:       "ends the call with a fault for a kind with a second dot",
				giveKind:   "dokimi.suite.x",
				wantField:  "kind",
				wantReason: `"dokimi.suite.x" does not match dokimi\.[a-z][a-z0-9-]*`,
			},
			{
				name:       "ends the call with a fault for the kind of the call records",
				giveKind:   "dokimi.assert",
				wantField:  "kind",
				wantReason: "dokimi.assert is the kind of the call records of assert",
			},
			{
				name:       "ends the call with a fault for a record that is a JSON array",
				giveRecord: `[{}]`,
				wantField:  "record",
				wantReason: "the record is not one JSON object",
			},
			{
				name:       "ends the call with a fault for a record that is no JSON",
				giveRecord: `{"seed":`,
				wantField:  "record",
				wantReason: "the record is not one JSON object",
			},
			{
				name:       "ends the call with a fault for a record of two objects",
				giveRecord: `{} {}`,
				wantField:  "record",
				wantReason: "the record is not one JSON object",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				kind, text := tt.giveKind, tt.giveRecord
				if kind == "" {
					kind = "dokimi.suite-run"
				}
				if text == "" {
					text = `{}`
				}
				seat := &matchertest.Seat{}
				matcher.WriteRecord(seat, recordOp, kind, text)
				got := recordFaultOf(t, seat)
				want := fault.Path{fault.Field(tt.wantField)}
				if got.Op != recordOp || got.Path.String() != want.String() || got.Reason != tt.wantReason {
					t.Fatalf("the fault is %+v, want %s: %s: %s", got, recordOp, want, tt.wantReason)
				}
			})
		}

		t.Run("writes nothing on a test's seat for a refused record", func(t *testing.T) {
			t.Parallel()

			seat := &attrSeat{}
			matcher.WriteRecord(seat, recordOp, "dokimi.suite", `[]`)
			if len(seat.written) != 0 {
				t.Fatalf("the seat received %v, want no attribute", seat.written)
			}
		})

		t.Run("reports no fault for a valid record on a seat outside a test", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.WriteRecord(seat, recordOp, "dokimi.suite", `{}`)
			if seat.Failed() || len(seat.Faults()) != 0 {
				t.Fatalf("the seat failed: %v, with the faults %v, want neither", seat.Failed(), seat.Faults())
			}
		})
	})
}

// TestRecordAllocs checks the allocation ceiling of WriteRecord on a seat
// that keeps call records, which writes nothing.
func TestRecordAllocs(t *testing.T) {
	checkAllocs(t, recordCases())
}

// BenchmarkRecord measures WriteRecord.
func BenchmarkRecord(b *testing.B) {
	benchAllocs(b, recordCases())
}

// recordCases returns a call of WriteRecord with its allocation ceiling.
func recordCases() []allocCase {
	return []allocCase{
		{name: "WriteRecord", allocs: 2, call: func(seat matcher.Seat) {
			matcher.WriteRecord(seat, recordOp, "dokimi.suite", `{"seed":"7"}`)
		}},
	}
}
