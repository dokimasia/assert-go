# go.dokimi.dev/assert/lint

## 0.1.1

### Patch Changes

- 46a134b: Attach the binaries of assertlint to each release, with deb, rpm and apk packages, a cask in dokimasia/homebrew-tap, checksums signed with cosign, SBOMs and SLSA provenance.
- bc4236a: Require Go 1.26.9, which fixes GO-2026-6604 in `os.Root` on Windows.

## 0.1.0

### Minor Changes

- bb56ca4: Move the repository to the baseline of ergon under the copyright of Dokimasia B.V., and fix the findings of its linters. The public API does not change.
