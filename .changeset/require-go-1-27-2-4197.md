---
"go.dokimi.dev/assert": patch
"go.dokimi.dev/assert/lint/golangci": patch
---

Require Go 1.27.2, which fixes GO-2026-6604 in `os.Root` on Windows.
