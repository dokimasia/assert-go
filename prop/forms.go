// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"context"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
)

// The ids of the property forms, which their records state.
const (
	equalID               = "prop-equal"
	notEqualID            = "prop-not-equal"
	trueID                = "prop-true"
	falseID               = "prop-false"
	nilID                 = "prop-nil"
	notNilID              = "prop-not-nil"
	lengthID              = "prop-length"
	emptyID               = "prop-empty"
	notEmptyID            = "prop-not-empty"
	containsID            = "prop-contains"
	notContainsID         = "prop-not-contains"
	containsInOrderID     = "prop-contains-in-order"
	permutationID         = "prop-permutation"
	hasPrefixID           = "prop-has-prefix"
	hasSuffixID           = "prop-has-suffix"
	matchesID             = "prop-matches"
	closeToID             = "prop-close-to"
	inRangeID             = "prop-in-range"
	pairwiseID            = "prop-pairwise"
	errAbsentID           = "prop-err-absent"
	errPresentID          = "prop-err-present"
	errIsID               = "prop-err-is"
	errIsNotID            = "prop-err-is-not"
	errAsID               = "prop-err-as"
	throwsID              = "prop-throws"
	notThrowsID           = "prop-not-throws"
	pureID                = "prop-pure"
	notPureID             = "prop-not-pure"
	nilContextSafeID      = "prop-nil-context-safe"
	honoursCancellationID = "prop-honours-cancellation"
	honoursDeadlineID     = "prop-honours-deadline"
	maxAllocsID           = "prop-max-allocs"
	idempotentID          = "prop-idempotent"
	accumulatesID         = "prop-accumulates"
	deterministicID       = "prop-deterministic"
	commutativeID         = "prop-commutative"
	associativeID         = "prop-associative"
	roundTripID           = "prop-round-trip"
)

// formIDs are the ids of every property form.
var formIDs = [...]string{
	equalID, notEqualID, trueID, falseID, nilID, notNilID, lengthID, emptyID, notEmptyID, containsID,
	notContainsID, containsInOrderID, permutationID, hasPrefixID, hasSuffixID, matchesID, closeToID, inRangeID,
	pairwiseID, errAbsentID, errPresentID, errIsID, errIsNotID, errAsID, throwsID, notThrowsID, pureID,
	notPureID, nilContextSafeID, honoursCancellationID, honoursDeadlineID, maxAllocsID, idempotentID,
	accumulatesID, deterministicID, commutativeID, associativeID, roundTripID,
}

// Equal runs [assert.Equal] on got(input) and want(input) for each input
// that the run generates. It is the property form prop-equal, and takes
// the relaxations of equal.
func Equal[T, U any](tb assert.TB, got, want func(T) U, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.Equal", id: equalID, relaxed: true, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, relaxations []matcher.Option) {
			matcher.Equal(c, matcher.Fatal, got(in[0]), want(in[0]), msg, relaxations...)
		})
}

// NotEqual runs [assert.NotEqual] on got(input) and want(input) for each
// input that the run generates. It is the property form prop-not-equal,
// and takes the relaxations of not-equal.
func NotEqual[T, U any](tb assert.TB, got, want func(T) U, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.NotEqual", id: notEqualID, relaxed: true, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, relaxations []matcher.Option) {
			matcher.NotEqual(c, matcher.Fatal, got(in[0]), want(in[0]), msg, relaxations...)
		})
}

// True runs [assert.True] on cond(input) for each input that the run
// generates. It is the property form prop-true.
func True[T any](tb assert.TB, cond func(T) bool, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.True", id: trueID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.True(c, matcher.Fatal, cond(in[0]), msg)
		})
}

// False runs [assert.False] on cond(input) for each input that the run
// generates. It is the property form prop-false.
func False[T any](tb assert.TB, cond func(T) bool, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.False", id: falseID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.False(c, matcher.Fatal, cond(in[0]), msg)
		})
}

// Nil runs [assert.Nil] on got(input) for each input that the run
// generates. It is the property form prop-nil.
func Nil[T, U any](tb assert.TB, got func(T) U, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.Nil", id: nilID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.Nil(c, matcher.Fatal, got(in[0]), msg)
		})
}

// NotNil runs [assert.NotNil] on got(input) for each input that the run
// generates. It is the property form prop-not-nil.
func NotNil[T, U any](tb assert.TB, got func(T) U, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.NotNil", id: notNilID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.NotNil(c, matcher.Fatal, got(in[0]), msg)
		})
}

// Length runs [assert.Length] on got(input) and want for each input that
// the run generates. It is the property form prop-length.
func Length[T, U any](tb assert.TB, got func(T) U, want int, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.Length", id: lengthID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.Length(c, matcher.Fatal, got(in[0]), want, msg)
		})
}

