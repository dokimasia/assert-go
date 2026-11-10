// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/record"
	"go.dokimi.dev/assert/internal/wrap"
)

// storeRoot is the directory of the stores of a package's tests, relative
// to the package's directory, beside testdata/golden.
const storeRoot = "testdata/prop"

// The reasons of the note on a stored case that decodes to other values
// than its entry records.
const (
	// replayedReason is the reason of a stored case whose run replayed the
	// counterexample of its entry.
	replayedReason = "the stored case decodes to other values than it records, and the run replayed its counterexample"
	// testedReason is the reason of a stored case whose run tested the values
	// that its choices decode to.
	testedReason = "the stored case decodes to other values than it records, and the run tested those"
)

// named is a seat whose test has a name, as testing.TB has.
type named interface {
	// Name returns the test's name, each subtest's after its parent's and a
	// slash.
	Name() string
}

// cleaner is a seat with cleanup functions, which run when its test ends,
// as testing.TB has them.
type cleaner interface {
	// Cleanup registers f to run when the test ends.
	Cleanup(f func())
}

// claimed is a property's claim on the entries of its contract in one
// store.
type claimed struct {
	// dir is the store's directory.
	dir string
	// contract is the property's contract.
	contract string
}

// claims are the claims of the properties of the tests that run, which
// each test releases when it ends.
var claims = struct {
	// mu guards active.
	mu sync.Mutex
	// active is the set of the claims.
	active map[claimed]struct{}
}{active: make(map[claimed]struct{})}

// directoryOf returns the directory of the store of a run under c on tb:
// the one that Store states, then testdata/prop/<name> for a seat with a
// name, and otherwise the empty string, which states no store. The name is
// the one of the first seat of the chain of tb that has one, as
// [assert.TB] states.
func directoryOf(tb assert.TB, c config) string {
	if c.storeSet {
		return c.store
	}
	if n, ok := wrap.As[named](tb); ok {
		return filepath.Join(storeRoot, n.Name())
	}
	return ""
}

// claim claims the entries of contract in the store dir for the test of tb
// until the test ends, and reports false when a property of another running
// test has claimed them. The claim ends through Cleanup of the first seat
// of the chain of tb that has it. A run without a store, and a chain
// without Cleanup, claim nothing and report true.
func claim(tb assert.TB, dir, contract string) bool {
	c, ok := wrap.As[cleaner](tb)
	if !ok || dir == "" {
		return true
	}
	key := claimed{dir: dir, contract: contract}
	claims.mu.Lock()
	defer claims.mu.Unlock()
	if _, taken := claims.active[key]; taken {
		return false
	}
	claims.active[key] = struct{}{}
	c.Cleanup(func() {
		claims.mu.Lock()
		defer claims.mu.Unlock()
		delete(claims.active, key)
	})
	return true
}

// entryOf returns the entry that keeps e, a failing case of the property
// contract, found at found.
func entryOf(contract string, e engine.Execution, found time.Time) store.Entry {
	return store.Entry{
		Definition:     record.Definition,
		Property:       contract,
		Identity:       identityOf(e.Identity),
		Choices:        e.Case.Choices(),
		Counterexample: drawsOf(e.Case.Draws()),
		Steps:          stepsOf(e.Case.Steps()),
		Found:          found,
	}
}

// stepsOf returns the steps of a case in the form of an entry, each with
// the number of draws before it.
func stepsOf(steps []engine.RecordedStep) []store.Step {
	out := make([]store.Step, len(steps))
	for i, s := range steps {
		out[i] = store.Step{Action: s.Action, Client: s.Client, Drain: s.Drain, Draws: s.Draws}
	}
	return out
}

// identityOf returns a failure identity in the form of an entry: the base
// name of its file, and its panic's type as an error.
func identityOf(i engine.Identity) store.Identity {
	id := store.Identity{Assertion: i.Assertion, Contract: i.Contract, Error: i.Panic}
	if i.Where != (assert.Where{}) {
		id.File, id.Line = filepath.Base(i.Where.File), i.Where.Line
	}
	return id
}

// drawsOf returns the draws of a case in the form of an entry: each label
// with the typed literal of its value, and without a value where no typed
// literal states it. A value that [Of] or [OfShape] derives is stated as its
// shape states it, such as a record for a struct, so the entry runs back
// through [Draws].
func drawsOf(draws []engine.Drawn) []store.Draw {
	out := make([]store.Draw, len(draws))
	for i, d := range draws {
		out[i].Label = d.Label
		if value, ok := literal.Encode(d.Neutral()); ok {
			out[i].Value = value
		}
	}
	return out
}

// storedCases returns each entry of stored as a stored case of a run, oldest
// first: its choices, the comparison of a case with its counterexample, and
// its counterexample as a trace.
func storedCases(stored store.Stored) []engine.Stored {
	out := make([]engine.Stored, len(stored.Entries))
	for i, e := range stored.Entries {
		trace, _ := traceOf(e)
		out[i] = engine.Stored{
			Choices: e.Choices,
			Matches: func(c *engine.Case) bool { return matches(e, c) },
			Trace:   trace,
		}
	}
	return out
}

