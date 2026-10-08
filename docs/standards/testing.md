<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# Testing

A test specifies behaviour that a caller can observe. Every rule in
this document applies to every package, the internal ones included.

## Black-box tests

- Every test file declares the package `x_test` for its package `x`, so
  a test calls only what the package exports.
- There is no `export_test.go` and no test file in the package `x`. Both
  declare an API that only the tests can call.
- An internal package's API is what it exports to the rest of the
  module. Its tests are black-box tests of that API.

## One test file per source file

- A source file `x.go` has its spec `x_test.go` beside it. The spec tests
  the behaviour that `x.go` implements, through the package's exported
  API.
- The tests of one source file's behaviour are all in its spec.
- Every source file has a spec except `doc.go`, which declares nothing.
  A generated file has one too: `kind.string_gen_test.go` pins every
  spelling that `kind.string_gen.go` returns.
- A source file with a build constraint has a spec with the same
  constraint.
- A test file without a source file of its name does not exist, except
  `helpers_test.go`.

## Helpers

- `helpers_test.go` contains the helpers, types and fixtures that more
  than one test file of the package uses. It is also the spec of
  `helpers.go` when the package has one.
- A helper that one test file uses is declared in that file.
- Every helper calls `t.Helper()` before anything else.

## Names

- A spec declares `Test<Subject>`, where the subject is the concept of
  its source file: `TestDecode` in `decode_test.go`.
- A subtest of the first level is named after the exported function or
  method under test, as `t.Run("List", …)`. Each of its subtests is one
  case.
- A case's name states one behaviour: a verb in the third person, the
  result, and the condition, as `returns the sixteen zones with UTC
  first`. It never starts with `should` and never contains `test`,
  `correctly` or `as expected`.
- A table of cases is a slice named `tests`, each case is `tt`, and its
  fields start with `give` and `want`.
- `Test<Subject>Allocs` checks the allocation ceilings of the functions
  of its subject, `Benchmark<Subject>` measures them, and
  `Fuzz<Function>` fuzzes one entry point.

## Determinism and isolation

- Every test and subtest calls `t.Parallel()`, except
  `Test<Subject>Allocs`, which measures with `testing.AllocsPerRun`,
  and a test that reads state of the whole process, whose doc comment
  states which state.
- A run is the same on every machine: seeds are fixed, an assertion
  never reads the wall clock, and no result depends on the local time
  zone.
- A test waits on a channel or on its context, never on
  `time.Sleep`.
- A test writes files under `t.TempDir()` and does not call the
  network.
- A test takes its context from `t.Context()`.

## Coverage

- Black-box tests cover 100% of the statements of every package.
- Code that no black-box test can execute is dead code or a missing
  test. Delete the code, or write the test that executes it.
- No statement is excluded from coverage, and no package has a lower
  threshold.

## Mutation

- `gremlins` mutates the code. The mutation stage of `ergon check`
  requires a 100% score and 100% mutator coverage for `internal/matcher`,
  `internal/equality` and `conformance`. The other packages are measured
  on demand, against the same bar.
- A mutant under which every test passes is a missing assertion, or
  code whose behaviour no caller can observe. Add the assertion, or
  delete the code.
- A new check is mutated by hand before it is merged: a gate, a
  validator, or a comparison of a conformance runner. The mutation
  breaks what the check guards, the check fails, and the code is
  restored. The commit message states the mutation.

## Benchmarks and allocation ceilings

- Every exported function and method has a benchmark under
  `bench.Start(b).MaxAllocs(n)`. A package that only test files import,
  such as `internal/matchertest`, has none, because no program runs its
  code. A registration, such as `prop.Register`, has none either: the
  definition allows one registration of a type in a test process, so no
  loop can call it twice.
- Its doc comment states the allocations in an `# Allocation contract`
  section.
- One table of cases states the ceilings of a subject. `internal/alloctest`
  checks the table in `Test<Subject>Allocs` with `expect.MaxAllocs`, so
  `go test` enforces every ceiling without `-bench` and reports each one
  that a change breaks. It measures the same table in `Benchmark<Subject>`
  under the contract.
- The tests of `internal/matcher` and `internal/equality` use the
  package `testing` alone. Their benchmarks report allocations through
  `testing.B`, and their `Test<Subject>Allocs` compares
  `testing.AllocsPerRun` with each ceiling.
- A ceiling is 0 for a function that allocates nothing. For any other
  function it is a quarter above the highest count that its check
  measures, rounded up to two significant digits, so that a platform or
  a release of Go that allocates a few more still passes. The check is
  `testing.AllocsPerRun` in the tests of `internal/matcher` and
  `internal/equality`, and `expect.MaxAllocs`, which rounds the average
  to the nearest whole number, everywhere else. The highest count is
  taken over an ordinary build and a coverage build on Linux, macOS and
  Windows, the systems that CI runs. A ceiling that rises states why in
  the commit message.
- A count is measured on a seat that writes no call record, such as a
  seat of `internal/matchertest`. An `assert.Recorder` records every
  call, so the count of a call on a recorder includes its call record.

## Fuzzing

- A function that decodes bytes or text from outside the program has a
  `Fuzz<Function>` target, such as the decoders of typed literals,
  tokens, store entries and shape files.
- `go test` runs every seed of the corpus. A fuzzing campaign runs on
  demand.

## Verdicts

- The tests of `internal/matcher` and `internal/equality` and the
  verdicts of `conformance` use the package `testing` alone. A defective
  core could pass a test that is written with itself.
- The verdicts of `conformance` and of `internal/matchertest` compare a
  value with code of their own, and never with `internal/equality`.
- Every other test uses this module's assertions.
- A test compares the values that the module reports: a failure's
  record, a fault's fields, and a call record. A seat of
  `internal/matchertest` takes each failure as its record and each fault
  as the error it is.
- The field `error` of a call record states the writer's text of a
  fault. A test builds the text that it expects with
  `matcher.RenderFault` from the fault that it expects.
- A test compares the writer's text only where no value states it: in a
  test of the writer or of a registered sentence, and in a test of what
  a seat without `Report` and `ReportFault` receives, such as the log of
  a fuzz input's test.

## Test data

- Golden files are stored under `testdata/golden` and compared with the
  `golden` package. `-update` rewrites them.
- The vendored definition is stored under `conformance/spec`. `make
  spec-sync` refreshes it, and nobody edits it by hand.