// Empty runs [assert.Empty] on got(input) for each input that the run
// generates. It is the property form prop-empty.
func Empty[T, U any](tb assert.TB, got func(T) U, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.Empty", id: emptyID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.Empty(c, matcher.Fatal, got(in[0]), msg)
		})
}

// NotEmpty runs [assert.NotEmpty] on got(input) for each input that the
// run generates. It is the property form prop-not-empty.
func NotEmpty[T, U any](tb assert.TB, got func(T) U, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.NotEmpty", id: notEmptyID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.NotEmpty(c, matcher.Fatal, got(in[0]), msg)
		})
}

// Contains runs [assert.Contains] on got(input) and needle for each input
// that the run generates. It is the property form prop-contains, and takes
// the relaxations of contains.
func Contains[T, U any](tb assert.TB, got func(T) U, needle any, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.Contains", id: containsID, relaxed: true, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, relaxations []matcher.Option) {
			matcher.Contains(c, matcher.Fatal, got(in[0]), needle, msg, relaxations...)
		})
}

// NotContains runs [assert.NotContains] on got(input) and needle for each
// input that the run generates. It is the property form prop-not-contains,
// and takes the relaxations of not-contains.
func NotContains[T, U any](tb assert.TB, got func(T) U, needle any, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(
		tb,
		form{op: "prop.NotContains", id: notContainsID, relaxed: true, labels: inputLabel},
		msg,
		caller(),
		opts,
		func(c *Case, in []T, relaxations []matcher.Option) {
			matcher.NotContains(c, matcher.Fatal, got(in[0]), needle, msg, relaxations...)
		},
	)
}

// ContainsInOrder runs [assert.ContainsInOrder] on got(input) and needles
// for each input that the run generates. It is the property form
// prop-contains-in-order.
func ContainsInOrder[T, U any](tb assert.TB, got func(T) U, needles []string, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.ContainsInOrder", id: containsInOrderID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.ContainsInOrder(c, matcher.Fatal, got(in[0]), needles, msg)
		})
}

// IsPermutation runs [assert.Permutation] on got(input) and want(input) for
// each input that the run generates. It is the property form
// prop-permutation, and takes the relaxations of permutation. Its name
// differs from the assertion's, because [Permutation] names a generator.
func IsPermutation[T, E any](tb assert.TB, got, want func(T) []E, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(
		tb,
		form{op: "prop.IsPermutation", id: permutationID, relaxed: true, labels: inputLabel},
		msg,
		caller(),
		opts,
		func(c *Case, in []T, relaxations []matcher.Option) {
			matcher.Permutation(c, matcher.Fatal, got(in[0]), want(in[0]), msg, relaxations...)
		},
	)
}

// HasPrefix runs [assert.HasPrefix] on got(input) and prefix for each
// input that the run generates. It is the property form prop-has-prefix.
func HasPrefix[T, U any](tb assert.TB, got func(T) U, prefix, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.HasPrefix", id: hasPrefixID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.HasPrefix(c, matcher.Fatal, got(in[0]), prefix, msg)
		})
}

// HasSuffix runs [assert.HasSuffix] on got(input) and suffix for each
// input that the run generates. It is the property form prop-has-suffix.
func HasSuffix[T, U any](tb assert.TB, got func(T) U, suffix, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.HasSuffix", id: hasSuffixID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.HasSuffix(c, matcher.Fatal, got(in[0]), suffix, msg)
		})
}

// Matches runs [assert.Matches] on got(input) and pattern for each input
// that the run generates. It is the property form prop-matches.
func Matches[T, U any](tb assert.TB, got func(T) U, pattern, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.Matches", id: matchesID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.Matches(c, matcher.Fatal, got(in[0]), pattern, msg)
		})
}

// CloseTo runs [assert.CloseTo] on got(input), want and tolerance for each
// input that the run generates. It is the property form prop-close-to.
func CloseTo[T, U any](tb assert.TB, got func(T) U, want, tolerance float64, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.CloseTo", id: closeToID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.CloseTo(c, matcher.Fatal, got(in[0]), want, tolerance, msg)
		})
}

// InRange runs [assert.InRange] on got(input), low and high for each input
// that the run generates. It is the property form prop-in-range.
func InRange[T, U any](tb assert.TB, got func(T) U, low, high float64, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.InRange", id: inRangeID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.InRange(c, matcher.Fatal, got(in[0]), low, high, msg)
		})
}

// Pairwise runs [assert.Pairwise] on got(input) and pred for each input
// that the run generates. It is the property form prop-pairwise.
func Pairwise[T, E any](tb assert.TB, got func(T) []E, pred func(earlier, later E) bool, msg string,
	opts ...FormOption,
) {
	tb.Helper()
	runForm(tb, form{op: "prop.Pairwise", id: pairwiseID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.Pairwise(c, matcher.Fatal, got(in[0]), pred, msg)
		})
}

