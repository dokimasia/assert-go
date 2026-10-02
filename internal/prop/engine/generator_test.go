// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// drawn is the label of the one value a test body draws.
const drawn = "value"

// The allocations of the combinators and of a draw, measured.
const (
	// newGeneratorAllocs are the allocations of NewGenerator: the decode
	// with its type erased.
	newGeneratorAllocs = 1
	// combinatorAllocs are the allocations of Bind, Composite and Just: the
	// decode and the decode with its type erased.
	combinatorAllocs = 2
	// mapAllocs are the allocations of Map of an integer: the decode, the
	// decode with its type erased, and the mapping of its values for the
	// explain phase.
	mapAllocs = 3
	// filterAllocs are the allocations of Filter: its attempt, the decode
	// and the decode with its type erased.
	filterAllocs = 3
	// drawAllocs are the allocations of a whole case that replays one draw,
	// its goroutine and its recorder included.
	drawAllocs = 11
)

// TestGenerator checks the combinators, the draw that records a value, and
// the spans each of them opens.
func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("NewGenerator", func(t *testing.T) {
		t.Parallel()

		pair := engine.NewGenerator("pair", decodePair)

		t.Run("returns a generator with the stated id", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, pair.ID(), "pair", "the id")
		})

		t.Run("returns a generator whose decode opens no span of its own", func(t *testing.T) {
			t.Parallel()
			got, e := decode(t, pair, integers(3, 4)...)
			assert.Equal(t, got, [2]uint64{3, 4}, "the two choices")
			assert.Empty(t, e.Case.Spans(), "no span")
		})
	})

	t.Run("ID", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the id the definition gives the generator", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, engine.Integer(0, 9).ID(), "integer", "the id")
		})
	})

	t.Run("Map", func(t *testing.T) {
		t.Parallel()

		t.Run("returns f of each value without a span of its own", func(t *testing.T) {
			t.Parallel()
			doubled := engine.Integer(0, 9).Map(func(v int) int { return 2 * v })
			got, e := decode(t, doubled, integers(4)...)
			assert.Equal(t, got, 8, "the mapped value")
			assert.Equal(t, labels(e.Case.Spans()), []string{"integer"}, "the source's span alone")
		})

		t.Run("returns a generator whose nearest passing value is f of the integer's", func(t *testing.T) {
			t.Parallel()
			doubled := engine.Integer(0, 10_000).Map(func(v int) int { return 2 * v })
			got := engine.Run(func(c *engine.Case) {
				if engine.Draw(c, doubled, drawn) >= 2002 {
					c.Report(assert.Failure{Assertion: "big"}, false)
				}
			}, settled())
			want := []engine.Explained{
				{Label: drawn, Value: 2002, Relevance: engine.ValueMatters, NearestPassing: 2000},
			}
			assert.Equal(t, got.Explanation, want, "twice the minimal integer and twice the one below it")
		})

		t.Run("returns a generator of no nearest passing value from one that is no integer", func(t *testing.T) {
			t.Parallel()
			labelled := engine.Boolean(1, 2).Map(func(v bool) string {
				if v {
					return "on"
				}
				return "off"
			})
			got := engine.Run(func(c *engine.Case) {
				if engine.Draw(c, labelled, drawn) == "on" {
					c.Report(assert.Failure{Assertion: "on"}, false)
				}
			}, settled())
			want := []engine.Explained{{Label: drawn, Value: "on", Relevance: engine.ValueMatters}}
			assert.Equal(t, got.Explanation, want, "the value that matters, without a nearest passing value")
		})
	})

	t.Run("Filter", func(t *testing.T) {
		t.Parallel()

		even := func(v int) bool { return v%2 == 0 }
		bytes := engine.Integer(0, 255)

		t.Run("returns the first value that keep accepts", func(t *testing.T) {
			t.Parallel()
			var got int
			evens := bytes.Filter(even)
			e := engine.Bridge(func(c *engine.Case) { got = engine.Draw(c, evens, drawn) }, []byte{3, 4}, nil)
			assert.Equal(t, got, 4, "the second attempt's value")
			assert.True(t, sameChoices(e.Case.Choices(), integers(4)), "the rejected attempt is gone from the record")
			assert.Equal(t, labels(e.Case.Spans()), []string{"filter", "integer"}, "the kept attempt's spans")
		})

		t.Run("rejects the case after three rejected attempts", func(t *testing.T) {
			t.Parallel()
			var attempts int
			counted := bytes.Map(func(v int) int { attempts++; return v }).Filter(even)
			e := engine.Bridge(func(c *engine.Case) { engine.Draw(c, counted, drawn) }, []byte{1, 3, 5, 7}, nil)
			assert.Equal(t, e.Status, engine.CaseRejected, "the filter gives up")
			assert.Equal(t, attempts, 3, "three attempts")
			assert.True(t, sameChoices(e.Case.Choices(), integers(5)), "the last attempt's choice")
		})

		t.Run("returns the replayed value of a kept attempt at the first attempt", func(t *testing.T) {
			t.Parallel()
			got, e := decode(t, bytes.Filter(even), integers(4)...)
			assert.Equal(t, got, 4, "the recorded value")
			assert.Equal(t, e.Status, engine.CasePassed, "the case passes")
		})

		t.Run("rejects a replayed value that keep refuses at every attempt", func(t *testing.T) {
			t.Parallel()
			_, e := decode(t, engine.Integer(0, 9).Filter(even), integers(3)...)
			assert.Equal(t, e.Status, engine.CaseRejected, "each attempt reads the recorded 3 again")
			assert.True(t, sameChoices(e.Case.Choices(), integers(3)), "the last attempt's choice")
		})

		t.Run("returns a collection that keep accepts", func(t *testing.T) {
			t.Parallel()
			pairs := engine.List(engine.Integer(0, 9), sizes(t, 0, 3)).Filter(func(v []int) bool { return len(v) >= 2 })
			got, e := decode(t, pairs, integers(1, 3, 1, 4, 0)...)
			assert.Equal(t, got, []int{3, 4}, "the recorded list")
			assert.True(t, sameChoices(e.Case.Choices(), integers(1, 3, 1, 4, 0)), "the recorded choices")
		})

		t.Run("draws each attempt from the source as the definition pins", func(t *testing.T) {
			t.Parallel()
			thirds := engine.Integer(0, 9).Filter(func(v int) bool { return v%3 == 0 })
			values, _ := generated(thirds, 42, 8)
			assert.Equal(t, values, []int{3, 6, 9, 0, 6, 6, 6, 6}, "the first eight cases of seed 42")
		})

		t.Run("reuses no value of a rejected attempt", func(t *testing.T) {
			t.Parallel()
			heavy := engine.List(engine.Integer(0, 9), sizes(t, 0, 4)).Filter(func(v []int) bool { return sum(v) > 10 })
			_, records := generated(heavy, 7, 6)
			want := [][]choice.Choice{
				integers(1, 8, 0),
				integers(1, 0, 1, 6, 1, 8, 1, 8, 0),
				integers(0),
				integers(1, 1, 0),
				integers(1, 2, 1, 2, 1, 9, 0),
				integers(1, 7, 1, 4, 1, 0, 1, 5, 0),
			}
			assert.True(t, sameRecords(records, want), "the records of the first six cases of seed 7")
		})
	})

	t.Run("Bind", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a value of the generator that the first value chooses", func(t *testing.T) {
			t.Parallel()
			fixed, err := choice.NewSizes(2, 2)
			assert.NoError(t, err, "the sizes are valid")
			pairs := engine.Integer(0, 9).Bind(func(v int) engine.Generator[[]int] {
				return engine.List(engine.Just(v), fixed)
			})
			got, e := decode(t, pairs, integers(7, 1, 1, 0)...)
			assert.Equal(t, got, []int{7, 7}, "two copies of the first value")
			assert.Equal(t, e.Case.Spans()[0], engine.Span{Label: "bind", Start: 0, End: 4, Parent: -1},
				"one span around both")
		})
	})

	t.Run("Composite", func(t *testing.T) {
		t.Parallel()

		t.Run("returns what f returns from its draws in one span", func(t *testing.T) {
			t.Parallel()
			digit := engine.Integer(0, 9)
			sum := engine.Composite(func(c *engine.Case) int {
				return engine.Draw(c, digit, "first") + engine.Draw(c, digit, "second")
			})
			got, e := decode(t, sum, integers(3, 4)...)
			assert.Equal(t, got, 7, "the sum")
			assert.Equal(t, e.Case.Spans()[0], engine.Span{Label: "composite", Start: 0, End: 2, Parent: -1},
				"the span")
			assert.Equal(t, drawLabels(e.Case.Draws()), []string{"first", "second", drawn},
				"the inner draws, then the outer")
		})
	})

	t.Run("Just", func(t *testing.T) {
		t.Parallel()

		t.Run("returns its value without a choice", func(t *testing.T) {
			t.Parallel()
			got, e := decode(t, engine.Just("x"), integers(3, 4)...)
			assert.Equal(t, got, "x", "the stated value")
			assert.Empty(t, e.Case.Choices(), "no choice")
			assert.Equal(t, labels(e.Case.Spans()), []string{"just"}, "one span")
		})
	})

	t.Run("Draw", func(t *testing.T) {
		t.Parallel()

		t.Run("records the label, the value and the first span of the generator", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				engine.Draw(c, engine.Integer(0, 9), "a")
				engine.Draw(c, engine.Just(1.5), "b")
			}, integers(6), nil)
			draws := e.Case.Draws()
			assert.Equal(t, [2]any{draws[0].Label, draws[0].Value}, [2]any{"a", 6}, "the first draw")
			assert.Equal(t, [2]any{draws[1].Span, draws[1].Value}, [2]any{1, 1.5}, "the second draw's span")
		})
	})
}

