// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"cmp"

	"go.dokimi.dev/assert/internal/matcher"
)

// Idempotent calls call with input twice, reads the state with observe
// after each call, and stops the test when the two readings differ.
//
//	assert.Idempotent(t,
//	    func(item Item) error { return store.Put(ctx, item) }, item,
//	    func() []Item { return store.List(ctx) },
//	    "a repeated Put leaves the store as one Put left it")
//
// first is the reading after one call, and second the reading after two.
// An error from call, or a panic of call or observe, fails the test in
// the field of the reading that the call did not produce, with the other
// nil. Return a copy from observe, as [Pure] requires.
//
// # Allocation contract
//
// A passing call with readings of one int allocates 24 times, in the
// comparison of the two readings as [Equal] compares them.
func Idempotent[I, S any](tb TB, call func(I) error, input I, observe func() S, msg string, opts ...Option) {
	tb.Helper()
	matcher.Idempotent(tb, matcher.Fatal, call, input, observe, msg, opts...)
}

// Accumulates reads an integer with observe, calls call with input twice,
// and stops the test when the first call leaves the integer unchanged or
// the two calls change it by different amounts.
//
//	assert.Accumulates(t,
//	    func(event Event) error { return log.Append(ctx, event) }, event,
//	    func() int { return log.Len() },
//	    "each Append adds one entry")
//
// first and second are the two changes: an int, or a *big.Int for a change
// beyond the range of int, which always fails. An error from call, or a
// panic of call or observe, fails the test in the field of the change that
// the call did not produce, with the other nil.
//
// # Allocation contract
//
// A passing call allocates nothing for changes within the range of int.
func Accumulates[I any](tb TB, call func(I) error, input I, observe func() int, msg string) {
	tb.Helper()
	matcher.Accumulates(tb, matcher.Fatal, call, input, observe, msg)
}

// Deterministic calls call with input 32 times, and stops the test at the
// first result that differs from the first result.
//
//	assert.Deterministic(t, func(v Value) ([]byte, error) { return Encode(v) }, value,
//	    "Encode returns the same bytes on every call")
//
// An error from call, or a panic of call, fails the test: in first on the
// first call and in second on any later one, with the other nil.
//
// # Allocation contract
//
// A passing call with results of one int allocates 744 times, in the 31
// comparisons of a result with the first.
func Deterministic[I, O any](tb TB, call func(I) (O, error), input I, msg string, opts ...Option) {
	tb.Helper()
	matcher.Deterministic(tb, matcher.Fatal, call, input, msg, opts...)
}

// Commutative stops the test when combine(a, b) differs from
// combine(b, a).
//
//	assert.Commutative(t, Counter.Merge, left, right, "Merge ignores the order of its operands")
//
// first is combine(a, b), and second combine(b, a). A panic of combine
// fails the test in the field of the result that it did not return, with
// the other nil.
//
// # Allocation contract
//
// A passing call with results of one int allocates 24 times, in the
// comparison of the two results.
func Commutative[T, R any](tb TB, combine func(a, b T) R, a, b T, msg string, opts ...Option) {
	tb.Helper()
	matcher.Commutative(tb, matcher.Fatal, combine, a, b, msg, opts...)
}

// Associative stops the test when combine(combine(a, b), c) differs from
// combine(a, combine(b, c)).
//
// first is the left grouping, and second the right one. A panic of
// combine fails the test in the field of the grouping that it did not
// complete, with the other nil.
//
// # Allocation contract
//
// A passing call over ints allocates 24 times, in the comparison of the
// two groupings.
func Associative[T any](tb TB, combine func(a, b T) T, a, b, c T, msg string, opts ...Option) {
	tb.Helper()
	matcher.Associative(tb, matcher.Fatal, combine, a, b, c, msg, opts...)
}

// RoundTrip converts input with forward and the result back with inverse,
// and stops the test when what comes back differs from input.
//
//	assert.RoundTrip(t, Encode, Decode, value, "Decode undoes Encode")
//
// want is input, and got what came back. An error from forward or inverse,
// or a panic of either, fails the test as got, with want nil.
//
// # Allocation contract
//
// A passing call on an int allocates 24 times, in the comparison of input
// with what came back.
func RoundTrip[I, E any](
	tb TB, forward func(I) (E, error), inverse func(E) (I, error), input I, msg string, opts ...Option,
) {
	tb.Helper()
	matcher.RoundTrip(tb, matcher.Fatal, forward, inverse, input, msg, opts...)
}

