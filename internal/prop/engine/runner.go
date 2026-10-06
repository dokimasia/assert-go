// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/coverage"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/internal/prop/tree"
	"go.dokimi.dev/assert/internal/record"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Outcome -linecomment -output=runner.string_gen.go

// The constants of a run.
const (
	// DefaultCases is the number of valid cases a run aims for when the
	// caller states no number.
	DefaultCases = 100
	// maxRejections is the number of rejected cases for every valid one
	// past which a run ends as rejected.
	maxRejections = 10
	// generationFactor is the number of random cases a phase generates at
	// most for every valid case it aims for.
	generationFactor = 10
	// prefixDivisor and maxPrefixCases bound the prefix cases: a run tries
	// them while at most cases/prefixDivisor of its cases are valid, and
	// never past maxPrefixCases valid cases.
	prefixDivisor  = 10
	maxPrefixCases = 50
	// minPrefixedChoices is the fewest choices a random case makes to have
	// a prefix case: one to replay and one to give its target.
	minPrefixedChoices = 2
)

// Outcome is how a run ended. Every outcome but [Passed] fails the test.
type Outcome uint8

const (
	// Passed is a run that found no failing case.
	Passed Outcome = 0 // passed
	// Counterexample is a run that found a failing case.
	Counterexample Outcome = 1 // counterexample
	// Flaky is a run whose body diverged, or whose failing case replayed
	// another way.
	Flaky Outcome = 2 // flaky
	// Rejected is a run that rejected more than ten cases for every valid
	// one.
	Rejected Outcome = 3 // rejected
	// CoverageUnmet is a run that refuted or left unmet a coverage
	// requirement.
	CoverageUnmet Outcome = 4 // coverage-unmet
	// Vacuous is a run whose cases requested no input.
	Vacuous Outcome = 5 // vacuous
)

// Valid reports whether o is one of the six outcomes.
func (o Outcome) Valid() bool {
	return o <= Vacuous
}

// Requirement is a label that must count at least Share of the valid
// cases.
type Requirement struct {
	// Label is the label the body classifies a case under.
	Label string
	// Share is the fraction of the valid cases the label must count.
	Share float64
}

// Shortfall is a requirement that a run refuted or left unmet, with its
// counts.
type Shortfall struct {
	// Requirement is the requirement the run missed.
	Requirement Requirement
	// Counted is the number of valid cases the label counted.
	Counted int
	// Valid is the number of valid cases.
	Valid int
	// Verdict is the test's verdict on the requirement.
	Verdict coverage.Verdict
}

// Settings are what a run is asked to do.
type Settings struct {
	// Seed is the seed of the random cases.
	Seed uint64
	// Cases is the number of valid cases the run aims for, 1 or more.
	Cases int
	// MaxChoices is the cap on the choices of one case.
	MaxChoices int
	// Requirements are the coverage requirements.
	Requirements []Requirement
	// Draws are the entries of the run's first case, whose draws take them
	// in order: each draw decodes its entry's value. A refused entry ends
	// the run before any other case.
	Draws []Entry
	// Examples are the cases whose values the caller states, which the run
	// tries after the case of Draws and before the stored cases, in order.
	Examples []Example
	// Stored are the choice sequences of stored cases, oldest first.
	Stored [][]choice.Choice
	// Shrink is the budget of runs that shrinking and explaining every
	// failure may spend. A budget of 0 turns both off.
	Shrink int
	// ShrinkTime is the time that shrinking and explaining may take, on
	// ShrinkClock. A time of 0 sets no limit.
	ShrinkTime time.Duration
	// ShrinkClock is the clock that ShrinkTime is measured on, and the
	// platform clock, assert.System, when nil. The cap bounds the job and
	// states nothing about the property, so it does not read Clock, which
	// advances only when the test advances it.
	ShrinkClock assert.Clock
	// Explain reports whether the run explains its counterexample.
	Explain bool
	// Clock is the clock of the test, which every case reads, and the
	// platform clock when nil.
	Clock assert.Clock
	// Workers is the number of cases, shrink candidates and explanation
	// fillings that run at once. Below 2 they run one at a time. A run on
	// more workers reports what a run on one reports, and its body must be
	// safe to run concurrently with itself.
	Workers int
	// Slot is the slot of the property's call record, which takes the
	// calls of every case that a run on one worker runs, in that order and
	// with the phase of each case. It is nil for a property that is not
	// recorded.
	Slot *record.Slot
	// Budget is how long a [Campaign] explores, on BudgetClock. A run
	// ignores it.
	Budget time.Duration
	// BudgetClock is the clock that Budget is measured on, and the platform
	// clock, assert.System, when nil.
	BudgetClock assert.Clock
	// Concluded receives each failure that a campaign concludes, as it
	// concludes it: a counterexample of the failure's minimal case, with the
	// other failures that its shrink found. It is nil for a campaign that
	// keeps its failures for its result alone. A run ignores it.
	Concluded func(Result)
}

