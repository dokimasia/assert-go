# go.dokimi.dev/assert

## 0.1.1

### Patch Changes

- 3171d0b: Give each allocation ceiling of the tests a quarter of headroom over the highest count that Linux, macOS and Windows measure.
- bc4236a: Require Go 1.27.2, which fixes GO-2026-6604 in `os.Root` on Windows.

## 0.1.0

### Minor Changes

- bb56ca4: Move the repository to the baseline of ergon under the copyright of Dokimasia B.V., and fix the findings of its linters. The public API does not change.

### Patch Changes

- 8e1a612: Require Go 1.27.1.
