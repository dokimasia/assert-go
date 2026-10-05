// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package random

// Keep decides whether a machine's swarm keeps an action. kept states
// whether swarm kept an earlier action, and remaining counts this action
// and the actions after it.
//
// Once an earlier action is kept, one Coin(num, den) decides. While none
// is, the last action is kept and the draw consumes nothing. Before it,
// each round tosses one Coin(num, den) for this action and one for each
// later action, and the rounds repeat until a coin comes up. The action is
// kept when its own coin of that round came up. The kept set then has the
// distribution of one coin per action, conditioned on keeping one action
// or more: at odds of 1 in 2, each non-empty set of n actions has
// probability 1 / (2^n - 1). num is above 0, and remaining is 1 or more.
func Keep(s *Source, num, den uint64, kept bool, remaining int) bool {
	if kept {
		return s.Coin(num, den)
	}
	if remaining == 1 {
		return true
	}
	for {
		own, some := s.Coin(num, den), false
		for range remaining - 1 {
			some = s.Coin(num, den) || some
		}
		if own || some {
			return own
		}
	}
}
