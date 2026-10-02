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
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/prop"
)

// The tests of the surfaces use the library's assertions, as a consumer of
// the library does. The assertion core is tested with testing alone,
// because a test written with a defective core can pass by the defect that
// it should find.

// abortingOnly names the members that the aborting surface declares and the
// recording surface does not, each with the reason. A type of the naming
// table's surface section needs no entry: the library declares it once for
// both surfaces, as the table states.
var abortingOnly = map[string]string{
	"Rejects":       "drives a check to failure, which needs a seat that stops",
	"Recorder":      "the seat both surfaces report through, declared once",
	"NewRecorder":   "the seat both surfaces report through, declared once",
	"NewControlled": "constructs the clock the table names, declared once",
	"TB":            "the seat interface, declared once and used by both",
}

// TestSurface compares the members of the two surfaces: every member of the
// aborting surface has a twin of the same name in the recording surface,
// unless the naming table or abortingOnly excuses it.
func TestSurface(t *testing.T) {
	t.Parallel()

	aborting, err := conformance.Members(conformance.Aborting)
	assert.NoError(t, err, "the aborting surface can be read")

	recording, err := conformance.Members(conformance.Recording)
	assert.NoError(t, err, "the recording surface can be read")

	t.Run("Members", func(t *testing.T) {
		t.Parallel()

		t.Run("returns members for both surfaces", func(t *testing.T) {
			t.Parallel()

			assert.NotEmpty(t, aborting, "the aborting surface declares members")
			assert.NotEmpty(t, recording, "the recording surface declares members")
		})

		t.Run("returns a recording twin for every aborting member", func(t *testing.T) {
			t.Parallel()

			named, err := conformance.SurfaceNames()
			assert.NoError(t, err, "the surface table can be read")
			covered := slices.Collect(maps.Values(named))

			for _, name := range aborting {
				if _, excused := abortingOnly[name]; excused {
					continue
				}
				// The library declares a name of the table once for both
				// surfaces, as the table states.
				if slices.Contains(covered, name) {
					continue
				}
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

		t.Run("returns an error for a surface whose directory does not exist", func(t *testing.T) {
			t.Parallel()

			_, err := conformance.Members(conformance.Surface(filepath.Join(t.TempDir(), "missing")))
			assert.HasError(t, err, "no directory, so no members")
			assert.Contains(t, err.Error(), "conformance: read", "the error names the read")
		})

		t.Run("returns an error for a surface of a file that does not parse", func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			assert.NoError(t, os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package"), 0o600),
				"the file is written")
			_, err := conformance.Members(conformance.Surface(dir))
			assert.HasError(t, err, "a file that does not parse declares no members")
			assert.Contains(t, err.Error(), "conformance: parse broken.go", "the error names the file")
		})

		t.Run("returns each excused member for the aborting surface alone", func(t *testing.T) {
			t.Parallel()

			for name, why := range abortingOnly {
				assert.Contains(t, aborting, name,
					"the aborting surface declares "+name+", excused because it "+why)
				assert.NotContains(t, recording, name,
					"the recording surface omits "+name+", excused because it "+why)
			}
		})
	})
}

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

	"contract.loop":  (*bench.Contract).Loop,
	"contract.check": (*bench.Contract).End,

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
	"prop.fuzz":            prop.Fuzz,
}

// TestSurfaceTable compares the pin map with the naming table: the map
// pins every id that the table names for Go, and the overlay declines
// every id that the table does not name, with a reason. Written with
// testing rather than with the library, because a verdict is not written
// with the subject.
func TestSurfaceTable(t *testing.T) {
	t.Parallel()

	names, err := conformance.SurfaceNames()
	if err != nil {
		t.Fatalf("the surface table can be read: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("the surface table states something")
	}

	overlay, err := conformance.Overlay()
	if err != nil {
		t.Fatalf("the overlay can be read: %v", err)
	}

	source, err := os.ReadFile("surface_test.go")
	if err != nil {
		t.Fatalf("this file can be read: %v", err)
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
			t.Errorf("%s: the table says %s and nothing here spells it; the pin and the table disagree",
				id, name)
		case name == "" && isPinned:
			t.Errorf("%s: declined in the overlay yet pinned here, which is a contradiction", id)
		}
	}
}

// rejected is the message every driven call passes.
const rejected = "the input is one the member rejects"

// silent names the members of the recording surface that report
// nothing, so no input drives one to fail, with the reason.
var silent = map[string]string{
	"Assertion":   "is the chain type, and the method drivers call its methods",
	"That":        "starts a chain",
	"Option":      "is the type of a comparison option",
	"EquateEmpty": "returns a comparison option",
	"EquateNaNs":  "returns a comparison option",
}

// escaped is the slice that the MaxAllocs driver allocates, which escape
// analysis then keeps on the heap.
var escaped []byte

// recordingFunctions calls each reporting function of the recording
// surface, keyed by its name, with an input the function rejects.
var recordingFunctions = map[string]func(tb assert.TB){
	"CloseTo": func(tb assert.TB) { expect.CloseTo(tb, 1.0, 2, 0.1, rejected) },
	"CompletesWithin": func(tb assert.TB) {
		expect.CompletesWithin(tb, time.Millisecond, func(context.Context) error {
			time.Sleep(2 * time.Millisecond)
			return nil
		}, rejected)
	},
	"Contains":        func(tb assert.TB) { expect.Contains(tb, "abc", "x", rejected) },
	"ContainsInOrder": func(tb assert.TB) { expect.ContainsInOrder(tb, "abc", []string{"c", "a"}, rejected) },
	"Empty":           func(tb assert.TB) { expect.Empty(tb, "a", rejected) },
	"Equal":           func(tb assert.TB) { expect.Equal(tb, 1, 2, rejected) },
	"ErrorAs":         func(tb assert.TB) { _ = expect.ErrorAs[*fs.PathError](tb, io.EOF, rejected) },
	"ErrorIs":         func(tb assert.TB) { expect.ErrorIs(tb, io.EOF, fs.ErrNotExist, rejected) },
	"ErrorIsNot":      func(tb assert.TB) { expect.ErrorIsNot(tb, io.EOF, io.EOF, rejected) },
	"Eventually": func(tb assert.TB) {
		expect.Eventually(tb, time.Millisecond, time.Millisecond, func(trial assert.TB) {
			expect.True(trial, false, rejected)
		}, rejected)
	},
	"EventuallyTrue": func(tb assert.TB) {
		expect.EventuallyTrue(tb, time.Millisecond, func() bool { return false }, rejected)
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
	"InRange": func(tb assert.TB) { expect.InRange(tb, 5.0, 0, 1, rejected) },
	"Length":  func(tb assert.TB) { expect.Length(tb, "ab", 3, rejected) },
	"Matches": func(tb assert.TB) { expect.Matches(tb, "abc", "^x", rejected) },
	"MaxAllocs": func(tb assert.TB) {
		expect.MaxAllocs(tb, func() { escaped = make([]byte, 64) }, 0, rejected)
	},
	"Nil": func(tb assert.TB) { expect.Nil(tb, 1, rejected) },
	"NilContextSafe": func(tb assert.TB) {
		expect.NilContextSafe(tb, func(ctx context.Context) error { return ctx.Err() }, rejected)
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
	"Pairwise": func(tb assert.TB) {
		expect.Pairwise(tb, []int{2, 1}, func(earlier, later int) bool { return earlier < later }, rejected)
	},
	"Panics": func(tb assert.TB) { expect.Panics(tb, func() {}, rejected) },
	"Pure": func(tb assert.TB) {
		calls := 0
		expect.Pure(tb, func() int { return calls }, func() { calls++ }, rejected)
	},
	"True": func(tb assert.TB) { expect.True(tb, false, rejected) },
}

// recordingMethods calls each method of the recording chain, keyed by
// its name, on a value the method rejects.
var recordingMethods = map[string]func(tb assert.TB){
	"CloseTo":         func(tb assert.TB) { expect.That(tb, 1.0).CloseTo(2, 0.1, rejected) },
	"Contains":        func(tb assert.TB) { expect.That(tb, "abc").Contains("x", rejected) },
	"ContainsInOrder": func(tb assert.TB) { expect.That(tb, "abc").ContainsInOrder([]string{"c", "a"}, rejected) },
	"Empty":           func(tb assert.TB) { expect.That(tb, "a").Empty(rejected) },
	"Equal":           func(tb assert.TB) { expect.That(tb, 1).Equal(2, rejected) },
	"HasPrefix":       func(tb assert.TB) { expect.That(tb, "abc").HasPrefix("x", rejected) },
	"HasSuffix":       func(tb assert.TB) { expect.That(tb, "abc").HasSuffix("x", rejected) },
	"InRange":         func(tb assert.TB) { expect.That(tb, 5.0).InRange(0, 1, rejected) },
	"Length":          func(tb assert.TB) { expect.That(tb, "ab").Length(3, rejected) },
	"Matches":         func(tb assert.TB) { expect.That(tb, "abc").Matches("^x", rejected) },
	"Nil":             func(tb assert.TB) { expect.That(tb, 1).Nil(rejected) },
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
// testing.AllocsPerRun, which panics while a parallel test runs, and
// the NoGoroutineLeaks driver reads every goroutine in the process.
func TestSurfaceRecording(t *testing.T) {
	members, err := conformance.Members(conformance.Recording)
	if err != nil {
		t.Fatalf("the recording surface can be read: %v", err)
	}

	names, err := conformance.Names()
	if err != nil {
		t.Fatalf("the naming table can be read: %v", err)
	}

	var methods []string
	for method := range reflect.TypeFor[*expect.Assertion[any]]().Methods() {
		methods = append(methods, method.Name)
	}

	t.Run("Members", func(t *testing.T) {
		t.Run("returns the functions that the drivers and the excuses name", func(t *testing.T) {
			named := slices.Concat(slices.Collect(maps.Keys(recordingFunctions)), slices.Collect(maps.Keys(silent)))
			slices.Sort(named)
			if !slices.Equal(named, members) {
				t.Errorf("the surface declares %v, and the drivers and excuses name %v", members, named)
			}
		})

		t.Run("matches each method of the chain with a driver", func(t *testing.T) {
			driven := slices.Sorted(maps.Keys(recordingMethods))
			if !slices.Equal(driven, methods) {
				t.Errorf("the chain declares %v, and the drivers name %v", methods, driven)
			}
		})
	})

	t.Run("Functions", func(t *testing.T) { drive(t, recordingFunctions, names) })
	t.Run("Methods", func(t *testing.T) { drive(t, recordingMethods, names) })
}

// drive calls each driver with a fresh seat. It requires a failure
// through Errorf, none through Fatalf, and a record whose assertion the
// naming table gives Go under the driver's key.
func drive(t *testing.T, drivers map[string]func(tb assert.TB), names map[conformance.ID]string) {
	t.Helper()

	for _, name := range slices.Sorted(maps.Keys(drivers)) {
		t.Run(name, func(t *testing.T) {
			if name == "MaxAllocs" && !matcher.AllocationsCounted() {
				t.Skip("this build does not count allocations, so no ceiling can fail")
			}

			seat := &matchertest.Seat{}
			drivers[name](seat)

			if fatals := seat.Fatals(); len(fatals) > 0 {
				t.Errorf("reported through Fatalf, which stops the test: %q", fatals)
			}
			records := seat.Records()
			if len(seat.Errs()) == 0 || len(records) == 0 {
				t.Fatal("reported nothing through Errorf, so the input did not fail it")
			}
			if got := names[conformance.ID(records[0].Assertion)]; got != name {
				t.Errorf("reported %s, which Go names %q, so the driver calls another member",
					records[0].Assertion, got)
			}
		})
	}
}
