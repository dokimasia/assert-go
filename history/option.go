// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"fmt"
	"time"
)

// The limits of a check when no option states them, which the definition
// fixes.
const (
	// defaultBudget is the steps that the search of one partition may spend.
	defaultBudget = 10_000_000
	// defaultMemoLimit is the bits that the memo of one partition may count,
	// 2^33, which is 1 GiB.
	defaultMemoLimit = 1 << 33
)

// Option configures one check of [Linearizable]. Each option states one
// setting, and a later option overrides an earlier one. The zero Option
// changes nothing.
//
// # Allocation contract
//
// An option allocates nothing when its call does not keep it, and the
// closure of its setting when it does. Applying the options of a check
// allocates nothing.
type Option struct {
	// set returns the settings of a check with the option's setting applied.
	set func(config) config
}

// Budget sets the steps that the search of one partition may spend,
// 10,000,000 by default. A step is one call of the spec's Next, and a step
// of a spec that [SpecFrom] returns counts for the calls that it applies.
// The search takes no step that would pass the budget, and ends as
// [Undecided] with [LimitSteps]. It panics for fewer than 1 step.
func Budget(steps int) Option {
	if steps < 1 {
		panic(fmt.Sprintf("history: Budget(%d) is below 1", steps))
	}
	return Option{set: func(c config) config {
		c.budget = steps
		return c
	}}
}

// MemoLimit sets the bits that the memo of one partition's search may count,
// 2^33 by default, which is 1 GiB. Each configuration in the memo counts one
// bit for each call of the partition, and the states that it lists count
// nothing. The search stores no configuration that would pass the limit,
// and ends as [Undecided] with [LimitMemo]. It panics for bits below 1.
func MemoLimit(bits int64) Option {
	if bits < 1 {
		panic(fmt.Sprintf("history: MemoLimit(%d) is below 1", bits))
	}
	return Option{set: func(c config) config {
		c.memoLimit = bits
		return c
	}}
}

// TimeLimit sets the time that the whole check may take, measured on the
// platform clock and not on the seat's. A time of 0, the default, sets no
// limit. The search of each partition reads the clock before its first step
// and every 1,024 steps after it, and once the time has passed it ends as
// [Undecided] with [LimitTime]. A time limit makes the outcome depend on the
// machine. It panics for a negative time.
func TimeLimit(d time.Duration) Option {
	if d < 0 {
		panic(fmt.Sprintf("history: TimeLimit(%v) is below 0", d))
	}
	return Option{set: func(c config) config {
		c.timeLimit = d
		return c
	}}
}

// Workers sets the number of partitions that are searched at once, 1 by
// default. A check on more workers reports what a check on one reports. It
// calls the spec's functions on several goroutines at once, so they must be
// safe for that. It panics for n below 1.
func Workers(n int) Option {
	if n < 1 {
		panic(fmt.Sprintf("history: Workers(%d) is below 1", n))
	}
	return Option{set: func(c config) config {
		c.workers = n
		return c
	}}
}

// Whole makes the check search the whole history as one partition, whatever
// keys its calls declare. A spec that states one state of the whole
// subject, as the spec of a machine does, needs every call in one search.
func Whole() Option {
	return Option{set: func(c config) config {
		c.whole = true
		return c
	}}
}

// Final makes the check store in states, for a check that passes, the states
// that the first linearization the search found leaves: the initial state
// for a history without calls, and the states after the linearization of
// its calls for a history of one partition. A check of more partitions, and
// a check that does not pass, stores nil. [Whole] puts every call in one
// partition.
//
// S must be the type of the spec's states. A check whose spec has states of
// another type ends the call with a fault.
func Final[S any](states *[]S) Option {
	return Option{set: func(c config) config {
		c.final = states
		return c
	}}
}

// Resume makes the check continue the search that cp keeps, and keep its own
// search in cp when it passes. A check continues the kept search when cp
// keeps the search of a check of the same history that passed under the
// same budget and memo limit, and the history has recorded events since
// without completing a call that the kept search read as pending. The
// continued search takes the steps that a search of every call takes after
// the order that the kept search found, so the check reports what a search
// of the whole history reports: its verdict, its steps, its frontier and
// its final states. In any other case the check searches every call. A
// check that does not pass keeps no search in cp. A nil cp changes nothing.
//
// Pass the same spec to every check that passes cp. The kept search
// contains the states that the spec of the earlier checks left.
//
// A check with Resume searches the whole history as one partition, and
// states [Whole]. A check without Whole ends the call with a fault, and so
// does a check whose spec has states of another type than S. A check with
// Resume calls the spec's Initial for a history without calls too.
func Resume[S any](cp *Checkpoint[S]) Option {
	if cp == nil {
		return Option{}
	}
	return Option{set: func(c config) config {
		c.resume = cp
		return c
	}}
}

// config is what the options of one check state.
type config struct {
	// budget is the steps that one partition's search may spend.
	budget int
	// memoLimit is the bits that one partition's memo may count.
	memoLimit int64
	// timeLimit is the time that the check may take, and 0 for no limit.
	timeLimit time.Duration
	// workers is the number of partitions searched at once.
	workers int
	// whole reports whether the check searches every call as one partition.
	whole bool
	// final is the *[]S of Final, which a passing check stores its states
	// in, and nil without Final.
	final any
	// resume is the *Checkpoint[S] of Resume, whose search the check
	// continues, and nil without Resume.
	resume any
}

// configure returns the defaults with each option of opts applied in order.
func configure(opts []Option) config {
	c := config{budget: defaultBudget, memoLimit: defaultMemoLimit, workers: 1}
	for _, o := range opts {
		if o.set != nil {
			c = o.set(c)
		}
	}
	return c
}
