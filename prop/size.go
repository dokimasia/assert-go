// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// lengths are the bounds that the options of one generator state on the
// length of its value.
type lengths struct {
	// least is the shortest length.
	least int
	// most is the longest length, when bounded is set.
	most int
	// bounded reports whether the options state a longest length.
	bounded bool
}

// sizes returns the lengths as the engine's sizes: from least to most, or
// least and more without a longest length. It panics when least exceeds
// most.
func (l lengths) sizes() choice.Sizes {
	var s choice.Sizes
	var err error
	if l.bounded {
		s, err = choice.NewSizes(l.least, l.most)
	} else {
		s, err = choice.NewUnboundedSizes(l.least)
	}
	if err != nil {
		panic(fmt.Sprintf("prop: MinSize(%d) and MaxSize(%d) state no length", l.least, l.most))
	}
	return s
}

// SizeOption bounds the length of a list, a dict, a string or a byte
// string. A SizeOption is also a [ListOption] and a [StringOption]. Without
// one, a length is 0 or more, and a later option overrides an earlier one
// of the same bound. The zero SizeOption changes nothing.
type SizeOption struct {
	// bound applies the option to the lengths being stated.
	bound func(*lengths)
}

var (
	_ ListOption   = SizeOption{}
	_ StringOption = SizeOption{}
)

// MinSize makes n the shortest length. A collection decodes its first n
// elements without a choice to stop. It panics for n below 0.
func MinSize(n int) SizeOption {
	if n < 0 {
		panic(fmt.Sprintf("prop: MinSize(%d) is below 0", n))
	}
	return SizeOption{bound: func(l *lengths) { l.least = n }}
}

// MaxSize makes n the longest length. A collection of n elements decodes
// no choice to continue. It panics for n below 0.
func MaxSize(n int) SizeOption {
	if n < 0 {
		panic(fmt.Sprintf("prop: MaxSize(%d) is below 0", n))
	}
	return SizeOption{bound: func(l *lengths) { l.most, l.bounded = n, true }}
}

// applyList applies the option to the options of a list.
func (o SizeOption) applyList(l *listing) {
	o.apply(&l.lengths)
}

// applyString applies the option to the options of a string.
func (o SizeOption) applyString(s *stringing) {
	o.apply(&s.lengths)
}

// apply applies the option to l.
func (o SizeOption) apply(l *lengths) {
	if o.bound != nil {
		o.bound(l)
	}
}

// sizesOf returns the sizes that opts state, in order.
func sizesOf(opts []SizeOption) choice.Sizes {
	var l lengths
	for _, o := range opts {
		o.apply(&l)
	}
	return l.sizes()
}