// Result is the end of a run.
type Result struct {
	// Outcome is how the run ended.
	Outcome Outcome
	// Cases is the number of valid cases.
	Cases int
	// Rejected is the number of rejected cases.
	Rejected int
	// Seed is the seed of the run.
	Seed uint64
	// Failing is the minimal failing case of a counterexample, or the case
	// that a flaky replay contradicted.
	Failing *Execution
	// Divergence is the difference that made the run flaky.
	Divergence *Divergence
	// Shortfall is the requirement that a coverage-unmet run missed.
	Shortfall *Shortfall
	// Others are the minimal cases of the other failures, in the order the
	// run found them.
	Others []Execution
	// Explanation explains each draw of the counterexample.
	Explanation []Explained
	// Token is the replay token of Failing.
	Token string
	// Runs is the number of runs that shrinking and explaining spent.
	Runs int
	// Stored are the runs of the stored cases, in order, up to the one that
	// ended the run.
	Stored []Execution
	// Refused is the refusal of the case of Settings.Draws, which ended the
	// run before any other case, and nil for any other run. It is a fault
	// whose path starts at the index of the refused entry and leads through
	// its label or its value. A refused run states nothing else: every other
	// field is zero.
	Refused error
}

// Run runs the phases of a property and returns how the run ended.
//
// The runner calls body once per case, and stops at the first failing
// case. The case of the Draws entries runs first, then the examples and the
// stored cases, all outside the case tree, then the case whose every choice
// is its target. Then random case i of the seed runs,
// for i = 0, 1, 2 and on, until Cases valid cases have run, the domain is
// exhausted, or ten times Cases random cases have run, repeats included.
// Each random case is followed by its prefix case and the next edge case.
// While a coverage requirement is undecided at a check, more random cases
// run up to the next check.
//
// A failing case is replayed, shrunk and explained. A failing example of
// values has no choices, and the run reports it as found, without a replay
// token. A run that found no failing case fails when it rejected more than
// ten cases for every valid one, when no case requested an input, or when it
// refuted or left unmet a coverage requirement, checked in that order. A run
// whose case of the Draws entries refuses an entry ends at once with
// [Result.Refused].
//
// The slot of s takes the calls of every case that a run on one worker
// runs, in that order, each under the phase of its case: the case of the
// Draws entries and the examples under example, then stored, simplest,
// random before the first coverage check and coverage after it, prefix,
// edge, and replay, shrink and explain for the runs that conclude a
// failing case.
func Run(body Body, s Settings) Result {
	s = withClocks(s)
	return conclude(body, s, explore(body, s))
}

// Conclude replays, shrinks and explains failing, a case whose status is
// [CaseFailed] and that a body failed outside a run, such as a case that
// [Bridge] decoded from a fuzzer's bytes. It returns a counterexample with
// no valid case counted, or a flaky result when the replay of failing
// differs from it.
func Conclude(body Body, s Settings, failing Execution) Result {
	s = withClocks(s)
	return conclude(body, s, Result{Outcome: Counterexample, Seed: s.Seed, Failing: &failing})
}

// RunReplay runs the one case that choices record, and reports a failure
// as found, without shrinking or explaining it. The calls of the case are
// recorded under phase: [record.Token] for the case of a replay token, and
// [record.Stored] for a stored case.
func RunReplay(body Body, s Settings, choices []choice.Choice, phase record.Phase) Result {
	s = withClocks(s)
	t := newTally(s)
	r, ended := t.take(execute(body, replaying{choices: choices}, s), phase)
	if !ended {
		if r, ended = t.missing(); !ended {
			r = t.result(Passed)
		}
		return r
	}
	if r.Failing != nil {
		r.Token = token.Encode(r.Failing.Case.Choices())
	}
	return r
}

// tally counts the cases of a run so far.
type tally struct {
	// seed is the seed of the run.
	seed uint64
	// valid and rejected count the valid and the rejected cases.
	valid, rejected int
	// requested reports whether any case requested an input.
	requested bool
	// labels count the valid cases under each label.
	labels map[string]int
	// slot is the slot of the property's call record, and nil for a
	// property that is not recorded.
	slot *record.Slot
}

// newTally returns the tally of a run of s.
func newTally(s Settings) *tally {
	return &tally{seed: s.Seed, labels: make(map[string]int), slot: s.Slot}
}

