// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/internal/record"
)

// drawn is the label of the one value a test body draws.
const drawn = "value"

// referenceSeed is the seed of every run whose outcome the tests pin from
// the definition's executable reference.
const referenceSeed = 7

// fourWorkers is the number of workers of the runs that a test compares
// with a run on one worker.
const fourWorkers = 4

// drawAllocs are the allocations of a whole case that replays one draw, its
// goroutine and its recorder included, measured.
const drawAllocs = 9

// The first values past the members of the engine's enums.
const (
	// invalidDifference is the first value past the three differences.
	invalidDifference engine.Difference = 3
	// invalidStatus is the first value past the six statuses.
	invalidStatus engine.Status = 6
	// invalidRelevance is the first value past the three relevances.
	invalidRelevance engine.Relevance = 3
	// invalidOutcome is the first value past the six outcomes.
	invalidOutcome engine.Outcome = 6
)

// digitRange are the bounds [0, 9] of a choice that a test body makes on the
// case itself.
var digitRange = choice.MustIntegerBounds(choice.Int{}, choice.UintOf(9))

// reported is the record that a test body reports, as an assertion would.
var reported = assert.Failure{
	Assertion: "equal",
	Contract:  "the totals match",
	Where:     assert.Where{File: "ledger_test.go", Line: 12},
}

// calls is the record of the calls of a seat of the test.
type calls = record.Calls

// keeping is a seat of the test that keeps the call records of its calls,
// as a recorder does.
type keeping struct {
	calls
}

// callRecord is what a test reads of the call record of a case's call: its
// run and its phase.
type callRecord struct {
	// Run is the run of the property's body that the call ran in.
	Run int `json:"run"`
	// Phase is the phase of the case.
	Phase string `json:"phase"`
}

// recordedRun calls run with s and the slot of a recorded property, and
// returns the call records of the calls of the property's cases, in the
// order of their numbers.
func recordedRun(t *testing.T, s engine.Settings, run func(engine.Settings)) []callRecord {
	t.Helper()
	k := &keeping{}
	record.Keep(&k.calls)
	s.Slot = record.Begin(k)
	run(s)
	lines := record.Lines(&k.calls)
	out := make([]callRecord, len(lines))
	for i, line := range lines {
		assert.NoError(t, json.Unmarshal([]byte(line), &out[i]), "the call record is JSON")
	}
	return out
}

// phasesOf returns the phase of each record, in order.
func phasesOf(records []callRecord) []string {
	out := make([]string, len(records))
	for i, r := range records {
		out[i] = r.Phase
	}
	return out
}

// ended returns body with a passing call of true at its end, which a case
// makes when its body runs to the end.
func ended(body engine.Body) engine.Body {
	return func(c *engine.Case) {
		body(c)
		assert.True(c, true, "the case ends")
	}
}

// errOdd is the error of halve for an odd integer.
var errOdd = errors.New("engine_test: the integer is odd")

// celsius is an integer type defined over int8.
type celsius int8

// property is a body that draws its values and returns the assertion of
// the failure they make, or "" for none.
type property func(c *engine.Case) string

// reference is what the definition's executable reference reports for a
// run: the explanation of each draw of the minimal case, the minimal case's
// token, the runs that shrinking and explaining spent, and the number of
// calls of the body with a SHA-256 digest of the token of each call's
// choices, joined by newlines.
type reference struct {
	// explanation is the explanation of each draw of the minimal case.
	explanation []engine.Explained
	// token is the minimal case's replay token.
	token string
	// runs are the runs that shrinking and explaining spent.
	runs int
	// calls is the number of calls of the body.
	calls int
	// digest is the digest of every call's choices.
	digest string
}

// other is a further failure of a run: its assertion, the values of its
// minimal case, and that case's token.
type other struct {
	// assertion is the failure's assertion.
	assertion string
	// values are the values the minimal case drew.
	values []any
	// token is the minimal case's token.
	token string
}

// runReport is what a run reports, without the cases it ran: how it ended
// with its counts, its minimal case's token, the runs that shrinking and
// explaining spent, the explanation, and the other failures.
type runReport struct {
	// summary is how the run ended, with its counts.
	summary engine.Result
	// token is the token of the minimal case.
	token string
	// runs are the runs that shrinking and explaining spent.
	runs int
	// explanation explains each draw of the minimal case.
	explanation []engine.Explained
	// others are the other failures.
	others []other
}

// decodePair returns two value choices of the digits, made on the case
// without a span.
func decodePair(c *engine.Case) [2]uint64 {
	first := c.Integer(digitRange).Magnitude()
	return [2]uint64{first, c.Integer(digitRange).Magnitude()}
}

