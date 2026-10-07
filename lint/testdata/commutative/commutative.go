// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package commutative

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type set struct{}

func (set) Union(a, b []int) []int { return nil }

func add(a, b int) int { return a + b }

func sum[T int | float64](a, b T) T { return a + b }

func next() int { return 1 }

func correct(t *testing.T) {
	assert.Commutative(t, add, 2, 3, "addition commutes")
}

func swapped(t *testing.T, a, b int, s set, xs, ys []int, f func(int, int) int) {
	assert.Equal(t, add(a, b), add(b, a), "addition commutes")              // want `commutative: state the check with Commutative`
	expect.Equal(t, s.Union(xs, ys), s.Union(ys, xs), "the union commutes") // want `commutative: state the check with Commutative`
	assert.Equal(t, f(a, b), f(b, a), "the function commutes")              // want `commutative: state the check with Commutative`
	assert.True(t, add(a, b) == add(b, a), "addition commutes")             // want `commutative: state the check with Commutative`
	assert.Equal(t, sum(a, b), sum(b, a), "the sum commutes")               // want `commutative: state the check with Commutative`
	assert.Equal(t, add(next(), b), add(b, next()), "addition commutes")    // want `commutative: state the check with Commutative`
	assert.Equal(t, max(a, b), max(b, a), "the maximum commutes")           // want `commutative: state the check with Commutative`
	if add(a, b) != add(b, a) {                                             // want `commutative: state the check with Commutative`
		t.Fatal("addition does not commute")
	}
	assert.Equal(t, add(a, b), add(a, b), "addition is deterministic") // want `deterministic: state the check with Deterministic`
	assert.Equal(t, add(a, a), add(a, a), "doubling is deterministic") // want `deterministic: state the check with Deterministic`
}
