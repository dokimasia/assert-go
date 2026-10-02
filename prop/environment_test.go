// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/prop"
)

// The environment variables that set the defaults of every property of a
// test run, which the definition names.
const (
	seedVariable    = "DOKIMI_ASSERT_PROP_SEED"
	profileVariable = "DOKIMI_ASSERT_PROP_PROFILE"
	replayVariable  = "DOKIMI_ASSERT_PROP_REPLAY"
)

// profileMessage is the problem of the profile nightly, which is neither
// default nor ci.
const profileMessage = `prop: DOKIMI_ASSERT_PROP_PROFILE "nightly" names neither the default nor the ci profile`

// TestEnvironment checks the environment variables that set the seed, the
// profile and the token to replay of every run. Each case sets the
// process's environment, so the cases run one at a time.
func TestEnvironment(t *testing.T) {
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
				rec := assert.NewRecorder()
				prop.ForAll(rec, contract, failsAtLeast(10000, 1001, big))
				want := "prop: DOKIMI_ASSERT_PROP_SEED " + strconv.Quote(tt.give) + " is no decimal number below 2^64"
				assert.Equal(t, rec.Message(), want, "the message names the variable")
				assert.Empty(t, rec.Failures(), "no run, so no record")
			})
		}

		t.Run("derives the seed from the contract under the ci profile", func(t *testing.T) {
			clean(t)
			t.Setenv(profileVariable, "ci")
			got := detailOf(failsAtLeast(10000, 1001, big))
			assert.Equal(t, got[seedField], any(strconv.FormatUint(random.Mix(contract), 10)), "the contract's seed")
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
				assert.NotEqual(t, one, other, "two seeds, equal once in 2^64 runs")
			})
		}

		t.Run("fails the run at once for a profile other than default and ci", func(t *testing.T) {
			clean(t)
			t.Setenv(profileVariable, "nightly")
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, failsAtLeast(10000, 1001, big), prop.Seed(7))
			assert.Equal(t, rec.Message(), profileMessage, "the profile is checked before Seed's seed")
		})

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
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, failsAtLeast(9, 5, big))
			assert.HasPrefix(t, rec.Message(), `prop: replay "token": token: not a token that an encoder writes`,
				"the message names the token")
		})
	})
}

// clean sets each variable of the environment of a run to the empty
// string, which states nothing, until the test ends.
func clean(t *testing.T) {
	t.Helper()
	for _, name := range []string{seedVariable, profileVariable, replayVariable} {
		t.Setenv(name, "")
	}
}
