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
//     files, and [Arities] the arity of each function and method.
//   - [Assertions], [Names], [SurfaceNames] and [RelaxationNames] read the
//     definition's tables, and [Overlay] reads this language's declines. The
//     completeness tests compare the members and their arities with them.
//   - [Cases] returns the corpus cases of the assertions. [Registry] drives
//     an assertion with a case's arguments in each [Form], with the options
//     that [Relaxations] maps the case's relaxations to. [RunSubject] drives
//     an assertion on both surfaces with a behaviour that [Subjects] builds.
//   - [Case.Check] compares the assertion's record and its call record
//     with the ones that the case states.
//
// # Properties
//
// [Vectors] returns the vectors of the property engine, and [Vector.Check]
// runs one against this implementation. Each [VectorKind] checks one part
// of the engine:
//
//   - [Decoding], [Generation] and [Bridge] decode a generator from stated
//     choices, from a seed, or from a fuzzer's bytes.
//   - [Shapes] decodes a shape from a seed, and [Inverse] runs a shape or a
//     generator back from a value to its choices.
//   - [Shrinking] shrinks the failure of a property of one draw.
//   - [Coverage] decides one coverage requirement.
//   - [Token] encodes choices as a replay token, or decodes one.
//   - [Store] writes a store entry, or reads the text of one file.
//   - [Behaviour] runs a body through [go.dokimi.dev/assert/prop.ForAll],
//     and [Draws] runs one under a case of stated entries.
//   - [CallRecords] runs a body through ForAll, and compares the call
//     records of the run.
//   - [Fixtures] reads a fixture type of the definition through
//     [go.dokimi.dev/assert/prop.ShapeOf]. The package declares the 34
//     fixture types, and registers the values and the variants that two of
//     them read.
//   - [Forms] runs a property form on the behaviours that [Subjects]
//     builds, over the generator that
//     [go.dokimi.dev/assert/prop.OfShape] returns for a stated shape.
//
// A vector states its generators, predicates and bodies in a closed
// vocabulary, and each value as a typed literal.
//
// # Histories
//
// [Vectors] returns the vectors of the history and its checker as well:
//
//   - [Seam] records a script through [go.dokimi.dev/assert/history.New], or
//     intervals through [go.dokimi.dev/assert/history.FromIntervals], and
//     compares the events in the history's JSON form, or the entry that the
//     history refuses.
//   - [Linearizable] checks a recorded script through
//     [go.dokimi.dev/assert/history.Linearizable] against a spec of
//     [Specs], which builds the six named specs of the definition, and
//     compares the detail of the record. A passing check reports no record,
//     so the runner checks the steps of a pass through two more checks
//     under a budget of the steps and of one step less.
//   - [Serializable] and [SnapshotIsolation] check a recorded script of
//     list-append transactions through
//     [go.dokimi.dev/assert/history.Serializable] or
//     [go.dokimi.dev/assert/history.HasSnapshotIsolation], and compare the
//     verdict and the detail of the record.
//
// # Machines
//
// [Vectors] returns the vectors of machines too. [Machines] runs one of the
// six machine subjects of the definition, which the runner builds natively,
// through [go.dokimi.dev/assert/stateful.Steps] in a run of
// [go.dokimi.dev/assert/prop.ForAll], under the vector's setup, settings and
// trace. It compares the detail of the run, each step of the
// counterexample among its draws, or the entry and the member of the
// trace's refusal. [Overlay] states how this library runs the concurrent
// section of a machine.
//
// # Files
//
// [Vectors] returns the vectors of the ten assertions that read files as
// well. The runner writes the tree literal of a vector's workspace into a
// directory below the vector's own, as [go.dokimi.dev/assert/files.Workspace]
// writes a tree, runs the assertion on a recorder, and compares the verdict
// and the detail of its call record:
//
//   - [TreeEqual] and [TreeContains] compare the workspace with a stated
//     tree, and [TreeUnchanged] calls a behaviour of [Subjects] on it.
//   - [GoldenMatchTree] writes the vector's golden tree below the vector's
//     directory, which must be the working directory of the process, and
//     compares the golden tree that an update leaves.
//   - [PathAbsent], [IsFile], [IsDir], [LinksTo], [HasContent] and [HasMode]
//     check a path of the workspace.
//
// The runner does not skip a vector. On a platform whose file systems record
// no permission bits, the package's tests skip the vectors that the
// definition's rules of conformance skip there: each vector of has-mode, and
// each one that states a mode or an executable file. The tests need a
// platform that creates symbolic links.
//
// # Errors
//
// [Vector.Check], [Case.Check] and [Case.Decoded] return a fault whose path
// starts at the vector's or the case's ID and leads through its JSON to the
// part at fault. That part is an input that does not parse or that the
// vocabulary does not state, or an output that differs from the run. When
// a library that decodes the input returns a fault, such as the decoder of
// typed literals, the fault keeps its reason and continues the path.
// [Members] and [Arities] return a fault for a [Surface] whose files do
// not read or parse.
//
// The readers of the definition return no error. The embed directive
// includes every file that a reader reads, and a file that does not decode
// leaves empty the values that it fails to state.
//
// # Dependency position
//
// Imports the standard library, this module's assert, expect, files, golden,
// history, prop and stateful packages, the fault, filetree and literal
// packages, and the internal packages of the property engine: choice,
// coverage, engine, matching, shape, store and token. Only tests import it. Besides its own tests, the
// store tests of prop import it to compare the version of a stored case
// with [Version].
package conformance
