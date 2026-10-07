// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package subject declares the subjects that the packages of the property
// forms check.
package subject

import (
	"context"
	"strconv"
)

// Store keeps a count of the values that Put receives.
type Store struct{ values []int }

func (s *Store) Put(n int) error {
	s.values = append(s.values, n)
	return nil
}

func (s *Store) Count() int { return len(s.values) }

func (s *Store) Snapshot() []int { return s.values }

func (s *Store) Get(n int) {}

func Double(n int) int { return 2 * n }

func Twice(n int) int { return n + n }

func Valid(n int) bool { return n >= 0 }

func Check(n int) error { return nil }

func Fetch(ctx context.Context, n int) error { return ctx.Err() }

func Name(n int) string { return "n" + strconv.Itoa(n) }

func Items(n int) []int { return []int{n} }

func Ratio(n int) float64 { return 0.5 }

func Pointer(n int) *int { return &n }

func Add(a, b int) int { return a + b }

func Encode(n int) (string, error) { return strconv.Itoa(n), nil }

func Decode(s string) (int, error) { return strconv.Atoi(s) }

func Work(n int) {}

func Explode(n int) { panic(n) }

func Buffer(n int) []byte { return make([]byte, n) }

func Consume(b []byte) {}
