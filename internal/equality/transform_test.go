// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package equality_test

import (
	"reflect"
	"testing"

	"go.dokimi.dev/assert/internal/equality"
	"go.dokimi.dev/assert/internal/matcher"
)

// The allocation ceilings of the transforms.
const (
	// withTransformAllocs is the ceiling of the allocations of WithTransform
	// of rules without a transform: the list of one transform.
	withTransformAllocs = 2
	// resultAllocs is the ceiling of the allocations of Result of a value
	// that matches no transform.
	resultAllocs = 0
)

// TestTransform checks how rules keep their transforms, and the result of a
// value under them.
func TestTransform(t *testing.T) {
	t.Parallel()

	byValueOnly := transformOf(func(c cached) int { return c.Value })
	bySize := transformOf(func(c cached) int { return c.size })
	x, y := reflect.ValueOf(cached{Value: 1, size: 5}), reflect.ValueOf(cached{Value: 1, size: 9})

	t.Run("WithTransform", func(t *testing.T) {
		t.Parallel()

		t.Run("replaces a transform of the same type", func(t *testing.T) {
			t.Parallel()
			r := equality.Rules{}.WithTransform(byValueOnly).WithTransform(bySize)
			if len(r.Transforms) != 1 {
				t.Fatalf("the rules keep %d transforms, want 1", len(r.Transforms))
			}
			if equality.Equal(x, y, r) {
				t.Fatal("Equal reports two sizes equal, want the transform by size to apply")
			}
		})

		t.Run("keeps a transform of another type", func(t *testing.T) {
			t.Parallel()
			r := equality.Rules{}.WithTransform(byValueOnly).WithTransform(halving.Transforms[0])
			if len(r.Transforms) != 2 {
				t.Fatalf("the rules keep %d transforms, want 2", len(r.Transforms))
			}
			if !equality.Equal(x, y, r) {
				t.Fatal("Equal reports two values unequal, want the transform by value to apply")
			}
		})

		t.Run("leaves the rules that it extends unchanged", func(t *testing.T) {
			t.Parallel()
			base := equality.Rules{}.WithTransform(byValueOnly).WithTransform(halving.Transforms[0])
			_ = base.WithTransform(bySize)
			if !equality.Equal(x, y, base) {
				t.Fatal("Equal reports two values unequal, want the transform by value of the base rules")
			}
		})
	})

	t.Run("Result", func(t *testing.T) {
		t.Parallel()

		var inner any = cached{Value: 2, size: 7}
		tests := []struct {
			name      string
			giveValue reflect.Value
			giveRules equality.Rules
			want      any
		}{
			{
				name:      "returns the result of the last transform that a value matches",
				giveValue: reflect.ValueOf(cached{Value: 1, size: 5}),
				giveRules: equality.Rules{Transforms: []equality.Transform{byValueOnly, bySize}},
				want:      5,
			},
			{
				name:      "returns a value that matches no transform",
				giveValue: reflect.ValueOf(3),
				giveRules: byValue,
				want:      3,
			},
			{
				name:      "returns the result of the value inside an interface",
				giveValue: reflect.ValueOf(&inner).Elem(),
				giveRules: byValue,
				want:      2,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := equality.Result(tt.giveValue, tt.giveRules)
				if got.Interface() != tt.want {
					t.Fatalf("Result returns %v, want %v", got, tt.want)
				}
			})
		}

		t.Run("returns the invalid value", func(t *testing.T) {
			t.Parallel()
			if got := equality.Result(reflect.Value{}, byValue); got.IsValid() {
				t.Fatalf("Result returns %v, want the invalid value", got)
			}
		})

		t.Run("returns a nil interface as it is", func(t *testing.T) {
			t.Parallel()
			var none any
			got := equality.Result(reflect.ValueOf(&none).Elem(), byValue)
			if !got.IsValid() || got.Kind() != reflect.Interface || !got.IsNil() {
				t.Fatalf("Result returns %v, want the nil interface", got)
			}
		})
	})
}

// TestTransformAllocs checks the allocation ceilings of WithTransform and of
// Result.
func TestTransformAllocs(t *testing.T) {
	byValueOnly := transformOf(func(c cached) int { return c.Value })
	value := reflect.ValueOf(3)
	calls := []struct {
		name   string
		call   func()
		allocs float64
	}{
		{
			name:   "WithTransform",
			call:   func() { _ = equality.Rules{}.WithTransform(byValueOnly) },
			allocs: withTransformAllocs,
		},
		{name: "Result", call: func() { _ = equality.Result(value, byValue) }, allocs: resultAllocs},
	}
	for _, c := range calls {
		got := testing.AllocsPerRun(allocRuns, c.call)
		if matcher.AllocationsCounted() && got > c.allocs {
			t.Errorf("%s allocates %v times, want at most %v", c.name, got, c.allocs)
		}
	}
}

// BenchmarkTransform measures WithTransform and Result.
func BenchmarkTransform(b *testing.B) {
	b.Run("WithTransform", func(b *testing.B) {
		byValueOnly := transformOf(func(c cached) int { return c.Value })
		b.ReportAllocs()
		for b.Loop() {
			_ = equality.Rules{}.WithTransform(byValueOnly)
		}
	})

	b.Run("Result", func(b *testing.B) {
		value := reflect.ValueOf(3)
		b.ReportAllocs()
		for b.Loop() {
			_ = equality.Result(value, byValue)
		}
	})
}
