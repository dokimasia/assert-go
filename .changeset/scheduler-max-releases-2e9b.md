---
"go.dokimi.dev/assert": minor
---

**Breaking:** Fail a case whose `Scheduler.Run` or `Scheduler.Advance` releases `stateful.MaxReleases` tasks, 1,000,000 by default, while a task is still ready.
