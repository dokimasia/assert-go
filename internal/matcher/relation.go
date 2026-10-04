// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"cmp"
	"errors"
	"math/big"
)

// repetitions is how many times [Deterministic] calls its subject,
// [StableOrder] iterates its subject and [Poisoned] reads its subject. A
// subject whose results vary can agree with itself by chance. Over 32
// iterations on go1.27.1, a map of two entries kept one order in 1.5% of
// 20,000 runs, and a map of four or more entries in none.
const repetitions = 32

// attempt calls fn and returns its value, or as failure the error that fn
// returned or the value that fn panicked with.
//
// A goroutine that [runtime.Goexit] ends does not return from attempt.
// recover returns nil for it, and the goroutine goes on ending, so a
// callable that calls Fatalf on a test's seat ends the test.
func attempt[T any](fn func() (T, error)) (value T, failure any) {
	defer func() {
		if raised := recover(); raised != nil {
			failure = raised
		}
	}()

	value, err := fn()
	return value, err
}

// failedAt returns the record of a failure of the step at index i of a
// member that compares first and second: the failure is first for the
// first step and second for every later one.
func failedAt(i int, failure any) map[string]any {
	if i == 0 {
		return map[string]any{"first": failure, "second": nil}
	}
	return map[string]any{"first": nil, "second": failure}
}

// both runs first and then second, and returns their values, or the
// record of the first of them that fails.
func both[T any](first, second func() (T, error)) (a, b T, failed map[string]any) {
	a, failure := attempt(first)
	if failure != nil {
		return a, b, failedAt(0, failure)
	}
	b, failure = attempt(second)
	if failure != nil {
		return a, b, failedAt(1, failure)
	}
	return a, b, nil
}

// settle reports failed, the record of a step that failed, or a record of
// first and second when they differ as [Equal] compares them under opts.
func settle(seat Seat, mode Mode, assertion, msg string, first, second any, failed map[string]any, opts []Option) {
	seat.Helper()

	if failed != nil {
		Fail(seat, mode, assertion, msg, failed)
		return
	}
	if !equal(first, second, opts) {
		Fail(seat, mode, assertion, msg, map[string]any{"first": first, "second": second})
		return
	}
	Pass(seat, mode, assertion, msg)
}

// agree runs run repetitions times, and reports the first result that
// differs from the first result as [Equal] compares them under opts, or
// the first run that fails.
func agree[T any](seat Seat, mode Mode, assertion, msg string, run func() (T, error), opts []Option) {
	seat.Helper()

	var first T
	for i := range repetitions {
		next, failure := attempt(run)
		if failure != nil {
			Fail(seat, mode, assertion, msg, failedAt(i, failure))
			return
		}
		if i == 0 {
			first = next
			continue
		}
		if !equal(first, next, opts) {
			Fail(seat, mode, assertion, msg, map[string]any{"first": first, "second": next})
			return
		}
	}
	Pass(seat, mode, assertion, msg)
}

// Idempotent calls call with input twice, reads the state with observe
// after each call, and reports when the two readings differ.
//
// Use it for an operation that a caller may repeat, such as a retried
// write. first is the reading after one call, and second the reading
// after two. opts relax the comparison for this call alone.
//
//	matcher.Idempotent(seat, matcher.Fatal,
//	    func(item Item) error { return store.Put(ctx, item) }, item,
//	    func() []Item { return store.List(ctx) },
//	    "a repeated Put leaves the store as one Put left it")
//
// An error that call returns, or a panic of call or observe, fails the
// assertion. The failure takes the place of the reading that the call did
// not produce, and the other reading is nil.
//
// # Allocation contract
//
// A passing call with readings of one int allocates 24 times, in the
// comparison of the two readings as [Equal] compares them.
func Idempotent[I, S any](
	seat Seat, mode Mode, call func(I) error, input I, observe func() S, msg string, opts ...Option,
) {
	seat.Helper()

	step := func() (S, error) {
		if err := call(input); err != nil {
			var zero S
			return zero, err
		}
		return observe(), nil
	}
	first, second, failed := both(step, step)
	settle(seat, mode, "idempotent", msg, first, second, failed, opts)
}

