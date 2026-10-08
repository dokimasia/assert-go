// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package engine runs a property: it decodes typed values from the choices
// of a case, runs the body over generated cases, shrinks a failing case
// and explains it.
//
// A [Generator] never reads the random source itself. It asks a [Case] for
// choices with bounds, and the case's provider supplies the value: drawn
// from the random source while generating, read back from recorded choices
// while replaying, given by the edge phase, or decoded from a fuzzer's
// bytes. The case records every value with its bounds and the spans of the
// generators that asked for it, and the shrinker edits that record.
//
// # Cases
//
// A body runs on a goroutine of its own. Its assertions report to a
// [Case], which forwards each report to an [assert.Recorder], so a fatal
// failure ends that goroutine through [runtime.Goexit], as testing ends a
// test. A rejection, a repeated case, a divergence and a case past its cap
// on choices end it the same way. Deferred calls run, and a recover returns
// nil during a Goexit, so a body that recovers every panic cannot run on
// past the end of its case.
//
// A failure's identity is its assertion and the innermost frame of the
// caller's code, and for a panic the type of its value with that frame.
// That frame is the first whose file is a test file or whose function is
// outside this module and the runtime, the frame that every assertion's
// record names.
//
// # Panics
//
// A generator constructor panics when its arguments state no domain, such
// as an integer generator whose lower bound exceeds its upper bound. The
// definition requires such a generator to fail when it is built, and
// regexp.MustCompile takes the same position for a literal.
//
// # Concurrency
//
// A [Generator] is a value that no case changes, so one generator serves
// any number of cases at once. Every method of [Case] is safe for
// concurrent use.
//
// [Settings.Workers] runs up to that many cases, shrink candidates and
// explanation fillings at once, and its body concurrently with itself. The
// runner takes the cases in the order one worker runs them, so a run on
// any number of workers reports what a run on one reports. A case that a
// worker runs ahead keeps every choice it made, the attempts that a filter
// removed from its record included, and the runner enters them into the
// case tree as the case would have walked it on one worker.
//
// # Running backwards
//
// Every generator of the definition runs backwards: [Invert] returns the
// choices that decode to a value, which a run tries as an example through
// [Settings.Examples]. A filter runs backwards through its source, and
// [Generator.MapBack] through the inverse it states. [Generator.Map],
// [Generator.Bind] and [Composite] apply functions without an inverse, so
// an example of such an input states its values, and its draws take them
// without a choice. The case of [Settings.Draws] runs each draw's generator
// backwards from the draw's entry.
//
// # Generators outside this package
//
// A package that builds a generator of the definition, as the matching
// package builds string-matching, constructs it with [NewInvertible], or
// with [NewGenerator] when it has no inverse. Its decode uses these
// primitives:
//
//   - [Case.Integer] makes a value choice, and [Case.Structure] a choice
//     that decides structure.
//   - [Case.Span] groups choices in a span with a label.
//   - [Collect] repeats choices as the elements of a collection.
//
// Its inverse returns a [Step] for each choice that its decode makes.
//
// # Machines
//
// A machine's run of steps, which a package outside this one builds, asks
// the case for its choices through these primitives:
//
//   - [Case.Keep] decides whether the case keeps an action of the swarm.
//   - [Case.Continue] decides whether the run takes another step, and
//     [Case.Weighted] chooses the action of the step.
//   - [Case.Uniform] chooses the client of a step of a concurrent section.
//   - [Case.SpanFrom] puts a step in a span from its continue flag, and
//     [Case.Step] records the step.
//
// [Case.Repeat] runs a case again on its choices, for a section whose
// clients run on threads. The case of [Settings.Draws] follows its entries
// through a [Trace], which turns each step entry into the choices of one
// step.
//
// # Campaigns
//
// [Campaign] runs the cases that [Run] tries first, and then explores until
// [Settings.Budget] has passed on [Settings.BudgetClock]: the random cases
// of the seed in order, and mutations of the cases that joined its pool by
// a new label, fingerprint or score, which [Case.Target] records. It
// concludes each failure of an identity of its own as Run concludes its
// failure, and goes on.
//
// # Errors
//
// Running a generator backwards returns a fault of the kind
// [ErrCannotInvert], whose path leads to the part of the value that no
// choice produces, or one of the kind [ErrNoInverse] for a generator on the
// way that has no inverse. The case of [Settings.Draws] ends with the
// refusal of an entry, a fault whose path starts at the entry's index and
// leads through its label or its value.
//
// # Clocks
//
// Every case reads [Settings.Clock], the test's clock. [Settings.ShrinkTime]
// is measured on [Settings.ShrinkClock], the platform clock by default. A
// test's controlled clock advances only when the test advances it, so a
// cap on that clock would never end a shrink.
//
// # Dependency position
//
// Imports the root package of this module and its history package, its
// internal matcher, fault, literal, record and cycle packages, its choice,
// random, alphabet, token, coverage and tree packages, and the standard
// library.
package engine
