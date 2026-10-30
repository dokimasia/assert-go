---
"go.dokimi.dev/assert": patch
---

Fix the allocation contract of `prop.RunSeed` to include the allocations of `os.Getenv` on Windows.