// Accumulates reads an integer with observe, calls call with input twice,
// reads the integer after each call, and reports when the first call
// leaves the integer unchanged or the two calls change it by different
// amounts.
//
// first and second are the changes of the first and the second call. A
// change is an int, or a *big.Int when it is beyond the range of int. No
// three readings of int differ by two equal changes of that size, so such
// a change always fails.
//
// An error that call returns, or a panic of call or observe, fails the
// assertion. The failure takes the place of the change that the call did
// not produce, and the other change is nil.
//
// # Allocation contract
//
// A passing call allocates nothing for changes within the range of int.
func Accumulates[I any](seat Seat, mode Mode, call func(I) error, input I, observe func() int, msg string) {
	seat.Helper()

	var before, after int
	first, second, failed := both(
		func() (any, error) {
			before = observe()
			if err := call(input); err != nil {
				return nil, err
			}
			after = observe()
			return change(before, after), nil
		},
		func() (any, error) {
			if err := call(input); err != nil {
				return nil, err
			}
			return change(after, observe()), nil
		})
	if failed != nil {
		Fail(seat, mode, "accumulates", msg, failed)
		return
	}
	// Two ints compare by value. A *big.Int compares by identity, so a pair
	// that includes one is never equal, and such a change always fails.
	if after == before || first != second {
		Fail(seat, mode, "accumulates", msg, map[string]any{"first": first, "second": second})
		return
	}
	Pass(seat, mode, "accumulates", msg)
}

// change returns to minus from exactly: an int, or a *big.Int when the
// difference is beyond the range of int.
func change(from, to int) any {
	d := to - from
	// The subtraction overflows exactly when from and to differ in sign and
	// d differs in sign from to.
	if (to^from)&(to^d) < 0 {
		return new(big.Int).Sub(big.NewInt(int64(to)), big.NewInt(int64(from)))
	}
	return d
}

// Deterministic calls call with input 32 times, and reports the first
// result that differs from the first result.
//
// first is the first result, and second the first result that differs.
// opts relax the comparison for this call alone. An error that call
// returns, or a panic of call, fails the assertion: in first on the first
// call and in second on any later one, with the other nil.
//
// # Allocation contract
//
// A passing call with results of one int allocates 744 times, in the 31
// comparisons of a result with the first.
func Deterministic[I, O any](seat Seat, mode Mode, call func(I) (O, error), input I, msg string, opts ...Option) {
	seat.Helper()
	agree(seat, mode, "deterministic", msg, func() (O, error) { return call(input) }, opts)
}

// Commutative reports when combine(a, b) differs from combine(b, a).
//
// first is combine(a, b), and second is combine(b, a). opts relax the
// comparison for this call alone. A panic of combine fails the assertion,
// in the field of the result that it did not return, with the other nil.
//
// # Allocation contract
//
// A passing call with results of one int allocates 24 times, in the
// comparison of the two results.
func Commutative[T, R any](seat Seat, mode Mode, combine func(a, b T) R, a, b T, msg string, opts ...Option) {
	seat.Helper()

	first, second, failed := both(
		func() (R, error) { return combine(a, b), nil },
		func() (R, error) { return combine(b, a), nil })
	settle(seat, mode, "commutative", msg, first, second, failed, opts)
}

// Associative reports when combine(combine(a, b), c) differs from
// combine(a, combine(b, c)).
//
// first is the left grouping, and second the right one. opts relax the
// comparison for this call alone. A panic of combine fails the assertion,
// in the field of the grouping that it did not complete, with the other
// nil.
//
// # Allocation contract
//
// A passing call over ints allocates 24 times, in the comparison of the
// two groupings.
func Associative[T any](seat Seat, mode Mode, combine func(a, b T) T, a, b, c T, msg string, opts ...Option) {
	seat.Helper()

	first, second, failed := both(
		func() (T, error) { return combine(combine(a, b), c), nil },
		func() (T, error) { return combine(a, combine(b, c)), nil })
	settle(seat, mode, "associative", msg, first, second, failed, opts)
}

