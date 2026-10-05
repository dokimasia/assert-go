// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/prop"
)

// maxAllocsFormAllocs are the allocations of a passing run of 100 cases of
// MaxAllocs, measured: the run's, and the count of each case.
const maxAllocsFormAllocs = 834

// formAllocs are the allocations of a passing run of 100 cases of each
// property form of formCases, measured: the run's own, and what the form's
// assertion and its subjects allocate in each case.
var formAllocs = map[string]uint64{
	"Equal": 833, "NotEqual": 833, "True": 833, "False": 833, "Nil": 833, "NotNil": 933, "Length": 933,
	"Empty": 1912, "NotEmpty": 2111, "Contains": 2211, "NotContains": 1912, "ContainsInOrder": 1262,
	"IsPermutation": 2542, "HasPrefix": 1261, "HasSuffix": 1261, "Matches": 3961, "CloseTo": 932,
	"InRange": 932, "Pairwise": 2344, "NoError": 833, "HasError": 833, "ErrorIs": 833, "ErrorIsNot": 833,
	"ErrorAs": 1033, "Panics": 833, "NotPanics": 833, "Pure": 833, "NotPure": 836, "NilContextSafe": 833,
	"HonoursCancellation": 1033, "HonoursDeadline": 1033, "Idempotent": 836, "Accumulates": 836,
	"Deterministic": 833, "Commutative": 855, "Associative": 1078, "RoundTrip": 1036,
}

// errSentinel is the error that the subjects of the error forms return.
var errSentinel = errors.New("sentinel")

// codedError is an error type that ErrorAs finds.
type codedError struct {
	// code is the error's code.
	code int
}

// Error states the code.
func (e *codedError) Error() string {
	return "code " + strconv.Itoa(e.code)
}

// formCase is one property form: its id, the id of its assertion, a call
// whose every case passes, and a call that fails.
type formCase struct {
	// name is the form's Go name.
	name string
	// id is the form's id.
	id string
	// assertion is the id of the form's assertion.
	assertion string
	// pass calls the form with a subject that passes every case.
	pass func(tb assert.TB)
	// fail calls the form with a subject that fails a case.
	fail func(tb assert.TB)
}

// TestForms checks that each property form passes a subject that its
// assertion passes for every input, and reports one record of its id for a
// subject that fails, whose failure is its assertion's record.
func TestForms(t *testing.T) {
	t.Parallel()

	for _, tt := range formCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			t.Run("reports no record for a subject that passes every case", func(t *testing.T) {
				t.Parallel()
				rec := assert.NewRecorder()
				tt.pass(rec)
				assert.False(t, rec.Failed(), "the run passes")
			})

			t.Run("records its call under its id and the calls of its cases under the call", func(t *testing.T) {
				t.Parallel()
				rec := assert.NewRecorder()
				tt.pass(rec)
				calls := decodedCalls(t, rec.Records())
				assert.Equal(t, []any{calls[0]["assertion"], calls[0]["verdict"]}, []any{tt.id, "pass"},
					"the form's call first")
				assert.Equal(t, []any{calls[1]["assertion"], calls[1]["parent"]}, []any{tt.assertion, 1.0},
					"a call of the form's assertion in a case")
			})

			t.Run("reports one record of the form whose failure is the assertion's record", func(t *testing.T) {
				t.Parallel()
				rec := assert.NewRecorder()
				tt.fail(rec)
				records := rec.Failures()
				assert.Length(t, records, 1, "one record")
				assert.Equal(t, records[0].Assertion, tt.id, "the form's id")
				assert.Equal(t, records[0].Contract, contractOfForm, "the form's message")
				assert.Equal(t, records[0].Detail[outcomeField], any(prop.Counterexample), "a counterexample")
				failure, ok := records[0].Detail[failureField].(assert.Failure)
				assert.True(t, ok, "the failure is a record")
				assert.Equal(t, failure.Assertion, tt.assertion, "the assertion's record")
			})
		})
	}

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()

		t.Run("reports the minimal input and the record of equal for it", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.Equal(rec, same, next, contractOfForm, prop.Seed(7))
			detail := rec.Failures()[0].Detail
			want := []prop.Entry{prop.Drawn{Label: "input", Value: int8(0), Relevance: prop.AnyValueFails}}
			assert.Equal(t, detail[counterexampleField], any(want), "the input 0, which any input fails like")
			failure := detail[failureField].(assert.Failure)
			assert.Equal(t, failure.Detail, map[string]any{"want": int8(1), "got": int8(0)},
				"the results of the minimal input")
		})
	})

	t.Run("Commutative", func(t *testing.T) {
		t.Parallel()

		t.Run("labels the two inputs a and b", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.Commutative(rec, subtract, contractOfForm, prop.Seed(7), prop.Explain(false))
			got := rec.Failures()[0].Detail[counterexampleField].([]prop.Entry)
			assert.Equal(t, labelsOf(got), []string{"a", "b"}, "the two inputs")
		})
	})

	t.Run("Associative", func(t *testing.T) {
		t.Parallel()

		t.Run("labels the three inputs a, b and c", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			prop.Associative(rec, subtract, contractOfForm, prop.Seed(7), prop.Explain(false))
			got := rec.Failures()[0].Detail[counterexampleField].([]prop.Entry)
			assert.Equal(t, labelsOf(got), []string{"a", "b", "c"}, "the three inputs")
		})
	})
}

