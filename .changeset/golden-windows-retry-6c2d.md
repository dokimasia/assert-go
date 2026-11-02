---
"go.dokimi.dev/assert": patch
---

Fix `golden` to retry a read or a rename of a golden file that another handle refuses on Windows.
