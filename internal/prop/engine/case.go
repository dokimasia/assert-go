// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"reflect"
	"runtime"
	"runtime/debug"
	"slices"
	"sync"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/tree"
	"go.dokimi.dev/assert/internal/record"
)

// calls is the record of the calls that a case's body makes on the case.
type calls = record.Calls

// MaxChoices is the most choices that one case may make by default. A
// sequence counts as one choice plus one for each element.
const MaxChoices = 8192

// stackReserve is the size in bytes of the frame that reserveStack puts on
// a case's goroutine before the body runs, which grows the stack to 16 KB.
const stackReserve = 8192

// The bounds that the case's own choices use.
var (
	// bitBounds are the bounds [0, 1] of a boolean.
	bitBounds = choice.MustIntegerBounds(choice.Int{}, choice.UintOf(1))
	// unsignedBounds are the bounds of the whole unsigned 64-bit range.
	unsignedBounds = choice.MustIntegerBounds(choice.Int{}, choice.UintOf(math.MaxUint64))
)

// stop is why a case ended before its body returned.
type stop uint8

const (
	// running is a case that has not stopped.
	running stop = 0
	// rejected is a case that the body rejected, or whose filter or unique
	// collection gave up.
	rejected stop = 1
	// overrun is a case that asked for more choices than its cap allows.
	overrun stop = 2
	// repeated is a case whose choices repeat a tested case.
	repeated stop = 3
	// diverged is a case whose body requested other choices after the same
	// values than an earlier case.
	diverged stop = 4
	// cancelled is a case that a run on more than one worker started ahead
	// and no longer needs.
	cancelled stop = 5
	// refused is a case of Settings.Draws that refused the entry of a draw.
	refused stop = 6
)

// walkStep is one step of a case's walk down the case tree: the bounds and
// the value of a choice, and the index that the choice took in the record.
type walkStep struct {
	// bounds are the bounds of the request.
	bounds choice.Bounds
	// value is the choice.
	value choice.Choice
	// at is the index of the choice in the record when the case made it.
	at int
}

// Drawn is one value that a body drew: its label, its value, and the span
// of its generator.
type Drawn struct {
	// Label is the label the body drew the value under.
	Label string
	// Value is the value the generator decoded.
	Value any
	// Span is the index of the first span the generator opened.
	Span int
	// source is the generator, for the explain phase to decode again.
	source erased
}

// Neutral returns the value as the generator states it: for a draw from a
// generator that [Generator.MapBack] built, the value that it maps from, as
// the generator it maps states that value, and the value itself for any
// other draw. A typed literal of the neutral value runs back to the draw's
// choices.
func (d Drawn) Neutral() any {
	return d.source.neutralOf(d.Value)
}

