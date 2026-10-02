// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"

	"go.dokimi.dev/assert/internal/prop/random"
)

// The environment variables that set the defaults of every property of a
// test run. An unset variable and an empty one state nothing.
const (
	// seedVariable states the seed, in decimal, of every run that [Seed]
	// does not seed.
	seedVariable = "DOKIMI_ASSERT_PROP_SEED"
	// profileVariable names the profile of the test run.
	profileVariable = "DOKIMI_ASSERT_PROP_PROFILE"
	// replayVariable states the token that every run replays when [Replay]
	// states none.
	replayVariable = "DOKIMI_ASSERT_PROP_REPLAY"
)

// profile is a set of defaults for a whole test run, which the variable
// DOKIMI_ASSERT_PROP_PROFILE names.
type profile string

const (
	// defaultProfile draws a random seed for each run. An unset or empty
	// variable names it.
	defaultProfile profile = "default"
	// ciProfile derives each run's seed from its property's contract, so an
	// unchanged test tries the same inputs on every run.
	ciProfile profile = "ci"
)

// profileOf returns the profile that the environment names. It returns an
// error for a name other than default and ci.
func profileOf() (profile, error) {
	name := profile(os.Getenv(profileVariable))
	if name == "" || name == defaultProfile {
		return defaultProfile, nil
	}
	if name == ciProfile {
		return ciProfile, nil
	}
	return "", fmt.Errorf("prop: %s %q names neither the default nor the ci profile", profileVariable, string(name))
}

// seedOf returns the seed of a run of contract under c: the seed that Seed
// states, then the one that DOKIMI_ASSERT_PROP_SEED states, then under the
// ci profile [random.Mix] of the contract, and otherwise a random seed. It
// checks the profile first, so a misspelled profile fails every run.
//
// It returns an error for a profile other than default and ci, and for a
// seed variable that is no decimal number below 2^64.
func seedOf(c config, contract string) (uint64, error) {
	p, err := profileOf()
	if err != nil {
		return 0, err
	}
	if c.seeded {
		return c.seed, nil
	}
	if text := os.Getenv(seedVariable); text != "" {
		seed, err := strconv.ParseUint(text, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("prop: %s %q is no decimal number below 2^64", seedVariable, text)
		}
		return seed, nil
	}
	if p == ciProfile {
		return random.Mix(contract), nil
	}
	return rand.Uint64(), nil
}

// tokenOf returns the token that a run replays: the one that Replay states,
// then the one that DOKIMI_ASSERT_PROP_REPLAY states. It reports false for
// a run that replays none.
func tokenOf(c config) (string, bool) {
	if c.replaying {
		return c.replay, true
	}
	tok := os.Getenv(replayVariable)
	return tok, tok != ""
}
