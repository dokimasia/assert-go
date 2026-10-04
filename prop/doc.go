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
// # Derived inputs
//
// [Of] returns the generator of a type's shape, which the definition fixes.
// Two types of one shape generate the same values from one seed in every
// language of the definition. The reader reads each Go type as a shape:
//
//   - bool reads as a bool.
//   - int8 to int64 and uint8 to uint64 read as an int of the type's width
//     and sign.
//   - int and uint read as an int of the platform's word.
//   - float32 and float64 read as a float of the type's width.
//   - string reads as a string.
//   - []byte reads as bytes.
//   - [N]byte reads as bytes of exactly N.
//   - uuid.UUID reads as a uuid.
//   - netip.Addr reads as an ip-address.
//   - *big.Int reads as an int of 128 bits.
//   - *big.Rat reads as a decimal of the scale that its tag states.
//   - time.Time reads as an instant at nanoseconds.
//   - time.Duration reads as a duration at nanoseconds.
//   - *time.Location reads as a zone.
//   - [WallTime] reads as a wall-time.
//   - A slice reads as a list.
//   - An array reads as a fixed-list of its length.
//   - A slice or an array of a type over uint8 reads as a list or a
//     fixed-list of integers, because only byte elements make bytes.
//   - A map to an empty struct reads as a set of its keys.
//   - Any other map reads as a map.
//   - A pointer reads as an optional.
//   - A struct reads as a record of its exported fields in declaration
//     order. Each field is named by its json tag, or by its Go name for a
//     tag that names none.
//   - A type over a basic kind reads as that kind over its whole range.
//   - A named type that refers to itself reads as a definition, named by
//     its package path and its name, and as a ref at each place it occurs.
//
// The tag prop:"-" leaves a field out. A field that the reader does not
// read keeps its zero value. The reader refuses a channel, a function, a
// complex number, a uintptr, an unsafe.Pointer and an interface without
// registered variants. Its fault is at the path of the part, such as
// Order.Lines[].Note.
//
// A field's prop tag states the constraints of its shape, as in
// `prop:"min=1,max=99"`. The keys are min, max, min_size, max_size,
// allow_nan, allow_infinity, unit, scale, version, pattern and alphabet. A
// pattern and an alphabet take the rest of the tag. The keys char, date,
// local-date-time, zoned-date-time, time-of-day and offset read a rune, a
// time.Time, a time.Duration or an integer as the shape of their name. The
// reader gives each key to the field's own shape when that shape has a
// parameter of the key's name. It gives any other key to the shape inside
// an optional, a list, a fixed-list or a set.
//
// [Register], [RegisterValues] and [RegisterVariants] state a type's
// generator, a type's values or an interface's variants for the test
// process, before the first property runs. [ShapeOf] writes a type's shape
// as a shape file, and [OfShape] generates from one. A value that Of or
// OfShape derives runs back to the choices that decode to it, so [Draws]
// runs the values that typed literals state, such as the counterexample of
// a store entry.
//
// # Property forms
//
// A property form runs one assertion of this module on each input that a
// run generates, such as [Equal] for [assert.Equal]. It takes a function of
// the input where the assertion takes the value it examines, and generates
// the input with the generator of the function's parameter type. A form of
// a relation generates the inputs that the relation takes: two for
// [Commutative] and three for [Associative].
//
//	prop.Equal(t, fastSort, referenceSort, "fastSort agrees with the reference")
//
// A form runs as [ForAll] does. Each case draws each generated input once,
// labelled input, or a, b and c, and runs the assertion with the case as its
// seat. A failing run reports one record of the form's id, such as
// prop-equal, with the detail fields of prop-for-all. Its failure is the
// record that the assertion reported for the minimal case.
//
// A form takes [FormOption] values: the [Option] values of the run, the
// relaxations of its assertion, and what [Using] and [Example] return. The
// input's generator is the one that Using states, then the one that
// [Register] states, then the generator of the type's shape. A form fails
// the test before any case runs for a relaxation of an assertion that takes
// none, for an input type that the reader refuses, and for a wrong example.
//
// # Runs
//
// A run tries the case of [Draws], the examples of a form, the property's
// stored cases, the case whose every choice is its target, then random
// cases of the run's seed, each followed by a prefix case and an edge case.
// It stops at the first failing case, which it replays, shrinks and
// explains. The defaults are 100 valid cases, a shrink budget of 2,000 runs
// and 30 seconds, a cap of 8,192 choices per case, and one worker. Each
// [Option] overrides one of them.
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
// JSON file, so the next run tries it first. The file states each drawn
// value as a typed literal. A value that Of or OfShape derives is the typed
// literal of its shape. The store is the directory
// testdata/prop/<test name> beside testdata/golden, for a seat with a Name
// method, or the directory that [Store] states. A run never overwrites or
// removes an entry: deleting one is a person's decision, as updating a
// golden file is. A damaged file fails the test, and a store that cannot be
// written is a note in the test's log.
//
// # Errors
//
// Every error of this package is a fault: its operation, such as
// prop.ShapeOf, the path to the part of the input that is wrong, and the
// reason. [ShapeOf] and [OfShape] return a fault of the kind [ErrShape]
// for a shape that the definition's rules refuse. A misuse fails the test
// before any case runs with the fault of the call, such as prop.ForAll or
// prop.Equal, at the variable, the option or the store that states the
// wrong input:
//
//	prop.ForAll: DOKIMI_ASSERT_PROP_SEED: "seven" is no decimal number below 2^64
//	prop.Equal: Example[1][0]: 12 is outside [0, 9]
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
// Imports the root package of this module, its internal fault, literal,
// matcher and record packages, its choice, coverage, engine, matching,
// random, shape, store, token and zone packages, and the standard library,
// testing included.
package prop
