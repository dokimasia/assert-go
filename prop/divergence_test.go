// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"errors"
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/prop"
)

// thisFile is the base name of this file, as a failure identity states it.
const thisFile = "divergence_test.go"

// TestDivergence checks which values are differences, and what a flaky run
// reports as its first difference, pinned to the definition's behaviour
// vectors, with each side in its text form.
func TestDivergence(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Difference
			want bool
		}{
			{name: "reports true for RequestDifference", give: prop.RequestDifference, want: true},
			{name: "reports true for VerdictDifference", give: prop.VerdictDifference, want: true},
			{name: "reports false past VerdictDifference", give: invalidDifference, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a difference")
			})
		}
	})

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of the difference", func(t *testing.T) {
			t.Parallel()
			got, err := prop.FingerprintDifference.MarshalText()
			assert.NoError(t, err, "every difference has a spelling")
			assert.Equal(t, string(got), "fingerprint", "the spelling of the definition")
		})
	})

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("reports the requests of a body that diverges with every detail field", func(t *testing.T) {
			t.Parallel()
			want := map[string]any{
				outcomeField:        prop.Flaky,
				casesField:          1,
				rejectedField:       0,
				seedField:           "7",
				counterexampleField: nil,
				failureField:        nil,
				choicesField:        nil,
				othersField:         nil,
				divergenceField: &prop.Divergence{
					What:     prop.RequestDifference,
					Recorded: "integer in [0, 9]",
					Replayed: "integer in [0, 1]",
				},
				coverageField: nil,
			}
			got := detailOf(diverges(prop.Integer(0, 9), prop.Boolean()), prop.Seed(7))
			assert.Equal(t, got, want, "the record of the definition's vector")
		})

		tests := []struct {
			name string
			body func(*prop.Case)
			want *prop.Divergence
		}{
			{
				name: "states signed integer bounds",
				body: diverges(prop.Integer(-5, 5), prop.Integer(-3, 3)),
				want: requested("integer in [-5, 5]", "integer in [-3, 3]"),
			},
			{
				name: "states the bounds of a float of width 64",
				body: diverges(prop.Float(0.0, 1.0), prop.Float(0.0, 2.0)),
				want: requested("float in [0, 1] of width 64", "float in [0, 2] of width 64"),
			},
			{
				name: "states the bounds of a float of width 32 that admits NaN",
				body: diverges(prop.Float[float32](0, 1, prop.AllowNaN()), prop.Float[float32](0, 2)),
				want: requested("float in [0, 1] of width 32 or NaN", "float in [0, 2] of width 32"),
			},
			{
				name: "states the bounds of a sequence with and without a longest length",
				body: diverges(prop.Bytes(prop.MaxSize(8)), prop.Bytes(prop.MinSize(2))),
				want: requested("sequence of 0 to 8 values below 256", "sequence of 2 or more values below 256"),
			},
			{
				name: "states no request for a body that ends where it requested before",
				body: diverges(prop.Integer(0, 9), prop.Just(0)),
				want: requested("integer in [0, 9]", nil),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, detailOf(tt.body, prop.Seed(7))[divergenceField], any(tt.want), "the first request")
			})
		}

		t.Run("reports a failure that passes on replay with every detail field", func(t *testing.T) {
			t.Parallel()
			var failed bool
			body := func(c *prop.Case) {
				if c.Draw(prop.Integer(0, 1000000000), drawn) > 1000 && !failed {
					failed = true
					fail(c, "once")
				}
			}
			want := map[string]any{
				outcomeField:        prop.Flaky,
				casesField:          1,
				rejectedField:       0,
				seedField:           "7",
				counterexampleField: []prop.Entry{prop.Drawn{Label: drawn, Value: 558560502}},
				failureField:        assert.Failure{Assertion: "once"},
				choicesField:        nil,
				othersField:         nil,
				divergenceField:     &prop.Divergence{What: prop.VerdictDifference, Index: 1, Recorded: "once"},
				coverageField:       nil,
			}
			assert.Equal(t, detailOf(body, prop.Seed(7)), want, "the record of the definition's vector")
		})

		t.Run("states the location of a message", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			body := once(func(c *prop.Case) { c.Fatalf("stop%s", here(&at)) })
			got := detailOf(body, prop.Seed(7))[divergenceField]
			assert.Equal(t, got, any(verdict(fmt.Sprintf("message at %s:%d", thisFile, at.Line), nil)),
				"the message's frame")
		})

		t.Run("states the type and the location of a panic", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			body := once(func(*prop.Case) { panic(errors.New("stop" + here(&at))) })
			got := detailOf(body, prop.Seed(7))[divergenceField]
			want := verdict(fmt.Sprintf("panic of *errors.errorString at %s:%d", thisFile, at.Line), nil)
			assert.Equal(t, got, any(want), "the panic's type and frame")
		})

		t.Run("states the assertion and the location of a record", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			body := once(func(c *prop.Case) { assert.True(c, false, "the flag is set"+here(&at)) })
			got := detailOf(body, prop.Seed(7))[divergenceField]
			assert.Equal(t, got, any(verdict(fmt.Sprintf("true at %s:%d", thisFile, at.Line), nil)),
				"the assertion and its frame")
		})

		t.Run("states the assertion and the contract of a record without a location", func(t *testing.T) {
			t.Parallel()
			body := once(func(c *prop.Case) { c.Report(assert.Failure{Assertion: big, Contract: "it fits"}, true) })
			got := detailOf(body, prop.Seed(7))[divergenceField]
			assert.Equal(t, got, any(verdict("big (it fits)", nil)), "the assertion and the contract")
		})

		t.Run("states the failure of a replay that fails another way", func(t *testing.T) {
			t.Parallel()
			var calls int
			body := func(c *prop.Case) {
				calls++
				if calls == 1 {
					fail(c, "first")
				}
				fail(c, "later")
			}
			got := detailOf(body, prop.Seed(7))[divergenceField]
			assert.Equal(t, got, any(verdict("first", "later")), "both failures")
		})

		t.Run("states the fingerprints of a replay that observes another", func(t *testing.T) {
			t.Parallel()
			var calls uint64
			body := func(c *prop.Case) {
				calls++
				c.Observe(calls)
				fail(c, always)
			}
			got := detailOf(body, prop.Seed(7))[divergenceField]
			want := &prop.Divergence{What: prop.FingerprintDifference, Recorded: uint64(1), Replayed: uint64(2)}
			assert.Equal(t, got, any(want), "the first fingerprint of each run")
		})

		t.Run("states no fingerprint where the recorded run observed fewer", func(t *testing.T) {
			t.Parallel()
			var calls int
			body := func(c *prop.Case) {
				calls++
				for range calls {
					c.Observe(7)
				}
				fail(c, always)
			}
			got := detailOf(body, prop.Seed(7))[divergenceField]
			want := &prop.Divergence{What: prop.FingerprintDifference, Index: 1, Replayed: uint64(7)}
			assert.Equal(t, got, any(want), "the replay's second fingerprint")
		})
	})
}

// TestDivergenceAllocs checks that Valid allocates nothing, and that
// MarshalText allocates its text.
func TestDivergenceAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.VerdictDifference.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = prop.VerdictDifference.MarshalText() }, 1, "MarshalText allocates its text")
}

// BenchmarkDivergence measures Valid under a ceiling of no allocation, and
// MarshalText.
func BenchmarkDivergence(b *testing.B) {
	b.Run("MarshalText", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(1)
		defer c.End()
		for c.Loop() {
			got, _ = prop.VerdictDifference.MarshalText()
		}
		assert.Equal(b, string(got), "verdict", "the spelling")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.VerdictDifference.Valid()
		}
		assert.True(b, got, "VerdictDifference is a difference")
	})
}

// requested returns the divergence of two requests at the first choice.
func requested(recorded, replayed any) *prop.Divergence {
	return &prop.Divergence{What: prop.RequestDifference, Recorded: recorded, Replayed: replayed}
}

// verdict returns the divergence of a replay of a failing case without a
// choice.
func verdict(recorded, replayed any) *prop.Divergence {
	return &prop.Divergence{What: prop.VerdictDifference, Recorded: recorded, Replayed: replayed}
}