// Case is the record of one call of a body: the choices it made, the
// requests and spans behind them, and what the body drew, classified,
// noted and observed.
//
// The body's assertions report to the case as an [assert.TB], an
// [assert.Reporter] and an [assert.Clocked]. The case forwards every report
// to an [assert.Recorder] under [assert.Recorder.WithGoexit], so a fatal
// failure ends the body's goroutine. A rejection, a repeated case, a
// divergence and a case past its cap end it the same way.
//
// When the body ends, however it ends, the case cancels its context and
// runs its cleanups on the body's goroutine, the last registered first. A
// cleanup is part of the case: its failures and its choices are the
// case's.
//
// The case keeps the call records of the body's assertions as one run of
// the property's body, which the property's record takes when the run
// takes the case's result.
//
// # Concurrency
//
// Every method is safe for concurrent use. A draw, a rejection and a
// fatal failure end the goroutine that makes them, so a body makes them on
// the goroutine it runs on.
type Case struct {
	calls

	// recorder keeps the body's failures in call order.
	recorder *assert.Recorder
	// done waits for the goroutine that finish runs the body on.
	done sync.WaitGroup
	// mu guards the fields below.
	mu sync.Mutex
	// provider supplies the value of every choice.
	provider provider
	// maxChoices is the cap on the choices of the case.
	maxChoices int
	// walker follows the case down the case tree, and is nil for a case
	// outside it.
	walker *tree.Walker
	// cost is the number of choices made so far, each sequence element
	// counted.
	cost int
	// choices are the values of the choices, in order.
	choices []choice.Choice
	// requests are what the case recorded of the requests behind the
	// choices, in order.
	requests []recorded
	// keepsWheres reports whether the case keeps where it made each request
	// and where it observed each fingerprint, in wheres and observed, as the
	// replay that confirms a failure does.
	keepsWheres      bool
	wheres, observed []Where
	// divergedAt is where the case made the request that diverged from the
	// case tree.
	divergedAt Where
	// keepsWalk reports whether the case keeps its walk, which a case that
	// runs outside the case tree and enters it afterwards does.
	keepsWalk bool
	// walk are the steps of the case's walk, the choices that a rewind
	// removed from the record included, in order, when the case keeps it.
	walk []walkStep
	// spans are the spans in the order they opened.
	spans []Span
	// open are the indices of the spans not closed yet, innermost last.
	open []int
	// draws are the values the body drew, in order.
	draws []Drawn
	// labels are the labels the body classified the case under, and nil
	// before the first.
	labels map[string]struct{}
	// notes are the messages the body attached to the case.
	notes []string
	// fingerprints are the fingerprints the body observed, in order.
	fingerprints []uint64
	// label is the label of the innermost draw that runs, when drawing.
	label   string
	drawing bool
	// place is the part and the step of a machine that are running, when
	// placed.
	place  Place
	placed bool
	// taken are the steps that the body's machine took, each with the number
	// of draws before it, in order.
	taken []RecordedStep
	// targets are the scores that the body recorded, the highest of each
	// label, and nil before the first.
	targets map[string]float64
	// history is the case's history, and nil until the body asks for it.
	history *history.History
	// repeat is the most runs of the case that the body asked for, and 0
	// when it asked for none.
	repeat int
	// settings are the settings of the run, under which a repeat of the case
	// runs.
	settings Settings
	// stop is why the case ended before its body returned.
	stop stop
	// dropped reports whether the run no longer needs the case, which then
	// ends at its next choice.
	dropped bool
	// divergence is the tree's report of a diverged case.
	divergence *tree.DivergenceError
	// panicked is the identity of a panic that ended the body.
	panicked *Identity
	// panicValue is the value of that panic.
	panicValue any
	// panicStack is the stack of the body's goroutine at that panic, as
	// runtime/debug.Stack formats it.
	panicStack []byte
	// raised is where the last panic that [Case.Raise] raises again was
	// raised first, and nil before one.
	raised *origin
	// recursion are the counts of base values of the recursive values
	// being decoded, a stack for each recursive generator.
	recursion map[any][]int
	// cleanups are the cleanups that have not run yet, in the order they
	// were registered.
	cleanups []func()
	// ended reports whether the body has ended, after which the case's
	// context is cancelled.
	ended bool
	// parent is the context that the case's context derives from, which
	// [WithContext] sets, and nil for context.Background().
	parent context.Context
	// replayed is the provider of a case that recycleReplaying starts. It is
	// a field of the case, so the provider costs no allocation.
	replayed replaying
	// inverting is the provider of a case of Settings.Draws, and nil for any
	// other case. It is set before the body starts and never changes.
	inverting *inverting
	// valuing is the provider of an example of values, and nil for any other
	// case. It is set before the body starts and never changes.
	valuing *valuing
	// refusal is the refusal that ended a case of Settings.Draws.
	refusal error
	// ctx is the case's context, and nil until a caller asks for it.
	ctx context.Context
	// cancelCtx cancels ctx.
	cancelCtx context.CancelFunc
}

var (
	_ assert.TB       = (*Case)(nil)
	_ assert.Reporter = (*Case)(nil)
	_ assert.Clocked  = (*Case)(nil)
)

// newCase returns an empty case whose values come from p, capped at
// s.MaxChoices, which walks w when it is not nil, reads the clock of s, and
// keeps its calls for the slot of s.
func newCase(p provider, s Settings, w *tree.Walker) *Case {
	c := new(Case)
	c.recycle(p, s, w)
	return c
}