// TestGeneratorZeroAlloc checks that a Generator's id allocates nothing,
// and the allocation ceiling of each combinator and of a replayed draw.
func TestGeneratorZeroAlloc(t *testing.T) {
	g := engine.Integer(0, 9)
	even := func(v int) bool { return v%2 == 0 }
	double := func(v int) int { return 2 * v }
	sum := func(c *engine.Case) int { return engine.Draw(c, g, drawn) }
	body := func(c *engine.Case) { engine.Draw(c, g, drawn) }
	choices := integers(7)
	assert.MaxAllocs(t, func() { _ = engine.NewGenerator("pair", decodePair) }, newGeneratorAllocs,
		"NewGenerator allocates its erased decode")
	assert.MaxAllocs(t, func() { _ = g.ID() }, 0, "ID allocates nothing")
	assert.MaxAllocs(t, func() { _ = g.Map(double) }, mapAllocs, "Map allocates its decodes and its mapping")
	assert.MaxAllocs(t, func() { _ = g.Filter(even) }, filterAllocs, "Filter allocates its attempt and its decodes")
	assert.MaxAllocs(t, func() { _ = g.Bind(engine.Just[int]) }, combinatorAllocs, "Bind allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Composite(sum) }, combinatorAllocs, "Composite allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Just(7) }, combinatorAllocs, "Just allocates its decodes")
	assert.MaxAllocs(t, func() { engine.Replay(body, choices, nil) }, drawAllocs, "a replayed draw allocates its case")
}

