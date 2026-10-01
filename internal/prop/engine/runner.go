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
}

// Run runs the phases of a property and returns how the run ended.
//
// The runner calls body once per case, and stops at the first failing
// case. The stored cases run first, outside the case tree, then the case
// whose every choice is its target. Then random case i of the seed runs,
// for i = 0, 1, 2 and on, until Cases valid cases have run, the domain is
// exhausted, or ten times Cases random cases have run, repeats included.
// Each random case is followed by its prefix case and the next edge case.
// While a coverage requirement is undecided at a check, more random cases
// run up to the next check.
//
// A failing case is replayed, shrunk and explained. A run that found no
// failing case fails when it rejected more than ten cases for every valid
// one, when no case requested an input, or when it refuted or left unmet
// a coverage requirement, checked in that order.
func Run(body Body, s Settings) Result {
	s = withClocks(s)
	return conclude(body, s, explore(body, s))
}

// RunReplay runs the one case that choices record, and reports a failure
// as found, without shrinking or explaining it.
func RunReplay(body Body, s Settings, choices []choice.Choice) Result {
	s = withClocks(s)
	t := &tally{seed: s.Seed, labels: make(map[string]int)}
	r, ended := t.take(execute(body, replaying{choices: choices}, s.MaxChoices, nil, s.Clock))
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
}

// take counts one case, and returns the result it ends the run with.
func (t *tally) take(e Execution) (Result, bool) {
	t.requested = t.requested || len(e.Case.Draws()) > 0 || len(e.Case.Choices()) > 0
	if e.Status == CaseFailed {
		r := t.result(Counterexample)
		r.Failing = &e
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
	if r, ended := p.t.take(e); ended {
		return r, true
	}
	prefixed := p.t.valid <= prefixCases(p.s.Cases) && len(choices) >= minPrefixedChoices && !p.tree.Exhausted()
	if prefixed {
		cut := 1 + source.Below(uint64(len(choices)-1))
		if r, ended := p.t.take(p.cases.replay(choices[:cut])); ended {
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
	return p.t.take(p.cases.edge(at))
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

// explore runs the phases until the run ends, without concluding a
// counterexample.
func explore(body Body, s Settings) Result {
	t := &tally{seed: s.Seed, labels: make(map[string]int)}
	for _, stored := range s.Stored {
		if r, ended := t.take(execute(body, replaying{choices: stored}, s.MaxChoices, nil, s.Clock)); ended {
			return r
		}
	}
	caseTree := tree.New(tree.NodeLimit)
	p := &phases{s: s, t: t, tree: caseTree, cases: newExecutor(body, s, caseTree), edges: boundaries[:]}
	defer p.cases.close()
	if r, ended := t.take(p.cases.replay(nil)); ended {
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
	return s
}

// conclude replays, shrinks and explains the failing case of a
// counterexample.
func conclude(body Body, s Settings, r Result) Result {
	if r.Outcome != Counterexample {
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