// recycle empties c, a case whose body's goroutine has ended, for a new
// case as newCase describes it. It keeps the storage of the record's
// choices, requests, spans and draws, and of the cleanups, and clears their
// elements. The caller no longer uses c's earlier record, and the slot took
// c's earlier calls.
func (c *Case) recycle(p provider, s Settings, w *tree.Walker) {
	clear(c.choices)
	clear(c.spans)
	clear(c.draws)
	clear(c.taken)
	clear(c.cleanups)
	*c = Case{
		recorder:   assert.NewRecorder().WithGoexit().WithClock(s.Clock),
		provider:   p,
		maxChoices: s.MaxChoices,
		walker:     w,
		choices:    c.choices[:0],
		requests:   c.requests[:0],
		spans:      c.spans[:0],
		open:       c.open[:0],
		draws:      c.draws[:0],
		taken:      c.taken[:0],
		cleanups:   c.cleanups[:0],
		settings:   s,
	}
	record.Run(&c.calls, s.Slot, nil)
}

// recycleReplaying empties c, as recycle does, for a case outside the case
// tree that replays choices, whose provider is c's own field.
func (c *Case) recycleReplaying(choices []choice.Choice, s Settings) {
	c.recycle(nil, s, nil)
	c.replayed = replaying{choices: choices}
	c.provider = &c.replayed
}

// steps returns how many steps the case's kept walk has made, which the
// call record of a case that runs ahead of the runner states.
func (c *Case) steps() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.walk)
}

// keepWalk makes c, a new case outside the case tree whose body has not
// started, keep its walk, which the runner enters into the case tree once
// the case has ended. Each call record that the case keeps then states the
// steps that its walk had made, so the runner keeps only the calls that a
// run on one worker makes.
func (c *Case) keepWalk() {
	c.keepsWalk = true
	if c.settings.Slot != nil {
		record.Run(&c.calls, c.settings.Slot, c.steps)
	}
}

// Helper forwards a helper mark to the case's recorder, which counts it.
// A record's location is the frame that its assertion or message states,
// so the mark moves no record.
func (c *Case) Helper() {
	c.recorder.Helper()
}

// Report keeps an assertion's record. An aborting record ends the
// calling goroutine.
func (c *Case) Report(f assert.Failure, aborting bool) {
	c.recorder.Report(f, aborting)
}

// Fatalf keeps a record without an assertion, whose contract is the message
// and whose location is the innermost frame of the caller's code, and ends
// the calling goroutine.
func (c *Case) Fatalf(format string, args ...any) {
	c.recorder.Report(plain(format, args), true)
}

// Errorf keeps a record without an assertion, whose contract is the message
// and whose location is the innermost frame of the caller's code. The case
// fails when its body returns.
func (c *Case) Errorf(format string, args ...any) {
	c.recorder.Report(plain(format, args), false)
}

// Clock returns the clock of the test that runs the case.
func (c *Case) Clock() assert.Clock {
	return c.recorder.Clock()
}

// Assume rejects the case when condition is false, which ends the calling
// goroutine. A rejected case is not counted and not shrunk.
func (c *Case) Assume(condition bool) {
	if !condition {
		c.end(rejected)
	}
}

// Classify counts the case under label. A label counted twice in one case
// counts once.
func (c *Case) Classify(label string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.labels == nil {
		c.labels = make(map[string]struct{})
	}
	c.labels[label] = struct{}{}
}

// Note attaches message to the case. Only a failing case reports its
// notes.
func (c *Case) Note(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notes = append(c.notes, message)
}

// Observe records a fingerprint of the subject's state, which a replay of
// the case compares. The replay that confirms a failure also records where
// it observed the fingerprint.
func (c *Case) Observe(fingerprint uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fingerprints = append(c.fingerprints, fingerprint)
	if c.keepsWheres {
		c.observed = append(c.observed, c.where())
	}
}

// History returns the case's history, which records the calls that the body
// makes to a subject. It is empty when the case starts, and the case makes
// it at the first call.
func (c *Case) History() *history.History {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.history == nil {
		c.history = history.New()
	}
	return c.history
}

// Target records score as a score that the case achieved under label. A case
// that records two scores under one label keeps the higher. A campaign
// explores near the cases with the highest score of each label, and every
// other run records the score and generates as if it were absent.
func (c *Case) Target(label string, score float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.targets == nil {
		c.targets = make(map[string]float64)
	}
	if best, scored := c.targets[label]; !scored || score > best {
		c.targets[label] = score
	}
}

