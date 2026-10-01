// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"
	"slices"
	"sync"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/tree"
)

// MaxChoices is the most choices that one case may make by default. A
// sequence counts as one choice plus one for each element.
const MaxChoices = 8192

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
// # Concurrency
//
// Every method is safe for concurrent use. A draw, a rejection and a
// fatal failure end the goroutine that makes them, so a body makes them on
// the goroutine it runs on.
type Case struct {
	// recorder keeps the body's failures in call order.
	recorder *assert.Recorder
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
	// requests are the requests behind the choices, in order.
	requests []request
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
	// labels are the labels the body classified the case under.
	labels map[string]struct{}
	// notes are the messages the body attached to the case.
	notes []string
	// fingerprints are the fingerprints the body observed, in order.
	fingerprints []uint64
	// stop is why the case ended before its body returned.
	stop stop
	// dropped reports whether the run no longer needs the case, which then
	// ends at its next choice.
	dropped bool
	// divergence is the tree's report of a diverged case.
	divergence *tree.DivergenceError
	// panicked is the identity of a panic that ended the body.
	panicked *Identity
	// recursion are the counts of base values of the recursive values
	// being decoded, a stack for each recursive generator.
	recursion map[any][]int
}

var (
	_ assert.TB       = (*Case)(nil)
	_ assert.Reporter = (*Case)(nil)
	_ assert.Clocked  = (*Case)(nil)
)

// newCase returns an empty case whose values come from p, capped at
// maxChoices, which walks w when it is not nil and reads clock.
func newCase(p provider, maxChoices int, w *tree.Walker, clock assert.Clock) *Case {
	return &Case{
		recorder:   assert.NewRecorder().WithGoexit().WithClock(clock),
		provider:   p,
		maxChoices: maxChoices,
		walker:     w,
		labels:     make(map[string]struct{}),
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
// the case compares.
func (c *Case) Observe(fingerprint uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fingerprints = append(c.fingerprints, fingerprint)
}

// Rand returns a source of random values whose every value is an integer
// choice over the whole unsigned 64-bit range.
func (c *Case) Rand() Source {
	return Source{c: c}
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

// choose returns the value for r and records it, with r. It ends the
// calling goroutine when the value takes the case past its cap, repeats a
// tested case or diverges from one.
func (c *Case) choose(r request) choice.Choice {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dropped {
		c.halt(cancelled)
	}
	value := c.provider.value(r, len(c.choices))
	c.cost += 1 + len(value.Sequence)
	if c.cost > c.maxChoices {
		c.halt(overrun)
	}
	c.choices = append(c.choices, value)
	c.requests = append(c.requests, r)
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
// err, with c.mu held: repeated for a choice that repeats a tested case,
// diverged for a request that differs from the recorded one, and running
// for a step that goes on. It keeps the divergence in c.divergence.
func (c *Case) walked(err error) stop {
	if errors.Is(err, tree.ErrRepeated) {
		return repeated
	}
	if errors.As(err, &c.divergence) {
		return diverged
	}
	return running
}

// integer returns a value choice in b.
func (c *Case) integer(b choice.IntegerBounds) choice.Int {
	return c.choose(request{bounds: choice.OfInteger(b)}).Integer
}

// reusable returns a value choice in b that the random phase may give an
// earlier value of the case with the same bounds.
func (c *Case) reusable(b choice.IntegerBounds) choice.Int {
	return c.choose(request{bounds: choice.OfInteger(b), reuse: true}).Integer
}

// structure returns a choice in b that decides structure, which the edge
// phase gives the value edge.
func (c *Case) structure(b choice.IntegerBounds, edge uint64) choice.Int {
	return c.choose(request{bounds: choice.OfInteger(b), structure: true, edge: choice.UintOf(edge)}).Integer
}

// coin returns a choice in [0, 1] that the random phase draws as a coin of
// num in den, and reports whether it is 1.
func (c *Case) coin(num, den uint64) bool {
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

// more returns the decision whether a collection of count elements under
// sizes gets another: a choice that decides structure, whose edge gives a
// collection one element.
func (c *Case) more(sizes choice.Sizes, count int) bool {
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

// openAt opens a span with c.mu held.
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

// position returns the index the next choice will have.
func (c *Case) position() int {
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
	c.spans, c.draws = c.spans[:at.spans], c.draws[:at.draws]
}

// recordAt returns the choices of the record as it stood after the first n
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

// leaves returns the count of base values of the recursive value that
// owner is decoding in the case, for the innermost decode in progress.
func (c *Case) leaves(owner any) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	stack := c.recursion[owner]
	return stack[len(stack)-1]
}

// addLeaf counts one more base value of owner's innermost recursive value
// in progress.
func (c *Case) addLeaf(owner any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recursion[owner][len(c.recursion[owner])-1]++
}

// enter starts a new count of base values for owner's next recursive
// value in the case, and returns the function that ends it.
func (c *Case) enter(owner any) func() {
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

// halt records why the case stopped and ends the calling goroutine, with
// c.mu held. The first reason is kept. The caller's deferred unlock runs
// as the goroutine ends.
func (c *Case) halt(s stop) {
	if c.stop == running {
		c.stop = s
	}
	runtime.Goexit()
}

// recoverPanic keeps the identity of a panic that ends the body. The
// runner defers it on the body's goroutine. A Goexit is no panic, and
// recover returns nil for it.
func (c *Case) recoverPanic() {
	v := recover()
	if v == nil {
		return
	}
	var pcs [maxFrames]uintptr
	n := runtime.Callers(1, pcs[:])
	identity := panicIdentity(v, pcs[:n])
	c.mu.Lock()
	defer c.mu.Unlock()
	c.panicked = &identity
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
	return s.c.integer(unsignedBounds).Magnitude()
}

// plain returns the record of a message without an assertion, at the
// innermost frame of the caller's code.
func plain(format string, args []any) assert.Failure {
	var pcs [maxFrames]uintptr
	n := runtime.Callers(1, pcs[:])
	return assert.Failure{Contract: fmt.Sprintf(format, args...), Where: callerWhere(pcs[:n])}
}