// TestFormsEnv checks that each property form names its faults by its Go
// name. It sets the profile variable of the process's environment, so its
// cases run one at a time.
func TestFormsEnv(t *testing.T) {
	t.Setenv(profileVariable, "nightly")
	for _, tt := range formCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("names a fault of its run by its Go name", func(t *testing.T) {
				seat := &matchertest.Seat{}
				tt.pass(seat)
				expectOnlyFault(t, seat.Faults(), profileFault("prop."+tt.name))
			})
		})
	}

	t.Run("MaxAllocs", func(t *testing.T) {
		t.Run("names a fault of its run by its Go name", func(t *testing.T) {
			seat := &matchertest.Seat{}
			prop.MaxAllocs(seat, func(int8) {}, 0, contractOfForm)
			expectOnlyFault(t, seat.Faults(), profileFault("prop.MaxAllocs"))
		})
	})
}

// TestMaxAllocsForm checks prop.MaxAllocs, which counts the allocations of
// the whole process and so runs in a test that does not run in parallel.
func TestMaxAllocsForm(t *testing.T) {
	allocates := func(x int8) {
		if x >= 5 {
			heap = make([]byte, 16)
		}
	}

	t.Run("reports no record for a function within the ceiling", func(t *testing.T) {
		rec := assert.NewRecorder()
		prop.MaxAllocs(rec, func(int8) {}, 0, contractOfForm, prop.Seed(7))
		assert.False(t, rec.Failed(), "the run passes")
	})

	t.Run("reports the smallest input that allocates past the ceiling in a build that counts them", func(t *testing.T) {
		rec := assert.NewRecorder()
		prop.MaxAllocs(rec, allocates, 0, contractOfForm, prop.Seed(7), prop.Explain(false))
		assert.Equal(t, rec.Failed(), matcher.AllocationsCounted(), "the run fails in a build that counts allocations")
		if matcher.AllocationsCounted() {
			records := rec.Failures()
			assert.Length(t, records, 1, "one record")
			assert.Equal(t, records[0].Assertion, "prop-max-allocs", "the form's id")
			assert.Equal(t, records[0].Detail[counterexampleField],
				any([]prop.Entry{prop.Drawn{Label: "input", Value: int8(5)}}), "the smallest input that allocates")
		}
	})

	t.Run("runs one case at a time on four workers, and reports what one worker reports", func(t *testing.T) {
		one, four := assert.NewRecorder(), assert.NewRecorder()
		prop.MaxAllocs(one, allocates, 0, contractOfForm, prop.Seed(7))
		prop.MaxAllocs(four, allocates, 0, contractOfForm, prop.Seed(7), prop.Workers(4))
		assert.Equal(t, one.Failed(), matcher.AllocationsCounted(), "the run fails in a build that counts allocations")
		assert.Equal(t, detailsOf(four.Failures()), detailsOf(one.Failures()), "the records of one worker")
	})
}

