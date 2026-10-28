# go.dokimi.dev/assert

## 0.11.0

### Minor Changes

- a94b83a: Implement definition 13.1.0 and write its version into every call record.
- f9f356f: Implement definition 13.2.0 and write its version into every call record.
- f6a47ec: Implement definition 13.3.0 and write its version into every call record.
- f9f356f: Add `MaxLiveBytes` to `assert` and `expect`, a ceiling on the heap that a call keeps reachable at each call of a probe.
- a94b83a: Add `Observer` and `Call` to `assert` and `expect`, so a seat observes every assertion call, a passing one included.
- f6a47ec: Let `history.Serializable` and `history.HasSnapshotIsolation` check histories of read-write register transactions.

## 0.10.0

### Minor Changes

- 3c2f787: Add `Recorder.WithTest`, and run a property under `prop.Around` on the seat of `Rejects` and on a recorder of a test.
- 3c2f787: **Breaking:** End the call of a property with the fault that ends a call in one of its cases.
- 3c2f787: Implement definition 13.0.0 and write its version into every call record.

### Patch Changes

- e86ec14: Fix `CompletesWithin` to fail a function that returns after the deadline of its context on every clock of a seat.

## 0.9.0

### Minor Changes

- ee39282: Read `complex64` and `complex128` in `prop.Of` and `prop.ShapeOf` as a record of the floats `real` and `imag`.
- ee39282: Read a `*big.Rat` without the tag key `scale` in `prop.Of` and `prop.ShapeOf` as a fraction of the integers `num` and `den`.
- ee39282: Implement definition 12.2.0 and write its version into every call record.
- ee39282: Read a `uintptr` in `prop.Of` and `prop.ShapeOf` as an unsigned int of the platform's word.

## 0.8.0

### Minor Changes

- e736be5: Implement definition 12.0.0 and write its version into every call record.
- 6e5a88c: Implement definition 12.1.0 and write its version into every call record.
- e736be5: **Breaking:** Decode each value of a type with the position generator inside the extension of a stated `prop.Recursive` generator of the type.
- 6e5a88c: Add `Skipf` to the seat of the check of `Rejects`, and end the call as skipped when the check skips before it fails.
- 6e5a88c: Write the reason of every skip into the field `reason` of its call record.

## 0.7.1

### Patch Changes

- 987000d: Fix `prop.Of` to refuse a type at the draw, so `prop.Using` can state the generator of an interface without variants.

## 0.7.0

### Minor Changes

- 3685cf0: Add `prop.Case.Skipf` to end the run of a property as skipped and skip its test.
- 3685cf0: Implement definition 11.0.0 and write its version into every call record.

## 0.6.0

### Minor Changes

- 98515bc: Add `AllocsChecked` to `assert` and `expect` to report whether a run checks allocation ceilings.
- ff0c1d8: Add `Fault` and `FaultReporter` to `assert` and `expect`, and `Recorder.ReportFault` and `Recorder.Faults`.
- 37ff68d: Implement definition 10.0.0 and write its version into every call record.
- 4f66d7d: Implement definition 10.1.0 and write its version into every call record.
- 7d6a547: Add `prop.Values` to return the values of a generator for a seed.
- ff5e075: Let `prop.Values` take the options of `prop.Using`.
- 3ef0907: **Breaking:** End the call of `Rejects` with the fault that ends its check.
- 98515bc: **Breaking:** Count an allocation ceiling ten times at most, in place of three.
- 4d42a76: Make `prop.Using` an `Option`, which `ForAll` and `Fuzz` take.

## 0.5.1

### Patch Changes

- c617260: Fix `files.Workspace`, `files.Write` and `golden.MatchTree` to make a link to a directory a directory link on Windows.

## 0.5.0

### Minor Changes

- 869c053: **Breaking:** Deprecate `prop.Bubble`, which now returns `prop.Around(synctest.Test)`.
- 869c053: Implement definition 9.0.0 and write its version into every call record.
- 869c053: Add `prop.Around` to run each run of a property's body inside a wrapper, such as `synctest.Test`.

## 0.4.0

### Minor Changes

- e35bb35: Add `prop.Bubble` to run each run of a property's body in a `testing/synctest` bubble of its own, under `ForAll`, the forms and `Fuzz`.
- 5967735: Add `Unwrap() TB` to a seat that wraps another, so that an assertion on the wrapper uses the call records and methods of the wrapped seat.

## 0.3.1

### Patch Changes

- 919b412: Fix `bench.Contract.Excluding` to stop the timer of `testing.B` while its work runs.

## 0.3.0

### Minor Changes

- cbb413f: **Breaking:** Time the iterations of a contract only under a latency ceiling, and panic for `MaxLatency` or `MaxMean` after the body has run.

### Patch Changes

- 5e9902f: Compare two slices of bytes in one step, as `bytes.Equal` does.
- c82a1c3: Document how `MaxAllocs` and `bench.Contract.MaxAllocs` differ at a ceiling of 0.
- 5297b64: Measure the budget of a campaign, the time of a shrink and the time limit of a history check in real time inside a `testing/synctest` bubble.
- d6af09b: Write a changed file of a golden tree through a temporary file and a rename, so a failed update leaves the file as it was.
- 208ba37: Write an updated golden file through a temporary file and a rename, so a failed update leaves the golden file as it was.

## 0.2.0

### Minor Changes

- 7860800: **Breaking:** Admit one allocation at a ceiling of 0, count up to three times, and state `total` in the failure of `MaxAllocs` and `MaxAllocsWithSetup`.
- 7860800: Implement definition 8.0.0 and write its version into every call record.

## 0.1.1

### Patch Changes

- 3171d0b: Give each allocation ceiling of the tests a quarter of headroom over the highest count that Linux, macOS and Windows measure.
- bc4236a: Require Go 1.27.2, which fixes GO-2026-6604 in `os.Root` on Windows.

## 0.1.0

### Minor Changes

- bb56ca4: Move the repository to the baseline of ergon under the copyright of Dokimasia B.V., and fix the findings of its linters. The public API does not change.

### Patch Changes

- 8e1a612: Require Go 1.27.1.
