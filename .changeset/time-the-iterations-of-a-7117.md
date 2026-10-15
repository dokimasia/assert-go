---
"go.dokimi.dev/assert": minor
---

**Breaking:** Time the iterations of a contract only under a latency ceiling, and panic for `MaxLatency` or `MaxMean` after the body has run.
