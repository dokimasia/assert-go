// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"context"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// The tests of the surfaces use the library's assertions, as a consumer of
// the library does. The assertion core is tested with testing alone,
// because a test written with a defective core can pass by the defect that
// it should find.

// TestSurface is the completeness gate. Every assertion of the definition
// is present under the name that the naming table gives Go, with the
// definition's arity, or the overlay declares it absent. Every member of
// the aborting surface has a twin of the same name in the recording
// surface, and the recording surface declares no member that the aborting
// one lacks, so the two surfaces declare the same members.
func TestSurface(t *testing.T) {
	t.Parallel()

	aborting, err := conformance.Members(conformance.Aborting)
	assert.NoError(t, err, "the aborting surface can be read")

	recording, err := conformance.Members(conformance.Recording)
	assert.NoError(t, err, "the recording surface can be read")

	assertions := conformance.Assertions()
	names := conformance.Names()
	overlay := conformance.Overlay()

	t.Run("Members", func(t *testing.T) {
		t.Parallel()

		t.Run("returns members for both surfaces", func(t *testing.T) {
			t.Parallel()

			assert.NotEmpty(t, aborting, "the aborting surface declares members")
			assert.NotEmpty(t, recording, "the recording surface declares members")
		})

		t.Run("returns the Go name of every assertion that the overlay does not declare absent", func(t *testing.T) {
			t.Parallel()

			for id, a := range assertions {
				name := names[id]
				where, member := split(name)

				surface, ok := resolve(a.Package, where)
				assert.True(t, ok, "assertion "+string(id)+" names a package this library has")

				members, err := conformance.Members(surface)
				assert.NoError(t, err, "the surface of "+string(id)+" can be read")

				present := declares(members, member)
				declared := overlay.Diverges(id)

				switch {
				case present && declared:
					t.Errorf("%s: the overlay declares it absent, but %s is implemented", id, name)
				case !present && !declared:
					t.Errorf("%s: %s is not implemented and no overlay entry declares why", id, name)
				}
			}
		})

		t.Run("returns the Go name of every relaxation that the naming table names", func(t *testing.T) {
			t.Parallel()

			for id, name := range conformance.RelaxationNames() {
				if name != "" && !declares(aborting, name) {
					t.Errorf("%s: %s is named and not implemented", id, name)
				}
			}
		})

		t.Run("returns a recording twin for every aborting member", func(t *testing.T) {
			t.Parallel()

			for _, name := range aborting {
				assert.Contains(t, recording, name,
					"the recording surface declares "+name)
			}
		})

		t.Run("returns no recording member without an aborting twin", func(t *testing.T) {
			t.Parallel()

			for _, name := range recording {
				assert.Contains(t, aborting, name,
					"the aborting surface declares "+name)
			}
		})

		t.Run("returns a fault for a surface whose directory does not exist", func(t *testing.T) {
			t.Parallel()

			_, err := conformance.Members(conformance.Surface(filepath.Join(t.TempDir(), "missing")))
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Empty(t, f.Path, "the surface itself does not read")
			assert.Equal(t, f.Reason, "the surface does not read", "the reason")
			assert.ErrorIs(t, err, fs.ErrNotExist, "the cause is the error of the file system")
		})

		t.Run("returns a fault at the file of a surface of a file that does not parse", func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			assert.NoError(t, os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package"), 0o600),
				"the file is written")
			_, err := conformance.Members(conformance.Surface(dir))
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, f.Path, fault.Path{fault.Field("broken.go")}, "the file that does not parse")
			assert.Equal(t, f.Reason, "the file does not parse", "the reason")
		})

		t.Run("returns each silent member for both surfaces", func(t *testing.T) {
			t.Parallel()

			for name, why := range silent {
				assert.Contains(t, aborting, name, "the aborting surface declares "+name+", which "+why)
				assert.Contains(t, recording, name, "the recording surface declares "+name+", which "+why)
			}
		})
	})

	t.Run("Arities", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the definition's arity for every assertion on each surface", func(t *testing.T) {
			t.Parallel()

			for id, a := range assertions {
				if overlay.Diverges(id) {
					continue
				}
				where, member := split(names[id])
				surface, ok := resolve(a.Package, where)
				assert.True(t, ok, "assertion "+string(id)+" names a package this library has")

				surfaces := []conformance.Surface{surface}
				if surface == conformance.Aborting {
					surfaces = append(surfaces, conformance.Recording)
				}
				want := a.Arity
				if excuse, excused := arityExcused[id]; excused {
					want = excuse.arity
				}

				for _, s := range surfaces {
					arities, err := conformance.Arities(s)
					assert.NoError(t, err, "the surface of "+string(id)+" can be read")

					got, declared := arities[member]
					switch {
					case !declared:
						t.Errorf("%s: %s declares no function or method %s", id, s, member)
					case got != want:
						t.Errorf("%s: %s in %s takes %d arguments, want %d", id, member, s, got, want)
					}
				}
			}
		})

		t.Run("excuses only an assertion whose arity differs", func(t *testing.T) {
			t.Parallel()

			for id, excuse := range arityExcused {
				assert.NotEqual(t, excuse.arity, assertions[id].Arity,
					"the arity of "+string(id)+" differs from the definition's, because Go "+excuse.why)
			}
		})

		t.Run("returns a fault for a surface whose directory does not exist", func(t *testing.T) {
			t.Parallel()

			_, err := conformance.Arities(conformance.Surface(filepath.Join(t.TempDir(), "missing")))
			f := assert.ErrorAs[*fault.Error](t, err, "a fault")
			assert.Equal(t, f.Reason, "the surface does not read", "the reason")
		})

		t.Run("returns the arity of each exported function and method by the definition's rule", func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			assert.NoError(t, os.WriteFile(filepath.Join(dir, "probe.go"), []byte(arityProbe), 0o600),
				"the file is written")

			arities, err := conformance.Arities(conformance.Surface(dir))
			assert.NoError(t, err, "the probe can be read")
			assert.Equal(t, arities, map[string]int{
				"Seated":      3,
				"Qualified":   1,
				"Stated":      3,
				"Inferred":    2,
				"Unseated":    2,
				"Later":       2,
				"Pair.Method": 1,
				"Probe.Paren": 1,
			}, "the seat, a variadic and an inferred type parameter do not count, and a stated one does")
		})
	})
}

