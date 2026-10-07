// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestAssertion runs the chain, a third surface over the same comparisons,
// through the shared cases of the functions. A method that differs from its
// function fails the shared case.
func TestAssertion(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.EqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				assert.That(s, got).Equal(want, msg)
			})
	})

	t.Run("NotEqual", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotEqualCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				assert.That(s, got).NotEqual(want, msg)
			})
	})

	t.Run("Nil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.That(s, got).Nil(msg)
			})
	})

	t.Run("NotNil", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotNilCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.That(s, got).NotNil(msg)
			})
	})

	t.Run("Length", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.LengthCases(),
			func(s *matchertest.Seat, got, want any, msg string) {
				assert.That(s, got).Length(want.(int), msg)
			})
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.EmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.That(s, got).Empty(msg)
			})
	})

	t.Run("NotEmpty", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NotEmptyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.That(s, got).NotEmpty(msg)
			})
	})

	t.Run("Contains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				assert.That(s, haystack).Contains(needle, msg)
			})
	})

	t.Run("NotContains", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.NotContainsCases(),
			func(s *matchertest.Seat, haystack, needle any, msg string) {
				assert.That(s, haystack).NotContains(needle, msg)
			})
	})

	t.Run("ContainsInOrder", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ContainsInOrderCases(),
			func(s *matchertest.Seat, got, needles any, msg string) {
				assert.That(s, got).ContainsInOrder(needles.([]string), msg)
			})
	})

	t.Run("HasPrefix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasPrefixCases(),
			func(s *matchertest.Seat, got, prefix any, msg string) {
				assert.That(s, got).HasPrefix(prefix.(string), msg)
			})
	})

	t.Run("HasSuffix", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.HasSuffixCases(),
			func(s *matchertest.Seat, got, suffix any, msg string) {
				assert.That(s, got).HasSuffix(suffix.(string), msg)
			})
	})

	t.Run("Matches", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.MatchesCases(),
			func(s *matchertest.Seat, got, pattern any, msg string) {
				assert.That(s, got).Matches(pattern.(string), msg)
			})
	})

	t.Run("CloseTo", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.CloseToCases(),
			func(s *matchertest.Seat, got, want, tolerance any, msg string) {
				assert.That(s, got).CloseTo(want.(float64), tolerance.(float64), msg)
			})
	})

	t.Run("InRange", func(t *testing.T) {
		t.Parallel()
		matchertest.RunTriple(t, matchertest.InRangeCases(),
			func(s *matchertest.Seat, got, low, high any, msg string) {
				assert.That(s, got).InRange(low.(float64), high.(float64), msg)
			})
	})

	t.Run("NoError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.NoErrorOfAnyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.That(s, got).NoError(msg)
			})
	})

	t.Run("HasError", func(t *testing.T) {
		t.Parallel()
		matchertest.RunOne(t, matchertest.HasErrorOfAnyCases(),
			func(s *matchertest.Seat, got any, msg string) {
				assert.That(s, got).HasError(msg)
			})
	})

	t.Run("ErrorIs", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsOfAnyCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				assert.That(s, got).ErrorIs(matchertest.AsError(target), msg)
			})
	})

	t.Run("ErrorIsNot", func(t *testing.T) {
		t.Parallel()
		matchertest.RunPair(t, matchertest.ErrorIsNotOfAnyCases(),
			func(s *matchertest.Seat, got, target any, msg string) {
				assert.That(s, got).ErrorIsNot(matchertest.AsError(target), msg)
			})
	})

	// The cases of That test the chain alone. The shared cases state what
	// one assertion reports, and these state how the methods of one chain
	// compose.
	t.Run("That", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the receiver from a method, so calls chain", func(t *testing.T) {
			t.Parallel()

			a := assert.That(&matchertest.Seat{}, 1)
			if a.Equal(1, "is one") != a {
				t.Fatal("Equal did not return the receiver")
			}
		})

		t.Run("reports the first failure first", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			assert.That(s, 1).Equal(2, "first failure").Equal(3, "second failure")

			if records := s.Records(); len(records) == 0 || records[0].Contract != "first failure" {
				t.Fatalf("Records() = %+v, want the record of the first failure first", records)
			}
		})

		t.Run("passes a chain of methods of different families", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			assert.That(s, "store: missing").
				HasPrefix("store: ", "starts with the package").
				Contains("missing", "states what happened").
				NotEqual("", "is not empty")

			if s.Failed() {
				t.Fatalf("reported %q, want none", s.First())
			}
		})

		t.Run("applies an option to the method that it is passed to", func(t *testing.T) {
			t.Parallel()

			s := &matchertest.Seat{}
			var nilSlice []int
			assert.That(s, nilSlice).Equal([]int{}, "opted in", assert.EquateEmpty())

			if s.Failed() {
				t.Fatalf("reported %q under EquateEmpty", s.First())
			}
		})
	})
}