// NoError runs [assert.NoError] on fn(input) for each input that the run
// generates. It is the property form prop-err-absent.
func NoError[T any](tb assert.TB, fn func(T) error, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.NoError", id: errAbsentID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.NoError(c, matcher.Fatal, fn(in[0]), msg)
		})
}

// HasError runs [assert.HasError] on fn(input) for each input that the run
// generates. It is the property form prop-err-present.
func HasError[T any](tb assert.TB, fn func(T) error, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.HasError", id: errPresentID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.HasError(c, matcher.Fatal, fn(in[0]), msg)
		})
}

// ErrorIs runs [assert.ErrorIs] on fn(input) and target for each input that
// the run generates. It is the property form prop-err-is.
func ErrorIs[T any](tb assert.TB, fn func(T) error, target error, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.ErrorIs", id: errIsID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.ErrorIs(c, matcher.Fatal, fn(in[0]), target, msg)
		})
}

// ErrorIsNot runs [assert.ErrorIsNot] on fn(input) and target for each
// input that the run generates. It is the property form prop-err-is-not.
func ErrorIsNot[T any](tb assert.TB, fn func(T) error, target error, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.ErrorIsNot", id: errIsNotID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.ErrorIsNot(c, matcher.Fatal, fn(in[0]), target, msg)
		})
}

// ErrorAs runs [assert.ErrorAs] of the error type E on fn(input) for each
// input that the run generates. It is the property form prop-err-as. A
// caller states E, as for assert.ErrorAs: prop.ErrorAs[*fs.PathError](t,
// open, msg).
func ErrorAs[E, T any](tb assert.TB, fn func(T) error, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.ErrorAs", id: errAsID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.ErrorAs[E](c, matcher.Fatal, fn(in[0]), msg)
		})
}

// Panics runs [assert.Panics] on a call of fn with input for each input
// that the run generates. It is the property form prop-throws.
func Panics[T any](tb assert.TB, fn func(T), msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.Panics", id: throwsID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.Panics(c, matcher.Fatal, func() { fn(in[0]) }, msg)
		})
}

// NotPanics runs [assert.NotPanics] on a call of fn with input for each
// input that the run generates. It is the property form prop-not-throws.
func NotPanics[T any](tb assert.TB, fn func(T), msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.NotPanics", id: notThrowsID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.NotPanics(c, matcher.Fatal, func() { fn(in[0]) }, msg)
		})
}

// Pure runs [assert.Pure] on observe and a call of fn with input for each
// input that the run generates. It is the property form prop-pure, and
// takes the relaxations of pure.
func Pure[T, S any](tb assert.TB, observe func() S, fn func(T), msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.Pure", id: pureID, relaxed: true, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, relaxations []matcher.Option) {
			matcher.Pure(c, matcher.Fatal, observe, func() { fn(in[0]) }, msg, relaxations...)
		})
}

// NotPure runs [assert.NotPure] on observe and a call of fn with input for
// each input that the run generates. It is the property form prop-not-pure,
// and takes the relaxations of not-pure.
func NotPure[T, S any](tb assert.TB, observe func() S, fn func(T), msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.NotPure", id: notPureID, relaxed: true, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, relaxations []matcher.Option) {
			matcher.NotPure(c, matcher.Fatal, observe, func() { fn(in[0]) }, msg, relaxations...)
		})
}

// NilContextSafe runs [assert.NilContextSafe] on fn with input for each
// input that the run generates. It is the property form
// prop-nil-context-safe.
func NilContextSafe[T any](tb assert.TB, fn func(ctx context.Context, in T) error, msg string,
	opts ...FormOption,
) {
	tb.Helper()
	runForm(tb, form{op: "prop.NilContextSafe", id: nilContextSafeID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.NilContextSafe(c, matcher.Fatal, func(ctx context.Context) error { return fn(ctx, in[0]) }, msg)
		})
}

// HonoursCancellation runs [assert.HonoursCancellation] on fn with input
// for each input that the run generates. It is the property form
// prop-honours-cancellation.
func HonoursCancellation[T any](tb assert.TB, fn func(ctx context.Context, in T) error, msg string,
	opts ...FormOption,
) {
	tb.Helper()
	runForm(
		tb,
		form{op: "prop.HonoursCancellation", id: honoursCancellationID, labels: inputLabel},
		msg,
		caller(),
		opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.HonoursCancellation(c, matcher.Fatal, func(ctx context.Context) error { return fn(ctx, in[0]) },
				msg)
		},
	)
}

