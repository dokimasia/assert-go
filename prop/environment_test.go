// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/prop"
)

// contractSeed is the seed that the contract derives, as a run's record
// states it.
var contractSeed = any(strconv.FormatUint(random.Mix(contract), 10))

// TestEnvironmentEnv checks the environment variables that set the seed, the
// profile and the token to replay of every run, a hermetic run, which reads
// none of them, and a run in a test binary that a mutation run
// instrumented. Each case sets the process's environment, so the cases run
// one at a time. In a test binary that a mutation run instrumented, every
// run seeds from its contract and runs no campaign, so a case of a random
// seed or of a campaign checks that instead.
func TestEnvironmentEnv(t *testing.T) {
	t.Run("ForAll", func(t *testing.T) {
		t.Run("takes the seed of a run that Seed does not seed from DOKIMI_ASSERT_PROP_SEED", func(t *testing.T) {
			clean(t)
			t.Setenv(seedVariable, "7")
			got := detailOf(failsAtLeast(10000, 1001, big))
			assert.Equal(t, got[seedField], any("7"), "the variable's seed")
			assert.Equal(t, got[choicesField], any("prop1:AOkH"), "the minimal case of seed 7")
		})

		t.Run("takes the seed that Seed states over the variable", func(t *testing.T) {
			clean(t)
			t.Setenv(seedVariable, "8")
			assert.Equal(t, detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7))[seedField], any("7"), "Seed's seed")
		})

		tests := []struct {
			name string
			give string
		}{
			{name: "fails the run at once for a seed that is no decimal number", give: "seven"},
			{name: "fails the run at once for a seed of 2^64", give: "18446744073709551616"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				clean(t)
				t.Setenv(seedVariable, tt.give)
				seat := &matchertest.Seat{}
				prop.ForAll(seat, contract, failsAtLeast(10000, 1001, big))
				expectOnlyFault(t, seat.Faults(), fault.Error{
					Op:     forAllOp,
					Path:   fault.Path{fault.Field(seedVariable)},
					Reason: strconv.Quote(tt.give) + " is no decimal number below 2^64",
				})
				assert.Empty(t, seat.Records(), "no run, so no record")
			})
		}

		t.Run("derives the seed from the contract under the ci profile", func(t *testing.T) {
			clean(t)
			t.Setenv(profileVariable, "ci")
			got := detailOf(failsAtLeast(10000, 1001, big))
			assert.Equal(t, got[seedField], contractSeed, "the contract's seed")
		})

		t.Run("takes the seed variable over the ci profile", func(t *testing.T) {
			clean(t)
			t.Setenv(profileVariable, "ci")
			t.Setenv(seedVariable, "7")
			assert.Equal(t, detailOf(failsAtLeast(10000, 1001, big))[seedField], any("7"), "the variable's seed")
		})

		profiles := []struct {
			name string
			give string
		}{
			{name: "draws a random seed for each run under the default profile", give: "default"},
			{name: "draws a random seed for each run without a profile", give: ""},
		}
		for _, tt := range profiles {
			t.Run(tt.name, func(t *testing.T) {
				clean(t)
				t.Setenv(profileVariable, tt.give)
				one := detailOf(failsAtLeast(10000, 1001, big))[seedField]
				other := detailOf(failsAtLeast(10000, 1001, big))[seedField]
				if matcher.Mutated() {
					assert.Equal(t, []any{one, other}, []any{contractSeed, contractSeed},
						"the contract's seed in each run of a mutation run")
					return
				}
				assert.NotEqual(t, one, other, "two seeds, equal once in 2^64 runs")
			})
		}

		t.Run("fails the run at once for a profile other than default, ci and campaign", func(t *testing.T) {
			clean(t)
			t.Setenv(profileVariable, "nightly")
			seat := &matchertest.Seat{}
			prop.ForAll(seat, contract, failsAtLeast(10000, 1001, big), prop.Seed(7))
			expectOnlyFault(t, seat.Faults(), profileFault(forAllOp))
		})

		t.Run("runs a campaign for the budget of the campaign profile, past the first failure", func(t *testing.T) {
			clean(t)
			t.Setenv(profileVariable, "campaign")
			t.Setenv(budgetVariable, "1")
			dir := t.TempDir()
			got := detailOf(failsFrom(1001), prop.Seed(7), prop.Store(dir))
			assert.Equal(t, got[failureField], any(assert.Failure{Assertion: big, Contract: fits}),
				"the campaign's failure")
			if matcher.Mutated() {
				assert.Equal(t, got[choicesField], any("prop1:AOkH"), "the minimal case of an ordinary run of seed 7")
				assert.Empty(t, loaded(t, dir).Entries, unwritten)
				return
			}
			assert.True(t, got[casesField].(int) > 100, "the valid cases of a second of exploration")
			assert.Length(t, loaded(t, dir).Entries, 1, "the failure, stored as the campaign concluded it")
		})

		budgets := []struct {
			name string
			give string
		}{
			{name: "fails the run at once for a campaign without a budget", give: ""},
			{name: "fails the run at once for a campaign of a budget of 0", give: "0"},
			{name: "fails the run at once for a campaign of a budget past 2^33 - 1 seconds", give: "8589934592"},
			{name: "fails the run at once for a campaign of a budget that is no whole number", give: "1.5"},
		}
		for _, tt := range budgets {
			t.Run(tt.name, func(t *testing.T) {
				clean(t)
				t.Setenv(profileVariable, "campaign")
				t.Setenv(budgetVariable, tt.give)
				seat := &matchertest.Seat{}
				prop.ForAll(seat, contract, failsAtLeast(10000, 1001, big), prop.Seed(7))
				if matcher.Mutated() {
					assert.Empty(t, seat.Faults(), "no fault, because a mutation run runs no campaign")
					return
				}
				expectOnlyFault(t, seat.Faults(), fault.Error{
					Op:     forAllOp,
					Path:   fault.Path{fault.Field(budgetVariable)},
					Reason: strconv.Quote(tt.give) + " is no whole number of seconds above 0",
				})
			})
		}

		t.Run("replays the token of DOKIMI_ASSERT_PROP_REPLAY", func(t *testing.T) {
			clean(t)
			t.Setenv(replayVariable, "prop1:AAc")
			got := detailOf(failsAtLeast(9, 5, big), prop.Seed(7))
			assert.Equal(t, counts(got), []any{prop.Counterexample, 0, 0}, "the one replayed case")
			assert.Equal(t, got[choicesField], any("prop1:AAc"), "the replayed token")
		})

		t.Run("replays the token that Replay states over the variable", func(t *testing.T) {
			clean(t)
			t.Setenv(replayVariable, "prop1:AAM")
			got := detailOf(failsAtLeast(9, 5, big), prop.Seed(7), prop.Replay("prop1:AAc"))
			assert.Equal(t, got[choicesField], any("prop1:AAc"), "Replay's token")
		})

		t.Run("fails the run at once for a variable token that no encoder writes", func(t *testing.T) {
			clean(t)
			t.Setenv(replayVariable, "token")
			seat := &matchertest.Seat{}
			prop.ForAll(seat, contract, failsAtLeast(9, 5, big))
			expectOnlyFault(t, seat.Faults(), fault.Error{
				Op:     forAllOp,
				Path:   fault.Path{fault.Field(replayVariable)},
				Kind:   token.ErrInvalid,
				Reason: `"token" does not start with prop1:`,
			})
		})

		t.Run("takes the seed that Seed states in a hermetic run", func(t *testing.T) {
			clean(t)
			t.Setenv(seedVariable, "8")
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7), prop.Hermetic())
			assert.Equal(t, got[seedField], any("7"), "Seed's seed")
		})

		hermetic := []struct {
			name     string
			variable string
			give     string
		}{
			{name: "reads no seed variable in a hermetic run", variable: seedVariable, give: "seven"},
			{name: "reads no profile in a hermetic run", variable: profileVariable, give: "nightly"},
			{name: "runs no campaign in a hermetic run", variable: profileVariable, give: "campaign"},
			{name: "reads no token to replay in a hermetic run", variable: replayVariable, give: "token"},
		}
		for _, tt := range hermetic {
			t.Run(tt.name, func(t *testing.T) {
				clean(t)
				t.Setenv(tt.variable, tt.give)
				got := detailOf(failsAtLeast(10000, 1001, big), prop.Hermetic())
				assert.Equal(t, got[choicesField], any("prop1:AOkH"), "the minimal case of an ordinary run")
			})
		}

		t.Run("draws a random seed in a hermetic run under the ci profile", func(t *testing.T) {
			clean(t)
			t.Setenv(profileVariable, "ci")
			got := detailOf(failsAtLeast(10000, 1001, big), prop.Hermetic())[seedField]
			if matcher.Mutated() {
				assert.Equal(t, got, contractSeed, "the contract's seed in a mutation run")
				return
			}
			assert.NotEqual(t, got, contractSeed, "a random seed, which is the contract's once in 2^64 runs")
		})

		t.Run("derives the seed from the contract in a test binary that a mutation run instrumented",
			func(t *testing.T) {
				if childtest.InChild(t) {
					assert.Equal(t, detailOf(failsAtLeast(10000, 1001, big))[seedField], contractSeed,
						"the contract's seed")
					assert.Equal(t, detailOf(failsAtLeast(10000, 1001, big), prop.Hermetic())[seedField], contractSeed,
						"the contract's seed in a hermetic run")
					return
				}
				clean(t)
				inMutationRun(t)
			})

		t.Run("takes the seed variable over the contract's seed in a mutation run", func(t *testing.T) {
			if childtest.InChild(t) {
				assert.Equal(t, detailOf(failsAtLeast(10000, 1001, big))[seedField], any("7"), "the variable's seed")
				return
			}
			clean(t)
			t.Setenv(seedVariable, "7")
			inMutationRun(t)
		})

		t.Run("runs no campaign in a mutation run, and reads no budget", func(t *testing.T) {
			if childtest.InChild(t) {
				got := detailOf(failsAtLeast(10000, 1001, big), prop.Seed(7))
				assert.Equal(t, got[choicesField], any("prop1:AOkH"), "the minimal case of an ordinary run")
				return
			}
			clean(t)
			t.Setenv(profileVariable, "campaign")
			inMutationRun(t)
		})
	})
}

// clean sets each variable of the environment of a run to the empty
// string, which states nothing, until the test ends.
func clean(t *testing.T) {
	t.Helper()
	for _, name := range []string{seedVariable, profileVariable, replayVariable, budgetVariable} {
		t.Setenv(name, "")
	}
}