// invertPair returns the steps of a pair that decodePair decodes: one value
// choice of the digits for each number, unchecked against the digits.
func invertPair(v any) ([]engine.Step, [2]uint64, error) {
	pair, ok := v.([2]uint64)
	if !ok {
		return nil, pair, fmt.Errorf("%v is no pair", v)
	}
	digit := func(d uint64) engine.Step {
		return engine.Step{Bounds: choice.OfInteger(digitRange), Value: unsigned(d)}
	}
	return []engine.Step{digit(pair[0]), digit(pair[1])}, pair, nil
}

// halve returns the integer that doubling maps to v, and errOdd for an odd
// v.
func halve(v int) (int, error) {
	if v%2 != 0 {
		return 0, errOdd
	}
	return v / 2, nil
}

// decode returns the value that g decodes from a case replaying choices,
// with the run of that case.
func decode[T any](tb testing.TB, g engine.Generator[T], choices ...choice.Choice) (T, engine.Execution) {
	tb.Helper()
	var got T
	e := engine.Replay(func(c *engine.Case) { got = engine.Draw(c, g, drawn) }, choices, nil)
	return got, e
}

// integers returns integer choices of the values.
func integers(values ...int64) []choice.Choice {
	out := make([]choice.Choice, len(values))
	for i, v := range values {
		out[i] = choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(v)}
	}
	return out
}

// unsigned returns the integer choice of v.
func unsigned(v uint64) choice.Choice {
	return choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(v)}
}

// float returns the float choice of v.
func float(v float64) choice.Choice {
	return choice.Choice{Kind: choice.Float, Float: v}
}

// sequence returns the sequence choice of the elements.
func sequence(elements ...uint32) choice.Choice {
	return choice.Choice{Kind: choice.Sequence, Sequence: elements}
}

// generated returns the values that g decodes in the first count cases of
// seed, and the choices that each case recorded.
func generated[T any](g engine.Generator[T], seed uint64, count int) ([]T, [][]choice.Choice) {
	values := make([]T, count)
	records := make([][]choice.Choice, count)
	for i := range count {
		e := engine.Generate(func(c *engine.Case) { values[i] = engine.Draw(c, g, drawn) }, seed, uint64(i), nil)
		records[i] = e.Case.Choices()
	}
	return values, records
}

// sameChoices reports whether a and b are the same choices in order, with
// floats compared by their bits.
func sameChoices(a, b []choice.Choice) bool {
	return slices.EqualFunc(a, b, choice.Choice.Equal)
}

// sameRecords reports whether a and b are the same choice sequences in
// order.
func sameRecords(a, b [][]choice.Choice) bool {
	return slices.EqualFunc(a, b, sameChoices)
}

// labels returns the labels of spans, in order.
func labels(spans []engine.Span) []string {
	out := make([]string, len(spans))
	for i, span := range spans {
		out[i] = span.Label
	}
	return out
}

// drawLabels returns the labels of draws, in order.
func drawLabels(draws []engine.Drawn) []string {
	out := make([]string, len(draws))
	for i, d := range draws {
		out[i] = d.Label
	}
	return out
}

// sizes returns the lengths from minSize to maxSize, failing the test when
// they are invalid.
func sizes(tb testing.TB, minSize, maxSize int) choice.Sizes {
	tb.Helper()
	s, err := choice.NewSizes(minSize, maxSize)
	assert.NoError(tb, err, "the sizes are valid")
	return s
}

// unbounded returns the lengths of minSize or more, failing the test when
// they are invalid.
func unbounded(tb testing.TB, minSize int) choice.Sizes {
	tb.Helper()
	s, err := choice.NewUnboundedSizes(minSize)
	assert.NoError(tb, err, "the sizes are valid")
	return s
}

// anyOf returns g with its values as any, so generators of different types
// share one one-of.
func anyOf[T any](g engine.Generator[T]) engine.Generator[any] {
	return g.Map(func(v T) any { return v })
}

// tree returns a recursive generator of a digit, or of a list of at most
// width positions, with at most maxLeaves digits in one value.
func tree(tb testing.TB, width, maxLeaves int) engine.Generator[any] {
	tb.Helper()
	upTo := sizes(tb, 0, width)
	return engine.Recursive(anyOf(engine.Integer(0, 9)), func(self engine.Generator[any]) engine.Generator[any] {
		return anyOf(engine.List(self, upTo))
	}, maxLeaves)
}

// leafCount returns the number of digits in a tree of digits and lists.
func leafCount(value any) int {
	items, ok := value.([]any)
	if !ok {
		return 1
	}
	count := 0
	for _, item := range items {
		count += leafCount(item)
	}
	return count
}