// RoundTrip converts input with forward, converts the result back with
// inverse, and reports when what comes back differs from input.
//
// want is input, and got what came back. opts relax the comparison for
// this call alone. An error that forward or inverse returns, or a panic of
// either, fails the assertion as got, with want nil.
//
// # Allocation contract
//
// A passing call on an int allocates 24 times, in the comparison of input
// with what came back.
func RoundTrip[I, E any](
	seat Seat, mode Mode, forward func(I) (E, error), inverse func(E) (I, error), input I, msg string,
	opts ...Option,
) {
	seat.Helper()

	got, failure := attempt(func() (I, error) {
		encoded, err := forward(input)
		if err != nil {
			var zero I
			return zero, err
		}
		return inverse(encoded)
	})
	if failure != nil {
		Fail(seat, mode, "round-trip", msg, map[string]any{"want": nil, "got": failure})
		return
	}
	if !equal(input, got, opts) {
		Fail(seat, mode, "round-trip", msg, map[string]any{"want": input, "got": got})
		return
	}
	Pass(seat, mode, "round-trip", msg)
}

// StableOrder calls iterate 32 times, and reports the first sequence that
// differs from the first sequence.
//
// first is the first sequence, and second the first sequence that
// differs. opts relax the comparison for this call alone. An error that
// iterate returns, or a panic of iterate, fails the assertion as
// [Deterministic] states for its call.
//
// # Allocation contract
//
// A passing call with sequences of three ints allocates 2,852 times, in the
// 31 comparisons of a sequence with the first.
func StableOrder[T any](seat Seat, mode Mode, iterate func() ([]T, error), msg string, opts ...Option) {
	seat.Helper()
	agree(seat, mode, "stable-order", msg, iterate, opts)
}

// NoDuplicates calls iterate once, and reports the first element that
// equals an earlier one.
//
// got is that element, and index its position. Each element is compared
// with every earlier one, which is n(n-1)/2 comparisons for n elements.
// opts relax the comparison for this call alone. An error that iterate
// returns, or a panic of iterate, fails the assertion as got, with index
// nil.
//
// # Allocation contract
//
// A passing call on three ints allocates 72 times, in its three
// comparisons.
func NoDuplicates[T any](seat Seat, mode Mode, iterate func() ([]T, error), msg string, opts ...Option) {
	seat.Helper()

	items, failure := attempt(iterate)
	if failure != nil {
		Fail(seat, mode, "no-duplicates", msg, map[string]any{"got": failure, "index": nil})
		return
	}
	for i := range items {
		for j := range i {
			if equal(items[j], items[i], opts) {
				Fail(seat, mode, "no-duplicates", msg, map[string]any{"got": items[i], "index": i})
				return
			}
		}
	}
	Pass(seat, mode, "no-duplicates", msg)
}

// Monotonic reads a value with observe, then calls advance and reads
// again, steps times, and reports the first reading that is below the
// reading before it or is NaN.
//
// index is the position of that reading, the reading before the first
// step at position 0. first is the reading before it, nil at position 0,
// and second that reading. A string orders as < orders it. A steps of 0
// or less advances nothing, so only the first reading is checked.
//
// An error that advance returns, or a panic of advance or observe, fails
// the assertion as second, with index and first nil.
//
// # Allocation contract
//
// A passing call allocates nothing besides what observe and advance
// allocate.
func Monotonic[N cmp.Ordered](seat Seat, mode Mode, observe func() N, advance func() error, steps int, msg string) {
	seat.Helper()

	failed := func(failure any) {
		Fail(seat, mode, "monotonic", msg, map[string]any{"index": nil, "first": nil, "second": failure})
	}
	previous, failure := attempt(func() (N, error) { return observe(), nil })
	if failure != nil {
		failed(failure)
		return
	}
	if isNaN(previous) {
		Fail(seat, mode, "monotonic", msg, map[string]any{"index": 0, "first": nil, "second": previous})
		return
	}
	for i := range steps {
		next, failure := attempt(func() (N, error) {
			if err := advance(); err != nil {
				var zero N
				return zero, err
			}
			return observe(), nil
		})
		if failure != nil {
			failed(failure)
			return
		}
		if isNaN(next) || next < previous {
			Fail(seat, mode, "monotonic", msg, map[string]any{"index": i + 1, "first": previous, "second": next})
			return
		}
		previous = next
	}
	Pass(seat, mode, "monotonic", msg)
}

