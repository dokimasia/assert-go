// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/prop"
)

// The contract of the test properties, the label of their draws, and the
// identities of their failures, as the definition's behaviour vectors
// name them.
const (
	// contract is the contract of every test property.
	contract = "the property holds"
	// drawn is the label of a body's draw.
	drawn = "value"
	// big is the identity of a failure at a large value.
	big = "big"
	// every is the identity of a failure at every value.
	every = "every"
	// always is the identity of a failure of every case.
	always = "always"
)

// The names of the detail fields of a failing run's record, which the
// definition pins.
const (
	outcomeField        = "outcome"
	casesField          = "cases"
	rejectedField       = "rejected"
	seedField           = "seed"
	counterexampleField = "counterexample"
	failureField        = "failure"
	choicesField        = "choices"
	othersField         = "others"
	divergenceField     = "divergence"
	coverageField       = "coverage"
)

// The labels of the coverage requirements of the definition's behaviour
// vectors.
const (
	// even counts the even values.
	even = "even"
	// small counts the values below 5.
	small = "small"
)

// The environment variables that set the defaults of every property of a
// test run, which the definition names.
const (
	seedVariable    = "DOKIMI_ASSERT_PROP_SEED"
	profileVariable = "DOKIMI_ASSERT_PROP_PROFILE"
	replayVariable  = "DOKIMI_ASSERT_PROP_REPLAY"
	budgetVariable  = "DOKIMI_ASSERT_PROP_BUDGET"
)

// contractOfForm is the contract of every test form.
const contractOfForm = "the form holds"

// The operations that name the faults of the package's functions.
const (
	forAllOp  = "prop.ForAll"
	shapeOfOp = "prop.ShapeOf"
	ofShapeOp = "prop.OfShape"
)

// duplicateReason is the reason of the fault of a second property of a test
// with the contract of the first in the same store.
const duplicateReason = `two properties of the test have the contract "the property holds", ` +
	"and would share their stored cases"

// The facts of the entries that the store tests write and read.
const (
	// fits is the contract of the record that a test body fails with.
	fits = "it fits"
	// fileMode is the mode of a file that a test writes into a store.
	fileMode = 0o644
)

// The dates of the entries that the store tests write and read.
var (
	// today is the time of the seats' controlled clocks.
	today = time.Date(2026, time.October, 1, 9, 30, 0, 0, time.UTC)
	// earlier is the date of an entry found before today.
	earlier = time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
)

// The types that the tests read into shapes.
type (
	// line is one line of an order.
	line struct {
		SKU string `json:"sku"`
		Qty int32  `json:"qty" prop:"min=1,max=99"`
	}
	// order is an order: its id, its lines and a note, a field that its tag
	// leaves out, and an unexported field.
	order struct {
		ID      uint32  `json:"id"`
		Lines   []line  `json:"lines"          prop:"max_size=3"`
		Note    *string `json:"note,omitempty" prop:"max_size=10"`
		Skipped string  `json:"skipped"        prop:"-"`
		hidden  int
	}
	// tree is a tree of integers, which refers to itself.
	tree struct {
		Value    int32  `json:"value"`
		Children []tree `json:"children"`
	}
	// ping and pong refer to each other.
	ping struct {
		Next *pong `json:"next"`
	}
	pong struct {
		Next ping `json:"next"`
	}
	// status is a string whose values RegisterValues states.
	status string
	// payment is an interface whose variants RegisterVariants states.
	payment interface{ isPayment() }
	// pending is the variant of payment without a payload.
	pending struct{}
	// paid is the variant of payment of an amount.
	paid int64
	// refunded is the variant of payment of a reason.
	refunded string
	// cancelled is the variant of payment that a pointer implements.
	cancelled struct {
		Reason string `json:"reason"`
	}
	// celsius is a type over a float.
	celsius float64
	// octet is a type over uint8, whose slices and arrays read as lists of
	// integers.
	octet uint8
)

func (pending) isPayment()    {}
func (paid) isPayment()       {}
func (refunded) isPayment()   {}
func (*cancelled) isPayment() {}

// The values of status.
const (
	statusPending status = "pending"
	statusPaid    status = "paid"
	statusShipped status = "shipped"
)

// The first value past the members of each enumeration of the package.
const (
	invalidDifference prop.Difference = 3
	invalidPart       prop.Part       = 6
	invalidRelevance  prop.Relevance  = 3
	invalidOutcome    prop.Outcome    = 6
	invalidVerdict    prop.Verdict    = 2
)

