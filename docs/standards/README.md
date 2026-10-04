# Standards

A standard states a rule that every package of this module follows. An
RFC decides one design and argues for it. A standard lists the rules
that apply to every change, and it changes only through an RFC that
names the rule it amends.

A change brings each file that it touches up to these standards.

| Standard | Covers |
|---|---|
| [Code](code.md) | Packages, files, one implementation per concept, interfaces, names, doc comments, state |
| [Testing](testing.md) | Black-box tests, one test file per source file, helpers, coverage, mutation, benchmarks, fuzzing |
| [Errors and output](errors.md) | Failure records, call records, faults, panics, and the one writer that turns records and faults into text |

## Enforcement

A tool checks a rule wherever one can. Review checks every other rule.

| Rule | Checked by |
|---|---|
| Formatting and import groups | `gofumpt`, `gci`, `goimports` and `golines`, through `ergon lint` |
| A test helper marks its frame | `thelper` |
| A test runs in parallel | `tparallel` |
| A test takes its context from the test | `usetesting` |
| 100% statement coverage of every package | The coverage stage of `ergon check` |
| 100% mutation score of `internal/matcher` and `conformance` | The mutation stage of `ergon check` |
| The text of `errors.New` starts with the package name | `ergon lint error-prefix` |
| A skipped test states an expiry date | `ergon lint skip-expiry` |
| A source file starts with the license header | `ergon lint license` |
| The public names match the definition's naming table | The completeness gate in `conformance` |
| Every allocation ceiling is met | The `Allocs` tests and the benchmark contracts |
| Every other rule of these standards | Review |
