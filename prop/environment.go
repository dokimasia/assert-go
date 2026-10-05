// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	"go.dokimi.dev/assert/internal/fault"
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
	// budgetVariable states how long each campaign of the campaign profile
	// runs, in whole seconds.
	budgetVariable = "DOKIMI_ASSERT_PROP_BUDGET"
)

// budgetBits is the bit size of the largest budget: 2^33 - 1 seconds fit in a
// time.Duration, and 2^34 - 1 seconds do not.
const budgetBits = 33

// replayOption is the name of the option that states a token to replay, at
// the front of the path of the token's fault.
const replayOption = "Replay"

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
	// campaignProfile runs every property of ForAll and of a form as a
	// campaign, for the budget that DOKIMI_ASSERT_PROP_BUDGET states, and
	// draws each seed as the default profile does.
	campaignProfile profile = "campaign"
)

// profileOf returns the profile that the environment names. It returns a
// fault at the variable for a name other than default, ci and campaign.
func profileOf() (profile, error) {
	name := profile(os.Getenv(profileVariable))
	if name == "" || name == defaultProfile {
		return defaultProfile, nil
	}
	if name == ciProfile || name == campaignProfile {
		return name, nil
	}
	return "", fault.At(fault.New("%q names none of the default, ci and campaign profiles", string(name)),
		fault.Field(profileVariable))
}

// budgetOf returns how long each campaign of the operation op runs: the
// whole seconds that DOKIMI_ASSERT_PROP_BUDGET states under the campaign
// profile, and 0 for a run that is no campaign. Fuzz runs no campaign, and
// reads no budget. The caller has checked the profile.
//
// It returns a fault at the variable for a budget that is no decimal number
// of seconds from 1 to 2^33 - 1.
func budgetOf(op string) (time.Duration, error) {
	if p, _ := profileOf(); p != campaignProfile || op == fuzzOp {
		return 0, nil
	}
	text := os.Getenv(budgetVariable)
	seconds, err := strconv.ParseUint(text, 10, budgetBits)
	if err != nil || seconds == 0 {
		return 0, fault.At(fault.New("%q is no whole number of seconds above 0", text), fault.Field(budgetVariable))
	}
	return time.Duration(seconds) * time.Second, nil
}

// seedOf returns the seed of a run of contract under c: the seed that Seed
// states, then the one that DOKIMI_ASSERT_PROP_SEED states, then under the
// ci profile [random.Mix] of the contract, and otherwise a random seed. It
// checks the profile first, so a misspelled profile fails every run.
//
// It returns a fault at the variable for a profile other than default, ci
// and campaign, and for a seed variable that is no decimal number below
// 2^64.
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
			return 0, fault.At(fault.New("%q is no decimal number below 2^64", text), fault.Field(seedVariable))
		}
		return seed, nil
	}
	if p == ciProfile {
		return random.Mix(contract), nil
	}
	return rand.Uint64(), nil
}

// tokenOf returns the token that a run replays and the name of where it is
// stated: the one that Replay states, then the one that
// DOKIMI_ASSERT_PROP_REPLAY states. It reports false for a run that replays
// none.
func tokenOf(c config) (tok, source string, ok bool) {
	if c.replaying {
		return c.replay, replayOption, true
	}
	tok = os.Getenv(replayVariable)
	return tok, replayVariable, tok != ""
}