// arityExcused names each assertion whose arity in Go differs from the
// definition's, with the arity that Go declares and the reason.
var arityExcused = map[conformance.ID]struct {
	arity int
	why   string
}{
	"no-task-leaks": {
		arity: 1,
		why: "marks the scope with the call and the check that the call returns, " +
			"and the definition counts the scope as an argument",
	},
}

// split separates a qualified name into the package it names and the
// member within it. An unqualified name has no package.
func split(name string) (pkg, member string) {
	where, rest, qualified := strings.Cut(name, ".")
	if !qualified {
		return "", name
	}
	return where, rest
}

// resolve returns the surface that an assertion's package names.
func resolve(declared, qualified string) (conformance.Surface, bool) {
	if declared == "" && qualified == "" {
		return conformance.Aborting, true
	}
	if declared == "" {
		declared = qualified
	}
	return conformance.Subpackage(declared)
}

// declares reports whether members contains name, or for a method the type
// that the method belongs to.
func declares(members []string, name string) bool {
	owner, _, isMethod := strings.Cut(name, ".")
	if isMethod {
		name = owner
	}
	return slices.Contains(members, name)
}

// arityProbe is a surface whose functions and methods state each case of
// the arity rule. It is parsed and never compiled, so it also states a
// receiver that names no type, which the compiler refuses.
const arityProbe = `package probe

type TB interface{ Helper() }

type Option func()

type Pair[K comparable, V any] struct{}

type Probe struct{}

type hidden struct{}

func (p (Probe)) Paren(tb TB, a int) {}

func (p []Probe) Unnamed(tb TB) {}

func Seated(tb TB, got, want any, msg string, opts ...Option) {}

func Qualified(tb assert.TB, msg string) {}

func Stated[T any](tb TB, err error, msg string) T { var zero T; return zero }

func Inferred[T any](tb TB, got T, msg string) {}

func Unseated(a, b int) {}

func Later(n int, tb TB) {}

func (p *Pair[K, V]) Method(k K) {}

func (hidden) Method(x int) {}

func unexported(tb TB) {}
`

