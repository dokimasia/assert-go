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
// An Option carries no state of its own and is safe to reuse across
// calls and across goroutines. Order does not matter, and passing one
// twice has the same effect as passing it once: each sets an
// independent flag rather than appending to a list.
type Option func(*config)

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
// a value that is present but empty are different answers, and a test
// may need to tell them apart.
func EquateEmpty() Option {
	return func(c *config) { c.equateEmpty = true }
}

// EquateNaNs makes a NaN float equal another NaN of the same type.
//
// The default keeps them unequal, following IEEE 754, where NaN
// compares unequal to every value including itself.
func EquateNaNs() Option {
	return func(c *config) { c.equateNaNs = true }
}

// Options builds the comparison option set for one call, applying opts
// in order. The returned slice is freshly allocated and the caller
// owns it.
//
// Two options always apply:
//
//   - [cmp.Exporter] admits unexported fields, so a struct compares on
//     everything it holds. Go reads these without unsafe access.
//   - A comparer answers two functions by code pointer. cmp reports
//     two non-nil functions as unequal even when they are the same
//     function, so this supplies comparison by identity rather than
//     overriding a working rule.
//
// Every other rule is cmp's own: floats compare exactly, cycles
// terminate, and values of different types never compare equal.
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

// bothFuncs reports whether x and y are both non-nil functions, which
// is the only pairing sameFunc answers. cmp's own default already
// handles nil against nil and nil against non-nil correctly.
func bothFuncs(x, y any) bool {
	return x != nil && y != nil &&
		reflect.TypeOf(x).Kind() == reflect.Func &&
		reflect.TypeOf(y).Kind() == reflect.Func
}

// sameFunc reports whether x and y point at the same code. Two
// closures over different variables share a code pointer, so this
// answers identity of the function rather than of the closure.
func sameFunc(x, y any) bool {
	return reflect.ValueOf(x).Pointer() == reflect.ValueOf(y).Pointer()
}