// HonoursDeadline runs [assert.HonoursDeadline] on fn with input for each
// input that the run generates. It is the property form
// prop-honours-deadline.
func HonoursDeadline[T any](tb assert.TB, fn func(ctx context.Context, in T) error, msg string,
	opts ...FormOption,
) {
	tb.Helper()
	runForm(tb, form{op: "prop.HonoursDeadline", id: honoursDeadlineID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.HonoursDeadline(c, matcher.Fatal, func(ctx context.Context) error { return fn(ctx, in[0]) }, msg)
		})
}

// MaxAllocs runs [assert.MaxAllocs] on a call of fn with input and ceiling
// for each input that the run generates, and shrinks to the smallest input
// that allocates more. It is the property form prop-max-allocs. The count
// covers the whole process, so the cases run one at a time whatever
// [Workers] states, and the test that calls it does not run in parallel.
// It checks no ceiling in the builds where assert.MaxAllocs checks none.
func MaxAllocs[T any](tb assert.TB, fn func(T), ceiling uint64, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(
		tb,
		form{op: "prop.MaxAllocs", id: maxAllocsID, serial: true, labels: inputLabel},
		msg,
		caller(),
		opts,
		func(c *Case, in []T, _ []matcher.Option) {
			matcher.MaxAllocs(c, matcher.Fatal, func() { fn(in[0]) }, ceiling, msg)
		},
	)
}

// Idempotent runs [assert.Idempotent] on call, input and observe for each
// input that the run generates. It is the property form prop-idempotent,
// and takes the relaxations of idempotent.
func Idempotent[I, S any](tb assert.TB, call func(I) error, observe func() S, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(
		tb,
		form{op: "prop.Idempotent", id: idempotentID, relaxed: true, labels: inputLabel},
		msg,
		caller(),
		opts,
		func(c *Case, in []I, relaxations []matcher.Option) {
			matcher.Idempotent(c, matcher.Fatal, call, in[0], observe, msg, relaxations...)
		},
	)
}

// Accumulates runs [assert.Accumulates] on call, input and observe for
// each input that the run generates. It is the property form
// prop-accumulates.
func Accumulates[I any](tb assert.TB, call func(I) error, observe func() int, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(tb, form{op: "prop.Accumulates", id: accumulatesID, labels: inputLabel}, msg, caller(), opts,
		func(c *Case, in []I, _ []matcher.Option) {
			matcher.Accumulates(c, matcher.Fatal, call, in[0], observe, msg)
		})
}

// Deterministic runs [assert.Deterministic] on call and input for each
// input that the run generates. It is the property form
// prop-deterministic, and takes the relaxations of deterministic.
func Deterministic[I, O any](tb assert.TB, call func(I) (O, error), msg string, opts ...FormOption) {
	tb.Helper()
	runForm(
		tb,
		form{op: "prop.Deterministic", id: deterministicID, relaxed: true, labels: inputLabel},
		msg,
		caller(),
		opts,
		func(c *Case, in []I, relaxations []matcher.Option) {
			matcher.Deterministic(c, matcher.Fatal, call, in[0], msg, relaxations...)
		},
	)
}

// Commutative runs [assert.Commutative] on combine and the two inputs a and
// b that each case generates. It is the property form prop-commutative,
// and takes the relaxations of commutative.
func Commutative[T, R any](tb assert.TB, combine func(a, b T) R, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(
		tb,
		form{op: "prop.Commutative", id: commutativeID, relaxed: true, labels: pairLabels},
		msg,
		caller(),
		opts,
		func(c *Case, in []T, relaxations []matcher.Option) {
			matcher.Commutative(c, matcher.Fatal, combine, in[0], in[1], msg, relaxations...)
		},
	)
}

// Associative runs [assert.Associative] on combine and the three inputs a,
// b and c that each case generates. It is the property form
// prop-associative, and takes the relaxations of associative.
func Associative[T any](tb assert.TB, combine func(a, b T) T, msg string, opts ...FormOption) {
	tb.Helper()
	runForm(
		tb,
		form{op: "prop.Associative", id: associativeID, relaxed: true, labels: tripleLabels},
		msg,
		caller(),
		opts,
		func(c *Case, in []T, relaxations []matcher.Option) {
			matcher.Associative(c, matcher.Fatal, combine, in[0], in[1], in[2], msg, relaxations...)
		},
	)
}

// RoundTrip runs [assert.RoundTrip] on forward, inverse and input for each
// input that the run generates. It is the property form prop-round-trip,
// and takes the relaxations of round-trip.
func RoundTrip[I, E any](tb assert.TB, forward func(I) (E, error), inverse func(E) (I, error), msg string,
	opts ...FormOption,
) {
	tb.Helper()
	runForm(
		tb,
		form{op: "prop.RoundTrip", id: roundTripID, relaxed: true, labels: inputLabel},
		msg,
		caller(),
		opts,
		func(c *Case, in []I, relaxations []matcher.Option) {
			matcher.RoundTrip(c, matcher.Fatal, forward, inverse, in[0], msg, relaxations...)
		},
	)
}
