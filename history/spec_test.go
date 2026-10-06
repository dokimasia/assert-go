// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
)

// The allocation ceilings of the spec, measured.
const (
	// specFromAllocs are the allocations of SpecFrom: the closure of the
	// spec's Next.
	specFromAllocs = 1
	// returnedAllocs are the allocations of Returned of a known int output.
	returnedAllocs = 0
)

// inc is the operation of the counter.
const inc = "inc"

// TestSpec checks the spec that SpecFrom builds from a subject, and what an
// operation may have returned.
func TestSpec(t *testing.T) {
	t.Parallel()

	t.Run("SpecFrom", func(t *testing.T) {
		t.Parallel()

		t.Run("accepts each call whose output the subject returns after the calls before it", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, detailOf(incs(1, 2), history.SpecFrom(counter)), "the counter returns 1 and then 2")
		})

		t.Run("rejects a call whose output differs from the subject's", func(t *testing.T) {
			t.Parallel()
			got := detailOf(incs(1, 3), history.SpecFrom(counter))
			assert.Equal(t, got[outcomeField], any(history.Violated), "the counter returns 2")
			assert.Equal(t, got[statesField], any([][]history.Operation{{{Name: inc}}}),
				"the state lists the first call without its output")
		})

		t.Run("counts d + 1 steps for a step from a state of d calls", func(t *testing.T) {
			t.Parallel()
			got := detailOf(incs(1, 2, 9), history.SpecFrom(counter))
			assert.Equal(t, got[stepsField], any(6), "steps of 1, 2 and 3")
			assert.Equal(t, got[statesField], any([][]history.Operation{{{Name: inc}, {Name: inc}}}),
				"the state lists the two calls that the subject accepted")
		})

		t.Run("accepts a call whose outcome is unknown", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, inc, nil).Unknown(errRefused)
			recordOK(h, 0, inc, nil, 1)
			assert.Nil(t, detailOf(h, history.SpecFrom(counter)), "the unknown call takes no effect before the read")
		})

		t.Run("applies the calls of the state in order before the call", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, "append", []any{"a"}, "a")
			recordOK(h, 0, "append", []any{"b"}, "ba")
			got := detailOf(h, history.SpecFrom(appender))
			assert.Equal(t, got[outcomeField], any(history.Violated), "the subject returns ab")
		})

		t.Run("compares a recorded output of nil with the subject's output", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, inc, nil, nil)
			got := detailOf(h, history.SpecFrom(counter))
			assert.Equal(t, got[outcomeField], any(history.Violated), "the counter returns 1, not nil")
		})

		t.Run("builds a subject for each step", func(t *testing.T) {
			t.Parallel()
			built := 0
			factory := func() history.Subject {
				built++
				return counter()
			}
			assert.Nil(t, detailOf(incs(1, 2), history.SpecFrom(factory)), "the counter returns 1 and then 2")
			assert.Equal(t, built, 2, "a subject for each of the two steps")
		})
	})

	t.Run("Returned", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give history.Operation
			v    any
			want bool
		}{
			{
				name: "returns true for a call that is not known, whatever v is",
				give: history.Operation{Name: "read"},
				v:    7,
				want: true,
			},
			{
				name: "returns true for a known output equal to v",
				give: history.Operation{Name: "read", Known: true, Output: 7},
				v:    7,
				want: true,
			},
			{
				name: "returns false for a known output that differs from v",
				give: history.Operation{Name: "read", Known: true, Output: 7},
				v:    8,
			},
			{
				name: "returns true for a known slice with the elements of v",
				give: history.Operation{Name: "read", Known: true, Output: []int{1, 2}},
				v:    []int{1, 2},
				want: true,
			},
			{
				name: "returns false for a known output of another type than v",
				give: history.Operation{Name: "read", Known: true, Output: int64(7)},
				v:    7,
			},
			{
				name: "returns false for a known output of nil against a value",
				give: history.Operation{Name: "read", Known: true},
				v:    0,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Returned(tt.v), tt.want, "what the call may have returned")
			})
		}
	})
}

// counter returns a subject whose every call returns the number of calls
// that it applied.
func counter() history.Subject {
	calls := 0
	return func(string, []any) any {
		calls++
		return calls
	}
}

// appender returns a subject whose every call appends its argument to the
// text and returns the text.
func appender() history.Subject {
	text := ""
	return func(_ string, args []any) any {
		text += args[0].(string)
		return text
	}
}

// incs returns a history of sequential calls of client 0 to inc that return
// outputs.
func incs(outputs ...int) *history.History {
	h := history.New()
	for _, output := range outputs {
		recordOK(h, 0, inc, nil, output)
	}
	return h
}

// The operation and the value of the allocation ceiling of Returned, built
// once, so that the ceiling counts Returned alone.
var (
	knownRead       = history.Operation{Name: "read", Known: true, Output: 1000}
	knownValue  any = 1000
	wasReturned bool
)

// specAllocs are the cases of the allocation ceilings of the spec.
var specAllocs = []alloctest.Case{
	{Name: "SpecFrom", Call: func(assert.TB) { _ = history.SpecFrom(counter) }, Allocs: specFromAllocs},
	{Name: "Returned", Call: func(assert.TB) { wasReturned = knownRead.Returned(knownValue) }, Allocs: returnedAllocs},
}

// TestSpecAllocs checks the allocation ceilings of SpecFrom and Returned.
func TestSpecAllocs(t *testing.T) {
	alloctest.Check(t, specAllocs)
}

// BenchmarkSpec measures SpecFrom and Returned under their ceilings.
func BenchmarkSpec(b *testing.B) {
	for _, c := range specAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
