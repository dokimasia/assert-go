// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"reflect"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Option relaxes one comparison rule for the call it is passed to.
//
// An Option has no state of its own, and is safe to reuse across calls
// and across goroutines. The order of options has no effect, and passing
// one twice has the effect of passing it once: each sets an independent
// flag.
type Option func(*config)

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

// config is the relaxation set an [Option] list builds up. The zero
// value applies no relaxation, which is the default comparison.
type config struct {
	// equateEmpty admits cmpopts.EquateEmpty.
	equateEmpty bool
	// equateNaNs admits cmpopts.EquateNaNs.
	equateNaNs bool
}

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
	return func(c *config) { c.equateEmpty = true }
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
	return func(c *config) { c.equateNaNs = true }
}

// Options returns the comparison options of one call, with opts applied
// in order. Each call returns a new slice, which the caller may change.
//
// Two options always apply:
//
//   - [cmp.Exporter] admits unexported fields, so a struct compares on
//     every field it contains. Go reads these without unsafe access.
//   - A comparer compares two functions by code pointer. cmp reports
//     two non-nil functions as unequal even when they are the same
//     function, so this comparer adds comparison by identity.
//
// Every other rule is cmp's own: floats compare exactly, cycles
// terminate, and values of different types never compare equal.
//
// # Allocation contract
//
// Options of one option allocates 9 times: the slice, the two options that
// always apply, which it builds on every call, and the option of opts.
func Options(opts ...Option) []cmp.Option {
	var c config
	for _, opt := range opts {
		opt(&c)
	}

	out := []cmp.Option{
		cmp.Exporter(func(reflect.Type) bool { return true }),
		cmp.FilterValues(bothFuncs, cmp.Comparer(sameFunc)),
	}
	if c.equateEmpty {
		out = append(out, cmpopts.EquateEmpty())
	}
	if c.equateNaNs {
		out = append(out, cmpopts.EquateNaNs())
	}
	return out
}

// bothFuncs reports whether x and y are both non-nil functions, the only
// pair that sameFunc compares. cmp's own default compares nil with nil
// and nil with a non-nil function.
func bothFuncs(x, y any) bool {
	return x != nil && y != nil &&
		reflect.TypeOf(x).Kind() == reflect.Func &&
		reflect.TypeOf(y).Kind() == reflect.Func
}

// sameFunc reports whether x and y point at the same code. Two
// closures over different variables share a code pointer, so sameFunc
// compares the identity of the function, not of the closure.
func sameFunc(x, y any) bool {
	return reflect.ValueOf(x).Pointer() == reflect.ValueOf(y).Pointer()
}
