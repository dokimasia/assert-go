// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/golden"
)

// pinned names each surface id the table gives Go, as a value whose
// type the compiler checks. Go can look nothing up by name at run
// time, so existence is proven here and the test below only checks
// that this map has not fallen behind the table.
//
// The collector seat is testing.T: Fatalf stops the test and Errorf
// records and carries on, which is that seat's contract, supplied by
// the platform. The compile-time conversion is the proof.
var pinned = map[conformance.ID]any{
	"seat":           (*assert.TB)(nil),
	"recorder-seat":  (*assert.Recorder)(nil),
	"collector-seat": assert.TB((*testing.T)(nil)),
	"scrubber":       golden.Scrubber(nil),
	"contract":       (*bench.Contract)(nil),

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

	source, err := os.ReadFile("surface_pin_test.go")
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
