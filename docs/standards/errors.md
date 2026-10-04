# Errors and output

Everything that this module reports is a value first and text second:

- Every assertion call ends in a verdict, and a failure is a record.
- An error is a fault.
- One writer turns records and faults into the text that a seat
  receives.
- A recorded run writes a call record of every assertion call into the
  test's output.

A different writer then changes the text of the whole module without a
change to any code that reports. Programs read the call records of a
recorded run, and never parse the text.

## Failures are records

- A failed assertion reports a `Failure`: the assertion's id, the
  caller's contract, the detail, and the call site. The definition fixes
  this record, and every language reports the same one.
- The detail contains exactly the fields that the definition declares
  for the assertion, under the declared names, with the Go values.
- Every assertion reports its verdict, pass or fail, through one
  function of `internal/matcher`. That function sends a failure to the
  seat and writes the call record of a recorded run. No assertion
  builds a sentence or calls `Fatalf`, `Errorf` or `Logf` with text of
  its own.
- A property reports one record of its assertion, `prop-for-all` or the
  form's id, whose detail contains the ten fields that the definition
  declares for a run.
- When the test's seat implements `Reporter`, it receives the record.
  Otherwise it receives the writer's text of the record, through
  `Fatalf` for the aborting surface and `Errorf` for the recording one.

## Call records

The definition's RFC-0016 fixes the call record. A run is
recorded when `DOKIMI_ASSERT_RECORD` is `1`, and recording is off by
default.

- A recorded run writes one call record for every assertion call: its
  number in the test, the assertion's id, the contract, the verdict,
  whether the surface aborts, the call site, and the detail of a
  failure. A property's call record states the detail of its run on a
  pass as well.
- The calls in a body are recorded under the call that ran the body:
  every case of a property, every attempt of `Eventually`, and the check
  that `Rejects` runs. A property writes the call records of its cases
  in the order of a run on one worker, with the phase of each case.
- A test's seat receives a call record through `Attr`, under the key
  `dokimi.assert.<seq>`. `go test -json` turns it into an `attr` event of
  the test, beside the events that state the test's status. A call
  record whose line does not fit the 4,096-byte line buffer of
  `test2json` is split over attributes of the same key.
- `assert.Recorder` keeps the call records of its calls whatever
  `DOKIMI_ASSERT_RECORD` states, and `Records` returns them. The
  recorder writes none of them to the test's output, because a test
  reads the outcomes of a recorder as its subject.
- The function writes the call record before it calls `Fatalf`, which
  ends the goroutine of the test.
- The module reads `DOKIMI_ASSERT_RECORD` once per process. A value
  other than an empty one, `0` or `1` is a fault that every assertion
  reports through `Fatalf`.
- With recording off, an assertion reads the switch and does not build a
  call record. Every allocation ceiling applies to a call on a seat that
  writes no call record, such as a test's seat with recording off. A
  recorder records every call, so no ceiling is measured on one.
- A fault does not state its test. The call record of the call that
  reported the fault is in the output of the test, under its name.

## Errors are faults

Every error that this module creates is a fault, a `fault.Error`:

```go
// Error is a fault, an error of this module: what failed, where in its
// input, and why.
type Error struct {
	// Op is the operation that failed, the package and the function, as
	// "prop.ShapeOf" or "literal.Decode".
	Op string
	// Path is where in the input the fault is, and empty for an input
	// without parts.
	Path Path
	// Kind is the sentinel that errors.Is matches, and nil when a caller
	// has no condition to act on.
	Kind error
	// Reason states what is wrong, in one clause.
	Reason string
	// Err is the cause, and nil for a fault without one.
	Err error
}
```

- `Error` returns `op: path: reason: cause`, and leaves out each part
  that is empty.
- `Unwrap` returns the cause, and `Is` matches the kind, so `errors.Is`
  and `errors.As` see through every fault.
- A reason is one clause in lowercase without a final period. It states
  what is wrong with the input: `the key min states no value`.
- A path is a list of segments, outermost first: a field, an index, a
  key, the element of a type, or a variant. It renders as a selector,
  as `order.Lines[].Note` or `cases[3].detail`. A writer for programs
  reads the segments, not the text.
- A sentinel exists for each condition that a caller tests, declared as
  `var ErrX = errors.New("pkg: the condition")`. A public package
  exports the sentinels of the faults that its functions return.
- Code and tests never compare the text that `Error` renders. They
  match errors with `errors.Is` and `errors.As`, and compare the fault's
  fields.
- Misuse of `ForAll`, such as a seed variable that is no number, is a
  fault that ends the call. A seat that implements
  `matcher.FaultReporter` receives the fault as the error it is, as a
  `Reporter` receives a failure's record. Any other seat receives the
  writer's text of the fault through `Fatalf`. The call record of a
  recorded run states the verdict `error` and the same text.

## Panics

- A constructor panics when its arguments state no domain, as
  `prop.Integer(5, 1)` does. Its message starts with the package, names
  the generator by the definition's id with its arguments, and states
  what they lack: `prop: integer(5, 1) states no value`.
- A registration panics when its arguments register nothing, or when it
  would change what a read of the registry has returned:
  `prop.RegisterValues` of no value, a second registration of one type,
  and a registration after the first run of a property. Its message
  starts with the package and names the registration with its type:
  `prop: Register[shop.Status] registers shop.Status a second time`.
- An internal function panics when its caller breaks a precondition that
  every caller in the module meets, such as an index of `alphabet.Rune`
  past the end of the alphabet. No input from outside the module can
  trigger such a panic.
- No other code panics on purpose.
- A `recover` exists only where a panic is what the code observes: an
  assertion about panics, or the end of a property's case.

## Writing

The writer is the one place where a record or a fault becomes text:

```go
// Writer writes the records and the faults that this module reports.
type Writer interface {
	// Failure returns the text of a failure's record.
	Failure(f Failure) string
	// Fault returns the text of a fault.
	Fault(err error) string
}
```

- Every text that this module sends to a seat comes from the writer:
  through `Fatalf`, `Errorf`, `Logf` and the test's output. Code that
  reports passes a record or a fault, never a format string.
- The matcher sends a call record to a seat through `Attr`, in the
  encoding that the definition fixes, and never through the writer.
- The text writer is the default. It writes the sentence that a Go
  reader expects: the contract, then the detail, a diff labelled
  `-want +got` for a mismatch, and a property's outcome, draws and
  replay token.
- `prop` registers the sentence of its 39 property assertions with
  `matcher.RegisterSentence` in its `init` function, because `matcher`
  does not import `prop`. The table of sentences is complete before any
  test starts, so the writer reads it without a lock.
- `matcher.Note` writes a note's text into the log of a seat that has
  `Logf`. `matcher.NoteFault` passes a fault that does not end the call
  to a `matcher.FaultReporter`, and writes the writer's text of the fault
  into the log of any other seat that has `Logf`. `prop` reports through
  them the notes of a failing case and each fault of a store that does
  not end the call.
- The package that declares a type of a record's detail, such as
  `prop.Drawn`, implements the type's text and its typed literal. The
  writer and the encoder of call records call them, and treat every
  package alike.
- No logging framework such as `log/slog` is part of the writer or of
  the call records: levels, contexts and attribute groups model nothing
  that a record states, and `slog`'s JSON handler does not write typed
  literals.
