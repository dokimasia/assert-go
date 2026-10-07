// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package lint

// measures are the functions whose value a check of a measurement reads,
// each with the rule that reports the check and the assertion that states
// it.
var measures = []struct {
	fn, rule, assertion string
}{
	{"testing.AllocsPerRun", "max-allocs", "MaxAllocs or MaxAllocsWithSetup"},
	{"runtime.NumGoroutine", "goroutine-leaks", "NoGoroutineLeaks"},
	{"(testing.BenchmarkResult).AllocsPerOp", "bench-max-allocs", "bench.Contract.MaxAllocs"},
	{"(testing.BenchmarkResult).AllocedBytesPerOp", "bench-max-bytes", "bench.Contract.MaxBytes"},
	{"(testing.BenchmarkResult).NsPerOp", "bench-max-mean", "bench.Contract.MaxMean"},
	{"(*testing.B).Elapsed", "bench-max-mean", "bench.Contract.MaxMean"},
}

// measurement reports a check of a value of a function of measures. The
// assertion measures as the definition fixes it: MaxAllocs counts 100 calls
// and rounds their average to the nearest whole number, NoGoroutineLeaks
// follows the goroutines that its scope starts, and a contract measures the
// iterations of its benchmark. The rule suggests no fix, because the
// hand-written check measures otherwise.
//
// Of a call of an assertion, the rule reads the value under test, the first
// value: a measurement that a check takes as its ceiling, such as the count
// of a baseline in MaxAllocs, is no value that the check measures.
func measurement(p *pass, c check) bool {
	values := c.values()
	if c.cond == nil {
		values = values[:min(1, len(values))]
	}
	for _, value := range values {
		for _, m := range measures {
			if n := p.produced(value, m.fn); n != nil {
				p.report(c.node, m.rule, m.assertion+" of "+p.brief(n), nil)
				return true
			}
		}
	}
	return false
}
