// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package conformance checks this library against the vendored definition
// of the standard.
//
// The definition is language-neutral. The directory spec/ vendors it at the
// version that [Version] reports. Each language repository checks its own
// implementation in a package of its own, because the check calls the
// library under test.
//
// # Assertions
//
// The standard requires two surfaces of the same assertions under the same
// names. One stops the test at the first failure, and the other records the
// failure and returns.
//
//   - [Members] returns the exported names of a [Surface] from its source
//     files.
//   - [Assertions], [Names], [SurfaceNames] and [RelaxationNames] read the
//     definition's tables, and [Overlay] reads this language's declines. The
//     completeness tests compare the members with them.
//   - [Cases] returns the corpus cases of the assertions. [Registry] and
//     [RunSubject] drive an assertion with a case's arguments or behaviour.
//   - [Case.Check] compares the assertion's record with the one that the
//     case states.
//
// # Properties
//
// [Vectors] returns the vectors of the property engine, and [Vector.Check]
// runs one against this implementation. Each [VectorKind] checks one part
// of the engine:
//
//   - [Decoding], [Generation] and [Bridge] decode a generator from stated
//     choices, from a seed, or from a fuzzer's bytes.
//   - [Shrinking] shrinks the failure of a property of one draw.
//   - [Coverage] decides one coverage requirement.
//   - [Token] encodes choices as a replay token, or decodes one.
//   - [Store] writes a store entry, or reads the text of one file.
//   - [Behaviour] runs a body through [go.dokimi.dev/assert/prop.ForAll].
//
// A vector states its generators, predicates and bodies in a closed
// vocabulary. It states each value as a typed literal, which [Decode] reads.
//
// # Dependency position
//
// Imports the standard library, github.com/google/go-cmp, this module's
// assert, expect and prop packages, and the internal packages of the
// property engine: choice, coverage, engine, pattern, store and token. Only
// tests import it. Besides its own tests, the store tests of prop import it
// to compare the version of a stored case with [Version].
package conformance
