// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"reflect"

	"go.dokimi.dev/assert/internal/equality"
)

// Option changes one comparison rule for the call it is passed to: it
// relaxes a rule, narrows the comparison of references to their identity,
// or compares the values of one type by a transform.
//
// An Option is safe to reuse across calls, and across goroutines when the
// transform of EquateBy is. Passing an option twice has the effect of passing
// it once. The order of options matters only for two transforms that one
// value matches, of which the last applies.
type Option func(equality.Rules) equality.Rules

// FormSeal is the parameter type of the method that makes a type an
// option of a property form. No package outside this module names it, so
// no type outside this module is such an option.
type FormSeal struct{}

// FormOption makes an Option an option of a property form, which relaxes
// the comparison of the form's assertion in every case of its run.
//
// # Allocation contract
//
// FormOption allocates nothing.
func (Option) FormOption(FormSeal) {}

// EquateEmpty makes a nil map or slice equal an empty one of the same
// type.
//
// The default keeps them distinct, because a value that is absent and
// a value that is present but empty are different results, and a test
// may need to tell them apart.
//
// # Allocation contract
//
// EquateEmpty allocates nothing.
func EquateEmpty() Option {
	return func(r equality.Rules) equality.Rules {
		r.EquateEmpty = true
		return r
	}
}

// EquateNaNs makes a NaN float equal another NaN of the same type.
//
// The default keeps them unequal, following IEEE 754, where NaN
// compares unequal to every value including itself.
//
// # Allocation contract
//
// EquateNaNs allocates nothing.
func EquateNaNs() Option {
	return func(r equality.Rules) equality.Rules {
		r.EquateNaNs = true
		return r
	}
}

// ByIdentity makes a pointer, a map and a slice equal another only when both
// are the same object: the same address, and for a slice the same length
// as well. The rule applies at every depth: at the top, in an element, in a
// map's value and in a field. A value that is no reference compares as
// without it, and EquateEmpty does not apply to a slice or a map under it.
//
// The default compares the values that two references refer to, so two
// allocations of one value are equal.
//
// # Allocation contract
//
// ByIdentity allocates nothing.
func ByIdentity() Option {
	return func(r equality.Rules) equality.Rules {
		r.ByIdentity = true
		return r
	}
}

// EquateBy makes a value of type T compare by f(value): at the top, in an
// element, in a key and a value of a map and in a field, also an unexported
// one. A value matches when its type is T, or implements T when T is an
// interface. At one place at most one EquateBy applies: the result compares
// as it is, and its parts compare under every EquateBy again. EquateBy
// applies before ByIdentity, and the last EquateBy of a type applies. A panic
// of f passes through the call.
//
// The record of Equal states f(want) and f(got) in place of want and got when
// they match, and the record of NotEqual f(got) in place of got.
//
// # Allocation contract
//
// EquateBy allocates its transform and the closure of the option. A call that
// takes the option allocates the list of its transforms, and each run of f
// allocates what f allocates and its result.
func EquateBy[T, R any](f func(T) R) Option {
	t := equality.Transform{
		In:    reflect.TypeFor[T](),
		Apply: func(v reflect.Value) reflect.Value { return reflect.ValueOf(f(v.Interface().(T))) },
	}
	return func(r equality.Rules) equality.Rules { return r.WithTransform(t) }
}

// rulesOf returns the rules of one call, with opts applied in order. Each
// option takes and returns the rules by value, so building them allocates
// nothing, except the list of the transforms of EquateBy.
func rulesOf(opts []Option) equality.Rules {
	var r equality.Rules
	for _, opt := range opts {
		r = opt(r)
	}
	return r
}