// pinned maps each surface id that the table gives Go to a value whose
// type the compiler checks. Go looks nothing up by name at run time, so
// the compiler proves that each name exists, and TestSurfaceTable checks
// that the map lists every id of the table.
//
// The collector seat is testing.T: Fatalf stops the test, and Errorf
// records the failure and returns, which is the seat's contract. The
// conversion to assert.TB proves it at compile time.
var pinned = map[conformance.ID]any{
	"seat":             (*assert.TB)(nil),
	"recorder-seat":    (*assert.Recorder)(nil),
	"collector-seat":   assert.TB((*testing.T)(nil)),
	"scrubber":         golden.Scrubber(nil),
	"contract":         (*bench.Contract)(nil),
	"clock":            (*assert.Clock)(nil),
	"controlled-clock": (*assert.Controlled)(nil),
	"system-clock":     assert.System{},
	"failure":          assert.Failure{},
	"where":            assert.Where{},
	"reporter":         (*assert.Reporter)(nil),
	"clocked":          (*assert.Clocked)(nil),
	"assertion":        (*assert.Assertion[int])(nil),

	"failure.assertion": assert.Failure{}.Assertion,
	"failure.contract":  assert.Failure{}.Contract,
	"failure.detail":    assert.Failure{}.Detail,
	"seat.report":       assert.Reporter.Report,

	"seat.clock":               assert.Clocked.Clock,
	"clock.now":                assert.Clock.Now,
	"clock.sleep":              assert.Clock.Sleep,
	"controlled-clock.advance": (*assert.Controlled).Advance,
	"contract.excluding":       (*bench.Contract).Excluding,
	"that":                     assert.That[int],
	"recorder-seat.failures":   (*assert.Recorder).Failures,

	"seat.helper": assert.TB.Helper,
	"seat.fail":   assert.TB.Fatalf,
	"seat.record": assert.TB.Errorf,

	"recorder-seat.failed":       (*assert.Recorder).Failed,
	"recorder-seat.message":      (*assert.Recorder).Message,
	"recorder-seat.messages":     (*assert.Recorder).Messages,
	"recorder-seat.helper-calls": (*assert.Recorder).HelperCalls,
	"recorder-seat.records":      (*assert.Recorder).Records,

	"contract.loop":         (*bench.Contract).Loop,
	"contract.check":        (*bench.Contract).End,
	"contract.warmup":       (*bench.Contract).Warmup,
	"contract.run-parallel": (*bench.Contract).RunParallel,

	"golden.scrub-timestamps":  golden.ScrubTimestamps,
	"golden.scrub-hashes":      golden.ScrubHashes,
	"golden.scrub-run-ids":     golden.ScrubRunIDs,
	"golden.scrub-json-fields": golden.ScrubJSONFields,
	"golden.should-update":     golden.ShouldUpdate,

	"generator":         (*prop.Generator[int])(nil),
	"case":              (*prop.Case)(nil),
	"case.draw":         (*prop.Case).Draw[int],
	"case.assume":       (*prop.Case).Assume,
	"case.classify":     (*prop.Case).Classify,
	"case.note":         (*prop.Case).Logf,
	"case.rand":         (*prop.Case).Rand,
	"case.observe":      (*prop.Case).Observe,
	"case.cleanup":      (*prop.Case).Cleanup,
	"case.cancellation": (*prop.Case).Context,
	"case.history":      (*prop.Case).History,
	"case.target":       (*prop.Case).Target,
	"generator.map":     prop.Generator[int].Map[string],
	"generator.filter":  prop.Generator[int].Filter,
	"generator.bind":    prop.Generator[int].Bind[string],

	"prop.integer":         prop.Integer[int],
	"prop.float":           prop.Float[float64],
	"prop.boolean":         prop.Boolean,
	"prop.just":            prop.Just[int],
	"prop.sampled-from":    prop.SampledFrom[int],
	"prop.one-of":          prop.OneOf[int],
	"prop.optional":        prop.Optional[int],
	"prop.list":            prop.List[int],
	"prop.dict":            prop.Dict[string, int],
	"prop.string":          prop.String,
	"prop.bytes":           prop.Bytes,
	"prop.duration":        prop.Duration,
	"prop.permutation":     prop.Permutation[int],
	"prop.string-matching": prop.StringMatching,
	"prop.recursive":       prop.Recursive[int],
	"prop.composite":       prop.Composite[int],
	"prop.cases":           prop.Cases,
	"prop.seed":            prop.Seed,
	"prop.replay":          prop.Replay,
	"prop.require":         prop.Require,
	"prop.shrink":          prop.Shrink,
	"prop.shrink-time":     prop.ShrinkTime,
	"prop.max-choices":     prop.MaxChoices,
	"prop.store":           prop.Store,
	"prop.explain":         prop.Explain,
	"prop.workers":         prop.Workers,
	"prop.hermetic":        prop.Hermetic,
	"prop.fuzz":            prop.Fuzz,

	"prop.of":                prop.Of[int],
	"prop.of-shape":          prop.OfShape,
	"prop.shape-of":          prop.ShapeOf[int],
	"prop.register":          prop.Register[int],
	"prop.register-values":   prop.RegisterValues[int],
	"prop.register-variants": prop.RegisterVariants[any],
	"prop.using":             prop.Using[int],
	"prop.example":           prop.Example[int],
	"prop.examples":          prop.Examples[int],
	"prop.draws":             prop.Draws,

	"history":                (*history.History)(nil),
	"call":                   history.Call{},
	"event":                  history.Event{},
	"outcome":                history.Outcome{},
	"model":                  history.Model[int]{},
	"op":                     history.Op{},
	"history.new":            history.New,
	"history.invoke":         (*history.History).Invoke,
	"history.events":         (*history.History).Events,
	"call.ok":                history.Call.OK,
	"call.fail":              history.Call.Fail,
	"call.unknown":           history.Call.Unknown,
	"model.init":             history.Model[int]{}.Init,
	"model.step":             history.Model[int]{}.Step,
	"model.equal":            history.Model[int]{}.Equal,
	"history.from-intervals": history.FromIntervals,
	"history.concurrently":   history.Concurrently,
	"history.model-from":     history.ModelFrom,
	"history.budget":         history.Budget,
	"history.memo-limit":     history.MemoLimit,
	"history.time-limit":     history.TimeLimit,
	"history.workers":        history.Workers,

	"machine":           stateful.Machine[int]{},
	"machine.model":     stateful.Machine[int]{}.Model,
	"machine.actions":   stateful.Machine[int]{}.Actions,
	"machine.invariant": stateful.Machine[int]{}.Invariant,
	"machine.settle":    stateful.Machine[int]{}.Settle,
	"action":            stateful.Action[int]{},
	"action.name":       stateful.Action[int]{}.Name,
	"action.weight":     stateful.Action[int]{}.Weight,
	"action.enabled":    stateful.Action[int]{}.Enabled,
	"action.drain":      stateful.Action[int]{}.Drain,
	"action.input":      stateful.Action[int]{}.Input,
	"action.run":        stateful.Action[int]{}.Run,
	"stateful.steps":    stateful.Steps[int],
	"steps.mean":        stateful.Mean,
	"steps.max":         stateful.Max,
	"steps.swarm":       stateful.Swarm,
	"steps.clients":     stateful.Clients,
	"steps.concurrent":  stateful.Concurrent,
	"steps.scheduler":   stateful.Tasks,
	"steps.repeat":      stateful.Repeat,
	"scheduler":         (*stateful.Scheduler)(nil),
	"scheduler.new":     stateful.NewScheduler,
	"scheduler.spawn":   (*stateful.Scheduler).Spawn,
	"scheduler.yield":   (*stateful.Scheduler).Yield,
	"scheduler.run":     (*stateful.Scheduler).Run,
	"scheduler.uniform": stateful.Uniform,
	"scheduler.pct":     stateful.PCT,

	"tree":             files.Tree{},
	"entry":            files.Entry{},
	"files.workspace":  files.Workspace,
	"files.read":       files.Read,
	"files.text":       files.Text,
	"files.bytes":      files.Bytes,
	"files.executable": files.Executable,
	"files.directory":  files.Dir,
	"files.link":       files.Link,
	"entry.with-mode":  files.Entry.WithMode,
}

