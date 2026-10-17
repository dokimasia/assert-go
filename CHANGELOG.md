# go.dokimi.dev/assert

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