// isNaN reports whether x is a NaN, the one value of an ordered type that
// differs from itself.
func isNaN[N cmp.Ordered](x N) bool {
	return x != x //nolint:gocritic // a NaN is the one value that differs from itself
}

// Total calls call with each element of domain, in order, and reports the
// first element for which it fails.
//
// index is the element's position, and got the error that call returned
// or the value that it panicked with. An empty domain passes.
//
// # Allocation contract
//
// A passing call allocates nothing besides what call allocates.
func Total[I any](seat Seat, mode Mode, call func(I) error, domain []I, msg string) {
	seat.Helper()

	for i, element := range domain {
		_, failure := attempt(func() (struct{}, error) { return struct{}{}, call(element) })
		if failure != nil {
			Fail(seat, mode, "total", msg, map[string]any{"index": i, "got": failure})
			return
		}
	}
	Pass(seat, mode, "total", msg)
}

// NotPure reads state with observe, calls fn, reads it again, and reports
// when the two readings are equal.
//
// It is the negation of [Pure], and takes the same arguments. got is the
// reading that did not change. opts relax the comparison for this call
// alone. A panic of observe or fn fails the assertion as got.
//
// # Allocation contract
//
// A passing call with readings of one int allocates 26 times, in the
// comparison of the two readings.
func NotPure[S any](seat Seat, mode Mode, observe func() S, fn func(), msg string, opts ...Option) {
	seat.Helper()

	var before, after S
	_, failure := attempt(func() (struct{}, error) {
		before = observe()
		fn()
		after = observe()
		return struct{}{}, nil
	})
	if failure != nil {
		Fail(seat, mode, "not-pure", msg, map[string]any{"got": failure})
		return
	}
	if equal(before, after, opts) {
		Fail(seat, mode, "not-pure", msg, map[string]any{"got": after})
		return
	}
	Pass(seat, mode, "not-pure", msg)
}

// FailsAfterClose calls closer, then call, and reports when call does not
// return an error that matches sentinel under [errors.Is].
//
// want is sentinel, and got what call returned. An error that closer
// returns, or a panic of closer or call, fails the assertion as got, with
// want nil.
//
// # Allocation contract
//
// A passing call allocates nothing besides what closer and call allocate.
func FailsAfterClose(seat Seat, mode Mode, closer, call func() error, sentinel error, msg string) {
	seat.Helper()

	got, failure := attempt(func() (error, error) {
		if err := closer(); err != nil {
			return nil, err
		}
		return call(), nil
	})
	if failure != nil {
		Fail(seat, mode, "after-close", msg, map[string]any{"want": nil, "got": failure})
		return
	}
	if !errors.Is(got, sentinel) {
		Fail(seat, mode, "after-close", msg, map[string]any{"want": sentinel, "got": got})
		return
	}
	Pass(seat, mode, "after-close", msg)
}

// Poisoned calls induce, then reads the subject with observe 32 times, and
// reports the first reading that returns no error.
//
// index is the position of that reading, and got what it returned. An
// error that induce causes is the caller's to ignore, so induce returns
// none. A panic of induce or observe fails the assertion as got, with
// index nil.
//
// # Allocation contract
//
// A passing call allocates nothing besides what induce and observe
// allocate.
func Poisoned(seat Seat, mode Mode, induce func(), observe func() error, msg string) {
	seat.Helper()

	_, failure := attempt(func() (struct{}, error) {
		induce()
		return struct{}{}, nil
	})
	if failure != nil {
		Fail(seat, mode, "poisoned", msg, map[string]any{"index": nil, "got": failure})
		return
	}
	for i := range repetitions {
		err, failure := attempt(func() (error, error) { return observe(), nil })
		if failure != nil {
			Fail(seat, mode, "poisoned", msg, map[string]any{"index": nil, "got": failure})
			return
		}
		if err == nil {
			Fail(seat, mode, "poisoned", msg, map[string]any{"index": i, "got": nil})
			return
		}
	}
	Pass(seat, mode, "poisoned", msg)
}