// TestSurfaceTable compares the pin map with the naming table: the map
// pins every id that the table names for Go and no id that the table does
// not state, and the overlay declines every id that the table does not
// name, with a reason. It is written with testing and not with the
// library, because a verdict is not written with the subject.
func TestSurfaceTable(t *testing.T) {
	t.Parallel()

	names := conformance.SurfaceNames()
	if len(names) == 0 {
		t.Fatal("the surface table states something")
	}

	overlay := conformance.Overlay()

	source, err := os.ReadFile("surface_test.go")
	if err != nil {
		t.Fatalf("this file can be read: %v", err)
	}

	for id := range pinned {
		if _, stated := names[id]; !stated {
			t.Errorf("%s: pinned here, and the table states no such id", id)
		}
	}

	for id, name := range names {
		declined := overlay.DeclinesSurface(id)
		_, isPinned := pinned[id]

		// The map is keyed by id, so it cannot notice the table
		// spelling something differently from what is pinned. The
		// spelling check is against this file's own text, on word
		// boundaries so a prefix of a longer name does not pass.
		// Nothing in a comment here may repeat a retired spelling,
		// because a comment is part of the text being searched.
		spelled := true
		if name != "" {
			leaf := name[strings.LastIndex(name, ".")+1:]
			spelled = regexp.MustCompile(`\b` + regexp.QuoteMeta(leaf) + `\b`).Match(source)
		}

		switch {
		case name != "" && declined:
			t.Errorf("%s: the table names %s and the overlay declines it, which is a contradiction",
				id, name)
		case name == "" && !declined:
			t.Errorf("%s: the table gives no Go name and the overlay does not decline it", id)
		case name != "" && !isPinned:
			t.Errorf("%s: %s is named and nothing here pins it; add it to the map", id, name)
		case name != "" && !spelled:
			t.Errorf("%s: the table names %s and nothing here spells it, so the pin and the table disagree",
				id, name)
		case name == "" && isPinned:
			t.Errorf("%s: declined in the overlay yet pinned here, which is a contradiction", id)
		}
	}
}

// rejected is the message every driven call passes.
const rejected = "the input is one the member rejects"

// silent names the functions of both surfaces that report nothing, so no
// input drives one to fail, with the reason. A type of the naming table's
// surface section reports nothing either, and needs no entry.
var silent = map[string]string{
	"Assertion":     "is the chain type, and the method drivers call its methods",
	"That":          "starts a chain",
	"Option":        "is the type of a comparison option",
	"EquateEmpty":   "returns a comparison option",
	"EquateNaNs":    "returns a comparison option",
	"ByIdentity":    "returns a comparison option",
	"NewRecorder":   "constructs the recorder seat",
	"NewControlled": "constructs the controlled clock",
}

// escaped is the slice that the MaxAllocs driver allocates, which escape
// analysis then keeps on the heap.
var escaped []byte