// chain keeps the chain that a call of That returns.
var chain *assert.Assertion[int]

// TestAssertionAllocs checks the allocation ceiling of That and of a
// passing call of each method of a chain.
func TestAssertionAllocs(t *testing.T) {
	alloctest.Check(t, assertionCases())
}

// BenchmarkAssertion measures That and a passing call of each method of a
// chain.
func BenchmarkAssertion(b *testing.B) {
	for _, c := range assertionCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// assertionCases returns a call of That and a call of That with one
// passing method of each kind, on the inputs of the functions' cases, with
// its allocation ceiling, measured.
func assertionCases() []alloctest.Case {
	items, none := []int{1, 2, 3}, []int{}
	needles := []string{"cart", "items"}
	var absent *int
	present := new(int)
	reading := 1.05
	var succeeded error
	wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", matchertest.ErrSample))
	return []alloctest.Case{
		{Name: "That", Call: func(tb assert.TB) { chain = assert.That(tb, 7) }, Allocs: 1},
		{Name: "Equal", Call: func(tb assert.TB) { assert.That(tb, 7).Equal(7, allocContract) }},
		{Name: "NotEqual", Call: func(tb assert.TB) { assert.That(tb, 7).NotEqual(8, allocContract) }},
		{Name: "Nil", Call: func(tb assert.TB) { assert.That(tb, absent).Nil(allocContract) }},
		{Name: "NotNil", Call: func(tb assert.TB) { assert.That(tb, present).NotNil(allocContract) }},
		{Name: "Length", Call: func(tb assert.TB) { assert.That(tb, items).Length(3, allocContract) }},
		{Name: "Empty", Allocs: 1, Call: func(tb assert.TB) { assert.That(tb, none).Empty(allocContract) }},
		{Name: "NotEmpty", Allocs: 1, Call: func(tb assert.TB) { assert.That(tb, items).NotEmpty(allocContract) }},
		{Name: "Contains", Allocs: 1, Call: func(tb assert.TB) {
			assert.That(tb, "a cart of three items").Contains("cart", allocContract)
		}},
		{Name: "NotContains", Allocs: 1, Call: func(tb assert.TB) {
			assert.That(tb, "a cart of three items").NotContains("truck", allocContract)
		}},
		{Name: "ContainsInOrder", Allocs: 1, Call: func(tb assert.TB) {
			assert.That(tb, "a cart of three items").ContainsInOrder(needles, allocContract)
		}},
		{Name: "HasPrefix", Allocs: 1, Call: func(tb assert.TB) {
			assert.That(tb, "store: missing").HasPrefix("store: ", allocContract)
		}},
		{Name: "HasSuffix", Allocs: 1, Call: func(tb assert.TB) {
			assert.That(tb, "store: missing").HasSuffix("missing", allocContract)
		}},
		{Name: "Matches", Allocs: 63, Call: func(tb assert.TB) {
			assert.That(tb, "order 42").Matches(`^order \d+$`, allocContract)
		}},
		{
			Name:   "CloseTo",
			Call:   func(tb assert.TB) { assert.That(tb, reading).CloseTo(1, 0.1, allocContract) },
			Allocs: 1,
		},
		{
			Name:   "InRange",
			Call:   func(tb assert.TB) { assert.That(tb, reading).InRange(0, 2, allocContract) },
			Allocs: 1,
		},
		{Name: "NoError", Call: func(tb assert.TB) { assert.That(tb, succeeded).NoError(allocContract) }},
		{Name: "HasError", Call: func(tb assert.TB) { assert.That(tb, wrapped).HasError(allocContract) }},
		{Name: "ErrorIs", Call: func(tb assert.TB) {
			assert.That(tb, wrapped).ErrorIs(matchertest.ErrSample, allocContract)
		}},
		{Name: "ErrorIsNot", Call: func(tb assert.TB) {
			assert.That(tb, wrapped).ErrorIsNot(matchertest.ErrOther, allocContract)
		}},
	}
}