// detailsOf returns the detail of each record, in order.
func detailsOf(records []assert.Failure) []map[string]any {
	out := make([]map[string]any, len(records))
	for i, r := range records {
		out[i] = r.Detail
	}
	return out
}

// TestFormsAllocs checks the allocation ceiling of a passing run of each
// property form.
func TestFormsAllocs(t *testing.T) {
	rec := &matchertest.Seat{}
	for _, tt := range formCases() {
		assert.MaxAllocs(t, func() { tt.pass(rec) }, formAllocs[tt.name], tt.name+" allocates its run")
	}
	assert.MaxAllocs(t, func() { prop.MaxAllocs(rec, func(int8) {}, 0, contractOfForm, prop.Seed(7)) },
		maxAllocsFormAllocs, "MaxAllocs allocates its run")
	assert.False(t, rec.Failed(), "every run passes")
}

// BenchmarkForms measures a passing run of 100 cases of each property
// form.
func BenchmarkForms(b *testing.B) {
	for _, tt := range formCases() {
		b.Run(tt.name, func(b *testing.B) {
			rec := &matchertest.Seat{}
			c := bench.Start(b).MaxAllocs(formAllocs[tt.name])
			defer c.End()
			for c.Loop() {
				tt.pass(rec)
			}
			assert.False(b, rec.Failed(), "every run passes")
		})
	}

	b.Run("MaxAllocs", func(b *testing.B) {
		rec := &matchertest.Seat{}
		c := bench.Start(b).MaxAllocs(maxAllocsFormAllocs)
		defer c.End()
		for c.Loop() {
			prop.MaxAllocs(rec, func(int8) {}, 0, contractOfForm, prop.Seed(7))
		}
		assert.False(b, rec.Failed(), "every run passes")
	})
}

// heap keeps an allocation of a test subject alive, so the compiler moves
// it to the heap.
var heap []byte

