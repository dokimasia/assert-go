---
"go.dokimi.dev/assert": patch
---

Cut the allocations of a passing `history.Linearizable` of two calls on one worker from 40 to 38.
