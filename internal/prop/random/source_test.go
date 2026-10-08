// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package random_test

import (
	"encoding/json"
	"math"
	"math/bits"
	"os"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/random"
)

// oracleFile contains the first 32 values of five seeds as Bob Jenkins's
// published C code prints them: decimal strings in a list per seed, keyed
// by the seed in decimal.
const oracleFile = "testdata/smallprng.json"

// TestSource checks the stream against the published C code, the stream of
// each case, and the uniform draws and coins that every draw builds on.
func TestSource(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the stream that the published C code prints for five seeds", func(t *testing.T) {
			t.Parallel()
			outputs := oracle(t)
			assert.Length(t, outputs, 5, "the oracle's seeds")
			for seed, want := range outputs {
				s := random.New(seed)
				got := make([]uint64, len(want))
				for i := range got {
					got[i] = s.Next()
				}
				assert.Equal(t, got, want, "the stream of seed "+strconv.FormatUint(seed, 10))
			}
		})
	})

	t.Run("ForCase", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the stream of the seed plus the index", func(t *testing.T) {
			t.Parallel()
			s, want := random.ForCase(40, 2), random.New(42)
			assert.Equal(t, s.Next(), want.Next(), "case 2 of seed 40 is seed 42")
		})

		t.Run("wraps the seed at 2^64", func(t *testing.T) {
			t.Parallel()
			s, want := random.ForCase(math.MaxUint64, 1), random.New(0)
			assert.Equal(t, s.Next(), want.Next(), "case 1 of the largest seed is seed 0")
		})
	})

	t.Run("Next", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the zero value's stream as zeros", func(t *testing.T) {
			t.Parallel()
			var s random.Source
			assert.Equal(t, [3]uint64{s.Next(), s.Next(), s.Next()}, [3]uint64{}, "the all-zero state")
		})
	})

	t.Run("UpTo", func(t *testing.T) {
		t.Parallel()

		t.Run("returns 0 for hi of 0 without consuming the stream", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(7), random.New(7)
			assert.Equal(t, s.UpTo(0), uint64(0), "the one value")
			assert.Equal(t, s.Next(), twin.Next(), "the next value of the stream")
		})

		t.Run("keeps the top bits and redraws a value above hi", func(t *testing.T) {
			t.Parallel()
			for _, hi := range []uint64{1, 2, 9, 999, 1<<63 + 4} {
				s, twin := random.New(hi), random.New(hi)
				shift := 64 - bits.Len64(hi)
				for range 200 {
					want := twin.Next() >> shift
					for want > hi {
						want = twin.Next() >> shift
					}
					assert.Equal(t, s.UpTo(hi), want, "a draw up to "+strconv.FormatUint(hi, 10))
				}
			}
		})

		t.Run("returns whole values for the largest hi", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(3), random.New(3)
			for range 10 {
				assert.Equal(t, s.UpTo(math.MaxUint64), twin.Next(), "a whole value of the stream")
			}
		})
	})

	t.Run("Below", func(t *testing.T) {
		t.Parallel()

		t.Run("returns 0 for n of 1 without consuming the stream", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(7), random.New(7)
			assert.Equal(t, s.Below(1), uint64(0), "the one value")
			assert.Equal(t, s.Next(), twin.Next(), "the next value of the stream")
		})

		t.Run("returns the draw of UpTo for n - 1", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(11), random.New(11)
			for n := range uint64(100) {
				assert.Equal(t, s.Below(n+2), twin.UpTo(n+1), "a draw below "+strconv.FormatUint(n+2, 10))
			}
		})

		t.Run("panics for n of 0", func(t *testing.T) {
			t.Parallel()
			s := random.New(1)
			assert.Panics(t, func() { s.Below(0) }, "an empty range has no value")
		})
	})

	t.Run("Coin", func(t *testing.T) {
		t.Parallel()

		t.Run("reports whether one draw below den is below num", func(t *testing.T) {
			t.Parallel()
			s, twin := random.New(9), random.New(9)
			for range 100 {
				assert.Equal(t, s.Coin(3, 8), twin.Below(8) < 3, "a coin of 3 in 8")
			}
		})

		t.Run("reports true for num equal to den", func(t *testing.T) {
			t.Parallel()
			s := random.New(9)
			for range 100 {
				assert.True(t, s.Coin(8, 8), "a certain coin")
			}
		})

		t.Run("reports false for num of 0", func(t *testing.T) {
			t.Parallel()
			s := random.New(9)
			for range 100 {
				assert.False(t, s.Coin(0, 8), "an impossible coin")
			}
		})

		t.Run("panics for num above den", func(t *testing.T) {
			t.Parallel()
			s := random.New(1)
			assert.Panics(t, func() { s.Coin(3, 2) }, "odds above 1")
		})

		t.Run("panics for den of 0", func(t *testing.T) {
			t.Parallel()
			s := random.New(1)
			assert.Panics(t, func() { s.Coin(0, 0) }, "odds over an empty range")
		})
	})
}

