// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
)

func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()

		t.Run("identical values report nothing", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			matcher.Equal(s, matcher.Fatal, 1, 1, "ints match")

			if len(s.fatals) != 0 {
				t.Fatalf("fatals = %v, want none", s.fatals)
			}
		})

		t.Run("the message leads, then want and got", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			matcher.Equal(s, matcher.Fatal, 1, 2, "ints match")

			if got, want := len(s.fatals), 1; got != want {
				t.Fatalf("len(fatals) = %d, want %d", got, want)
			}
			if !strings.HasPrefix(s.fatals[0], "ints match:") {
				t.Fatalf("message %q does not lead with the contract message", s.fatals[0])
			}
			for _, field := range []string{"want", "got"} {
				if !strings.Contains(s.fatals[0], field) {
					t.Fatalf("message %q does not name %q", s.fatals[0], field)
				}
			}
		})

		t.Run("Soft mode records instead of stopping", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			matcher.Equal(s, matcher.Soft, 1, 2, "ints match")

			if len(s.fatals) != 0 {
				t.Fatalf("fatals = %v, want none in Soft mode", s.fatals)
			}
			if got, want := len(s.errs), 1; got != want {
				t.Fatalf("len(errs) = %d, want %d", got, want)
			}
		})

		t.Run("a nil slice does not equal an empty one", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			var nilSlice []int
			matcher.Equal(s, matcher.Fatal, nilSlice, []int{}, "nil is not empty")

			if got, want := len(s.fatals), 1; got != want {
				t.Fatalf("len(fatals) = %d, want %d", got, want)
			}
		})

		t.Run("EquateEmpty accepts a nil slice against an empty one", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			var nilSlice []int
			matcher.Equal(s, matcher.Fatal, nilSlice, []int{}, "opted in", matcher.EquateEmpty())

			if len(s.fatals) != 0 {
				t.Fatalf("fatals = %v, want none under EquateEmpty", s.fatals)
			}
		})

		t.Run("a cyclic structure stops", func(t *testing.T) {
			t.Parallel()

			type node struct {
				Name string
				Next *node
			}
			cycle := func() *node {
				a, b := &node{Name: "a"}, &node{Name: "b"}
				a.Next, b.Next = b, a
				return a
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				matcher.Equal(&seat{}, matcher.Fatal, cycle(), cycle(), "cycles stop")
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Equal did not return on a cyclic structure")
			}
		})
	})

	t.Run("NotEqual", func(t *testing.T) {
		t.Parallel()

		t.Run("matching values report got", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			matcher.NotEqual(s, matcher.Fatal, 1, 1, "values differ")

			if got, want := len(s.fatals), 1; got != want {
				t.Fatalf("len(fatals) = %d, want %d", got, want)
			}
			if !strings.Contains(s.fatals[0], "got") {
				t.Fatalf("message %q does not name %q", s.fatals[0], "got")
			}
		})

		t.Run("differing values report nothing", func(t *testing.T) {
			t.Parallel()

			s := &seat{}
			matcher.NotEqual(s, matcher.Fatal, 1, 2, "values differ")

			if len(s.fatals) != 0 {
				t.Fatalf("fatals = %v, want none", s.fatals)
			}
		})
	})
}
