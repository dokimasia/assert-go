// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"
	"sync"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The allocations of the case's methods, measured.
const (
	// reportAllocs are the allocations of a recording Report: the rendered
	// record and the message the recorder formats from it.
	reportAllocs = 2
	// errorfAllocs are the allocations of Errorf: the message, the frames
	// searched for the caller's code, the rendered record and the message
	// the recorder formats from it.
	errorfAllocs = 5
	// copyAllocs are the allocations of an accessor: the copy it returns.
	copyAllocs = 1
	// fatalCaseAllocs are the allocations of a whole replayed case whose
	// body calls Fatalf.
	fatalCaseAllocs = 12
	// valueCaseAllocs are the allocations of a whole replayed case whose
	// body draws one value from its source.
	valueCaseAllocs = 7
)

// reported is the record that a test body reports, as an assertion would.
var reported = assert.Failure{
	Assertion: "equal",
	Contract:  "the totals match",
	Where:     assert.Where{File: "ledger_test.go", Line: 12},
}

// TestCase checks the case as the TB of a body's assertions: how each kind
// of failure ends it, and what it records.
func TestCase(t *testing.T) {
	t.Parallel()

	t.Run("Helper", func(t *testing.T) {
		t.Parallel()

		t.Run("returns without ending the case", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Helper()
				after = true
			}, nil, nil)
			assert.True(t, after, "the body runs on")
			assert.Equal(t, e.Status, engine.CasePassed, "the case passes")
		})
	})

	t.Run("Report", func(t *testing.T) {
		t.Parallel()

		t.Run("ends the case at an aborting record", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Report(reported, true)
				after = true
			}, nil, nil)
			assert.False(t, after, "the body ends at the record")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails")
			assert.Equal(t, e.Identity, engine.Identity{Assertion: "equal", Where: reported.Where},
				"the assertion and its location")
		})

		t.Run("lets the body run on after a recording record", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Report(reported, false)
				after = true
			}, nil, nil)
			assert.True(t, after, "the body runs on")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails when its body returns")
		})

		t.Run("keeps every record in call order", func(t *testing.T) {
			t.Parallel()
			second := assert.Failure{
				Assertion: "true",
				Contract:  "the ledger balances",
				Where:     assert.Where{File: "ledger_test.go", Line: 20},
			}
			e := engine.Replay(func(c *engine.Case) {
				c.Report(reported, false)
				c.Report(second, false)
			}, nil, nil)
			assert.Equal(t, e.Case.Failures(), []assert.Failure{reported, second}, "both records")
			assert.Equal(t, e.Identity.Assertion, "equal", "the first record is the case's failure")
		})
	})

	t.Run("Fatalf", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a record of the message at the caller's line", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			e := engine.Replay(func(c *engine.Case) { c.Fatalf("stop at %d", site(&at)) }, nil, nil)
			want := assert.Failure{Contract: fmt.Sprintf("stop at %d", at.Line), Where: at}
			assert.Equal(t, e.Case.Failures(), []assert.Failure{want}, "a record without an assertion")
		})

		t.Run("ends the calling goroutine", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Fatalf("stop")
				after = true
			}, nil, nil)
			assert.False(t, after, "the body ends at Fatalf")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails")
		})

		t.Run("ends only the goroutine that calls it", func(t *testing.T) {
			t.Parallel()
			var escaped, returned bool
			e := engine.Replay(func(c *engine.Case) {
				var wg sync.WaitGroup
				wg.Go(func() {
					c.Fatalf("stop")
					escaped = true
				})
				wg.Wait()
				returned = true
			}, nil, nil)
			assert.False(t, escaped, "the other goroutine ends at Fatalf")
			assert.True(t, returned, "the body runs on")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails when its body returns")
		})
	})

	t.Run("Errorf", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a record of the message at the caller's line", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Errorf("late at %d", site(&at))
				after = true
			}, nil, nil)
			want := assert.Failure{Contract: fmt.Sprintf("late at %d", at.Line), Where: at}
			assert.Equal(t, e.Case.Failures(), []assert.Failure{want}, "a record without an assertion")
			assert.True(t, after, "the body runs on")
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails when its body returns")
		})

		t.Run("keeps the zero location for a goroutine without the caller's code", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) { go c.Errorf("detached") }, nil, nil)
			arrived := func() bool { return len(e.Case.Failures()) == 1 }
			assert.EventuallyTrue(t, 10*time.Second, arrived, "the message arrives")
			assert.Equal(t, e.Case.Failures()[0].Where, assert.Where{}, "no frame of the caller's code")
		})
	})

	t.Run("Clock", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the clock of the test", func(t *testing.T) {
			t.Parallel()
			start := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)
			var got assert.Clock
			engine.Replay(func(c *engine.Case) { got = c.Clock() }, nil, assert.NewControlled(start))
			assert.Equal(t, got.Now(), start, "the controlled clock's time")
		})

		t.Run("returns the system clock for no clock", func(t *testing.T) {
			t.Parallel()
			var got assert.Clock
			engine.Replay(func(c *engine.Case) { got = c.Clock() }, nil, nil)
			assert.Equal[assert.Clock](t, got, assert.System{}, "the runtime clock")
		})
	})

	t.Run("Assume", func(t *testing.T) {
		t.Parallel()

		t.Run("rejects the case for false", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Assume(false)
				after = true
			}, nil, nil)
			assert.False(t, after, "the body ends at the rejection")
			assert.Equal(t, e.Status, engine.CaseRejected, "the case is rejected")
		})

		t.Run("returns for true", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				c.Assume(true)
				after = true
			}, nil, nil)
			assert.True(t, after, "the body runs on")
			assert.Equal(t, e.Status, engine.CasePassed, "the case passes")
		})

		t.Run("rejects the case from another goroutine when the body returns", func(t *testing.T) {
			t.Parallel()
			var returned bool
			e := engine.Replay(func(c *engine.Case) {
				var wg sync.WaitGroup
				wg.Go(func() { c.Assume(false) })
				wg.Wait()
				returned = true
			}, nil, nil)
			assert.True(t, returned, "the body runs on")
			assert.Equal(t, e.Status, engine.CaseRejected, "the case is rejected")
		})
	})

	t.Run("Classify", func(t *testing.T) {
		t.Parallel()

		t.Run("counts a label once in a case", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Classify("small")
				c.Classify("even")
				c.Classify("small")
			}, nil, nil)
			assert.Equal(t, e.Case.Labels(), []string{"even", "small"}, "each label once, sorted")
		})
	})

	t.Run("Note", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps the notes in order", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Note("opened the ledger")
				c.Note("appended two entries")
			}, nil, nil)
			assert.Equal(t, e.Case.Notes(), []string{"opened the ledger", "appended two entries"}, "both notes")
		})
	})

	t.Run("Observe", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps the fingerprints in order", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Observe(3)
				c.Observe(1)
			}, nil, nil)
			assert.Equal(t, e.Case.Fingerprints(), []uint64{3, 1}, "both fingerprints")
		})
	})

	t.Run("Rand", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a source that math/rand/v2 draws the case's choices from", func(t *testing.T) {
			t.Parallel()
			var got uint64
			body := func(c *engine.Case) { got = rand.New(c.Rand()).Uint64() }
			e := engine.Replay(body, []choice.Choice{unsigned(5)}, nil)
			assert.Equal(t, got, uint64(5), "the replayed choice")
			assert.True(t, sameChoices(e.Case.Choices(), []choice.Choice{unsigned(5)}), "one choice")
		})
	})

	t.Run("Uint64", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the case's next choice over the unsigned range", func(t *testing.T) {
			t.Parallel()
			var got []uint64
			recorded := []choice.Choice{unsigned(math.MaxUint64), unsigned(0)}
			e := engine.Replay(func(c *engine.Case) {
				source := c.Rand()
				got = []uint64{source.Uint64(), source.Uint64()}
			}, recorded, nil)
			assert.Equal(t, got, []uint64{math.MaxUint64, 0}, "the replayed choices")
			assert.True(t, sameChoices(e.Case.Choices(), recorded), "one choice for each value")
		})

		t.Run("returns 0 for a negative choice", func(t *testing.T) {
			t.Parallel()
			var got uint64
			e := engine.Replay(func(c *engine.Case) { got = c.Rand().Uint64() }, integers(-5), nil)
			assert.Equal(t, got, uint64(0), "the target of the unsigned range")
			assert.True(t, sameChoices(e.Case.Choices(), []choice.Choice{unsigned(0)}), "the coerced choice")
		})
	})

	t.Run("Choices", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Choices()[0] = unsigned(1)
			assert.True(t, sameChoices(c.Choices(), integers(7)), "the recorded choice")
		})
	})

	t.Run("Spans", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Spans()[0].Label = "changed"
			assert.Equal(t, labels(c.Spans()), []string{"integer"}, "the recorded span")
		})
	})

	t.Run("Draws", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Draws()[0].Label = "changed"
			assert.Equal(t, drawLabels(c.Draws()), []string{drawn}, "the recorded draw")
		})
	})

	t.Run("Labels", func(t *testing.T) {
		t.Parallel()

		t.Run("returns no label for an unclassified case", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(*engine.Case) {}, nil, nil)
			assert.Empty(t, e.Case.Labels(), "no label")
		})
	})

	t.Run("Notes", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Notes()[0] = "changed"
			assert.Equal(t, c.Notes(), []string{"seen"}, "the recorded note")
		})
	})

	t.Run("Fingerprints", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Fingerprints()[0] = 0
			assert.Equal(t, c.Fingerprints(), []uint64{42}, "the recorded fingerprint")
		})
	})

	t.Run("Failures", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a copy that the caller may change", func(t *testing.T) {
			t.Parallel()
			c := leaked()
			c.Failures()[0].Assertion = "changed"
			assert.Equal(t, c.Failures(), []assert.Failure{reported}, "the recorded failure")
		})
	})
}