// recordingFunctions calls each reporting function of the recording
// surface, keyed by its name, with an input the function rejects.
var recordingFunctions = map[string]func(tb assert.TB){
	"Accumulates": func(tb assert.TB) {
		expect.Accumulates(tb, func(int) error { return nil }, 0, func() int { return 0 }, rejected)
	},
	"Associative": func(tb assert.TB) {
		expect.Associative(tb, func(a, b int) int { return a - b }, 2, 3, 5, rejected)
	},
	"CloseTo": func(tb assert.TB) { expect.CloseTo(tb, 1.0, 2, 0.1, rejected) },
	"Commutative": func(tb assert.TB) {
		expect.Commutative(tb, func(a, b int) int { return a - b }, 2, 3, rejected)
	},
	"CompletesWithin": func(tb assert.TB) {
		expect.CompletesWithin(tb, time.Millisecond, func(context.Context) error {
			time.Sleep(2 * time.Millisecond)
			return nil
		}, rejected)
	},
	"Contains":        func(tb assert.TB) { expect.Contains(tb, "abc", "x", rejected) },
	"ContainsInOrder": func(tb assert.TB) { expect.ContainsInOrder(tb, "abc", []string{"c", "a"}, rejected) },
	"Deterministic": func(tb assert.TB) {
		expect.Deterministic(tb, func(int) (int, error) { return 0, io.EOF }, 0, rejected)
	},
	"Empty":      func(tb assert.TB) { expect.Empty(tb, "a", rejected) },
	"Equal":      func(tb assert.TB) { expect.Equal(tb, 1, 2, rejected) },
	"ErrorAs":    func(tb assert.TB) { _ = expect.ErrorAs[*fs.PathError](tb, io.EOF, rejected) },
	"ErrorIs":    func(tb assert.TB) { expect.ErrorIs(tb, io.EOF, fs.ErrNotExist, rejected) },
	"ErrorIsNot": func(tb assert.TB) { expect.ErrorIsNot(tb, io.EOF, io.EOF, rejected) },
	"Eventually": func(tb assert.TB) {
		expect.Eventually(tb, time.Millisecond, time.Millisecond, func(trial assert.TB) {
			expect.True(trial, false, rejected)
		}, rejected)
	},
	"EventuallyTrue": func(tb assert.TB) {
		expect.EventuallyTrue(tb, time.Millisecond, func() bool { return false }, rejected)
	},
	"FailsAfterClose": func(tb assert.TB) {
		expect.FailsAfterClose(tb, func() error { return nil }, func() error { return nil }, io.EOF, rejected)
	},
	"False":     func(tb assert.TB) { expect.False(tb, true, rejected) },
	"HasError":  func(tb assert.TB) { expect.HasError(tb, nil, rejected) },
	"HasPrefix": func(tb assert.TB) { expect.HasPrefix(tb, "abc", "x", rejected) },
	"HasSuffix": func(tb assert.TB) { expect.HasSuffix(tb, "abc", "x", rejected) },
	"HonoursCancellation": func(tb assert.TB) {
		expect.HonoursCancellation(tb, func(context.Context) error { return nil }, rejected)
	},
	"HonoursDeadline": func(tb assert.TB) {
		expect.HonoursDeadline(tb, func(context.Context) error { return nil }, rejected)
	},
	"Idempotent": func(tb assert.TB) {
		expect.Idempotent(tb, func(int) error { return io.EOF }, 0, func() int { return 0 }, rejected)
	},
	"InRange": func(tb assert.TB) { expect.InRange(tb, 5.0, 0, 1, rejected) },
	"Length":  func(tb assert.TB) { expect.Length(tb, "ab", 3, rejected) },
	"Matches": func(tb assert.TB) { expect.Matches(tb, "abc", "^x", rejected) },
	"MaxAllocs": func(tb assert.TB) {
		expect.MaxAllocs(tb, func() { escaped = make([]byte, 64) }, 0, rejected)
	},
	"MaxAllocsWithSetup": func(tb assert.TB) {
		expect.MaxAllocsWithSetup(tb, func() int { return 64 }, func(n int) { escaped = make([]byte, n) }, 0, rejected)
	},
	"Monotonic": func(tb assert.TB) {
		expect.Monotonic(tb, func() int { return 0 }, func() error { return io.EOF }, 1, rejected)
	},
	"Nil": func(tb assert.TB) { expect.Nil(tb, 1, rejected) },
	"NilContextSafe": func(tb assert.TB) {
		expect.NilContextSafe(tb, func(ctx context.Context) error { return ctx.Err() }, rejected)
	},
	"NoDuplicates": func(tb assert.TB) {
		expect.NoDuplicates(tb, func() ([]int, error) { return []int{1, 1}, nil }, rejected)
	},
	"NoError": func(tb assert.TB) { expect.NoError(tb, io.EOF, rejected) },
	"NoGoroutineLeaks": func(tb assert.TB) {
		check := expect.NoGoroutineLeaks(tb, rejected)
		release := make(chan struct{})
		go func() { <-release }()
		check()
		close(release)
	},
	"NotContains": func(tb assert.TB) { expect.NotContains(tb, "abc", "b", rejected) },
	"NotEmpty":    func(tb assert.TB) { expect.NotEmpty(tb, "", rejected) },
	"NotEqual":    func(tb assert.TB) { expect.NotEqual(tb, 1, 1, rejected) },
	"NotNil":      func(tb assert.TB) { expect.NotNil(tb, nil, rejected) },
	"NotPanics":   func(tb assert.TB) { expect.NotPanics(tb, func() { panic(rejected) }, rejected) },
	"NotPure":     func(tb assert.TB) { expect.NotPure(tb, func() int { return 0 }, func() {}, rejected) },
	"Pairwise": func(tb assert.TB) {
		expect.Pairwise(tb, []int{2, 1}, func(earlier, later int) bool { return earlier < later }, rejected)
	},
	"Panics":      func(tb assert.TB) { expect.Panics(tb, func() {}, rejected) },
	"Permutation": func(tb assert.TB) { expect.Permutation(tb, []int{1}, []int{2}, rejected) },
	"Poisoned":    func(tb assert.TB) { expect.Poisoned(tb, func() {}, func() error { return nil }, rejected) },
	"Pure": func(tb assert.TB) {
		calls := 0
		expect.Pure(tb, func() int { return calls }, func() { calls++ }, rejected)
	},
	"Rejects": func(tb assert.TB) { expect.Rejects(tb, rejected, func(assert.TB) {}) },
	"RoundTrip": func(tb assert.TB) {
		expect.RoundTrip(tb, func(int) (int, error) { return 0, io.EOF },
			func(v int) (int, error) { return v, nil }, 1, rejected)
	},
	"StableOrder": func(tb assert.TB) {
		expect.StableOrder(tb, func() ([]int, error) { return nil, io.EOF }, rejected)
	},
	"Total": func(tb assert.TB) { expect.Total(tb, func(int) error { return io.EOF }, []int{1}, rejected) },
	"True":  func(tb assert.TB) { expect.True(tb, false, rejected) },
}

