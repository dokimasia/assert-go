// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// rejectionsPerValue is the most cases that Values decodes and rejects for
// each value that it returns, the ratio at which a run gives up.
const rejectionsPerValue = 10

// Values returns n values of g for seed: the values of the random cases of
// seed, in order, without the cases that g rejects. Case i draws from the
// random source of seed + i, as a random case of a run with seed does, so
// the values are the outputs of the generation and shapes vectors of the
// definition for the seed. Two calls with one seed return the same values,
// and a call with a larger n returns the same first values.
//
// A benchmark takes its inputs from Values, so that every run measures the
// same inputs:
//
//	ids := prop.Values(prop.Of[string](), 42, 100)
//	for b.Loop() {
//	    for _, id := range ids {
//	        _, _ = store.Get(ctx, id)
//	    }
//	}
//
// Values runs no property. It takes no simplest case and no boundary case,
// does not shrink, records no assertion, writes no stored case and reads
// none of the variables DOKIMI_ASSERT_PROP_*.
//
// Each option of [Using] makes its generator the generator of its type, as
// it does in a run: of each draw of [Of] of the type, and of each value of
// the type in a value that a draw of Of decodes. Inside the stated generator
// of a type, a draw of Of of that type takes the generator without it.
// Without such an option, a draw of Of in g uses the registration or the
// shape of its type. The zero Option changes nothing.
//
//	members := prop.SampledFrom(Pending, Paid, Shipped)
//	orders := prop.Values(prop.Of[Order](), 42, 100, prop.Using(members))
//
// # Panics
//
// Values panics for n below 0, for an option that states a setting other
// than Using, and when g rejects more than ten cases for each of the n
// values. A panic of g passes through.
//
// # Allocation contract
//
// Values allocates the slice of n values, and for each case that it decodes
// the case, the goroutine and the record of the case: 14 allocations for
// one value of an integer. The options of Using add the generators of the
// call, as they add them to a run.
func Values[T any](g Generator[T], seed uint64, n int, opts ...Option) []T {
	if n < 0 {
		panic(fmt.Sprintf("prop: Values(g, %d, %d) states a count below 0", seed, n))
	}
	for _, o := range opts {
		if o.set != nil && !o.fromUsing {
			panic(fmt.Sprintf("prop: Values(g, %d, %d) states an option other than Using", seed, n))
		}
	}
	decode := engine.Generator[T](g)
	if gs := generatorsOf(configure(opts).using); gs != nil {
		decode = engine.Within(decode, gs.engineUsing)
	}
	out := make([]T, 0, n)
	rejected := 0
	for index := uint64(0); len(out) < n; index++ {
		var value T
		e := engine.Generate(func(c *engine.Case) { value = decode.Decode(c) }, seed, index, matcher.System{})
		switch e.Status {
		case engine.CaseRejected:
			rejected++
			if rejected > rejectionsPerValue*n {
				panic(fmt.Sprintf("prop: Values(g, %d, %d) stops after g rejects %d cases for %d values",
					seed, n, rejected, n))
			}
		case engine.CaseFailed:
			panic(e.Panic)
		default:
			out = append(out, value)
		}
	}
	return out
}
