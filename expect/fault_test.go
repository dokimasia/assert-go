// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"errors"
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// errMissingOption is the fault of a helper that misses an option.
var errMissingOption = errors.New("the check calls Restart without the option Restartable")

// textOnly is a seat without ReportFault. It passes the calls of TB to the
// seat that it embeds.
type textOnly struct {
	assert.TB
}

// TestFault checks where the fault of a helper arrives on the recording
// surface.
func TestFault(t *testing.T) {
	t.Parallel()

	t.Run("FaultReporter", func(t *testing.T) {
		t.Parallel()

		t.Run("is the fault reporter of the aborting surface", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, reflect.TypeFor[expect.FaultReporter](), reflect.TypeFor[assert.FaultReporter](),
				"one fault reporter")
		})
	})

	t.Run("Fault", func(t *testing.T) {
		t.Parallel()

		t.Run("passes the fault to the seat as a fault that ends the call", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			expect.Fault(seat, errMissingOption)
			assert.Length(t, seat.Faults(), 1, "the seat keeps one fault")
			expect.ErrorIs(t, seat.Faults()[0], errMissingOption, "the seat keeps the fault of the helper")
			expect.Equal(t, seat.Fatals(), []string{matcher.RenderFault(errMissingOption)},
				"the seat fails through Fatalf for a fault that ends the call")
		})

		t.Run("sends the writer's text of the fault through Fatalf of a seat that takes no fault", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			expect.Fault(textOnly{seat}, errMissingOption)
			expect.Equal(t, seat.Fatals(), []string{matcher.RenderFault(errMissingOption)},
				"the seat receives the writer's text through Fatalf")
			expect.Empty(t, seat.Errs(), "the seat receives nothing through Errorf")
			expect.Empty(t, seat.Faults(), "the seat receives no fault as an error")
		})

		t.Run("writes no call record", func(t *testing.T) {
			t.Parallel()

			r := expect.NewRecorder()
			expect.Fault(r, errMissingOption)
			expect.Empty(t, r.Records(), "the recorder keeps no call record")
		})

		t.Run("panics for a nil fault", func(t *testing.T) {
			t.Parallel()

			got := expect.Panics(t, func() { expect.Fault(expect.NewRecorder(), nil) }, "a nil fault")
			expect.Equal(t, got, any("expect: Fault(tb, nil) states no fault"), "the panic names the call")
		})
	})
}

// TestFaultAllocs checks the allocation ceiling of Fault on a recorder.
func TestFaultAllocs(t *testing.T) {
	alloctest.Check(t, faultCases())
}

// BenchmarkFault measures Fault on a recorder.
func BenchmarkFault(b *testing.B) {
	for _, c := range faultCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// faultCases returns a call of Fault on a recorder, which keeps the fault and
// its text, with its allocation ceiling.
func faultCases() []alloctest.Case {
	r := expect.NewRecorder()
	return []alloctest.Case{
		{Name: "Fault", Call: func(assert.TB) { expect.Fault(r, errMissingOption) }, Allocs: 2},
	}
}
