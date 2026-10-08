// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"errors"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The allocations of the combinators, measured.
const (
	// newGeneratorAllocs are the allocations of NewGenerator and
	// NewInvertible: the decode with its type erased.
	newGeneratorAllocs = 1
	// combinatorAllocs are the allocations of Bind and Composite: the
	// decode and the decode with its type erased.
	combinatorAllocs = 2
	// justAllocs are the allocations of Just: the decode, the decode with
	// its type erased, and the inverse.
	justAllocs = 3
	// mapAllocs are the allocations of Map of an integer: the decode, the
	// decode with its type erased, and the mapping of its values for the
	// explain phase.
	mapAllocs = 3
	// mapBackAllocs are the allocations of MapBack of an integer: the
	// allocations of Map, the inverse, and the neutral value of a draw.
	mapBackAllocs = 5
	// eraseAllocs are the allocations of Erase of a generator with an
	// inverse: the inverse with its type erased.
	eraseAllocs = 1
	// filterAllocs are the allocations of Filter: its attempt, the decode,
	// the decode with its type erased, and the inverse.
	filterAllocs = 4
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
			e := engine.Bridge(
				func(c *engine.Case) { got = engine.Draw(c, evens, drawn) },
				[]byte{3, 4},
				engine.Settings{},
			)
			assert.Equal(t, got, 4, "the second attempt's value")
			assert.True(t, sameChoices(e.Case.Choices(), integers(4)), "the rejected attempt is gone from the record")
			assert.Equal(t, labels(e.Case.Spans()), []string{"filter", "integer"}, "the kept attempt's spans")
		})

		t.Run("rejects the case after three rejected attempts", func(t *testing.T) {
			t.Parallel()
			var attempts int
			counted := bytes.Map(func(v int) int { attempts++; return v }).Filter(even)
			e := engine.Bridge(
				func(c *engine.Case) { engine.Draw(c, counted, drawn) },
				[]byte{1, 3, 5, 7},
				engine.Settings{},
			)
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

	t.Run("Neutral", func(t *testing.T) {
		t.Parallel()

		digit := engine.Integer(0, 9)
		double := func(v int) int { return 2 * v }
		doubled := digit.MapBack(double, halve)

		t.Run("returns the value of a draw from a generator that states its values itself", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, drawnFrom(digit, 4), any(4), "the drawn value")
			assert.Equal(t, neutralOf(t, digit, 4), any(4), "the value itself")
		})

		t.Run("returns the value that a draw from MapBack maps from", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, drawnFrom(doubled, 4), any(8), "the mapped value")
			assert.Equal(t, neutralOf(t, doubled, 4), any(4), "the integer it maps from")
		})

		t.Run("returns the value of the innermost generator through MapBack of MapBack", func(t *testing.T) {
			t.Parallel()
			text := doubled.MapBack(strconv.Itoa, strconv.Atoi)
			assert.Equal(t, drawnFrom(text, 4), any("8"), "the text of the doubled value")
			assert.Equal(t, neutralOf(t, text, 4), any(4), "the integer of the innermost generator")
		})

		t.Run("returns the value as its source states it through Filter and Erase", func(t *testing.T) {
			t.Parallel()
			kept := engine.Erase(doubled.Filter(func(v int) bool { return v < 100 }))
			assert.Equal(t, neutralOf(t, kept, 4), any(4), "the integer that the filtered value maps from")
		})

		t.Run("returns the value itself through Map, which states no inverse", func(t *testing.T) {
			t.Parallel()
			again := doubled.Map(double)
			assert.Equal(t, neutralOf(t, again, 4), any(16), "the value that Map returns")
		})

		t.Run("returns the value itself when back refuses it", func(t *testing.T) {
			t.Parallel()
			refused := digit.MapBack(double, func(int) (int, error) { return 0, errors.New("no way back") })
			assert.Equal(t, neutralOf(t, refused, 4), any(8), "the mapped value")
		})
	})
}

// drawnFrom returns the value that g decodes from a case replaying the
// integer choice v.
func drawnFrom[T any](g engine.Generator[T], v int64) any {
	var got T
	engine.Replay(func(c *engine.Case) { got = engine.Draw(c, g, drawn) }, integers(v), nil)
	return got
}

// neutralOf returns the neutral value of the one draw from g of a case
// replaying the integer choice v.
func neutralOf[T any](tb testing.TB, g engine.Generator[T], v int64) any {
	tb.Helper()
	e := engine.Replay(func(c *engine.Case) { engine.Draw(c, g, drawn) }, integers(v), nil)
	draws := e.Case.Draws()
	assert.Length(tb, draws, 1, "one draw")
	return draws[0].Neutral()
}

// TestGeneratorAllocs checks that a Generator's id allocates nothing,
// and the allocation ceiling of each combinator and of a replayed draw.
func TestGeneratorAllocs(t *testing.T) {
	g := engine.Integer(0, 9)
	even := func(v int) bool { return v%2 == 0 }
	double := func(v int) int { return 2 * v }
	sum := func(c *engine.Case) int { return engine.Draw(c, g, drawn) }
	body := func(c *engine.Case) { engine.Draw(c, g, drawn) }
	choices := integers(7)
	assert.MaxAllocs(t, func() { _ = engine.NewGenerator("pair", decodePair) }, newGeneratorAllocs,
		"NewGenerator allocates its erased decode")
	assert.MaxAllocs(t, func() { _ = engine.NewInvertible("pair", decodePair, invertPair) }, newGeneratorAllocs,
		"NewInvertible allocates its erased decode")
	assert.MaxAllocs(t, func() { _ = engine.Erase(g) }, eraseAllocs, "Erase allocates its erased inverse")
	assert.MaxAllocs(t, func() { _ = g.ID() }, 0, "ID allocates nothing")
	assert.MaxAllocs(t, func() { _ = g.Map(double) }, mapAllocs, "Map allocates its decodes and its mapping")
	assert.MaxAllocs(t, func() { _ = g.MapBack(double, halve) }, mapBackAllocs,
		"MapBack allocates its decodes, its mapping and its inverse")
	assert.MaxAllocs(t, func() { _ = g.Filter(even) }, filterAllocs,
		"Filter allocates its attempt, its decodes and its inverse")
	assert.MaxAllocs(t, func() { _ = g.Bind(engine.Just[int]) }, combinatorAllocs, "Bind allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Composite(sum) }, combinatorAllocs, "Composite allocates its decodes")
	assert.MaxAllocs(t, func() { _ = engine.Just(7) }, justAllocs, "Just allocates its decodes and its inverse")
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

	b.Run("NewInvertible", func(b *testing.B) {
		var got engine.Generator[[2]uint64]
		c := bench.Start(b).MaxAllocs(newGeneratorAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.NewInvertible("pair", decodePair, invertPair)
		}
		assert.Equal(b, got.ID(), "pair", "the id")
	})

	b.Run("Erase", func(b *testing.B) {
		var got engine.Generator[any]
		c := bench.Start(b).MaxAllocs(eraseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Erase(g)
		}
		assert.Equal(b, got.ID(), "integer", "the id of the generator")
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

	b.Run("MapBack", func(b *testing.B) {
		var got engine.Generator[int]
		double := func(v int) int { return 2 * v }
		c := bench.Start(b).MaxAllocs(mapBackAllocs)
		defer c.End()
		for c.Loop() {
			got = g.MapBack(double, halve)
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
		c := bench.Start(b).MaxAllocs(justAllocs)
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
