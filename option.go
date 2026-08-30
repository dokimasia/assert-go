// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Option relaxes one comparison rule for the call it is passed to.
//
// An Option carries no state and is safe to reuse across calls and
// across goroutines. Order does not matter, passing one twice has the
// same effect as passing it once, and an Option never leaks into a
// call it was not passed to.
//
// [go.dokimi.dev/assert/expect] names the same type, so an option
// built by either package works with both.
type Option = matcher.Option

// EquateEmpty makes a nil map or slice equal an empty one of the same
// type, for the call it is passed to.
//
// The default keeps them distinct, because a value that is absent and
// a value that is present but empty are different answers, and a test
// may need to tell them apart.
//
//	assert.Equal(t, got, []int{}, "List returns no items", assert.EquateEmpty())
func EquateEmpty() Option { return matcher.EquateEmpty() }

// EquateNaNs makes a NaN float equal another NaN of the same type, for
// the call it is passed to.
//
// The default keeps them unequal, following IEEE 754, where NaN
// compares unequal to every value including itself.
func EquateNaNs() Option { return matcher.EquateNaNs() }