// matches reports whether the case c decodes to the counterexample of e:
// draws that [alike] reports alike, and the steps of e for an entry with
// steps. An entry without steps records the draws alone, so the steps of c
// do not count against it.
func matches(e store.Entry, c *engine.Case) bool {
	if !alike(e.Counterexample, drawsOf(c.Draws())) {
		return false
	}
	return len(e.Steps) == 0 || slices.Equal(e.Steps, stepsOf(c.Steps()))
}

// traceOf returns the counterexample of e as the entries of a trace, in
// request order, with the value that the typed literal of each draw decodes
// to. A draw without a value is an entry without a value, which a composite
// takes by its label and every other draw refuses. traceOf returns no trace
// and a fault at the first draw whose value is no typed literal.
func traceOf(e store.Entry) ([]engine.Entry, error) {
	trace := make([]engine.Entry, 0, len(e.Counterexample)+len(e.Steps))
	for d, s := range e.Entries() {
		if s != nil {
			step := engine.MachineStep{Action: s.Action, Client: s.Client, Drain: s.Drain}
			trace = append(trace, engine.Entry{Step: &step})
			continue
		}
		if d.Value == nil {
			trace = append(trace, engine.Entry{Label: d.Label, Unstated: true})
			continue
		}
		value, err := literal.Decode(d.Value)
		if err != nil {
			return nil, fault.At(err, fault.Index(len(trace)), fault.Field(valueMember))
		}
		trace = append(trace, engine.Entry{Label: d.Label, Value: value})
	}
	return trace, nil
}

// storeFaults returns the faults that a run notes about its store, after r
// replayed the stored cases of stored: the fault of each file that the run
// skips, and of each stored case that decodes to other values than its
// entry records.
func (p property) storeFaults(stored store.Stored, r engine.Result) []error {
	return slices.Concat(p.skipped(stored), p.differences(stored.Entries, r.Stored))
}

// skipped returns the fault of each file of stored that the run skips, as a
// fault of the property's operation at the store's directory.
func (p property) skipped(stored store.Stored) []error {
	faults := make([]error, len(stored.Skipped))
	for i, err := range stored.Skipped {
		faults[i] = fault.In(p.op, fault.At(err, fault.Field(p.dir)))
	}
	return faults
}

// differences returns a fault of the property's operation at the file of
// each stored entry whose case of the choices decodes to other values than
// the entry records. The fault of a run that replayed the counterexample of
// the entry states the replay. The fault of a run that tested the values of
// the choices has the refusal of the counterexample as its cause, unless
// the case of the choices failed or ended with a fault. runs are the runs
// of the stored cases in the order of entries, up to the one that ended the
// run.
func (p property) differences(entries []store.Entry, runs []engine.StoredRun) []error {
	faults := make([]error, 0, len(runs))
	for i, run := range runs {
		e := entries[i]
		var note *fault.Error
		switch {
		case run.Traced:
			note = fault.New(replayedReason)
		case !matches(e, run.Case):
			note = fault.New(testedReason)
			if refusal := refusalOf(e, run); refusal != nil {
				note = note.Because(fault.At(refusal, fault.Field(store.FieldCounterexample)))
			}
		default:
			continue
		}
		faults = append(faults, fault.In(p.op, fault.At(note, fault.Field(p.dir), fault.Field(e.Name()))))
	}
	return faults
}

// refusalOf returns why run, the run of the stored case of e, did not count
// the case of its trace: the refusal of the run, or the fault of a
// counterexample that is no trace. It returns nil for a case of the choices
// that failed or ended with a fault. Such a case counts without a trace.
func refusalOf(e store.Entry, run engine.StoredRun) error {
	if run.Refusal != nil || run.Status == engine.CaseFailed || run.Status == engine.CaseFaulted {
		return run.Refusal
	}
	_, err := traceOf(e)
	return err
}

// alike reports whether replayed states the labels of recorded in order,
// and the value of each draw whose value recorded states. Two values are
// alike when their typed literals state one value, as the definition
// compares two literals. The spelling of a number or an escape in a file
// that another implementation wrote does not count, and a recorded -0
// differs from a replayed +0.
func alike(recorded, replayed []store.Draw) bool {
	if len(recorded) != len(replayed) {
		return false
	}
	for i, r := range recorded {
		if r.Label != replayed[i].Label || (r.Value != nil && !sameValue(r.Value, replayed[i].Value)) {
			return false
		}
	}
	return true
}

// sameValue reports whether a and b are typed literals of one value: two
// literals that decode to values with one canonical text. A replayed value
// b is nil for a value that no typed literal states, which is the same as
// no stored value, and a stored value a that does not decode differs from
// every value.
func sameValue(a, b json.RawMessage) bool {
	if b == nil {
		return false
	}
	x, err := literal.Decode(a)
	if err != nil {
		return false
	}
	y, err := literal.Decode(b)
	return err == nil && literal.Canonical(x) == literal.Canonical(y)
}
