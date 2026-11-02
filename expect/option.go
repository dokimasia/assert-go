// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import "go.dokimi.dev/assert/internal/matcher"

// Option changes one comparison rule for the call it is passed to:
// [EquateEmpty] and [EquateNaNs] relax a rule, [ByIdentity] narrows one,
// and [EquateBy] compares the values of one type by a transform.
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

// EquateBy makes a value of type T compare by f(value), for the call it is
// passed to. It applies at every depth: at the top, in an element, in a key
// and a value of a map and in a field, also an unexported one. A value
// matches when its type is T, or implements T when T is an interface. At one
// place at most one EquateBy applies: the result compares as it is, and its
// parts compare under every EquateBy again. EquateBy applies before
// [ByIdentity], and the last EquateBy of a type applies.
//
// The record of [Equal] states f(want) and f(got) in place of want and got
// when they match, so its diff lists the differences of the results, and the
// record of [NotEqual] states f(got). A panic of f passes through the call.
// f must not change its argument, and its result must not contain its
// argument.
//
//	expect.Equal(t, got, want, "the stored book is the written book",
//	    expect.EquateBy(prototext.Format))
//
// EquateBy keeps f, so the option is safe for concurrent use when f is. It
// panics for a nil f.
//
// # Allocation contract
//
// EquateBy allocates twice: its transform and the option. Each run of f
// allocates what f allocates and its result.
func EquateBy[T, R any](f func(T) R) Option {
	if f == nil {
		panic("expect: EquateBy(nil) states no transform")
	}
	return matcher.EquateBy(f)
}
