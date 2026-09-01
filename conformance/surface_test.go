// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/golden"
)

// This package is a consumer of the library, so its tests are written
// with it. The assertion core cannot do the same: a package that tests
// itself with itself lets one bug hide another.

// abortingOnly names members the recording surface is not expected to
// carry, with the reason. An entry here is a claim someone can argue
// with, which a silent omission is not.
//
// A type the naming table's surface section covers needs no entry: it
// is declared once for both surfaces and the table already says so.
var abortingOnly = map[string]string{
	"Rejects":       "drives a check to failure, which needs a seat that stops",
	"Recorder":      "the seat both surfaces report through, declared once",
	"NewRecorder":   "the seat both surfaces report through, declared once",
	"NewControlled": "constructs the clock the table names, declared once",
	"TB":            "the seat interface, declared once and used by both",
}

func TestSurface(t *testing.T) {
	t.Parallel()

	aborting, err := conformance.Members(conformance.Aborting)
	assert.NoError(t, err, "the aborting surface can be read")

	recording, err := conformance.Members(conformance.Recording)
	assert.NoError(t, err, "the recording surface can be read")

	t.Run("Members", func(t *testing.T) {
		t.Parallel()

		t.Run("both surfaces declare something", func(t *testing.T) {
			t.Parallel()

			assert.NotEmpty(t, aborting, "the aborting surface declares members")
			assert.NotEmpty(t, recording, "the recording surface declares members")
		})

		t.Run("every aborting member has a recording twin", func(t *testing.T) {
			t.Parallel()

			named, err := conformance.SurfaceNames()
			assert.NoError(t, err, "the surface table can be read")
			covered := slices.Collect(maps.Values(named))

			for _, name := range aborting {
				if _, excused := abortingOnly[name]; excused {
					continue
				}
				// A name the table carries is declared once for both
				// surfaces, which is what the table states.
				if slices.Contains(covered, name) {
					continue
				}
				assert.Contains(t, recording, name,
					"the recording surface carries "+name)
			}
		})

		t.Run("the recording surface adds nothing of its own", func(t *testing.T) {
			t.Parallel()

			for _, name := range recording {
				assert.Contains(t, aborting, name,
					"the aborting surface carries "+name)
			}
		})

		t.Run("every excused member is absent as claimed", func(t *testing.T) {
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

// pinned names each surface id the table gives Go, as a value whose
// type the compiler checks. Go can look nothing up by name at run
// time, so existence is proven here and the test below only checks
// that this map has not fallen behind the table.
//
// The collector seat is testing.T: Fatalf stops the test and Errorf
// records and carries on, which is that seat's contract, supplied by
// the platform. The compile-time conversion is the proof.
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
}

// TestSurfaceTable holds the pin map to the naming table: every id the
// table names for Go is pinned, and every id it does not name is
// declined by the overlay with a reason. Written with testing rather
// than with the library, because a verdict is not written with the
// subject.
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
