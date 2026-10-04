// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

// TestProperty checks how a call of ForAll delivers the faults of its store
// and its record to its seat.
func TestProperty(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("passes the fault of each skipped file to the seat in the order of the files", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "b.json"), `{"store": 2}`)
			write(t, filepath.Join(dir, "a.json"), `{"store": 3}`)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Seed(7), prop.Store(dir))
			faults := seat.Faults()
			assert.Length(t, faults, 2, "a fault for each file")
			expectFault(t, faults[0], laterFault(forAllOp, dir, "a.json", "3"))
			expectFault(t, faults[1], laterFault(forAllOp, dir, "b.json", "2"))
		})

		t.Run("drops the notes of a seat without a log", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, draws(prop.Integer(0, 9)), prop.Seed(7), prop.Store(dir))
			assert.False(t, rec.Failed(), "the run passes without a note")
		})

		t.Run("reports a failing run and the fault of each skipped file", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, failsFrom(1001), prop.Seed(7), prop.Store(dir))
			expectOnlyFault(t, seat.Faults(), laterFault(forAllOp, dir, "later.json", "2"))
			assert.Length(t, seat.Records(), 1, "the record")
		})
	})
}
