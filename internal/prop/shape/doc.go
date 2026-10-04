// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package shape reads the definition's shapes into generators of the
// engine. A shape is a language-neutral description of a type's values: a
// JSON object whose "shape" key names one of the 27 shapes of the
// vocabulary. A shape file is one such object. Besides its own keys it may
// state "definitions", the shapes that a "ref" names, and "source", the
// language and the type that it was read from, which no reader compares.
//
// [Read] turns a shape file into a generator. Two implementations that read
// one shape decode the same value from the same choices, so two types with
// one shape produce the same values from one seed in every language.
//
// # Values
//
// A generator of a shape decodes to a neutral value: the Go value that the
// typed literal of the definition's value decodes to.
//
//   - bool is a bool, an int of a width up to 64 an int64, or a uint64 when
//     the shape is unsigned, and an int of width 128 a *big.Int.
//   - float is a float64, or a float32 at width 32.
//   - char and string are a string, and bytes, uuid and ip-address a
//     []byte in network order.
//   - list, fixed-list and set are a []any, and map a [literal.Pairs] of
//     its entries in the order the case generated them.
//   - optional is nil or the value, record a [literal.Record], enum a
//     [literal.Variant], and literal one of its stated values.
//   - decimal, date, time-of-day, duration and offset are an int64: the
//     unscaled value, the days since 1970-01-01, the units since midnight,
//     the units, and the seconds east of UTC.
//   - instant, local-date-time, zoned-date-time and wall-time are a
//     [literal.Record] of their parts, and zone is the zone's name.
//
// Each generator runs backwards from its neutral value, and from the value
// that a typed literal of one decodes to.
//
// # Recursion
//
// A value of a recursive shape has a budget of 100 nodes. A node is the
// value of a definition that refers back to itself through a chain of refs.
// Once a value
// has used its budget, every container that refers back takes its exit: an
// optional is absent, a list, a set or a map takes no further element, and
// an enum takes its first variant that does not refer back. The exit is
// still a choice, with bounds that admit only the exit, so a replay walks
// the same positions. A definition whose chain of refs back to itself
// passes through no such container has no finite value, and fails the read.
//
// # Errors
//
// [Read] returns a fault of the kind [ErrShape] for a document that states
// a shape the vocabulary does not have, a key that a shape does not take, a
// parameter that it cannot read, a ref to no definition, or a definition
// without a finite value. The fault's path leads through the document to
// the part that does not read, as fields[0][1].max. A constraint never
// becomes a filter.
//
// A generator's inverse returns a fault of the kind [engine.ErrCannotInvert]
// whose path leads through the value to the part that no choice produces,
// as .id does to a record's field id.
//
// # Concurrency
//
// A generator that [Read] returns changes no state of its own. The budget
// of each value counts on the case that decodes it, so one generator serves
// any number of cases at once.
//
// # Dependency position
//
// Imports the fault, engine, matching, choice, literal and zone packages of
// this module, and the standard library.
package shape