// recordingMethods calls each method of the recording chain, keyed by
// its name, on a value the method rejects.
var recordingMethods = map[string]func(tb assert.TB){
	"CloseTo":         func(tb assert.TB) { expect.That(tb, 1.0).CloseTo(2, 0.1, rejected) },
	"Contains":        func(tb assert.TB) { expect.That(tb, "abc").Contains("x", rejected) },
	"ContainsInOrder": func(tb assert.TB) { expect.That(tb, "abc").ContainsInOrder([]string{"c", "a"}, rejected) },
	"Empty":           func(tb assert.TB) { expect.That(tb, "a").Empty(rejected) },
	"Equal":           func(tb assert.TB) { expect.That(tb, 1).Equal(2, rejected) },
	"ErrorIs":         func(tb assert.TB) { expect.That(tb, io.EOF).ErrorIs(fs.ErrNotExist, rejected) },
	"ErrorIsNot":      func(tb assert.TB) { expect.That(tb, io.EOF).ErrorIsNot(io.EOF, rejected) },
	"HasError":        func(tb assert.TB) { expect.That[error](tb, nil).HasError(rejected) },
	"HasPrefix":       func(tb assert.TB) { expect.That(tb, "abc").HasPrefix("x", rejected) },
	"HasSuffix":       func(tb assert.TB) { expect.That(tb, "abc").HasSuffix("x", rejected) },
	"InRange":         func(tb assert.TB) { expect.That(tb, 5.0).InRange(0, 1, rejected) },
	"Length":          func(tb assert.TB) { expect.That(tb, "ab").Length(3, rejected) },
	"Matches":         func(tb assert.TB) { expect.That(tb, "abc").Matches("^x", rejected) },
	"Nil":             func(tb assert.TB) { expect.That(tb, 1).Nil(rejected) },
	"NoError":         func(tb assert.TB) { expect.That(tb, io.EOF).NoError(rejected) },
	"NotContains":     func(tb assert.TB) { expect.That(tb, "abc").NotContains("b", rejected) },
	"NotEmpty":        func(tb assert.TB) { expect.That(tb, "").NotEmpty(rejected) },
	"NotEqual":        func(tb assert.TB) { expect.That(tb, 1).NotEqual(1, rejected) },
	"NotNil":          func(tb assert.TB) { expect.That[any](tb, nil).NotNil(rejected) },
}

// TestSurfaceRecording drives every reporting member of the recording
// surface with an input it rejects. Each member must report through
// Errorf and never through Fatalf, and its record must name the member
// the driver is keyed by. The shared suites accept a failure from
// either path, so this is the only test that tells the two modes apart.
//
// The member list comes from the surface's source and the chain's
// method set, so a member added without a driver fails here.
//
// It does not run in parallel. The MaxAllocs driver calls
// testing.AllocsPerRun, which panics while a parallel test runs, the
// MaxAllocsWithSetup driver counts the allocations of the whole process,
// and the NoGoroutineLeaks driver reads every goroutine in the process.
func TestSurfaceRecording(t *testing.T) {
	members, err := conformance.Members(conformance.Recording)
	if err != nil {
		t.Fatalf("the recording surface can be read: %v", err)
	}

	names := conformance.Names()
	typed := slices.Collect(maps.Values(conformance.SurfaceNames()))

	var methods []string
	for method := range reflect.TypeFor[*expect.Assertion[any]]().Methods() {
		methods = append(methods, method.Name)
	}

	t.Run("Members", func(t *testing.T) {
		t.Run("returns the members that a driver, an excuse or the table names", func(t *testing.T) {
			checkNamed(t, members, recordingFunctions, typed)
		})

		t.Run("matches each method of the chain with a driver", func(t *testing.T) {
			driven := slices.Sorted(maps.Keys(recordingMethods))
			if !slices.Equal(driven, methods) {
				t.Errorf("the chain declares %v, and the drivers name %v", methods, driven)
			}
		})
	})

	t.Run("Functions", func(t *testing.T) { drive(t, recordingFunctions, names, false) })
	t.Run("Methods", func(t *testing.T) { drive(t, recordingMethods, names, false) })
}