// TestCaseZeroAlloc checks the allocation ceilings of the case's methods.
// A method that ends the calling goroutine, and a value that adds a choice,
// are measured on a whole replayed case.
func TestCaseZeroAlloc(t *testing.T) {
	c := leaked()
	stop := func(c *engine.Case) { c.Fatalf("stop") }
	value := func(c *engine.Case) { c.Rand().Uint64() }
	assert.MaxAllocs(t, func() { c.Helper() }, 0, "Helper allocates nothing")
	assert.MaxAllocs(t, func() { c.Report(reported, false) }, reportAllocs, "Report allocates its message")
	assert.MaxAllocs(t, func() { engine.Replay(stop, nil, nil) }, fatalCaseAllocs, "a case that calls Fatalf")
	assert.MaxAllocs(t, func() { c.Errorf("late") }, errorfAllocs, "Errorf allocates its message and its frames")
	assert.MaxAllocs(t, func() { _ = c.Clock() }, 0, "Clock allocates nothing")
	assert.MaxAllocs(t, func() { c.Assume(true) }, 0, "Assume allocates nothing")
	assert.MaxAllocs(t, func() { c.Classify("small") }, 0, "Classify allocates nothing for a known label")
	assert.MaxAllocs(t, func() { c.Note("seen") }, 0, "Note allocates only to grow the record")
	assert.MaxAllocs(t, func() { c.Observe(42) }, 0, "Observe allocates only to grow the record")
	assert.MaxAllocs(t, func() { _ = c.Rand() }, 0, "Rand allocates nothing")
	assert.MaxAllocs(t, func() { engine.Replay(value, nil, nil) }, valueCaseAllocs, "a case that draws one value")
	assert.MaxAllocs(t, func() { _ = c.Choices() }, copyAllocs, "Choices allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Spans() }, copyAllocs, "Spans allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Draws() }, copyAllocs, "Draws allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Labels() }, copyAllocs, "Labels allocates its list")
	assert.MaxAllocs(t, func() { _ = c.Notes() }, copyAllocs, "Notes allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Fingerprints() }, copyAllocs, "Fingerprints allocates its copy")
	assert.MaxAllocs(t, func() { _ = c.Failures() }, copyAllocs, "Failures allocates its copy")
}

// BenchmarkCase measures each method of the case. A method that ends the
// calling goroutine, and a value that adds a choice, are measured on a
// whole replayed case.
func BenchmarkCase(b *testing.B) {
	b.Run("Helper", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Helper()
		}
		assert.Length(b, cs.Failures(), 1, "the case's one record")
	})

	b.Run("Report", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(reportAllocs)
		defer c.End()
		for c.Loop() {
			cs.Report(reported, false)
		}
		assert.Equal(b, cs.Failures()[0], reported, "the first record")
	})

	b.Run("Fatalf", func(b *testing.B) {
		var got engine.Execution
		stop := func(c *engine.Case) { c.Fatalf("stop") }
		c := bench.Start(b).MaxAllocs(fatalCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(stop, nil, nil)
		}
		assert.Equal(b, got.Status, engine.CaseFailed, "the case fails")
	})

	b.Run("Errorf", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(errorfAllocs)
		defer c.End()
		for c.Loop() {
			cs.Errorf("late")
		}
		assert.Equal(b, cs.Failures()[1].Contract, "late", "the message's record")
	})

	b.Run("Clock", func(b *testing.B) {
		var got assert.Clock
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = cs.Clock()
		}
		assert.Equal[assert.Clock](b, got, assert.System{}, "the runtime clock")
	})

	b.Run("Assume", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Assume(true)
		}
		assert.Length(b, cs.Choices(), 1, "no choice added")
	})

	b.Run("Classify", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Classify("small")
		}
		assert.Equal(b, cs.Labels(), []string{"small"}, "one label")
	})

	b.Run("Note", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Note("seen")
		}
		assert.Equal(b, cs.Notes()[0], "seen", "the first note")
	})

	b.Run("Observe", func(b *testing.B) {
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			cs.Observe(42)
		}
		assert.Equal(b, cs.Fingerprints()[0], uint64(42), "the first fingerprint")
	})

	b.Run("Rand", func(b *testing.B) {
		var got engine.Source
		cs := leaked()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = cs.Rand()
		}
		assert.Equal(b, got, cs.Rand(), "the case's source")
	})

	b.Run("Uint64", func(b *testing.B) {
		var got engine.Execution
		value := func(c *engine.Case) { c.Rand().Uint64() }
		c := bench.Start(b).MaxAllocs(valueCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(value, nil, nil)
		}
		assert.Length(b, got.Case.Choices(), 1, "one choice")
	})

	b.Run("Choices", func(b *testing.B) {
		var got []choice.Choice
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Choices()
		}
		assert.True(b, sameChoices(got, integers(7)), "the recorded choice")
	})

	b.Run("Spans", func(b *testing.B) {
		var got []engine.Span
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Spans()
		}
		assert.Equal(b, labels(got), []string{"integer"}, "the recorded span")
	})

	b.Run("Draws", func(b *testing.B) {
		var got []engine.Drawn
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Draws()
		}
		assert.Equal(b, drawLabels(got), []string{drawn}, "the recorded draw")
	})

	b.Run("Labels", func(b *testing.B) {
		var got []string
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Labels()
		}
		assert.Equal(b, got, []string{"small"}, "the label")
	})

	b.Run("Notes", func(b *testing.B) {
		var got []string
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Notes()
		}
		assert.Equal(b, got, []string{"seen"}, "the note")
	})

	b.Run("Fingerprints", func(b *testing.B) {
		var got []uint64
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Fingerprints()
		}
		assert.Equal(b, got, []uint64{42}, "the fingerprint")
	})

	b.Run("Failures", func(b *testing.B) {
		var got []assert.Failure
		cs := leaked()
		c := bench.Start(b).MaxAllocs(copyAllocs)
		defer c.End()
		for c.Loop() {
			got = cs.Failures()
		}
		assert.Equal(b, got, []assert.Failure{reported}, "the record")
	})
}

// leaked returns the case of a replayed body that drew 7 from the digits,
// and classified, noted, observed and reported once, for a caller to read
// and to call after the body returned.
func leaked() *engine.Case {
	e := engine.Replay(func(c *engine.Case) {
		engine.Draw(c, engine.Integer(0, 9), drawn)
		c.Classify("small")
		c.Note("seen")
		c.Observe(42)
		c.Report(reported, false)
	}, integers(7), nil)
	return e.Case
}

// site stores the file and the line of its caller in at and returns the
// line, so a call inside another call's arguments states where that call
// is.
func site(at *assert.Where) int {
	_, at.File, at.Line, _ = runtime.Caller(1)
	return at.Line
}
