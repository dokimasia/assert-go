// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import "go.dokimi.dev/assert/internal/equality"

// Option relaxes one comparison rule for the call it is passed to.
//
// An Option has no state of its own, and is safe to reuse across calls
// and across goroutines. The order of options has no effect, and passing
// one twice has the effect of passing it once: each sets an independent
// flag.
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

// rulesOf returns the rules of one call, with opts applied in order. Each
// option takes and returns the rules by value, so building them allocates
// nothing.
func rulesOf(opts []Option) equality.Rules {
	var r equality.Rules
	for _, opt := range opts {
		r = opt(r)
	}
	return r
}
