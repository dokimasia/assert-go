// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package cycle_test

import (
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/cycle"
)

// enterAllocs are the allocations of an Enter and a Leave on a path that
// has contained a container before.
const enterAllocs = 0

// point is a struct whose first field is at the struct's address.
type point struct {
	X int
}

// TestPath checks the containers that a path contains, and the steps back
// to one that a walk meets again.
func TestPath(t *testing.T) {
	t.Parallel()

	t.Run("Enter", func(t *testing.T) {
		t.Parallel()

		t.Run("adds a container and reports true", func(t *testing.T) {
			t.Parallel()
			var p cycle.Path
			back, entered := p.Enter(reflect.ValueOf(map[string]int{}))
			assert.Equal(t, []any{back, entered}, []any{0, true}, "no steps back, and the map entered")
		})

		t.Run("returns the steps back to a container that the path contains", func(t *testing.T) {
			t.Parallel()
			var p cycle.Path
			outer, inner := reflect.ValueOf([]any{1}), reflect.ValueOf(map[string]int{})
			p.Enter(outer)
			p.Enter(inner)
			back, entered := p.Enter(inner)
			assert.Equal(t, []any{back, entered}, []any{1, false}, "one step back to the map entered last")
			back, entered = p.Enter(outer)
			assert.Equal(t, []any{back, entered}, []any{2, false}, "two steps back to the outer slice")
		})

		t.Run("adds a shorter slice of the same array as another container", func(t *testing.T) {
			t.Parallel()
			var p cycle.Path
			items := []int{1, 2}
			p.Enter(reflect.ValueOf(items))
			_, entered := p.Enter(reflect.ValueOf(items[:1]))
			assert.True(t, entered, "a slice of one element differs from the slice of two")
		})

		t.Run("adds a pointer of another type to the same address as another container", func(t *testing.T) {
			t.Parallel()
			var p cycle.Path
			v := &point{X: 1}
			p.Enter(reflect.ValueOf(v))
			_, entered := p.Enter(reflect.ValueOf(&v.X))
			assert.True(t, entered, "a *int differs from the *point at its address")
		})

		t.Run("panics for a value that is no pointer, map or slice", func(t *testing.T) {
			t.Parallel()
			var p cycle.Path
			assert.Panics(t, func() { p.Enter(reflect.ValueOf(1)) }, "an int has no address to enter")
		})
	})

	t.Run("Leave", func(t *testing.T) {
		t.Parallel()

		t.Run("removes the container that the walk entered last", func(t *testing.T) {
			t.Parallel()
			var p cycle.Path
			outer, inner := reflect.ValueOf([]any{1}), reflect.ValueOf(map[string]int{})
			p.Enter(outer)
			p.Enter(inner)
			p.Leave()
			_, entered := p.Enter(inner)
			assert.True(t, entered, "the map enters again once the walk left it")
			back, _ := p.Enter(outer)
			assert.Equal(t, back, 2, "the outer slice is still on the path")
		})

		t.Run("panics on an empty path", func(t *testing.T) {
			t.Parallel()
			var p cycle.Path
			assert.Panics(t, p.Leave, "an empty path has no container to remove")
		})
	})
}

// TestPathAllocs checks the allocation ceiling of an Enter and a Leave.
func TestPathAllocs(t *testing.T) {
	var p cycle.Path
	v := reflect.ValueOf(map[string]int{})
	p.Enter(v)
	p.Leave()
	assert.MaxAllocs(t, func() { p.Enter(v); p.Leave() }, enterAllocs,
		"Enter reuses the storage of a container that the path contained")
}

// BenchmarkPath measures an Enter and a Leave on a path that has contained
// a container before.
func BenchmarkPath(b *testing.B) {
	var p cycle.Path
	v := reflect.ValueOf(map[string]int{})
	p.Enter(v)
	p.Leave()

	b.Run("Enter and Leave", func(b *testing.B) {
		c := bench.Start(b).MaxAllocs(enterAllocs)
		defer c.End()
		for c.Loop() {
			p.Enter(v)
			p.Leave()
		}
	})
}