// take counts one case, whose calls the property's record takes under
// phase, and returns the result it ends the run with.
func (t *tally) take(e Execution, phase record.Phase) (Result, bool) {
	e.take(t.slot, phase)
	if e.Status == CaseRefused {
		return Result{Refused: e.Refusal}, true
	}
	t.requested = t.requested || e.Case.requested()
	if e.Status == CaseFailed {
		failing := e
		r := t.result(Counterexample)
		r.Failing = &failing
		return r, true
	}
	if e.Status == CaseDiverged {
		r := t.result(Flaky)
		r.Divergence = e.Divergence
		return r, true
	}
	if e.Status == CaseRejected {
		t.rejected++
	}
	if e.Status == CasePassed {
		t.valid++
		for _, label := range e.Case.Labels() {
			t.labels[label]++
		}
	}
	return Result{}, false
}

// result returns a result of outcome with the counts so far.
func (t *tally) result(outcome Outcome) Result {
	return Result{Outcome: outcome, Cases: t.valid, Rejected: t.rejected, Seed: t.seed}
}

// missing returns the result of too many rejections or of no input, if
// either ends the run.
func (t *tally) missing() (Result, bool) {
	if t.rejected > maxRejections*t.valid {
		return t.result(Rejected), true
	}
	if !t.requested {
		return t.result(Vacuous), true
	}
	return Result{}, false
}

// phases runs the cases of a run after its stored cases, in the order one
// worker runs them.
type phases struct {
	// s are the settings of the run.
	s Settings
	// t counts the cases.
	t *tally
	// tree is the case tree.
	tree *tree.Tree
	// cases runs each case into the tree.
	cases executor
	// edges are the boundaries of the edge cases still to run.
	edges []boundary
	// index is the index of the next random case.
	index uint64
	// phase is the phase of the random cases: random before the first
	// coverage check, and coverage after it.
	phase record.Phase
}

// random runs the next random case, then its prefix case and the next
// edge case.
//
// A random case has a prefix case while at most prefixCases of the run's
// cases are valid and the random case made two choices or more. The
// prefix case replays the first c of the random case's n choices, with
// c = 1 + Below(n - 1) drawn from the random case's source after the case,
// and every later request takes its target.
func (p *phases) random() (Result, bool) {
	e, choices, source := p.cases.random(p.index)
	p.index++
	if r, ended := p.t.take(e, p.phase); ended {
		return r, true
	}
	prefixed := p.t.valid <= prefixCases(p.s.Cases) && len(choices) >= minPrefixedChoices && !p.tree.Exhausted()
	if prefixed {
		cut := 1 + source.Below(uint64(len(choices)-1))
		if r, ended := p.t.take(p.cases.replay(choices[:cut]), record.Prefix); ended {
			return r, true
		}
	}
	return p.nextEdge()
}

// nextEdge runs the next edge case, unless none is left or the domain is
// exhausted.
func (p *phases) nextEdge() (Result, bool) {
	if len(p.edges) == 0 || p.tree.Exhausted() {
		return Result{}, false
	}
	at := p.edges[0]
	p.edges = p.edges[1:]
	return p.t.take(p.cases.edge(at), record.Edge)
}

// runTo runs random cases until target cases are valid, the domain is
// exhausted, or ten times target random cases have run, and then the edge
// cases left.
func (p *phases) runTo(target int) (Result, bool) {
	for p.t.valid < target && !p.tree.Exhausted() && p.index < uint64(generationFactor*target) {
		if r, ended := p.random(); ended {
			return r, true
		}
	}
	for len(p.edges) > 0 && !p.tree.Exhausted() {
		if r, ended := p.nextEdge(); ended {
			return r, true
		}
	}
	return Result{}, false
}

// RunStored runs the stored cases of s, oldest first, as [Run] runs them
// before its other cases, and returns how they ended: a pass that counts
// them, or the result that the first of them that does not pass ends a run
// with, a failing case as found, without shrinking or explaining it. The
// result states the runs of the stored cases up to that one, and the calls
// of each case are recorded under [record.Stored].
func RunStored(body Body, s Settings) Result {
	s = withClocks(s)
	t := newTally(s)
	r, ended := stored(body, s, t)
	if !ended {
		runs := r.Stored
		r = t.result(Passed)
		r.Stored = runs
		return r
	}
	if r.Failing != nil {
		r.Token = token.Encode(r.Failing.Case.Choices())
	}
	return r
}

// explore runs the phases until the run ends, without concluding a
// counterexample, and returns the result with the runs of the stored cases.
func explore(body Body, s Settings) Result {
	t := newTally(s)
	if r, ended := known(body, s, t); ended {
		return r
	}
	r, ended := stored(body, s, t)
	if ended {
		return r
	}
	runs := r.Stored
	r = generated(body, s, t)
	r.Stored = runs
	return r
}