// Targets returns a copy of the scores that the case recorded, the highest
// of each label.
func (c *Case) Targets() map[string]float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return maps.Clone(c.targets)
}

// Rand returns a source of random values whose every value is an integer
// choice over the whole unsigned 64-bit range.
func (c *Case) Rand() Source {
	return Source{c: c}
}

// Cleanup registers f to run when the body ends. The case runs its
// cleanups on the body's goroutine, the last registered first, and a
// cleanup that a cleanup registers runs before the case ends. A cleanup
// registered after the case ended never runs.
func (c *Case) Cleanup(f func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanups = append(c.cleanups, f)
}

// Context returns the case's context. It derives from the context that
// [WithContext] gave the body, and from context.Background() for a body
// without one, and the case cancels it when the body ends, before the
// cleanups run. A second call returns the same context, and a context
// first asked for once the body has ended is cancelled already.
func (c *Case) Context() context.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ctx == nil {
		parent := c.parent
		if parent == nil {
			parent = context.Background()
		}
		c.ctx, c.cancelCtx = context.WithCancel(parent)
		if c.ended {
			c.cancelCtx()
		}
	}
	return c.ctx
}

// derive makes ctx the context that the case's context derives from.
func (c *Case) derive(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.parent = ctx
}

// Choices returns a copy of the choices the case made, in order.
func (c *Case) Choices() []choice.Choice {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.choices)
}

// Spans returns a copy of the spans of the case, in the order they opened.
func (c *Case) Spans() []Span {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.spans)
}

// Draws returns a copy of the values the body drew, in order.
func (c *Case) Draws() []Drawn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.draws)
}

// Labels returns the labels the case was classified under, sorted.
func (c *Case) Labels() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	labels := make([]string, 0, len(c.labels))
	for label := range c.labels {
		labels = append(labels, label)
	}
	slices.Sort(labels)
	return labels
}

// Notes returns a copy of the notes attached to the case, in order.
func (c *Case) Notes() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.notes)
}

// Fingerprints returns a copy of the fingerprints the body observed, in
// order.
func (c *Case) Fingerprints() []uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.fingerprints)
}

// Failures returns every record the body's assertions and messages kept,
// in call order.
func (c *Case) Failures() []assert.Failure {
	return c.recorder.Failures()
}

// requested reports whether the body drew a value or made a choice. It
// allocates nothing.
func (c *Case) requested() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.draws) > 0 || len(c.choices) > 0
}

// record returns the choices the case made, without a copy. The caller
// reads it once the body's goroutine has ended, and changes nothing in it.
func (c *Case) record() []choice.Choice {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.choices
}

// choose returns the value for r and records it, with r. It ends the
// calling goroutine when the value takes the case past its cap, repeats a
// tested case or diverges from one, and once the case has stopped, as a
// cleanup's draw after such a stop does. A sequence whose minimum length
// alone takes the case past its cap ends it before the provider makes the
// value, which would make that many elements.
func (c *Case) choose(r request) choice.Choice {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dropped {
		c.halt(cancelled)
	}
	if c.stop != running {
		c.halt(c.stop)
	}
	if r.bounds.Kind() == choice.Sequence && r.bounds.Sequence().Sizes().Min() >= c.maxChoices-c.cost {
		c.halt(overrun)
	}
	value := c.provider.value(r, len(c.choices))
	c.cost += 1 + len(value.Sequence)
	if c.cost > c.maxChoices {
		c.halt(overrun)
	}
	c.choices = append(c.choices, value)
	c.requests = append(c.requests, recorded{bounds: r.bounds, structure: r.structure})
	if c.keepsWheres {
		c.wheres = append(c.wheres, c.where())
	}
	if c.keepsWalk {
		c.walk = append(c.walk, walkStep{bounds: r.bounds, value: value, at: len(c.choices) - 1})
	}
	if c.walker == nil {
		return value
	}
	if s := c.walked(c.walker.Step(r.bounds, value)); s != running {
		c.halt(s)
	}
	return value
}

// walked returns why the case stops at a step of its walk that returned
// err: repeated for a choice that repeats a tested case, diverged for a
// request that differs from the recorded one, and running for a step that
// goes on. It keeps the divergence in c.divergence, and where the case made
// the request in c.divergedAt. The caller has locked c.mu.
func (c *Case) walked(err error) stop {
	if errors.Is(err, tree.ErrRepeated) {
		return repeated
	}
	if errors.As(err, &c.divergence) {
		c.divergedAt = c.where()
		return diverged
	}
	return running
}