// site stores the file and the line of its caller in at and returns the
// line, so a call inside another call's arguments states where that call
// is.
func site(at *assert.Where) int {
	_, at.File, at.Line, _ = runtime.Caller(1)
	return at.Line
}

// settled returns the settings of a run of the reference seed that shrinks
// and explains with the default budget, and tries the stored case of
// choices first when there are any.
func settled(choices ...choice.Choice) engine.Settings {
	s := engine.Settings{
		Seed:       referenceSeed,
		Cases:      engine.DefaultCases,
		MaxChoices: engine.MaxChoices,
		Shrink:     engine.DefaultShrink,
		Explain:    true,
	}
	if len(choices) > 0 {
		s.Stored = [][]choice.Choice{choices}
	}
	return s
}

// require returns the adjustment of settings that states requirements.
func require(requirements ...engine.Requirement) func(*engine.Settings) {
	return func(s *engine.Settings) { s.Requirements = requirements }
}

// traced runs p under s, and returns the result and the token of the
// choices of each call of the body, in call order. The entry of a call is
// taken as the call ends, so a call that a rejection or a repeat ends has
// one too.
func traced(p property, s engine.Settings) (engine.Result, []string) {
	var trace []string
	result := engine.Run(func(c *engine.Case) {
		defer func() { trace = append(trace, token.Encode(c.Choices())) }()
		if failed := p(c); failed != "" {
			c.Report(assert.Failure{Assertion: failed}, false)
		}
	}, s)
	return result, trace
}

// recorded runs body under s, and returns the result and the token of the
// choices of each call of the body when the call ends, however it ends.
func recorded(body engine.Body, s engine.Settings) (engine.Result, []string) {
	var trace []string
	result := engine.Run(func(c *engine.Case) {
		defer func() { trace = append(trace, token.Encode(c.Choices())) }()
		body(c)
	}, s)
	return result, trace
}

// matchesReference checks a counterexample and the trace of its run
// against what the reference reports for the same run.
func matchesReference(tb testing.TB, got engine.Result, trace []string, want reference) {
	tb.Helper()
	assert.Equal(tb, got.Outcome, engine.Counterexample, "a counterexample")
	assert.Equal(tb, got.Explanation, want.explanation, "each draw of the minimal case, explained", assert.EquateNaNs())
	assert.Equal(tb, got.Token, want.token, "the minimal case's token")
	assert.Equal(tb, got.Runs, want.runs, "the runs that shrinking and explaining spent")
	assert.Equal(tb, len(trace), want.calls, "the calls of the body")
	assert.Equal(tb, digestOf(trace), want.digest, "the choices of every call, in order")
}

// digestOf returns the SHA-256 digest of the trace joined by newlines, in
// hexadecimal.
func digestOf(trace []string) string {
	sum := sha256.Sum256([]byte(strings.Join(trace, "\n")))
	return hex.EncodeToString(sum[:])
}

// summary returns r without its cases, its explanation, its token and its
// runs: how the run ended and its counts.
func summary(r engine.Result) engine.Result {
	return engine.Result{
		Outcome:    r.Outcome,
		Cases:      r.Cases,
		Rejected:   r.Rejected,
		Seed:       r.Seed,
		Divergence: r.Divergence,
		Shortfall:  r.Shortfall,
	}
}

// reportOf returns what r reports, without the cases it ran.
func reportOf(r engine.Result) runReport {
	return runReport{summary: summary(r), token: r.Token, runs: r.Runs, explanation: r.Explanation, others: others(r)}
}

// others returns the further failures of a run, in the order it found
// them.
func others(r engine.Result) []other {
	out := make([]other, len(r.Others))
	for i, e := range r.Others {
		out[i] = other{e.Identity.Assertion, drawValues(e.Case.Draws()), token.Encode(e.Case.Choices())}
	}
	return out
}

// drawValues returns the values of draws, in order.
func drawValues(draws []engine.Drawn) []any {
	out := make([]any, len(draws))
	for i, d := range draws {
		out[i] = d.Value
	}
	return out
}

// failsWhen returns assertion when failing is true, and "" otherwise.
func failsWhen(failing bool, assertion string) string {
	if failing {
		return assertion
	}
	return ""
}

// above returns the property that fails with "above" for a value of g
// above 1000.
func above(g engine.Generator[int]) property {
	return func(c *engine.Case) string {
		return failsWhen(engine.Draw(c, g, "n") > 1000, "above")
	}
}

// sum returns the sum of values.
func sum(values []int) int {
	total := 0
	for _, v := range values {
		total += v
	}
	return total
}
