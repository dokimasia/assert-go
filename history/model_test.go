// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
)

// modelFromAllocs are the allocations of ModelFrom, measured: the closure of
// the model's Step.
const modelFromAllocs = 1

// inc is the operation of the counter.
const inc = "inc"

// TestModel checks the model that ModelFrom builds from a subject.
func TestModel(t *testing.T) {
	t.Parallel()

	t.Run("ModelFrom", func(t *testing.T) {
		t.Parallel()

		t.Run("accepts each call whose output the subject returns after the calls before it", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, detailOf(incs(1, 2), history.ModelFrom(counter)), "the counter returns 1 and then 2")
		})

		t.Run("rejects a call whose output differs from the subject's", func(t *testing.T) {
			t.Parallel()
			got := detailOf(incs(1, 3), history.ModelFrom(counter))
			assert.Equal(t, got[outcomeField], any(history.Violated), "the counter returns 2")
			assert.Equal(t, got[statesField], any([][]history.Op{{{Operation: inc}}}),
				"the state lists the first call without its output")
		})

		t.Run("counts d + 1 steps for a step from a state of d calls", func(t *testing.T) {
			t.Parallel()
			got := detailOf(incs(1, 2, 9), history.ModelFrom(counter))
			assert.Equal(t, got[stepsField], any(6), "steps of 1, 2 and 3")
			assert.Equal(t, got[statesField], any([][]history.Op{{{Operation: inc}, {Operation: inc}}}),
				"the state lists the two calls that the subject accepted")
		})

		t.Run("accepts a call whose outcome is unknown", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			h.Invoke(0, inc, nil).Unknown(errRefused)
			recordOK(h, 0, inc, nil, 1)
			assert.Nil(t, detailOf(h, history.ModelFrom(counter)), "the unknown call takes no effect before the read")
		})

		t.Run("applies the calls of the state in order before the call", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, "append", []any{"a"}, "a")
			recordOK(h, 0, "append", []any{"b"}, "ba")
			got := detailOf(h, history.ModelFrom(appender))
			assert.Equal(t, got[outcomeField], any(history.Violated), "the subject returns ab")
		})

		t.Run("compares a recorded output of nil with the subject's output", func(t *testing.T) {
			t.Parallel()
			h := history.New()
			recordOK(h, 0, inc, nil, nil)
			got := detailOf(h, history.ModelFrom(counter))
			assert.Equal(t, got[outcomeField], any(history.Violated), "the counter returns 1, not nil")
		})

		t.Run("builds a subject for each step", func(t *testing.T) {
			t.Parallel()
			built := 0
			factory := func() history.Subject {
				built++
				return counter()
			}
			assert.Nil(t, detailOf(incs(1, 2), history.ModelFrom(factory)), "the counter returns 1 and then 2")
			assert.Equal(t, built, 2, "a subject for each of the two steps")
		})
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

// modelAllocs are the cases of the allocation ceiling of ModelFrom.
var modelAllocs = []alloctest.Case{
	{Name: "ModelFrom", Call: func(assert.TB) { _ = history.ModelFrom(counter) }, Allocs: modelFromAllocs},
}

// TestModelAllocs checks the allocation ceiling of ModelFrom.
func TestModelAllocs(t *testing.T) {
	alloctest.Check(t, modelAllocs)
}

// BenchmarkModel measures ModelFrom under its ceiling.
func BenchmarkModel(b *testing.B) {
	for _, c := range modelAllocs {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
