// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"math"
	"time"

	"go.dokimi.dev/assert/internal/prop/engine"
)

// defaultShrinkTime is the time that shrinking and explaining may take when
// no option states one: 30 seconds.
const defaultShrinkTime time.Duration = 30000000000

// Option configures a run of [ForAll] or [Fuzz]. Each option states one
// setting, and a later option overrides an earlier one, except [Require],
// which adds a requirement. The zero Option changes nothing.
//
// # Allocation contract
//
// An option allocates nothing when its call does not keep it, and the
// closure of its setting when it does. Applying the options of a run
// allocates nothing, except a requirement that [Require] adds.
type Option struct {
	// set returns the settings of a run with the option's setting applied.
	set func(config) config
}

// Cases sets the number of valid cases a run aims for, 100 by default. A
// run also stops once it has tested every input, and after ten times n
// generated cases. It panics for n below 1.
func Cases(n int) Option {
	if n < 1 {
		panic(fmt.Sprintf("prop: Cases(%d) is below 1", n))
	}
	return Option{set: func(c config) config {
		c.cases = n
		return c
	}}
}

// Seed sets the seed of the run. Without it, the variable
// DOKIMI_ASSERT_PROP_SEED states the seed in decimal, then the ci profile
// derives it from the contract, and otherwise each run draws one. A failing
// run reports its seed.
func Seed(s uint64) Option {
	return Option{set: func(c config) config {
		c.seed, c.seeded = s, true
		return c
	}}
}

// Replay makes [ForAll] run the one case that token records, and nothing
// else. The case is not shrunk. A token that no encoder writes fails the
// test. Without it, the variable DOKIMI_ASSERT_PROP_REPLAY states the token
// to replay.
func Replay(token string) Option {
	return Option{set: func(c config) config {
		c.replay, c.replaying = token, true
		return c
	}}
}

// Require adds a coverage requirement: label must count at least share of
// the valid cases, as [Case.Classify] counts them. A Wilson score test
// decides it after the stated number of cases, and again each time the
// number doubles, up to eight times the number, with at most one false
// refutation in 10^9 runs. A run that refutes it or leaves it unmet fails
// as [CoverageUnmet]. It panics for a share outside [0, 1].
func Require(label string, share float64) Option {
	if math.IsNaN(share) || share < 0 || share > 1 {
		panic(fmt.Sprintf("prop: Require(%q, %v) states a share outside [0, 1]", label, share))
	}
	requirement := engine.Requirement{Label: label, Share: share}
	return Option{set: func(c config) config {
		c.requirements = append(c.requirements, requirement)
		return c
	}}
}

// Shrink sets the budget of runs of the body that shrinking and explaining
// every failure of a run may spend, 2,000 by default. A budget of 0 turns
// both off, and a failing case is reported as found. It panics for a
// negative budget.
func Shrink(runs int) Option {
	if runs < 0 {
		panic(fmt.Sprintf("prop: Shrink(%d) is below 0", runs))
	}
	return Option{set: func(c config) config {
		c.shrink = runs
		return c
	}}
}

// ShrinkTime sets the time that shrinking and explaining may take, 30
// seconds by default, measured on the platform clock and not on the seat's.
// A shrink that runs out of time reports the smallest case it found. A time
// of 0 sets no limit. It panics for a negative time.
func ShrinkTime(d time.Duration) Option {
	if d < 0 {
		panic(fmt.Sprintf("prop: ShrinkTime(%v) is below 0", d))
	}
	return Option{set: func(c config) config {
		c.shrinkTime = d
		return c
	}}
}

// MaxChoices sets the most choices that one case may make, 8,192 by
// default. A sequence counts as one choice and one for each element. A case
// that requests more is rejected. It panics for n below 1.
func MaxChoices(n int) Option {
	if n < 1 {
		panic(fmt.Sprintf("prop: MaxChoices(%d) is below 1", n))
	}
	return Option{set: func(c config) config {
		c.maxChoices = n
		return c
	}}
}