// TestSourceAllocs checks that no function or method of Source
// allocates.
func TestSourceAllocs(t *testing.T) {
	s := random.New(42)
	assert.MaxAllocs(t, func() { _ = random.New(42) }, 0, "New allocates nothing")
	assert.MaxAllocs(t, func() { _ = random.ForCase(42, 7) }, 0, "ForCase allocates nothing")
	assert.MaxAllocs(t, func() { _ = s.Next() }, 0, "Next allocates nothing")
	assert.MaxAllocs(t, func() { _ = s.UpTo(999) }, 0, "UpTo allocates nothing")
	assert.MaxAllocs(t, func() { _ = s.Below(1000) }, 0, "Below allocates nothing")
	assert.MaxAllocs(t, func() { _ = s.Coin(1, 8) }, 0, "Coin allocates nothing")
}

// BenchmarkSource measures each function and method of Source under a
// ceiling of no allocation.
func BenchmarkSource(b *testing.B) {
	b.Run("New", func(b *testing.B) {
		var got random.Source
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = random.New(42)
		}
		want := random.New(42)
		assert.Equal(b, got.Next(), want.Next(), "the stream of seed 42")
	})

	b.Run("ForCase", func(b *testing.B) {
		var got random.Source
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = random.ForCase(40, 2)
		}
		want := random.New(42)
		assert.Equal(b, got.Next(), want.Next(), "the stream of seed 42")
	})

	b.Run("Next", func(b *testing.B) {
		var got uint64
		s := random.New(42)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.Next()
		}
		assert.NotEqual(b, got, uint64(0), "a value of the stream")
	})

	b.Run("UpTo", func(b *testing.B) {
		var got uint64
		s := random.New(42)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.UpTo(999)
		}
		assert.True(b, got <= 999, "a value up to 999")
	})

	b.Run("Below", func(b *testing.B) {
		var got uint64
		s := random.New(42)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.Below(1000)
		}
		assert.True(b, got < 1000, "a value below 1000")
	})

	b.Run("Coin", func(b *testing.B) {
		var got bool
		s := random.New(42)
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.Coin(8, 8)
		}
		assert.True(b, got, "a certain coin")
	})
}

// oracle returns the values of the oracle file by seed, failing the test
// when the file does not read or does not parse.
func oracle(tb testing.TB) map[uint64][]uint64 {
	tb.Helper()
	data, err := os.ReadFile(oracleFile)
	assert.NoError(tb, err, "the oracle reads")
	var file struct {
		Outputs map[string][]string `json:"outputs"`
	}
	assert.NoError(tb, json.Unmarshal(data, &file), "the oracle parses")
	outputs := make(map[uint64][]uint64, len(file.Outputs))
	for seed, values := range file.Outputs {
		key, err := strconv.ParseUint(seed, 10, 64)
		assert.NoError(tb, err, "a seed is an unsigned 64-bit integer")
		for _, v := range values {
			value, err := strconv.ParseUint(v, 10, 64)
			assert.NoError(tb, err, "a value is an unsigned 64-bit integer")
			outputs[key] = append(outputs[key], value)
		}
	}
	return outputs
}
