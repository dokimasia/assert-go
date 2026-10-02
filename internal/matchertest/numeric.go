// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import "math"

// CloseToCases are the cases every surface's tolerance assertion must
// produce. Drive them with [RunTriple], reading the arguments as
// (got, want, tolerance).
//
// The NaN cases fail an implementation that compares the difference
// alone. Every comparison against a NaN is false, so diff > tolerance
// passes a NaN.
func CloseToCases() []Case {
	return []Case{
		{Name: "a value inside the tolerance passes", Args: []any{1.02, 1.0, 0.05}},
		{Name: "the tolerance is inclusive", Args: []any{1.5, 1.0, 0.5}},
		{Name: "an integer reads as a number", Args: []any{10, 10.2, 0.5}},
		{Name: "a defined float type reads as a number", Args: []any{celsius(20.1), 20.0, 0.5}},
		{Name: "a negative difference is still a difference", Args: []any{0.98, 1.0, 0.05}},
		{
			Name:      "a value outside the tolerance reports",
			Args:      []any{2.0, 1.0, 0.05},
			Fails:     true,
			Assertion: "close-to",
		},
		{
			Name:      "a NaN value fails whatever the tolerance",
			Args:      []any{math.NaN(), 1.0, math.Inf(1)},
			Fails:     true,
			Assertion: "close-to",
		},
		{
			Name:      "a NaN want fails",
			Args:      []any{1.0, math.NaN(), 10.0},
			Fails:     true,
			Assertion: "close-to",
		},
		{
			Name:      "a NaN tolerance fails",
			Args:      []any{1.0, 1.0, math.NaN()},
			Fails:     true,
			Assertion: "close-to",
		},
		{
			Name:      "a value that is not a number reports",
			Args:      []any{"1", 1.0, 0.5},
			Fails:     true,
			Assertion: "close-to",
		},
	}
}

// InRangeCases are the cases every surface's range assertion must
// produce. Drive them with [RunTriple], reading the arguments as
// (got, low, high).
func InRangeCases() []Case {
	return []Case{
		{Name: "a value inside passes", Args: []any{8080, 1024.0, 65535.0}},
		{Name: "a range of one value contains that value", Args: []any{5, 5.0, 5.0}},
		{Name: "the low bound is included", Args: []any{1024, 1024.0, 65535.0}},
		{Name: "the high bound is included", Args: []any{65535, 1024.0, 65535.0}},
		{Name: "an 8-bit integer reads as a number", Args: []any{int8(5), 0.0, 10.0}},
		{Name: "an unsigned integer reads as a number", Args: []any{uint64(5), 0.0, 10.0}},
		{Name: "a 32-bit float reads as a number", Args: []any{float32(5), 0.0, 10.0}},
		{
			Name:      "a value below reports",
			Args:      []any{80, 1024.0, 65535.0},
			Fails:     true,
			Assertion: "in-range",
		},
		{
			Name:      "a value above reports",
			Args:      []any{70000, 1024.0, 65535.0},
			Fails:     true,
			Assertion: "in-range",
		},
		{
			Name:      "NaN is in no range",
			Args:      []any{math.NaN(), 0.0, 100.0},
			Fails:     true,
			Assertion: "in-range",
		},
		{
			Name:      "infinity is outside a bounded range",
			Args:      []any{math.Inf(1), 0.0, 100.0},
			Fails:     true,
			Assertion: "in-range",
		},
		{
			Name:      "an inverted range contains no value",
			Args:      []any{5, 10.0, 1.0},
			Fails:     true,
			Assertion: "in-range",
		},
		{
			Name:      "a NaN low bound contains no value",
			Args:      []any{5, math.NaN(), 10.0},
			Fails:     true,
			Assertion: "in-range",
		},
		{
			Name:      "a NaN high bound contains no value",
			Args:      []any{5, 0.0, math.NaN()},
			Fails:     true,
			Assertion: "in-range",
		},
		{
			Name:      "a value that is not a number reports",
			Args:      []any{"5", 0.0, 10.0},
			Fails:     true,
			Assertion: "in-range",
		},
	}
}