// BenchmarkGenerator measures the id of a generator, each combinator, and
// a draw from a replayed case.
func BenchmarkGenerator(b *testing.B) {
	g := engine.Integer(0, 9)

	b.Run("NewGenerator", func(b *testing.B) {
		var got engine.Generator[[2]uint64]
		c := bench.Start(b).MaxAllocs(newGeneratorAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.NewGenerator("pair", decodePair)
		}
		assert.Equal(b, got.ID(), "pair", "the id")
	})

	b.Run("ID", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = g.ID()
		}
		assert.Equal(b, got, "integer", "the id")
	})

	b.Run("Map", func(b *testing.B) {
		var got engine.Generator[int]
		double := func(v int) int { return 2 * v }
		c := bench.Start(b).MaxAllocs(mapAllocs)
		defer c.End()
		for c.Loop() {
			got = g.Map(double)
		}
		assert.Equal(b, got.ID(), "integer", "the id of the source")
	})

	b.Run("Filter", func(b *testing.B) {
		var got engine.Generator[int]
		even := func(v int) bool { return v%2 == 0 }
		c := bench.Start(b).MaxAllocs(filterAllocs)
		defer c.End()
		for c.Loop() {
			got = g.Filter(even)
		}
		assert.Equal(b, got.ID(), "filter", "the id")
	})

	b.Run("Bind", func(b *testing.B) {
		var got engine.Generator[int]
		c := bench.Start(b).MaxAllocs(combinatorAllocs)
		defer c.End()
		for c.Loop() {
			got = g.Bind(engine.Just[int])
		}
		assert.Equal(b, got.ID(), "bind", "the id")
	})

	b.Run("Composite", func(b *testing.B) {
		var got engine.Generator[int]
		sum := func(c *engine.Case) int { return engine.Draw(c, g, drawn) }
		c := bench.Start(b).MaxAllocs(combinatorAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Composite(sum)
		}
		assert.Equal(b, got.ID(), "composite", "the id")
	})

	b.Run("Just", func(b *testing.B) {
		var got engine.Generator[int]
		c := bench.Start(b).MaxAllocs(combinatorAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Just(7)
		}
		assert.Equal(b, got.ID(), "just", "the id")
	})

	b.Run("Draw", func(b *testing.B) {
		var got int
		choices := integers(7)
		body := func(c *engine.Case) { got = engine.Draw(c, g, drawn) }
		c := bench.Start(b).MaxAllocs(drawAllocs)
		defer c.End()
		for c.Loop() {
			engine.Replay(body, choices, nil)
		}
		assert.Equal(b, got, 7, "the replayed value")
	})
}

