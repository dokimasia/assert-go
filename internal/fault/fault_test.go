// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package fault_test

import (
	"errors"
	"io"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
)

// errEmpty is a sentinel that the tests give a fault as its kind.
var errEmpty = errors.New("test: the bounds admit no value")

// sink receives the error that a measured call returns, so that the error
// escapes to the heap as it does when a caller returns it.
var sink error

// The allocation ceilings of the functions of a fault, measured with a
// result that escapes.
const (
	// newAllocs is the ceiling of the allocations of New and Of: the fault
	// and its reason.
	newAllocs = 3
	// atAllocs is the ceiling of the allocations of At: the fault and its
	// path.
	atAllocs = 3
	// inAllocs is the ceiling of the allocations of In: the fault.
	inAllocs = 2
	// errorAllocs is the ceiling of the allocations of Error of a fault of
	// every part: the text of its path and the text it returns.
	errorAllocs = 3
)

// TestFault checks how a fault is built, how it gains its path and its
// operation, how it renders, and how errors.Is and errors.As see it.
func TestFault(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a fault whose reason is the formatted text", func(t *testing.T) {
			t.Parallel()
			f := fault.New("the key %s states no value", "min")
			assert.Equal(t, *f, fault.Error{Reason: "the key min states no value"}, "a fault of the reason alone")
		})

		t.Run("states a value that contains itself with the cycle marked", func(t *testing.T) {
			t.Parallel()
			m := map[string]any{}
			m["self"] = m
			f := fault.New("%v is no list", m)
			assert.Equal(t, f.Reason, "map[self:<cycle>] is no list", "the reason")
		})
	})

	t.Run("Of", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a fault of the kind whose reason is the formatted text", func(t *testing.T) {
			t.Parallel()
			f := fault.Of(errEmpty, "the bounds [%d, %d] admit no value", 1, 0)
			assert.Equal(t, *f, fault.Error{Kind: errEmpty, Reason: "the bounds [1, 0] admit no value"},
				"a fault of the kind and the reason")
		})
	})

	t.Run("Because", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the fault with err as its cause", func(t *testing.T) {
			t.Parallel()
			f := fault.New("the text is no JSON")
			assert.True(t, f.Because(io.EOF) == f, "Because returns the fault it was called on")
			assert.Equal(t, f.Err, io.EOF, "the cause")
		})
	})

	t.Run("At", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a fault with the segments in front of the path of a fault", func(t *testing.T) {
			t.Parallel()
			inner := fault.At(fault.New("is no character"), fault.Index(1), fault.Field("note"))
			err := fault.At(inner, fault.Field("order"), fault.Field("lines"))
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, f.Path,
				fault.Path{fault.Field("order"), fault.Field("lines"), fault.Index(1), fault.Field("note")},
				"the outer segments first, in the order given")
			assert.Equal(t, f.Reason, "is no character", "the reason of the fault")
		})

		t.Run("returns a copy and leaves the fault it was given unchanged", func(t *testing.T) {
			t.Parallel()
			inner := fault.New("is no character")
			_ = fault.At(inner, fault.Field("order"))
			assert.Empty(t, inner.Path, "the fault it was given has no path")
		})

		t.Run("returns a fault at the segments whose cause is an error that is no fault", func(t *testing.T) {
			t.Parallel()
			segs := []fault.Segment{fault.Field("entries"), fault.Index(0)}
			err := fault.At(io.EOF, segs...)
			segs[0] = fault.Field("changed")
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, *f, fault.Error{Path: fault.Path{fault.Field("entries"), fault.Index(0)}, Err: io.EOF},
				"a fault of a copy of the segments and the cause")
		})
	})

	t.Run("In", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a fault without an operation with the operation", func(t *testing.T) {
			t.Parallel()
			err := fault.In("prop.ShapeOf", fault.New("the key min states no value"))
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, *f, fault.Error{Op: "prop.ShapeOf", Reason: "the key min states no value"},
				"the fault with its operation")
		})

		t.Run("returns a new fault whose cause is a fault of another operation", func(t *testing.T) {
			t.Parallel()
			inner := fault.In("literal.Decode", fault.New("the record states no fields"))
			err := fault.In("prop.OfShape", inner)
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, *f, fault.Error{Op: "prop.OfShape", Err: inner},
				"a fault whose cause names its own operation")
		})

		t.Run("returns a new fault whose cause is an error that is no fault", func(t *testing.T) {
			t.Parallel()
			err := fault.In("store.Load", io.EOF)
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, *f, fault.Error{Op: "store.Load", Err: io.EOF}, "a fault of the operation and the cause")
		})
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give *fault.Error
			want string
		}{
			{
				name: "returns the operation, the path, the reason and the cause joined by colons",
				give: &fault.Error{
					Op: "prop.ShapeOf",
					Path: fault.Path{
						fault.Field("order"),
						fault.Field("Lines"),
						fault.Element(),
						fault.Field("Note"),
					},
					Reason: "the key min states no value",
					Err:    io.EOF,
				},
				want: "prop.ShapeOf: order.Lines[].Note: the key min states no value: EOF",
			},
			{
				name: "returns the reason alone for a fault of no other part",
				give: &fault.Error{Reason: "the key min states no value"},
				want: "the key min states no value",
			},
			{
				name: "returns the text of the kind in place of a missing reason",
				give: &fault.Error{Op: "choice.NewSizes", Kind: errEmpty},
				want: "choice.NewSizes: test: the bounds admit no value",
			},
			{
				name: "returns the reason and not the kind of a fault of both",
				give: &fault.Error{Kind: errEmpty, Reason: "the sizes [2, 1] admit no size"},
				want: "the sizes [2, 1] admit no size",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Error(), tt.want, "the text of the fault")
			})
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the cause", func(t *testing.T) {
			t.Parallel()
			f := fault.New("the file is no entry").Because(io.EOF)
			assert.Equal(t, f.Unwrap(), io.EOF, "the cause")
			assert.ErrorIs(t, f, io.EOF, "errors.Is sees the cause")
		})
	})

	t.Run("Is", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for the kind of the fault", func(t *testing.T) {
			t.Parallel()
			f := fault.Of(errEmpty, "the sizes [2, 1] admit no size")
			assert.True(t, f.Is(errEmpty), "the kind")
			assert.ErrorIs(t, fault.At(f, fault.Field("lines")), errEmpty, "errors.Is sees the kind through a path")
		})

		t.Run("reports false for another error", func(t *testing.T) {
			t.Parallel()
			f := fault.Of(errEmpty, "the sizes [2, 1] admit no size")
			assert.False(t, f.Is(io.EOF), "another error is not the kind")
			assert.ErrorIsNot(t, f, io.EOF, "errors.Is does not match another error")
		})
	})
}

