// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/internal/prop/engine"
)

// Fuzz registers body as the fuzz target of f. Each input's bytes decode
// into the choices of one case by the definition's bridge rules, so every
// input is a valid case, and a property over one byte string sees the
// fuzzer's bytes almost unchanged.
//
// Before it registers the target, Fuzz replays the property's stored cases,
// oldest first, from the store of the fuzz test, and fails f with the
// record of the first that fails, as found. go test without -fuzz then runs
// the stored cases and the seed corpus that f.Add states.
//
// A failing input's case is replayed, shrunk and explained as [ForAll]
// does with a failing case, written to the store, and reported through the
// input's *testing.T with its replay token. The report and the notes on the
// store are written to the test's output without a source location: the
// fuzzing machinery calls the target through reflect, so no frame of the
// caller's code is on the stack. The record states no valid case. Fuzz
// runs no replay token, and no coverage requirement: [Replay],
// DOKIMI_ASSERT_PROP_REPLAY, [Cases] and [Require] apply to ForAll alone.
//
// Fuzz fails f at once, without registering the target, for each fault for
// which ForAll fails without a run.
func Fuzz(f *testing.F, contract string, body func(*Case), opts ...Option) {
	f.Helper()
	p, err := newProperty(f, contract, caller(), configure(opts))
	if err != nil {
		f.Fatalf("%v", err)
	}
	if !claim(f, p.dir, contract) {
		f.Fatalf(duplicate, contract)
	}
	stored, err := p.load()
	if err != nil {
		f.Fatalf("%v", err)
	}
	note(f, stored.Skipped)
	replayed := bodyOf(f.Context(), body)
	for _, entry := range stored.Entries {
		p.report(f, engine.RunReplay(replayed, p.settings, entry.Choices))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		run := bodyOf(t.Context(), body)
		e := engine.Bridge(run, data, p.settings.Clock)
		if e.Status != engine.CaseFailed {
			return
		}
		r := engine.Conclude(run, p.settings, e)
		if r.Outcome == engine.Counterexample {
			for _, n := range p.save(r) {
				fmt.Fprintln(t.Output(), n)
			}
		}
		p.reportInput(t, r)
	})
}
