// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package prop checks properties over generated inputs: [ForAll] runs a
// body against many generated cases and fails with the smallest
// counterexample it finds, and [Fuzz] runs the same body under go test
// -fuzz.
//
// A body receives a [Case], draws its inputs from generators with
// [Case.Draw], and asserts on the case as on any seat:
//
//	func TestRoundTrip(t *testing.T) {
//		prop.ForAll(t, "decoding undoes encoding", func(c *prop.Case) {
//			v := c.Draw(prop.List(prop.Integer[int64](-1000, 1000), prop.MaxSize(16)), "values")
//			got, err := Decode(Encode(v))
//			assert.NoError(c, err, "decoding succeeds")
//			assert.Equal(c, got, v, "decoding returns the encoded values")
//		})
//	}
//
// The definition fixes the random source, the decoding of every generator,
// the phases of a run and the shrink passes, so one seed gives the same
// inputs, and one failure the same counterexample, in every implementation
// of the definition.
//
// # Generators
//
// A [Generator] decodes a value from the choices of a case. The
// generators are [Integer], [Float], [Boolean], [Just], [SampledFrom],
// [OneOf], [Optional], [List], [Dict], [String], [Bytes], [Duration],
// [Permutation], [StringMatching] and [Recursive]. [Generator.Map],
// [Generator.Filter], [Generator.Bind] and [Composite] build a generator
// from others. A constructor panics when its arguments state no domain,
// such as Integer(5, 1), because the definition requires such a generator
// to fail when it is built.
//
// # Runs
//
// A run tries the property's stored cases, the case whose every choice is
// its target, then random cases of the run's seed, each followed by a
// prefix case and an edge case. It stops at the first failing case, which
// it replays, shrinks and explains. The defaults are 100 valid cases, a
// shrink budget of 2,000 runs and 30 seconds, a cap of 8,192 choices per
// case, and one worker. Each [Option] overrides one of them.
//
// The seed is random for each run, unless [Seed] states one, the variable
// DOKIMI_ASSERT_PROP_SEED states one in decimal, or the variable
// DOKIMI_ASSERT_PROP_PROFILE names the ci profile, which derives the seed
// from the property's contract. The profile default keeps the random seed.
// [Replay], or the variable DOKIMI_ASSERT_PROP_REPLAY, runs the one case of
// a replay token.
//
// # Store
//
// A test's store keeps the smallest failing case of each failure as one
// JSON file, so the next run tries it first. It is the directory
// testdata/prop/<test name> beside testdata/golden, for a seat with a Name
// method, or the directory that [Store] states. A run never overwrites or
// removes an entry: deleting one is a person's decision, as updating a
// golden file is. A damaged file fails the test, and a store that cannot be
// written is a note in the test's log.
//
// # Concurrency
//
// A [Generator] is a value that no case changes, so one generator serves
// any number of cases at once. Every method of [Case] is safe for
// concurrent use. [Workers] runs cases, shrink candidates and explanation
// fillings at once, and the body concurrently with itself. A run on any
// number of workers reports what a run on one reports.
//
// # Dependency position
//
// Imports the root package of this module, its internal matcher package,
// its choice, coverage, engine, pattern, random, store and token packages,
// and the standard library, testing included.
package prop