// Store sets the directory of the run's store, relative to the test's
// directory. "" turns the store off. Without it, the store of a test whose
// seat has a Name method is testdata/prop/<name>, beside testdata/golden,
// and a seat without one has no store.
func Store(dir string) Option {
	return Option{set: func(c config) config {
		c.store, c.storeSet = dir, true
		return c
	}}
}

// Explain sets whether a run explains its counterexample, on by default.
// The explanation states for each draw whether any value fails there, and
// for an integer or a duration whose value matters the nearest value that
// passes.
func Explain(enabled bool) Option {
	return Option{set: func(c config) config {
		c.explain = enabled
		return c
	}}
}

// Draws makes the run's first case decode its draws and take its machine's
// steps from entries: a JSON array of the entries of a counterexample, in
// request order. A draw entry states a label and a typed literal, as a store
// entry records a draw. A step entry states the action of a step, with its
// client in a concurrent section and the drain mark in the drain.
//
//	prop.Draws(`[
//		{"label": "capacity", "value": {"type": "int", "value": 2}},
//		{"step": "put"},
//		{"label": "v", "value": {"type": "int", "value": 0}},
//		{"step": "put", "client": 1},
//		{"step": "deliver", "drain": true}
//	]`)
//
// Each draw takes the next entry, and its choices are the ones that decode
// to the entry's value under the draw's generator, so a value written by
// hand runs, and shrinks when it fails, as a generated one does. A machine
// of package stateful turns each step entry into the choices of one step. A
// draw past the last entry takes its target. A draw whose label differs from
// its entry's, a draw whose generator does not produce the entry's value,
// and a step entry that the machine cannot take at its position fail the
// test before any other case runs, and the failure names the entry. Entries
// that are no such array fail the test without a run.
func Draws(entries string) Option {
	return Option{set: func(c config) config {
		c.draws, c.drawn = entries, true
		return c
	}}
}

// Workers sets the number of cases, shrink candidates and explanation
// fillings that run at once, 1 by default. A run on more workers reports
// what a run on one reports, and runs the body concurrently with itself, so
// the body must be safe for that. It panics for n below 1.
func Workers(n int) Option {
	if n < 1 {
		panic(fmt.Sprintf("prop: Workers(%d) is below 1", n))
	}
	return Option{set: func(c config) config {
		c.workers = n
		return c
	}}
}

// config is what the options of one run state.
type config struct {
	// cases is the number of valid cases the run aims for.
	cases int
	// seed is the seed that Seed states, when seeded is set.
	seed uint64
	// seeded reports whether Seed stated the seed.
	seeded bool
	// replay is the token that Replay states, when replaying is set.
	replay string
	// replaying reports whether Replay stated a token.
	replaying bool
	// requirements are the coverage requirements, in the order stated.
	requirements []engine.Requirement
	// shrink is the budget of shrink runs.
	shrink int
	// shrinkTime is the time a shrink may take.
	shrinkTime time.Duration
	// maxChoices is the cap on the choices of one case.
	maxChoices int
	// store is the directory that Store states, when storeSet is set.
	store string
	// storeSet reports whether Store stated the directory.
	storeSet bool
	// explain reports whether the run explains its counterexample.
	explain bool
	// workers is the number of workers.
	workers int
	// draws are the entries that Draws states, when drawn is set.
	draws string
	// drawn reports whether Draws stated entries.
	drawn bool
}

// configure returns the defaults with each option of opts applied in
// order.
func configure(opts []Option) config {
	c := config{
		cases:      engine.DefaultCases,
		shrink:     engine.DefaultShrink,
		shrinkTime: defaultShrinkTime,
		maxChoices: engine.MaxChoices,
		explain:    true,
		workers:    1,
	}
	for _, o := range opts {
		if o.set != nil {
			c = o.set(c)
		}
	}
	return c
}
