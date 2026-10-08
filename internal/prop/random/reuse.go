// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package random

// reuseOdds is the denominator of the coin that sends a reusable draw to
// an earlier value of its case: one draw in 4.
const reuseOdds = 4

// Reuse decides whether a reusable draw takes one of the earlier values of
// its case with the same bounds, of which there are earlier. It returns
// the index of that value and true, from a Coin(1, 4) and then
// Below(earlier). It returns false when the coin fails, and false without
// consuming the stream when earlier is 0.
func Reuse(s *Source, earlier int) (int, bool) {
	if earlier == 0 || !s.Coin(1, reuseOdds) {
		return 0, false
	}
	return int(s.Below(uint64(earlier))), true
}
