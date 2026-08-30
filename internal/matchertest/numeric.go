// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import "math"

// CloseToCases are the cases every surface's tolerance assertion must
// produce. Drive them with [RunTriple], reading the arguments as
// (got, want, tolerance).
//
// The NaN cases are the ones a naive implementation fails. Every
// comparison against NaN is false, so a bare `diff > tolerance` passes
// a NaN instead of rejecting it.
func CloseToCases() []Case {
	return []Case{
		{Name: "a value inside the tolerance passes", Args: []any{1.02, 1.0, 0.05}},
		{Name: "the tolerance is inclusive", Args: []any{1.5, 1.0, 0.5}},
		{Name: "an integer answers as a number", Args: []any{10, 10.2, 0.5}},
		{Name: "a defined float type answers as a number", Args: []any{celsius(20.1), 20.0, 0.5}},
		{Name: "a negative difference is still a difference", Args: []any{0.98, 1.0, 0.05}},
		{
			Name:     "a value outside the tolerance reports",
			Args:     []any{2.0, 1.0, 0.05},
			Fails:    true,
			Contains: []string{"within"},
		},
		{
			Name:     "a NaN value fails whatever the tolerance",
			Args:     []any{math.NaN(), 1.0, math.Inf(1)},
			Fails:    true,
			Contains: []string{"NaN"},
		},
		{
			Name:     "a NaN want fails",
			Args:     []any{1.0, math.NaN(), 10.0},
			Fails:    true,
			Contains: []string{"NaN"},
		},
		{
			Name:     "a NaN tolerance fails",
			Args:     []any{1.0, 1.0, math.NaN()},
			Fails:    true,
			Contains: []string{"NaN"},
		},
		{
			Name:     "a non-numeric value reports rather than panicking",
			Args:     []any{"1", 1.0, 0.5},
			Fails:    true,
			Contains: []string{"requires a number"},
		},
	}
}

// InRangeCases are the cases every surface's range assertion must
// produce. Drive them with [RunTriple], reading the arguments as
// (got, low, high).
func InRangeCases() []Case {
	return []Case{
		{Name: "a value inside passes", Args: []any{8080, 1024.0, 65535.0}},
		{Name: "the low bound is included", Args: []any{1024, 1024.0, 65535.0}},
		{Name: "the high bound is included", Args: []any{65535, 1024.0, 65535.0}},
		{Name: "a signed width answers", Args: []any{int8(5), 0.0, 10.0}},
		{Name: "an unsigned width answers", Args: []any{uint64(5), 0.0, 10.0}},
		{Name: "a 32-bit float answers", Args: []any{float32(5), 0.0, 10.0}},
		{
			Name:     "a value below reports",
			Args:     []any{80, 1024.0, 65535.0},
			Fails:    true,
			Contains: []string{"80"},
		},
		{
			Name:     "a value above reports",
			Args:     []any{70000, 1024.0, 65535.0},
			Fails:    true,
			Contains: []string{"70000"},
		},
		{
			Name:     "NaN is in no range",
			Args:     []any{math.NaN(), 0.0, 100.0},
			Fails:    true,
			Contains: []string{"NaN"},
		},
		{
			Name:     "infinity is outside a bounded range",
			Args:     []any{math.Inf(1), 0.0, 100.0},
			Fails:    true,
			Contains: []string{"Inf"},
		},
		{
			Name:     "an inverted range always fails and says so",
			Args:     []any{5, 10.0, 1.0},
			Fails:    true,
			Contains: []string{"empty range"},
		},
		{
			Name:     "a non-numeric value reports rather than panicking",
			Args:     []any{"5", 0.0, 10.0},
			Fails:    true,
			Contains: []string{"requires a number"},
		},
	}
}
