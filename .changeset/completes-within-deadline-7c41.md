---
"go.dokimi.dev/assert": patch
---

Fix `CompletesWithin` to fail a function that returns after the deadline of its context on every clock of a seat.