// init registers the values of status and the variants of payment in the
// test process's registry, before any property runs.
func init() {
	prop.RegisterValues(statusPending, statusPaid, statusShipped)
	prop.RegisterVariants[payment](pending{}, paid(0), refunded(""), &cancelled{})
}

// testSeat is a seat of internal/matchertest with a test's name and
// cleanups, as testing.TB has them, and a clock that reads today. It takes
// each failure as its record and each fault as the error it is.
type testSeat struct {
	matchertest.Seat
	// name is the test's name.
	name string
	// clock is the seat's clock, which reads today.
	clock *assert.Controlled
	// mu guards cleanups.
	mu sync.Mutex
	// cleanups are the functions that end runs, in the order registered.
	cleanups []func()
}

// newTestSeat returns a seat of the test name whose clock reads today.
func newTestSeat(name string) *testSeat {
	return &testSeat{name: name, clock: assert.NewControlled(today)}
}

// Name returns the test's name.
func (s *testSeat) Name() string {
	return s.name
}

// Clock returns the seat's clock, which reads today.
func (s *testSeat) Clock() assert.Clock {
	return s.clock
}

// Cleanup registers f to run when the test ends.
func (s *testSeat) Cleanup(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanups = append(s.cleanups, f)
}

// end runs the registered functions, the last first, as a test's end does.
func (s *testSeat) end() {
	s.mu.Lock()
	cleanups := s.cleanups
	s.cleanups = nil
	s.mu.Unlock()
	for _, f := range slices.Backward(cleanups) {
		f()
	}
}

// sentences is a seat without a Report or a ReportFault method, as
// testing.TB is. It keeps the message of each failure, each fault and each
// log line, in call order.
type sentences struct {
	// mu guards messages.
	mu sync.Mutex
	// messages are the messages of the failures and the log, in call order.
	messages []string
}

// Helper does nothing: the seat states no location.
func (*sentences) Helper() {}

// Fatalf keeps the message and returns.
func (s *sentences) Fatalf(format string, args ...any) {
	s.keep(format, args)
}

// Errorf keeps the message.
func (s *sentences) Errorf(format string, args ...any) {
	s.keep(format, args)
}

// Logf keeps the message.
func (s *sentences) Logf(format string, args ...any) {
	s.keep(format, args)
}

// keep adds the formatted message to the messages.
func (s *sentences) keep(format string, args []any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, fmt.Sprintf(format, args...))
}

// all returns a copy of the messages.
func (s *sentences) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.messages...)
}

// detailOf runs body as the property contract on a recorder under opts, and
// returns the detail of the one record that the run reported, or nil for a
// run that reported none.
func detailOf(body func(*prop.Case), opts ...prop.Option) map[string]any {
	rec := assert.NewRecorder()
	prop.ForAll(rec, contract, body, opts...)
	records := rec.Failures()
	if len(records) == 0 {
		return nil
	}
	return records[0].Detail
}

// callsOf runs body as the property contract on a recorder under opts, and
// returns the JSON object of each call record that the recorder keeps, the
// property's own first.
func callsOf(t *testing.T, body func(*prop.Case), opts ...prop.Option) []map[string]any {
	t.Helper()
	rec := assert.NewRecorder()
	prop.ForAll(rec, contract, body, opts...)
	return decodedCalls(t, rec.Records())
}

// decodedCalls returns the JSON object of each call record of lines.
func decodedCalls(t *testing.T, lines []string) []map[string]any {
	t.Helper()
	out := make([]map[string]any, len(lines))
	for i, line := range lines {
		assert.NoError(t, json.Unmarshal([]byte(line), &out[i]), "the call record is JSON")
	}
	return out
}

// recordedDetail runs body as the property contract on a recorder under
// opts, and returns the JSON object of the detail that the property's call
// record states.
func recordedDetail(t *testing.T, body func(*prop.Case), opts ...prop.Option) map[string]any {
	t.Helper()
	detail, _ := callsOf(t, body, opts...)[0]["detail"].(map[string]any)
	return detail
}

// counts returns the outcome and the counts of valid and rejected cases of
// a record's detail.
func counts(detail map[string]any) []any {
	return []any{detail[outcomeField], detail[casesField], detail[rejectedField]}
}

// draws returns the body that draws one value of g and never fails.
func draws[T any](g prop.Generator[T]) func(*prop.Case) {
	return func(c *prop.Case) { c.Draw(g, drawn) }
}

// failsAtLeast returns the body that draws an integer in [0, most], and
// fails with identity at a value of least or more.
func failsAtLeast(most, least int, identity string) func(*prop.Case) {
	g := prop.Integer(0, most)
	return func(c *prop.Case) {
		if c.Draw(g, drawn) >= least {
			fail(c, identity)
		}
	}
}