// formCases returns a passing and a failing call of each property form but
// MaxAllocs, which runs one case at a time.
func formCases() []formCase {
	seven := prop.Seed(7)
	return []formCase{
		{
			name: "Equal", id: "prop-equal", assertion: "equal",
			pass: func(tb assert.TB) { prop.Equal(tb, same, same, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.Equal(tb, same, next, contractOfForm, seven) },
		},
		{
			name: "NotEqual", id: "prop-not-equal", assertion: "not-equal",
			pass: func(tb assert.TB) { prop.NotEqual(tb, same, next, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.NotEqual(tb, same, same, contractOfForm, seven) },
		},
		{
			name: "True", id: "prop-true", assertion: "true",
			pass: func(tb assert.TB) { prop.True(tb, func(int8) bool { return true }, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.True(tb, func(x int8) bool { return x < 5 }, contractOfForm, seven) },
		},
		{
			name: "False", id: "prop-false", assertion: "false",
			pass: func(tb assert.TB) { prop.False(tb, func(int8) bool { return false }, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.False(tb, func(x int8) bool { return x >= 5 }, contractOfForm, seven) },
		},
		{
			name: "Nil", id: "prop-nil", assertion: "nil",
			pass: func(tb assert.TB) { prop.Nil(tb, func(int8) *int8 { return nil }, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.Nil(tb, func(x int8) *int8 { return &x }, contractOfForm, seven) },
		},
		{
			name: "NotNil", id: "prop-not-nil", assertion: "not-nil",
			pass: func(tb assert.TB) { prop.NotNil(tb, func(x int8) *int8 { return &x }, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.NotNil(tb, func(int8) *int8 { return nil }, contractOfForm, seven) },
		},
		{
			name: "Length", id: "prop-length", assertion: "length",
			pass: func(tb assert.TB) {
				prop.Length(tb, func(int8) []int8 { return make([]int8, 3) }, 3, contractOfForm, seven)
			},
			fail: func(tb assert.TB) { prop.Length(tb, sameList, 0, contractOfForm, seven) },
		},
		{
			name: "Empty", id: "prop-empty", assertion: "empty",
			pass: func(tb assert.TB) { prop.Empty(tb, func([]int8) []int8 { return nil }, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.Empty(tb, sameList, contractOfForm, seven) },
		},
		{
			name: "NotEmpty", id: "prop-not-empty", assertion: "not-empty",
			pass: func(tb assert.TB) { prop.NotEmpty(tb, prependZero, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.NotEmpty(tb, sameList, contractOfForm, seven) },
		},
		{
			name: "Contains", id: "prop-contains", assertion: "contains",
			pass: func(tb assert.TB) { prop.Contains(tb, prependZero, int8(0), contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.Contains(tb, sameList, int8(0), contractOfForm, seven) },
		},
		{
			name: "NotContains", id: "prop-not-contains", assertion: "not-contains",
			pass: func(tb assert.TB) {
				prop.NotContains(tb, func([]int8) []int8 { return nil }, int8(0), contractOfForm, seven)
			},
			fail: func(tb assert.TB) { prop.NotContains(tb, prependZero, int8(0), contractOfForm, seven) },
		},
		{
			name: "ContainsInOrder", id: "prop-contains-in-order", assertion: "contains-in-order",
			pass: func(tb assert.TB) { prop.ContainsInOrder(tb, wrap, []string{"a", "b"}, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.ContainsInOrder(tb, wrap, []string{"b", "a"}, contractOfForm, seven) },
		},
		{
			name: "IsPermutation", id: "prop-permutation", assertion: "permutation",
			pass: func(tb assert.TB) { prop.IsPermutation(tb, sortedList, sameList, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.IsPermutation(tb, prependZero, sameList, contractOfForm, seven) },
		},
		{
			name: "HasPrefix", id: "prop-has-prefix", assertion: "has-prefix",
			pass: func(tb assert.TB) { prop.HasPrefix(tb, wrap, "a", contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.HasPrefix(tb, sameText, "a", contractOfForm, seven) },
		},
		{
			name: "HasSuffix", id: "prop-has-suffix", assertion: "has-suffix",
			pass: func(tb assert.TB) { prop.HasSuffix(tb, wrap, "b", contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.HasSuffix(tb, sameText, "b", contractOfForm, seven) },
		},
		{
			name: "Matches", id: "prop-matches", assertion: "matches",
			pass: func(tb assert.TB) { prop.Matches(tb, wrap, "^a", contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.Matches(tb, sameText, "^a", contractOfForm, seven) },
		},
		{
			name: "CloseTo", id: "prop-close-to", assertion: "close-to",
			pass: func(tb assert.TB) { prop.CloseTo(tb, widen, 0, 128, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.CloseTo(tb, widen, 0, 0.5, contractOfForm, seven) },
		},
		{
			name: "InRange", id: "prop-in-range", assertion: "in-range",
			pass: func(tb assert.TB) { prop.InRange(tb, widen, -128, 127, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.InRange(tb, widen, 0, 127, contractOfForm, seven) },
		},
		{
			name: "Pairwise", id: "prop-pairwise", assertion: "pairwise",
			pass: func(tb assert.TB) { prop.Pairwise(tb, sortedList, ascending, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.Pairwise(tb, sameList, ascending, contractOfForm, seven) },
		},
		{
			name: "NoError", id: "prop-err-absent", assertion: "err-absent",
			pass: func(tb assert.TB) { prop.NoError(tb, succeeds, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.NoError(tb, failsFromFive, contractOfForm, seven) },
		},
		{
			name: "HasError", id: "prop-err-present", assertion: "err-present",
			pass: func(tb assert.TB) { prop.HasError(tb, fails, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.HasError(tb, succeeds, contractOfForm, seven) },
		},
		{
			name: "ErrorIs", id: "prop-err-is", assertion: "err-is",
			pass: func(tb assert.TB) { prop.ErrorIs(tb, fails, errSentinel, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.ErrorIs(tb, succeeds, errSentinel, contractOfForm, seven) },
		},
		{
			name: "ErrorIsNot", id: "prop-err-is-not", assertion: "err-is-not",
			pass: func(tb assert.TB) { prop.ErrorIsNot(tb, succeeds, errSentinel, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.ErrorIsNot(tb, fails, errSentinel, contractOfForm, seven) },
		},
		{
			name: "ErrorAs", id: "prop-err-as", assertion: "err-as",
			pass: func(tb assert.TB) {
				prop.ErrorAs[*codedError](tb, func(int8) error { return &codedError{code: 1} }, contractOfForm, seven)
			},
			fail: func(tb assert.TB) { prop.ErrorAs[*codedError](tb, succeeds, contractOfForm, seven) },
		},
		{
			name: "Panics", id: "prop-throws", assertion: "throws",
			pass: func(tb assert.TB) { prop.Panics(tb, func(int8) { panic("always") }, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.Panics(tb, func(int8) {}, contractOfForm, seven) },
		},
		{
			name: "NotPanics", id: "prop-not-throws", assertion: "not-throws",
			pass: func(tb assert.TB) { prop.NotPanics(tb, func(int8) {}, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.NotPanics(tb, panicsFromFive, contractOfForm, seven) },
		},
		{
			name: "Pure", id: "prop-pure", assertion: "pure",
			pass: func(tb assert.TB) {
				prop.Pure(tb, func() int { return 0 }, func(int8) {}, contractOfForm, seven)
			},
			fail: func(tb assert.TB) {
				var calls int
				prop.Pure(tb, func() int { return calls }, func(int8) { calls++ }, contractOfForm, seven)
			},
		},
		{
			name: "NotPure", id: "prop-not-pure", assertion: "not-pure",
			pass: func(tb assert.TB) {
				var calls int
				prop.NotPure(tb, func() int { return calls }, func(int8) { calls++ }, contractOfForm, seven)
			},
			fail: func(tb assert.TB) {
				prop.NotPure(tb, func() int { return 0 }, func(int8) {}, contractOfForm, seven)
			},
		},
		{
			name: "NilContextSafe", id: "prop-nil-context-safe", assertion: "nil-context-safe",
			pass: func(tb assert.TB) { prop.NilContextSafe(tb, declines, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.NilContextSafe(tb, awaits, contractOfForm, seven) },
		},
		{
			name: "HonoursCancellation", id: "prop-honours-cancellation", assertion: "honours-cancellation",
			pass: func(tb assert.TB) { prop.HonoursCancellation(tb, reads, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.HonoursCancellation(tb, declines, contractOfForm, seven) },
		},
		{
			name: "HonoursDeadline", id: "prop-honours-deadline", assertion: "honours-deadline",
			pass: func(tb assert.TB) { prop.HonoursDeadline(tb, reads, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.HonoursDeadline(tb, declines, contractOfForm, seven) },
		},
		{
			name: "Idempotent", id: "prop-idempotent", assertion: "idempotent",
			pass: func(tb assert.TB) {
				var state int8
				prop.Idempotent(tb, func(x int8) error { state = x; return nil }, func() int8 { return state },
					contractOfForm, seven)
			},
			fail: func(tb assert.TB) {
				var calls int
				prop.Idempotent(tb, func(int8) error { calls++; return nil }, func() int { return calls },
					contractOfForm, seven)
			},
		},
		{
			name: "Accumulates", id: "prop-accumulates", assertion: "accumulates",
			pass: func(tb assert.TB) {
				var calls int
				prop.Accumulates(tb, func(int8) error { calls++; return nil }, func() int { return calls },
					contractOfForm, seven)
			},
			fail: func(tb assert.TB) {
				prop.Accumulates(tb, func(int8) error { return nil }, func() int { return 0 }, contractOfForm, seven)
			},
		},
		{
			name: "Deterministic", id: "prop-deterministic", assertion: "deterministic",
			pass: func(tb assert.TB) {
				prop.Deterministic(tb, func(x int8) (int8, error) { return x, nil }, contractOfForm, seven)
			},
			fail: func(tb assert.TB) {
				var calls int
				prop.Deterministic(tb, func(int8) (int, error) { calls++; return calls, nil }, contractOfForm, seven)
			},
		},
		{
			name: "Commutative", id: "prop-commutative", assertion: "commutative",
			pass: func(tb assert.TB) { prop.Commutative(tb, add, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.Commutative(tb, subtract, contractOfForm, seven) },
		},
		{
			name: "Associative", id: "prop-associative", assertion: "associative",
			pass: func(tb assert.TB) { prop.Associative(tb, add, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.Associative(tb, subtract, contractOfForm, seven) },
		},
		{
			name: "RoundTrip", id: "prop-round-trip", assertion: "round-trip",
			pass: func(tb assert.TB) { prop.RoundTrip(tb, render, parse, contractOfForm, seven) },
			fail: func(tb assert.TB) { prop.RoundTrip(tb, renderMagnitude, parse, contractOfForm, seven) },
		},
	}
}

// sameList returns x.
func sameList(x []int8) []int8 {
	return x
}

// prependZero returns x after a zero.
func prependZero(x []int8) []int8 {
	return append([]int8{0}, x...)
}

// sortedList returns a sorted copy of x.
func sortedList(x []int8) []int8 {
	return slices.Sorted(slices.Values(x))
}

// ascending reports whether earlier is at most later.
func ascending(earlier, later int8) bool {
	return earlier <= later
}

// sameText returns s.
func sameText(s string) string {
	return s
}

// wrap returns s between an a and a b.
func wrap(s string) string {
	return "a" + s + "b"
}

// widen returns x as a float64.
func widen(x int8) float64 {
	return float64(x)
}

// succeeds returns no error.
func succeeds(int8) error {
	return nil
}

// fails returns errSentinel.
func fails(int8) error {
	return errSentinel
}

// failsFromFive returns errSentinel for an x of 5 or more.
func failsFromFive(x int8) error {
	if x >= 5 {
		return errSentinel
	}
	return nil
}

// panicsFromFive panics for an x of 5 or more.
func panicsFromFive(x int8) {
	if x >= 5 {
		panic("five or more")
	}
}

// declines returns no error, whatever its handle states.
func declines(context.Context, int8) error {
	return nil
}

// reads returns the error of its handle.
func reads(ctx context.Context, _ int8) error {
	return ctx.Err()
}

// awaits waits for its handle to end, which a nil handle cannot.
func awaits(ctx context.Context, _ int8) error {
	<-ctx.Done()
	return nil
}

// add returns a plus b.
func add(a, b int8) int8 {
	return a + b
}

// render returns the decimal text of x.
func render(x int16) (string, error) {
	return strconv.Itoa(int(x)), nil
}

// renderMagnitude returns the decimal text of x without its sign.
func renderMagnitude(x int16) (string, error) {
	return strconv.Itoa(max(int(x), -int(x))), nil
}

// parse returns the int16 of the decimal text s.
func parse(s string) (int16, error) {
	n, err := strconv.ParseInt(s, 10, 16)
	return int16(n), err
}

// labelsOf returns the labels of the entries of a counterexample of draws,
// in order.
func labelsOf(entries []prop.Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.(prop.Drawn).Label
	}
	return out
}