// Integer returns a value choice in b and records it. The random phase
// draws it anew, with no reuse of an earlier value, and the edge phase
// gives it the case's boundary. Like every choice, it ends the calling
// goroutine when it takes the case past its cap, repeats a tested case or
// diverges from one.
func (c *Case) Integer(b choice.IntegerBounds) choice.Int {
	return c.choose(request{bounds: choice.OfInteger(b)}).Integer
}

// Reusable returns a value choice in b and records it, as [Case.Integer]
// does, except that the random phase may give it an earlier value of the
// case with the same bounds. The second of two keys or two identifiers
// then equals the first in at least one case in four.
func (c *Case) Reusable(b choice.IntegerBounds) choice.Int {
	return c.choose(request{bounds: choice.OfInteger(b), reuse: true}).Integer
}

// Structure returns a choice in b that decides structure, such as an index
// among alternatives, and records it. The edge phase gives it the value
// edge. It ends the calling goroutine as [Case.Integer] does.
func (c *Case) Structure(b choice.IntegerBounds, edge uint64) choice.Int {
	return c.choose(request{bounds: choice.OfInteger(b), structure: true, edge: choice.UintOf(edge)}).Integer
}

// Coin returns a choice in [0, 1] that the random phase draws as a coin of
// num in den, records it, and reports whether it is 1. num is at most den,
// and den is above 0. It ends the calling goroutine as [Case.Integer] does.
func (c *Case) Coin(num, den uint64) bool {
	r := request{bounds: choice.OfInteger(bitBounds), drawing: byCoin, num: num, den: den}
	return c.choose(r).Integer == choice.UintOf(1)
}

// float returns a float choice in b.
func (c *Case) float(b choice.FloatBounds) float64 {
	return c.choose(request{bounds: choice.OfFloat(b)}).Float
}

// sequence returns a sequence choice in b.
func (c *Case) sequence(b choice.SequenceBounds) []uint32 {
	return c.choose(request{bounds: choice.OfSequence(b)}).Sequence
}

// more returns the decision whether a collection or a run of steps of count
// elements under sizes gets another, around the length average: a choice
// that decides structure, whose edge gives one element.
func (c *Case) more(sizes choice.Sizes, count, average int) bool {
	edge := uint64(0)
	if count == 0 {
		edge = 1
	}
	r := request{
		bounds:    choice.OfInteger(sizes.FlagBounds(count)),
		structure: true,
		edge:      choice.UintOf(edge),
		drawing:   byFlag,
		sizes:     sizes,
		count:     count,
		average:   average,
	}
	return c.choose(r).Integer == choice.UintOf(1)
}

// openSpan opens a span labelled label at the next choice, and returns its
// index for closeSpan.
func (c *Case) openSpan(label string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.openAt(label, len(c.choices))
}

// openSpanAt opens a span labelled label that starts at the choice at
// start, which the case recorded before, and returns its index for
// closeSpan. An element's span starts at the continue flag decided before
// it.
func (c *Case) openSpanAt(label string, start int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.openAt(label, start)
}

// openAt opens a span labelled label that starts at the choice at start,
// and returns its index. The caller has locked c.mu.
func (c *Case) openAt(label string, start int) int {
	parent := -1
	if len(c.open) > 0 {
		parent = c.open[len(c.open)-1]
	}
	index := len(c.spans)
	c.spans = append(c.spans, Span{Label: label, Start: start, End: start, Depth: len(c.open), Parent: parent})
	c.open = append(c.open, index)
	return index
}

// closeSpan closes the span at index, which ends at the next choice.
func (c *Case) closeSpan(index int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.open = c.open[:len(c.open)-1]
	c.spans[index].End = len(c.choices)
}

// Span calls f inside a span labelled label, which starts at the next
// choice and ends after the last choice f makes. The span closes when f
// returns or ends the goroutine. The shrinker deletes, lifts and sorts
// whole spans.
func (c *Case) Span(label string, f func()) {
	span := c.openSpan(label)
	defer c.closeSpan(span)
	f()
}

