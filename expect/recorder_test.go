// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
)

// recorder keeps the recorder that a call of NewRecorder returns.
var recorder *expect.Recorder

// TestRecorder checks the recorder of this surface.
func TestRecorder(t *testing.T) {
	t.Parallel()

	t.Run("Recorder", func(t *testing.T) {
		t.Parallel()

		t.Run("is the recorder of the aborting surface", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, reflect.TypeFor[expect.Recorder](), reflect.TypeFor[assert.Recorder](), "one recorder")
		})
	})

	t.Run("NewRecorder", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a recorder that keeps every failure and lets the caller run on", func(t *testing.T) {
			t.Parallel()

			r := expect.NewRecorder()
			expect.True(r, false, "the first fails")
			expect.True(r, false, "the second fails")
			expect.Length(t, r.Failures(), 2, "both failures")
			expect.Equal(t, r.Failures()[1].Contract, "the second fails", "the second in call order")
		})
	})
}

// TestRecorderAllocs checks the allocation ceiling of NewRecorder.
func TestRecorderAllocs(t *testing.T) {
	alloctest.Check(t, recorderCases())
}

// BenchmarkRecorder measures NewRecorder.
func BenchmarkRecorder(b *testing.B) {
	for _, c := range recorderCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// recorderCases returns a call of NewRecorder, with its allocation ceiling,
// measured.
func recorderCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "NewRecorder", Call: func(assert.TB) { recorder = expect.NewRecorder() }, Allocs: 1},
	}
}