// fail ends the case with an aborting record of identity, without a
// location, as a body of the definition's vectors fails.
func fail(c *prop.Case, identity string) {
	c.Report(assert.Failure{Assertion: identity}, true)
}

// failsFrom returns the body that draws an integer in [0, 10000], and ends
// the case at a value of least or more with a record whose identity an
// entry keeps: the assertion big and the contract fits.
func failsFrom(least int) func(*prop.Case) {
	g := prop.Integer(0, 10000)
	return func(c *prop.Case) {
		if c.Draw(g, drawn) >= least {
			c.Report(assert.Failure{Assertion: big, Contract: fits}, true)
		}
	}
}

// detached is the body that draws a digit and fails with a message from a
// goroutine without a frame of the caller's code, a failure that no entry
// can keep.
func detached(c *prop.Case) {
	c.Draw(prop.Integer(0, 9), drawn)
	go c.Errorf("detached")
	for len((*engine.Case)(c).Failures()) == 0 {
		runtime.Gosched()
	}
}

// classifies returns the body that draws an integer in [0, most], and
// classifies the case under label when matches reports true for the value.
func classifies(most int, label string, matches func(int) bool) func(*prop.Case) {
	g := prop.Integer(0, most)
	return func(c *prop.Case) {
		if matches(c.Draw(g, drawn)) {
			c.Classify(label)
		}
	}
}

// isEven reports whether v is even.
func isEven(v int) bool {
	return v%2 == 0
}

// same returns x.
func same(x int8) int8 {
	return x
}

// next returns x plus one.
func next(x int8) int8 {
	return x + 1
}

// subtract returns a minus b.
func subtract(a, b int8) int8 {
	return a - b
}

// diverges returns the body that draws a value of first on its first call
// and a value of then on every later call.
func diverges[T, U any](first prop.Generator[T], then prop.Generator[U]) func(*prop.Case) {
	var calls int
	return func(c *prop.Case) {
		calls++
		if calls == 1 {
			c.Draw(first, drawn)
			return
		}
		c.Draw(then, drawn)
	}
}

// placed returns the body that runs body at place p of a machine's steps,
// as the steps of a machine set it.
func placed(p engine.Place, body func(*prop.Case)) func(*prop.Case) {
	return func(c *prop.Case) {
		(*engine.Case)(c).SetPlace(p)
		body(c)
	}
}

// once returns the body that calls failing on its first call and passes on
// every later call, so the replay of its failing case passes.
func once(failing func(*prop.Case)) func(*prop.Case) {
	var calls int
	return func(c *prop.Case) {
		calls++
		if calls == 1 {
			failing(c)
		}
	}
}

// here stores the file and the line of its caller in at and returns the
// empty string, so that a call inside another call's arguments states
// where that call is.
func here(at *assert.Where) string {
	_, at.File, at.Line, _ = runtime.Caller(1)
	return ""
}

// decoded returns the value that g decodes in a run that replays one
// integer choice for each value, and the run's outcome.
func decoded[T any](g prop.Generator[T], values ...uint64) (T, prop.Outcome) {
	var got T
	detail := replayed(func(c *prop.Case) { got = c.Draw(g, drawn) }, values...)
	if detail == nil {
		return got, prop.Passed
	}
	outcome, _ := detail[outcomeField].(prop.Outcome)
	return got, outcome
}

// first returns the value that g decodes in a run that replays one integer
// choice for each value.
func first[T any](g prop.Generator[T], values ...uint64) T {
	got, _ := decoded(g, values...)
	return got
}

// replayed runs body on a recorder, replaying the case of one integer
// choice for each value, and returns the detail of the run's record, or
// nil for a run that passed.
func replayed(body func(*prop.Case), values ...uint64) map[string]any {
	return detailOf(body, prop.Replay(tokenOf(values...)))
}

// tokenOf returns the replay token of one integer choice for each value.
func tokenOf(values ...uint64) string {
	choices := make([]choice.Choice, len(values))
	for i, v := range values {
		choices[i] = integer(v)
	}
	return token.Encode(choices)
}

// replayedOf returns the value that g decodes in a run that replays
// choices.
func replayedOf[T any](g prop.Generator[T], choices ...choice.Choice) T {
	var got T
	prop.ForAll(assert.NewRecorder(), contract, func(c *prop.Case) { got = c.Draw(g, drawn) },
		prop.Replay(token.Encode(choices)))
	return got
}

