// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

// TestProperty checks how a call of ForAll delivers its notes and its
// record to its seat.
func TestProperty(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("logs each note to a seat with a log in the order of the files", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "b.json"), `{"store": 2}`)
			write(t, filepath.Join(dir, "a.json"), `{"store": 3}`)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, draws(prop.Integer(0, 9)), prop.Seed(7), prop.Store(dir))
			assert.Length(t, seat.logged(), 2, "a note on each file")
			assert.HasPrefix(t, seat.logged()[0], "a.json", "the first file first")
			assert.HasPrefix(t, seat.logged()[1], "b.json", "the second file second")
		})

		t.Run("drops the notes of a seat without a log", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			rec := assert.NewRecorder()
			prop.ForAll(rec, contract, draws(prop.Integer(0, 9)), prop.Seed(7), prop.Store(dir))
			assert.False(t, rec.Failed(), "the run passes without a note")
		})

		t.Run("reports a failing run after its notes", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, filepath.Join(dir, "later.json"), `{"store": 2}`)
			seat := newTestSeat(t.Name())
			prop.ForAll(seat, contract, failsFrom(1001), prop.Seed(7), prop.Store(dir))
			assert.Length(t, seat.logged(), 1, "the note")
			assert.Length(t, seat.Failures(), 1, "the record")
		})
	})
}