// abortingFunctions calls each reporting function of the aborting surface,
// keyed by its name, with an input the function rejects.
var abortingFunctions = map[string]func(tb assert.TB){
	"Accumulates": func(tb assert.TB) {
		assert.Accumulates(tb, func(int) error { return nil }, 0, func() int { return 0 }, rejected)
	},
	"Associative": func(tb assert.TB) {
		assert.Associative(tb, func(a, b int) int { return a - b }, 2, 3, 5, rejected)
	},
	"CloseTo": func(tb assert.TB) { assert.CloseTo(tb, 1.0, 2, 0.1, rejected) },
	"Commutative": func(tb assert.TB) {
		assert.Commutative(tb, func(a, b int) int { return a - b }, 2, 3, rejected)
	},
	"CompletesWithin": func(tb assert.TB) {
		assert.CompletesWithin(tb, time.Millisecond, func(context.Context) error {
			time.Sleep(2 * time.Millisecond)
			return nil
		}, rejected)
	},
	"Contains":        func(tb assert.TB) { assert.Contains(tb, "abc", "x", rejected) },
	"ContainsInOrder": func(tb assert.TB) { assert.ContainsInOrder(tb, "abc", []string{"c", "a"}, rejected) },
	"Deterministic": func(tb assert.TB) {
		assert.Deterministic(tb, func(int) (int, error) { return 0, io.EOF }, 0, rejected)
	},
	"Empty":      func(tb assert.TB) { assert.Empty(tb, "a", rejected) },
	"Equal":      func(tb assert.TB) { assert.Equal(tb, 1, 2, rejected) },
	"ErrorAs":    func(tb assert.TB) { _ = assert.ErrorAs[*fs.PathError](tb, io.EOF, rejected) },
	"ErrorIs":    func(tb assert.TB) { assert.ErrorIs(tb, io.EOF, fs.ErrNotExist, rejected) },
	"ErrorIsNot": func(tb assert.TB) { assert.ErrorIsNot(tb, io.EOF, io.EOF, rejected) },
	"Eventually": func(tb assert.TB) {
		assert.Eventually(tb, time.Millisecond, time.Millisecond, func(trial assert.TB) {
			assert.True(trial, false, rejected)
		}, rejected)
	},
	"EventuallyTrue": func(tb assert.TB) {
		assert.EventuallyTrue(tb, time.Millisecond, func() bool { return false }, rejected)
	},
	"FailsAfterClose": func(tb assert.TB) {
		assert.FailsAfterClose(tb, func() error { return nil }, func() error { return nil }, io.EOF, rejected)
	},
	"False":     func(tb assert.TB) { assert.False(tb, true, rejected) },
	"HasError":  func(tb assert.TB) { assert.HasError(tb, nil, rejected) },
	"HasPrefix": func(tb assert.TB) { assert.HasPrefix(tb, "abc", "x", rejected) },
	"HasSuffix": func(tb assert.TB) { assert.HasSuffix(tb, "abc", "x", rejected) },
	"HonoursCancellation": func(tb assert.TB) {
		assert.HonoursCancellation(tb, func(context.Context) error { return nil }, rejected)
	},
	"HonoursDeadline": func(tb assert.TB) {
		assert.HonoursDeadline(tb, func(context.Context) error { return nil }, rejected)
	},
	"Idempotent": func(tb assert.TB) {
		assert.Idempotent(tb, func(int) error { return io.EOF }, 0, func() int { return 0 }, rejected)
	},
	"InRange": func(tb assert.TB) { assert.InRange(tb, 5.0, 0, 1, rejected) },
	"Length":  func(tb assert.TB) { assert.Length(tb, "ab", 3, rejected) },
	"Matches": func(tb assert.TB) { assert.Matches(tb, "abc", "^x", rejected) },
	"MaxAllocs": func(tb assert.TB) {
		assert.MaxAllocs(tb, func() { escaped = make([]byte, 64) }, 0, rejected)
	},
	"MaxAllocsWithSetup": func(tb assert.TB) {
		assert.MaxAllocsWithSetup(tb, func() int { return 64 }, func(n int) { escaped = make([]byte, n) }, 0, rejected)
	},
	"Monotonic": func(tb assert.TB) {
		assert.Monotonic(tb, func() int { return 0 }, func() error { return io.EOF }, 1, rejected)
	},
	"Nil": func(tb assert.TB) { assert.Nil(tb, 1, rejected) },
	"NilContextSafe": func(tb assert.TB) {
		assert.NilContextSafe(tb, func(ctx context.Context) error { return ctx.Err() }, rejected)
	},
	"NoDuplicates": func(tb assert.TB) {
		assert.NoDuplicates(tb, func() ([]int, error) { return []int{1, 1}, nil }, rejected)
	},
	"NoError": func(tb assert.TB) { assert.NoError(tb, io.EOF, rejected) },
	"NoGoroutineLeaks": func(tb assert.TB) {
		check := assert.NoGoroutineLeaks(tb, rejected)
		release := make(chan struct{})
		go func() { <-release }()
		check()
		close(release)
	},
	"NotContains": func(tb assert.TB) { assert.NotContains(tb, "abc", "b", rejected) },
	"NotEmpty":    func(tb assert.TB) { assert.NotEmpty(tb, "", rejected) },
	"NotEqual":    func(tb assert.TB) { assert.NotEqual(tb, 1, 1, rejected) },
	"NotNil":      func(tb assert.TB) { assert.NotNil(tb, nil, rejected) },
	"NotPanics":   func(tb assert.TB) { assert.NotPanics(tb, func() { panic(rejected) }, rejected) },
	"NotPure":     func(tb assert.TB) { assert.NotPure(tb, func() int { return 0 }, func() {}, rejected) },
	"Pairwise": func(tb assert.TB) {
		assert.Pairwise(tb, []int{2, 1}, func(earlier, later int) bool { return earlier < later }, rejected)
	},
	"Panics":      func(tb assert.TB) { assert.Panics(tb, func() {}, rejected) },
	"Permutation": func(tb assert.TB) { assert.Permutation(tb, []int{1}, []int{2}, rejected) },
	"Poisoned":    func(tb assert.TB) { assert.Poisoned(tb, func() {}, func() error { return nil }, rejected) },
	"Pure": func(tb assert.TB) {
		calls := 0
		assert.Pure(tb, func() int { return calls }, func() { calls++ }, rejected)
	},
	"Rejects": func(tb assert.TB) { assert.Rejects(tb, rejected, func(assert.TB) {}) },
	"RoundTrip": func(tb assert.TB) {
		assert.RoundTrip(tb, func(int) (int, error) { return 0, io.EOF },
			func(v int) (int, error) { return v, nil }, 1, rejected)
	},
	"StableOrder": func(tb assert.TB) {
		assert.StableOrder(tb, func() ([]int, error) { return nil, io.EOF }, rejected)
	},
	"Total": func(tb assert.TB) { assert.Total(tb, func(int) error { return io.EOF }, []int{1}, rejected) },
	"True":  func(tb assert.TB) { assert.True(tb, false, rejected) },
}