// decodedBy returns the value that g decodes from a case that replays
// choices.
func decodedBy[T any](g prop.Generator[T], choices ...choice.Choice) T {
	var got T
	engine.Replay(func(c *engine.Case) { got = engine.Draw(c, engine.Generator[T](g), drawn) }, choices, nil)
	return got
}

// refusal returns the error of the inverse of the generator of T for v.
func refusal[T any](v T) error {
	_, err := engine.Invert(engine.Generator[T](prop.Of[T]()), v)
	return err
}

// shapeRefusal returns the error of the inverse of the generator of the
// shape file text for v.
func shapeRefusal(tb testing.TB, text string, v any) error {
	tb.Helper()
	g, err := prop.OfShape(text)
	assert.NoError(tb, err, "the shape reads")
	_, err = engine.Invert(engine.Generator[any](g), v)
	return err
}

// integer returns the integer choice of v.
func integer(v uint64) choice.Choice {
	return choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(v)}
}

// signed returns the integer choice of v.
func signed(v int64) choice.Choice {
	return choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(v)}
}

// floating returns the float choice of v.
func floating(v float64) choice.Choice {
	return choice.Choice{Kind: choice.Float, Float: v}
}

// sequence returns the sequence choice of elements.
func sequence(elements ...uint32) choice.Choice {
	return choice.Choice{Kind: choice.Sequence, Sequence: elements}
}

// shapeTree returns the JSON value of the shape file that read returns,
// without its source, failing the test when the type does not read.
func shapeTree(tb testing.TB, read func() (string, error)) any {
	tb.Helper()
	text, err := read()
	assert.NoError(tb, err, "the type reads")
	var tree map[string]any
	assert.NoError(tb, json.Unmarshal([]byte(text), &tree), "the shape file is JSON")
	assert.Contains(tb, tree, "source", "the shape file names its source")
	delete(tree, "source")
	return tree
}

// jsonTree returns the JSON value of text.
func jsonTree(tb testing.TB, text string) any {
	tb.Helper()
	var v any
	assert.NoError(tb, json.Unmarshal([]byte(text), &v), "the expected value is JSON")
	return v
}

// fieldV returns the path of the field V of the struct whose field V is of
// the type typ and states the prop tag tag, as the reader names the field.
func fieldV(typ, tag string) fault.Path {
	return fault.Path{fault.Field(fmt.Sprintf("struct { V %s %q }", typ, `prop:"`+tag+`"`)), fault.Field("V")}
}

// expectFault checks that err is a fault of the operation, the path, the
// kind and the reason that want states. It leaves the cause unchecked.
func expectFault(tb testing.TB, err error, want fault.Error) {
	tb.Helper()
	got, ok := errors.AsType[*fault.Error](err)
	assert.True(tb, ok, "the error is a fault")
	assert.Equal(tb, fault.Error{Op: got.Op, Path: got.Path, Kind: got.Kind, Reason: got.Reason}, want,
		"the operation, the path, the kind and the reason of the fault")
}

// expectOnlyFault checks that faults contains one fault, of the operation,
// the path, the kind and the reason that want states.
func expectOnlyFault(tb testing.TB, faults []error, want fault.Error) {
	tb.Helper()
	assert.Length(tb, faults, 1, "one fault")
	expectFault(tb, faults[0], want)
}

// laterFault returns the fault of op at the field store of the file name in
// the store dir, an entry of format, which a run skips.
func laterFault(op, dir, name, format string) fault.Error {
	return fault.Error{
		Op:     op,
		Path:   fault.Path{fault.Field(dir), fault.Field(name), fault.Field("store")},
		Kind:   store.ErrLater,
		Reason: "the entry is of format " + format,
	}
}

// profileFault returns the fault of op for the profile nightly, which is
// neither default nor ci.
func profileFault(op string) fault.Error {
	return fault.Error{
		Op:     op,
		Path:   fault.Path{fault.Field(profileVariable)},
		Reason: `"nightly" names none of the default, ci and campaign profiles`,
	}
}

// loaded returns the entries of the property contract in the store dir.
func loaded(t *testing.T, dir string) store.Stored {
	t.Helper()
	stored, err := store.Load(dir, contract)
	assert.NoError(t, err, "the store reads")
	return stored
}

// save writes e to the store dir.
func save(t *testing.T, dir string, e store.Entry) {
	t.Helper()
	written, err := store.Save(dir, e)
	assert.NoError(t, err, "the entry is saved")
	assert.True(t, written, "the entry's file is new")
}

// write writes content to the file at path.
func write(t *testing.T, path, content string) {
	t.Helper()
	assert.NoError(t, os.WriteFile(path, []byte(content), fileMode), "the file is written")
}