// stored runs the stored cases of s oldest first, counted by t. It returns
// the result that ends the run and true, or a result of the runs alone and
// false. Either states the runs of the stored cases, up to the one that
// ended the run.
func stored(body Body, s Settings, t *tally) (Result, bool) {
	runs := make([]Execution, 0, len(s.Stored))
	for _, choices := range s.Stored {
		e := execute(body, replaying{choices: choices}, s)
		runs = append(runs, e)
		if r, ended := t.take(e, record.Stored); ended {
			r.Stored = runs
			return r, true
		}
	}
	return Result{Stored: runs}, false
}

// known runs the cases whose values the caller states, outside the case
// tree: the case of the Draws entries, when there are entries, and then
// the examples. It returns the result that ends the run.
func known(body Body, s Settings, t *tally) (Result, bool) {
	if len(s.Draws) > 0 {
		if r, ended := t.take(executeDraws(body, s), record.Example); ended {
			return r, true
		}
	}
	for _, ex := range s.Examples {
		if r, ended := t.take(executeExample(body, s, ex), record.Example); ended {
			return r, true
		}
	}
	return Result{}, false
}

// generated runs the simplest case and the random and edge cases into the
// case tree, after the stored cases that t counted, until the run ends.
func generated(body Body, s Settings, t *tally) Result {
	caseTree := tree.New(tree.NodeLimit)
	p := &phases{s: s, t: t, tree: caseTree, cases: newExecutor(body, s, caseTree), edges: boundaries[:]}
	defer p.cases.close()
	if r, ended := t.take(p.cases.replay(nil), record.Simplest); ended {
		return r
	}
	if r, ended := checks(p); ended {
		return r
	}
	return t.result(Passed)
}

// checks runs the random phase to each check, and returns the result that
// ends the run. A run without requirements ends at the first check. The
// last check and every check of an exhausted domain decide every
// requirement, so the run ends at the last check at the latest.
func checks(p *phases) (Result, bool) {
	multiples := coverage.Checks()
	last := len(multiples) - 1
	for position := 0; ; position++ {
		p.phase = record.Coverage
		if position == 0 {
			p.phase = record.Random
		}
		r, ended := p.runTo(multiples[position] * p.s.Cases)
		if !ended {
			r, ended = p.t.missing()
		}
		if ended || len(p.s.Requirements) == 0 {
			return r, ended
		}
		stage := coverage.Interim
		if position == last {
			stage = coverage.Final
		}
		if p.tree.Exhausted() {
			stage = coverage.Exhausted
		}
		shortfall, met := shortfallOf(p.t, p.s.Requirements, stage)
		if shortfall != nil {
			r := p.t.result(CoverageUnmet)
			r.Shortfall = shortfall
			return r, true
		}
		if met {
			return Result{}, false
		}
	}
}

// shortfallOf returns the first refuted or unmet requirement, and whether
// every requirement is met.
func shortfallOf(t *tally, requirements []Requirement, stage coverage.Stage) (*Shortfall, bool) {
	met := true
	for _, requirement := range requirements {
		counted := t.labels[requirement.Label]
		verdict := coverage.Decide(counted, t.valid, requirement.Share, stage)
		if verdict == coverage.Refuted || verdict == coverage.Unmet {
			return &Shortfall{Requirement: requirement, Counted: counted, Valid: t.valid, Verdict: verdict}, false
		}
		met = met && verdict == coverage.Met
	}
	return nil, met
}

// prefixCases returns the most valid cases a run may have and still try a
// prefix case.
func prefixCases(cases int) int {
	return min(cases/prefixDivisor, maxPrefixCases)
}

// withClocks returns s with the platform clock for each clock it does not
// state.
func withClocks(s Settings) Settings {
	if s.Clock == nil {
		s.Clock = assert.System{}
	}
	if s.ShrinkClock == nil {
		s.ShrinkClock = assert.System{}
	}
	if s.BudgetClock == nil {
		s.BudgetClock = assert.System{}
	}
	return s
}

// conclude replays, shrinks and explains the failing case of a
// counterexample. It reports an example of values as found, without a
// token.
func conclude(body Body, s Settings, r Result) Result {
	if r.Outcome != Counterexample || r.Failing.Case.Valued() {
		return r
	}
	if s.Shrink == 0 {
		r.Token = token.Encode(r.Failing.Case.Choices())
		return r
	}
	if d := confirm(body, *r.Failing, s); d != nil {
		r.Outcome, r.Divergence = Flaky, d
		return r
	}
	sh := newShrinker(body, *r.Failing, s)
	sh.shrinkAll()
	first := sh.failures[r.Failing.Identity]
	for _, identity := range sh.found[1:] {
		r.Others = append(r.Others, sh.failures[identity].execution)
	}
	if s.Explain {
		r.Explanation = explain(sh, first, s.Seed)
	}
	r.Failing, r.Token, r.Runs = &first.execution, token.Encode(first.execution.Case.Choices()), sh.runs
	return r
}
