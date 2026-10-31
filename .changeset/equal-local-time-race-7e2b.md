---
"go.dokimi.dev/assert": patch
---

Fix `Equal` and every comparison of values to read `time.Local` after the time package fills it.