// draw records a value that g decoded under label, from the span at index
// span.
func (c *Case) draw(label string, value any, span int, g erased) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.draws = append(c.draws, Drawn{Label: label, Value: value, Span: span, source: g})
}

// nextSpan returns the index the next span will have.
func (c *Case) nextSpan() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.spans)
}

// Position returns the index that the case's next choice will have. A span
// that [Case.SpanFrom] opens starts at such an index.
func (c *Case) Position() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.choices)
}

// mark is a point of a case's record that rewind returns to.
type mark struct {
	// choices, spans and draws are the lengths of the record at the mark.
	choices, spans, draws int
}

// mark returns the current point of the record.
func (c *Case) mark() mark {
	c.mu.Lock()
	defer c.mu.Unlock()
	return mark{choices: len(c.choices), spans: len(c.spans), draws: len(c.draws)}
}

// rewind removes from the record every choice, span and draw made after
// at, which closed every span it opened. The choices still count towards
// the case's cap, and the tree walk keeps them.
func (c *Case) rewind(at mark) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.choices, c.requests = c.choices[:at.choices], c.requests[:at.choices]
	c.wheres = c.wheres[:min(len(c.wheres), at.choices)]
	c.spans, c.draws = c.spans[:at.spans], c.draws[:at.draws]
}

// recordAt returns the choices that the record contained after the first n
// steps of the case's kept walk, before the rewinds of the later steps.
func (c *Case) recordAt(n int) []choice.Choice {
	c.mu.Lock()
	defer c.mu.Unlock()
	var record []choice.Choice
	for _, s := range c.walk[:n] {
		record = append(record[:s.at], s.value)
	}
	return record
}

// Count returns the count of owner's innermost value in progress in the
// case, and 0 when no value of owner is in progress.
func (c *Case) Count(owner any) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	stack := c.recursion[owner]
	if len(stack) == 0 {
		return 0
	}
	return stack[len(stack)-1]
}

// Add adds one to the count of owner's innermost value in progress. A value
// of owner is in progress: an [Case.Enter] of owner whose end has not run.
func (c *Case) Add(owner any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recursion[owner][len(c.recursion[owner])-1]++
}

// Enter starts a count of 0 for owner's next value in the case, and returns
// the function that ends it. A generator whose values nest, such as a
// recursive one, counts what each value spends with [Case.Add] and reads
// it with [Case.Count]. The values of one owner nest, so the innermost one
// in progress counts.
func (c *Case) Enter(owner any) func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.recursion == nil {
		c.recursion = make(map[any][]int)
	}
	c.recursion[owner] = append(c.recursion[owner], 0)
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.recursion[owner] = c.recursion[owner][:len(c.recursion[owner])-1]
	}
}

// cancel ends the case at its next choice. A run on more than one worker
// cancels the cases it started ahead and no longer needs.
func (c *Case) cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dropped = true
}

// end records why the case stopped and ends the calling goroutine.
func (c *Case) end(s stop) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.halt(s)
}

// halt records why the case stopped and ends the calling goroutine. The
// caller has locked c.mu, and its deferred unlock runs as the goroutine
// ends. The case keeps the earliest reason.
func (c *Case) halt(s stop) {
	if c.stop == running {
		c.stop = s
	}
	runtime.Goexit()
}

// run calls body on c, on the goroutine that finish starts for it. It keeps
// a panic that ends the body, then runs the case's cleanups, and marks the
// goroutine done when the body and the cleanups have returned, panicked or
// ended the goroutine. It reserves the goroutine's stack before it calls
// the body.
func (c *Case) run(body Body) {
	defer c.done.Done()
	defer c.cleanUp()
	defer c.recoverPanic()
	reserveStack()
	body(c)
}

// origin is where a panic was raised first: its value, and the frames and
// the stack of the goroutine that raised it.
type origin struct {
	// value is the value of the panic.
	value any
	// pcs are the frames of the goroutine at the panic.
	pcs []uintptr
	// stack is the stack of the goroutine at the panic, as runtime/debug.Stack
	// formats it.
	stack []byte
}

