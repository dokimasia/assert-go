# Code

These rules cover how the module is divided into packages and files,
how a concept is implemented once, and how code is documented.

## Packages

- A package has one responsibility. The first sentence of its doc
  comment states it.
- `doc.go` contains the package doc comment and nothing else. Its
  `# Dependency position` section names every package that the package
  imports.
- Packages import in one direction. The compiler's cycle check enforces
  the order, and each package's dependency position states its place in
  it.
- The public packages are `assert`, `expect`, `golden`, `bench`, `prop`
  and `history`. Everything else is under `internal/`.
- A public name either is the name that the definition's naming table
  gives Go, or states in Go a part of the definition that the table
  does not name, such as a generator's parameter as an option or the
  type of a record's field. An RFC of this repository decides each name
  of the second kind.
- A new member of the definition starts as a change in `assert-spec`:
  an RFC, the naming table and every language's overlay. This module
  implements the member after the definition states it.
- The module depends on the standard library alone. A dependency needs
  an RFC.

## Files

- A file contains one concept and is named after it: the type that it
  declares, or the operation that it implements. `decode.go` decodes a
  typed literal, and `registry.go` contains the registry.
- A file that contains two concepts is split into two files.
- A function that belongs to the concept of one file is declared in
  that file, also when other files call it: the steps of an inverse are
  in the engine's `inverse.go`.
- `helpers.go` contains the unexported helpers that more than one file
  of the package calls and that belong to no file's concept. A helper
  that one file calls is declared in that file.
- A generated file has the suffix of its generator, as
  `kind.string_gen.go` has `stringer`'s. It is committed, and
  `go generate` writes it again. `stringer` runs with `-linecomment`
  when the definition fixes the spelling of a value.

## One implementation per concept

- Each concept has one implementation in the module. A second copy, in
  the same package or in another, is a defect.
- The surfaces `assert` and `expect` are the exception that RFC-0001
  records. Each declares its own functions and chain methods, because
  `go doc` does not list the methods of a type declared in an internal
  package. Every one of them calls the one comparison in
  `internal/matcher`.
- When code repeats the same steps for many cases, write a table of the
  parts that vary and one function over the table.
- A value that the definition fixes, such as an assertion id, a field
  name or a bound, is a named constant declared once.
- The package that declares a type implements the contracts that the
  definition states for it, such as its typed literal. Every other
  package calls that implementation.

## Interfaces and dependencies

- A type has one reason to change, as a file does.
- A new case is a new table entry or a new implementation. It is never a
  new branch in a second switch.
- Every implementation of an interface passes the interface's shared
  suite. `internal/matchertest` drives the core and both surfaces with
  one set of cases.
- An interface is declared beside its consumer and declares only the
  methods that the consumer calls. `matcher.Seat` has three methods. A
  second capability is a second interface, as `Reporter` and
  `FaultReporter` are.
- The core calls a test only through the seat, the clock and the
  writer, and never depends on `testing.T`. It calls the package
  `testing` for `AllocsPerRun` alone.

## Names

- A public name is the name that the naming table gives Go.
- An id, a field name or a spelling of the definition appears in code as
  the definition spells it.
- Each type has one receiver name of one or two letters.
- A name does not abbreviate, except `ctx`, `err`, `tb`, `t` and `b`,
  which Go code uses throughout.

## Doc comments

- Every declaration has a doc comment. An unexported declaration may
  omit it when its name and its signature make the mechanism clear.
- A doc comment states the contract that a caller relies on, and not
  the implementation.
- A doc comment uses the sections `# Allocation contract`,
  `# Concurrency`, `# Errors` and `# Panics` where they apply.
- Comments, doc comments, test names and error messages state the
  present contract. History belongs in the commit message, and none of
  them cites an RFC, an ADR or a ticket.

## State

- Package-level mutable state exists only where the definition scopes a
  thing to the test process, such as the registry of `prop`. A sync
  primitive guards it, and its doc comment states the scope.
- A type whose methods are safe for concurrent use states it in a
  `# Concurrency` section. Without the section, a type is safe for one
  goroutine at a time.