// decodePair returns two value choices of the digits, made on the case
// without a span.
func decodePair(c *engine.Case) [2]uint64 {
	first := c.Integer(digitRange).Magnitude()
	return [2]uint64{first, c.Integer(digitRange).Magnitude()}
}

// decode returns the value that g decodes from a case replaying choices,
// with the run of that case.
func decode[T any](tb testing.TB, g engine.Generator[T], choices ...choice.Choice) (T, engine.Execution) {
	tb.Helper()
	var got T
	e := engine.Replay(func(c *engine.Case) { got = engine.Draw(c, g, drawn) }, choices, nil)
	return got, e
}

// integers returns integer choices of the values.
func integers(values ...int64) []choice.Choice {
	out := make([]choice.Choice, len(values))
	for i, v := range values {
		out[i] = choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(v)}
	}
	return out
}

// unsigned returns the integer choice of v.
func unsigned(v uint64) choice.Choice {
	return choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(v)}
}

// float returns the float choice of v.
func float(v float64) choice.Choice {
	return choice.Choice{Kind: choice.Float, Float: v}
}

// sequence returns the sequence choice of the elements.
func sequence(elements ...uint32) choice.Choice {
	return choice.Choice{Kind: choice.Sequence, Sequence: elements}
}

// generated returns the values that g decodes in the first count cases of
// seed, and the choices that each case recorded.
func generated[T any](g engine.Generator[T], seed uint64, count int) ([]T, [][]choice.Choice) {
	values := make([]T, count)
	records := make([][]choice.Choice, count)
	for i := range count {
		e := engine.Generate(func(c *engine.Case) { values[i] = engine.Draw(c, g, drawn) }, seed, uint64(i), nil)
		records[i] = e.Case.Choices()
	}
	return values, records
}

// sameChoices reports whether a and b are the same choices in order, with
// floats compared by their bits.
func sameChoices(a, b []choice.Choice) bool {
	return slices.EqualFunc(a, b, choice.Choice.Equal)
}

// sameRecords reports whether a and b are the same choice sequences in
// order.
func sameRecords(a, b [][]choice.Choice) bool {
	return slices.EqualFunc(a, b, sameChoices)
}

// labels returns the labels of spans, in order.
func labels(spans []engine.Span) []string {
	out := make([]string, len(spans))
	for i, span := range spans {
		out[i] = span.Label
	}
	return out
}

// drawLabels returns the labels of draws, in order.
func drawLabels(draws []engine.Drawn) []string {
	out := make([]string, len(draws))
	for i, d := range draws {
		out[i] = d.Label
	}
	return out
}

// sizes returns the lengths from minSize to maxSize, failing the test when
// they are invalid.
func sizes(tb testing.TB, minSize, maxSize int) choice.Sizes {
	tb.Helper()
	s, err := choice.NewSizes(minSize, maxSize)
	assert.NoError(tb, err, "the sizes are valid")
	return s
}

// unbounded returns the lengths of minSize or more, failing the test when
// they are invalid.
func unbounded(tb testing.TB, minSize int) choice.Sizes {
	tb.Helper()
	s, err := choice.NewUnboundedSizes(minSize)
	assert.NoError(tb, err, "the sizes are valid")
	return s
}