// TestFaultAllocs checks the allocation ceilings of the functions of a
// fault.
func TestFaultAllocs(t *testing.T) {
	f := fault.Of(errEmpty, "the sizes are empty")
	full := &fault.Error{
		Op: "prop.ShapeOf", Path: fault.Path{fault.Field("order"), fault.Index(1)},
		Reason: "the key min states no value", Err: io.EOF,
	}
	assert.MaxAllocs(t, func() { sink = fault.New("the sizes are empty") }, newAllocs,
		"New allocates the fault and its reason")
	assert.MaxAllocs(t, func() { sink = fault.Of(errEmpty, "the sizes are empty") }, newAllocs,
		"Of allocates the fault and its reason")
	assert.MaxAllocs(t, func() { sink = f.Because(io.EOF) }, 0, "Because allocates nothing")
	assert.MaxAllocs(t, func() { sink = fault.At(f, fault.Field("lines")) }, atAllocs,
		"At allocates the fault and its path")
	assert.MaxAllocs(t, func() { sink = fault.At(full, fault.Field("lines"), fault.Index(2), fault.Index(0)) },
		atAllocs, "At allocates the fault and its path for three segments in front of two")
	assert.MaxAllocs(t, func() { sink = fault.At(io.EOF, fault.Field("lines")) }, atAllocs,
		"At allocates a fault and its path for an error that is no fault")
	assert.MaxAllocs(t, func() { sink = fault.In("prop.ShapeOf", f) }, inAllocs, "In allocates the fault")
	assert.MaxAllocs(t, func() { sink = fault.In("prop.ShapeOf", io.EOF) }, inAllocs,
		"In allocates a fault for an error that is no fault")
	assert.MaxAllocs(t, func() { _ = full.Error() }, errorAllocs, "Error allocates the text of the path and its own")
	assert.MaxAllocs(t, func() { _ = full.Unwrap() }, 0, "Unwrap allocates nothing")
	assert.MaxAllocs(t, func() { _ = full.Is(errEmpty) }, 0, "Is allocates nothing")
}

// BenchmarkFault measures each function of a fault.
func BenchmarkFault(b *testing.B) {
	full := &fault.Error{
		Op: "prop.ShapeOf", Path: fault.Path{fault.Field("order"), fault.Index(1)},
		Reason: "the key min states no value", Err: io.EOF,
	}

	b.Run("New", func(b *testing.B) {
		var got *fault.Error
		c := bench.Start(b).MaxAllocs(newAllocs)
		defer c.End()
		for c.Loop() {
			got = fault.New("the sizes are empty")
		}
		assert.Equal(b, got.Reason, "the sizes are empty", "the reason")
	})

	b.Run("Of", func(b *testing.B) {
		var got *fault.Error
		c := bench.Start(b).MaxAllocs(newAllocs)
		defer c.End()
		for c.Loop() {
			got = fault.Of(errEmpty, "the sizes are empty")
		}
		assert.ErrorIs(b, got, errEmpty, "the kind")
	})

	b.Run("Because", func(b *testing.B) {
		f := fault.New("the sizes are empty")
		var got *fault.Error
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = f.Because(io.EOF)
		}
		assert.Equal(b, got.Err, io.EOF, "the cause")
	})

	b.Run("At", func(b *testing.B) {
		var got error
		c := bench.Start(b).MaxAllocs(atAllocs)
		defer c.End()
		for c.Loop() {
			got = fault.At(full, fault.Field("lines"))
		}
		assert.ErrorIs(b, got, io.EOF, "the cause of the fault")
	})

	b.Run("In", func(b *testing.B) {
		var got error
		c := bench.Start(b).MaxAllocs(inAllocs)
		defer c.End()
		for c.Loop() {
			got = fault.In("prop.OfShape", full)
		}
		assert.ErrorIs(b, got, io.EOF, "the cause of the fault")
	})

	b.Run("Error", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(errorAllocs)
		defer c.End()
		for c.Loop() {
			got = full.Error()
		}
		assert.Equal(b, got, "prop.ShapeOf: order[1]: the key min states no value: EOF", "the text")
	})

	b.Run("Unwrap", func(b *testing.B) {
		var got error
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = full.Unwrap()
		}
		assert.Equal(b, got, io.EOF, "the cause")
	})

	b.Run("Is", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = full.Is(io.EOF)
		}
		assert.False(b, got, "the cause is no kind")
	})
}
