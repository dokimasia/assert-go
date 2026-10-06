// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import "go.dokimi.dev/assert/internal/matcher"

// Option changes one comparison rule for the call it is passed to:
// [EquateEmpty] and [EquateNaNs] relax a rule, and [ByIdentity] narrows
// one.
//
// It is the same type [go.dokimi.dev/assert.Option] names, so an
// option built by either package works with both.
type Option = matcher.Option

// EquateEmpty makes a nil map or slice equal an empty one of the same
// type, for the call it is passed to.
//
// The default keeps them distinct, because an absent value and a present
// but empty value are different results, and a test may need to tell
// them apart.
//
// # Allocation contract
//
// EquateEmpty allocates nothing.
func EquateEmpty() Option { return matcher.EquateEmpty() }

// EquateNaNs makes a NaN float equal another NaN of the same type, for
// the call it is passed to.
//
// The default keeps them unequal, following IEEE 754, where NaN
// compares unequal to every value including itself.
//
// # Allocation contract
//
// EquateNaNs allocates nothing.
func EquateNaNs() Option { return matcher.EquateNaNs() }

// ByIdentity makes a pointer, a map and a slice equal another only when
// both are the same object, for the call it is passed to: the same address,
// and for a slice the same length as well. It applies at every depth: at
// the top, in an element, in a map's value and in a field. A value that is
// no reference compares as without it, and a function compares by its code
// pointer under every rule.
//
// The default compares the values that two references refer to, so two
// allocations of one value are equal.
//
// # Allocation contract
//
// ByIdentity allocates nothing.
func ByIdentity() Option { return matcher.ByIdentity() }