// StableOrder calls iterate 32 times, and stops the test at the first
// sequence that differs from the first sequence.
//
//	assert.StableOrder(t, func() ([]string, error) { return registry.Names(), nil },
//	    "Names lists the registry in one order")
//
// Thirty-two iterations pass a Go map of two entries in 1.5% of runs, so
// a test of an order built from a map uses a map of four entries or more.
// An error from iterate, or a panic of it, fails the test as
// [Deterministic] states for its call.
//
// # Allocation contract
//
// A passing call with sequences of three ints allocates 2,852 times, in
// the 31 comparisons of a sequence with the first.
func StableOrder[T any](tb TB, iterate func() ([]T, error), msg string, opts ...Option) {
	tb.Helper()
	matcher.StableOrder(tb, matcher.Fatal, iterate, msg, opts...)
}

// NoDuplicates calls iterate once, and stops the test at the first element
// that equals an earlier one.
//
//	assert.NoDuplicates(t, func() ([]Item, error) { return store.List(ctx) },
//	    "List returns each item once across its pages")
//
// got is that element, and index its position. Each element is compared
// with every earlier one, which is n(n-1)/2 comparisons for n elements. An
// error from iterate, or a panic of it, fails the test as got, with index
// nil.
//
// # Allocation contract
//
// A passing call on three ints allocates 72 times, in its three
// comparisons.
func NoDuplicates[T any](tb TB, iterate func() ([]T, error), msg string, opts ...Option) {
	tb.Helper()
	matcher.NoDuplicates(tb, matcher.Fatal, iterate, msg, opts...)
}

// Monotonic reads a value with observe, then calls advance and reads again,
// steps times, and stops the test at the first reading that is below the
// reading before it or is NaN.
//
//	assert.Monotonic(t, func() int64 { return clock.Now().UnixNano() },
//	    func() error { return clock.Tick() }, 100, "the clock never runs backwards")
//
// index is the position of that reading, the first reading at position 0.
// first is the reading before it, nil at position 0, and second that
// reading. A string orders as < orders it. A steps of 0 or less checks
// the first reading alone. An error from advance, or a panic of advance or
// observe, fails the test as second, with index and first nil.
//
// # Allocation contract
//
// A passing call allocates nothing besides what observe and advance
// allocate.
func Monotonic[N cmp.Ordered](tb TB, observe func() N, advance func() error, steps int, msg string) {
	tb.Helper()
	matcher.Monotonic(tb, matcher.Fatal, observe, advance, steps, msg)
}

// Total calls call with each element of domain, in order, and stops the
// test at the first element for which call returns an error or panics.
//
//	assert.Total(t, func(code int) error { _, err := Describe(code); return err },
//	    knownCodes, "Describe accepts every known code")
//
// index is the element's position, and got the error or the panic value.
// An empty domain passes.
//
// # Allocation contract
//
// A passing call allocates nothing besides what call allocates.
func Total[I any](tb TB, call func(I) error, domain []I, msg string) {
	tb.Helper()
	matcher.Total(tb, matcher.Fatal, call, domain, msg)
}

// NotPure reads state with observe, calls fn, reads it again, and stops
// the test when the two readings are equal.
//
// It is the negation of [Pure], and takes the same arguments. got is the
// reading that did not change. A panic of observe or fn fails the test as
// got.
//
// # Allocation contract
//
// A passing call with readings of one int allocates 26 times, in the
// comparison of the two readings.
func NotPure[S any](tb TB, observe func() S, fn func(), msg string, opts ...Option) {
	tb.Helper()
	matcher.NotPure(tb, matcher.Fatal, observe, fn, msg, opts...)
}

// FailsAfterClose calls closer, then call, and stops the test when call
// does not return an error that matches sentinel under [errors.Is].
//
//	assert.FailsAfterClose(t, f.Close,
//	    func() error { _, err := f.Read(buf); return err }, os.ErrClosed,
//	    "a closed file refuses a read")
//
// want is sentinel, and got what call returned. An error from closer, or a
// panic of closer or call, fails the test as got, with want nil.
//
// # Allocation contract
//
// A passing call allocates nothing besides what closer and call allocate.
func FailsAfterClose(tb TB, closer, call func() error, sentinel error, msg string) {
	tb.Helper()
	matcher.FailsAfterClose(tb, matcher.Fatal, closer, call, sentinel, msg)
}

// Poisoned calls induce, then reads the subject with observe 32 times, and
// stops the test at the first reading that returns no error.
//
//	assert.Poisoned(t, func() { disk.FailWrites() },
//	    func() error { return store.Put(ctx, item) },
//	    "a store that lost its disk refuses every write")
//
// index is the position of that reading, and got what it returned. induce
// returns no error, because the fault that it induces is often itself a
// failed call. A panic of induce or observe fails the test as got, with
// index nil.
//
// # Allocation contract
//
// A passing call allocates nothing besides what induce and observe
// allocate.
func Poisoned(tb TB, induce func(), observe func() error, msg string) {
	tb.Helper()
	matcher.Poisoned(tb, matcher.Fatal, induce, observe, msg)
}