// abortingMethods calls each method of the aborting chain, keyed by its
// name, on a value the method rejects.
var abortingMethods = map[string]func(tb assert.TB){
	"CloseTo":         func(tb assert.TB) { assert.That(tb, 1.0).CloseTo(2, 0.1, rejected) },
	"Contains":        func(tb assert.TB) { assert.That(tb, "abc").Contains("x", rejected) },
	"ContainsInOrder": func(tb assert.TB) { assert.That(tb, "abc").ContainsInOrder([]string{"c", "a"}, rejected) },
	"Empty":           func(tb assert.TB) { assert.That(tb, "a").Empty(rejected) },
	"Equal":           func(tb assert.TB) { assert.That(tb, 1).Equal(2, rejected) },
	"ErrorIs":         func(tb assert.TB) { assert.That(tb, io.EOF).ErrorIs(fs.ErrNotExist, rejected) },
	"ErrorIsNot":      func(tb assert.TB) { assert.That(tb, io.EOF).ErrorIsNot(io.EOF, rejected) },
	"HasError":        func(tb assert.TB) { assert.That[error](tb, nil).HasError(rejected) },
	"HasPrefix":       func(tb assert.TB) { assert.That(tb, "abc").HasPrefix("x", rejected) },
	"HasSuffix":       func(tb assert.TB) { assert.That(tb, "abc").HasSuffix("x", rejected) },
	"InRange":         func(tb assert.TB) { assert.That(tb, 5.0).InRange(0, 1, rejected) },
	"Length":          func(tb assert.TB) { assert.That(tb, "ab").Length(3, rejected) },
	"Matches":         func(tb assert.TB) { assert.That(tb, "abc").Matches("^x", rejected) },
	"Nil":             func(tb assert.TB) { assert.That(tb, 1).Nil(rejected) },
	"NoError":         func(tb assert.TB) { assert.That(tb, io.EOF).NoError(rejected) },
	"NotContains":     func(tb assert.TB) { assert.That(tb, "abc").NotContains("b", rejected) },
	"NotEmpty":        func(tb assert.TB) { assert.That(tb, "").NotEmpty(rejected) },
	"NotEqual":        func(tb assert.TB) { assert.That(tb, 1).NotEqual(1, rejected) },
	"NotNil":          func(tb assert.TB) { assert.That[any](tb, nil).NotNil(rejected) },
}

// TestSurfaceAborting drives every reporting member of the aborting surface
// with an input it rejects. Each member must report through Fatalf and
// never through Errorf, and its record must name the member the driver is
// keyed by. It is the twin of TestSurfaceRecording, and it does not run in
// parallel for the same reasons.
//
// The member list comes from the surface's source and the chain's method
// set, so a member added without a driver fails here. A member that
// reports nothing is excused by silent, or by a type of the naming table's
// surface section.
func TestSurfaceAborting(t *testing.T) {
	members, err := conformance.Members(conformance.Aborting)
	if err != nil {
		t.Fatalf("the aborting surface can be read: %v", err)
	}

	names := conformance.Names()
	typed := slices.Collect(maps.Values(conformance.SurfaceNames()))

	var methods []string
	for method := range reflect.TypeFor[*assert.Assertion[any]]().Methods() {
		methods = append(methods, method.Name)
	}

	t.Run("Members", func(t *testing.T) {
		t.Run("returns the members that a driver, an excuse or the table names", func(t *testing.T) {
			checkNamed(t, members, abortingFunctions, typed)
		})

		t.Run("matches each method of the chain with a driver", func(t *testing.T) {
			driven := slices.Sorted(maps.Keys(abortingMethods))
			if !slices.Equal(driven, methods) {
				t.Errorf("the chain declares %v, and the drivers name %v", methods, driven)
			}
		})
	})

	t.Run("Functions", func(t *testing.T) { drive(t, abortingFunctions, names, true) })
	t.Run("Methods", func(t *testing.T) { drive(t, abortingMethods, names, true) })
}

// checkNamed requires that the members of a surface are exactly the
// functions that drivers keys, the members that silent excuses, and the
// types of the naming table's surface section that the surface declares.
func checkNamed(t *testing.T, members []string, drivers map[string]func(tb assert.TB), typed []string) {
	t.Helper()

	named := slices.Concat(slices.Collect(maps.Keys(drivers)), slices.Collect(maps.Keys(silent)))
	for _, name := range members {
		if slices.Contains(typed, name) && !slices.Contains(named, name) {
			named = append(named, name)
		}
	}
	slices.Sort(named)
	if !slices.Equal(named, members) {
		t.Errorf("the surface declares %v, and the drivers, the excuses and the table name %v", members, named)
	}
}

// drive calls each driver with a fresh seat. It requires a failure through
// Fatalf and none through Errorf when aborting is set, the reverse when it
// is not, and a record whose assertion the naming table gives Go under the
// driver's key.
func drive(t *testing.T, drivers map[string]func(tb assert.TB), names map[conformance.ID]string, aborting bool) {
	t.Helper()

	for _, name := range slices.Sorted(maps.Keys(drivers)) {
		t.Run(name, func(t *testing.T) {
			if strings.HasPrefix(name, "MaxAllocs") && !matcher.AllocationsCounted() {
				t.Skip("this build does not count allocations, so no ceiling can fail")
			}

			seat := &matchertest.Seat{}
			drivers[name](seat)

			reported, other := seat.Errs(), seat.Fatals()
			path, otherPath := "Errorf", "Fatalf"
			if aborting {
				reported, other = other, reported
				path, otherPath = otherPath, path
			}
			if len(other) > 0 {
				t.Errorf("reported through %s: %q", otherPath, other)
			}
			records := seat.Records()
			if len(reported) == 0 || len(records) == 0 {
				t.Fatalf("reported nothing through %s, so the input did not fail it", path)
			}
			if got := names[conformance.ID(records[0].Assertion)]; got != name {
				t.Errorf("reported %s, which Go names %q, so the driver calls another member",
					records[0].Assertion, got)
			}
		})
	}
}
