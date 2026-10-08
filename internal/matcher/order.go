// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// Pairwise reports when any adjacent pair of items fails pred, which
// receives them in slice order as (earlier, later).
//
// It states an ordering without naming one: pass a less-than for
// ascending, an after for reverse-chronological, or any relation two
// neighbours must satisfy.
//
//	matcher.Pairwise(seat, matcher.Fatal, stamps,
//	    func(earlier, later time.Time) bool { return earlier.Before(later) },
//	    "events are in chronological order")
//
// A slice of nought or one item passes: it has no adjacent pair to
// break. The failure names the index and both values of the first
// break, so a reader sees where the order went wrong.
//
// # Allocation contract
//
// A passing call allocates nothing besides what pred allocates.
func Pairwise[T any](seat Seat, mode Mode, items []T, pred func(earlier, later T) bool, msg string) {
	seat.Helper()

	for i := 1; i < len(items); i++ {
		if !pred(items[i-1], items[i]) {
			Fail(seat, mode, "pairwise", msg, map[string]any{
				indexField: i - 1, firstField: items[i-1], secondField: items[i],
			})
			return
		}
	}
	Pass(seat, mode, "pairwise", msg)
}