// Raise panics with v on the calling goroutine, as a panic that another
// goroutine of the case raised where pcs and stack were taken. When the
// panic ends the body or a cleanup, the case keeps v with the identity and
// the stack of that goroutine, so two panics raised again keep the places
// where they were raised. A scheduler raises the panic of a task again
// this way. A body that recovers the panic recovers v.
func (c *Case) Raise(v any, pcs []uintptr, stack []byte) {
	c.mu.Lock()
	c.raised = &origin{value: v, pcs: pcs, stack: stack}
	c.mu.Unlock()
	panic(v)
}

// recoverPanic keeps the identity, the value and the stack of a panic that
// ends the body or a cleanup: those of the goroutine that raised it first,
// for a panic that [Case.Raise] raised again, and its own otherwise. The
// first panic of the case is kept. The runner defers it on the body's
// goroutine. A Goexit is no panic, and recover returns nil for it.
func (c *Case) recoverPanic() {
	v := recover()
	if v == nil {
		return
	}
	var pcs [maxFrames]uintptr
	n := runtime.Callers(1, pcs[:])
	frames, stack := pcs[:n], debug.Stack()
	c.mu.Lock()
	defer c.mu.Unlock()
	if o := c.raised; o != nil && reflect.DeepEqual(o.value, v) {
		frames, stack = o.pcs, o.stack
	}
	identity := panicIdentity(v, frames)
	if c.panicked == nil {
		c.panicked, c.panicValue, c.panicStack = &identity, v, stack
	}
}

// cleanUp cancels the case's context and runs the cleanups, the last
// registered first. A cleanup that panics or ends the goroutine leaves the
// others to the deferred cleanUpRest, which runs them. A cleanup that a
// cleanup registers runs as well.
func (c *Case) cleanUp() {
	defer c.cleanUpRest()
	defer c.recoverPanic()
	c.endBody()
	for f := c.nextCleanup(); f != nil; f = c.nextCleanup() {
		f()
	}
}

// cleanUpRest runs the cleanups that a panic or an end of the goroutine in
// a cleanup left.
func (c *Case) cleanUpRest() {
	c.mu.Lock()
	rest := len(c.cleanups) > 0
	c.mu.Unlock()
	if rest {
		c.cleanUp()
	}
}

// endBody marks the body ended and cancels the case's context.
func (c *Case) endBody() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ended = true
	if c.cancelCtx != nil {
		c.cancelCtx()
	}
}

// nextCleanup removes the cleanup registered last and returns it, or nil
// when none is left.
func (c *Case) nextCleanup() func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	last := len(c.cleanups) - 1
	if last < 0 {
		return nil
	}
	f := c.cleanups[last]
	c.cleanups[last] = nil
	c.cleanups = c.cleanups[:last]
	return f
}

// Source is a source of random values whose every value is an integer
// choice of its case over the whole unsigned 64-bit range. It satisfies
// math/rand/v2's Source.
type Source struct {
	// c is the case that makes the choices.
	c *Case
}

var _ rand.Source = Source{}

// Uint64 returns the next integer choice of the case.
func (s Source) Uint64() uint64 {
	return s.c.Integer(unsignedBounds).Magnitude()
}

// reserveStack grows the calling goroutine's stack until a frame of
// stackReserve bytes fits on it, while the goroutine is shallow.
//
// A case's goroutine starts with the stack size that the runtime averaged
// over the stacks it scanned at its last collection, which is 2 KB when no
// body was running then. The engine's frames for one draw take about
// 4.5 KB, so the stack grows inside the first draw, and the runtime copies
// and adjusts every frame of the body and the engine. Growing it before the
// body runs copies two frames. It costs the zeroing of the frame, and the
// 16 KB stack has room for about 10 KB of the body's own frames beside one
// draw before it grows again.
//
//go:noinline
func reserveStack() {
	var frame [stackReserve]byte
	keep(frame[:])
}

// keep is an opaque use of frame, which keeps reserveStack's frame on the
// stack. It does not read the frame.
//
//go:noinline
func keep([]byte) {}

// plain returns the record of a message without an assertion, at the
// innermost frame of the caller's code.
func plain(format string, args []any) assert.Failure {
	var pcs [maxFrames]uintptr
	n := runtime.Callers(1, pcs[:])
	return assert.Failure{Contract: fmt.Sprintf(format, args...), Where: matcher.CallerWhere(pcs[:n])}
}
