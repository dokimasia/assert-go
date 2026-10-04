// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/record"
)

// storeRoot is the directory of the stores of a package's tests, relative
// to the package's directory, beside testdata/golden.
const storeRoot = "testdata/prop"

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
// name, and otherwise the empty string, which states no store.
func directoryOf(tb assert.TB, c config) string {
	if c.storeSet {
		return c.store
	}
	if n, ok := tb.(named); ok {
		return filepath.Join(storeRoot, n.Name())
	}
	return ""
}

// claim claims the entries of contract in the store dir for the test of tb
// until the test ends, and reports false when a property of another running
// test has claimed them. A run without a store, and a seat without Cleanup,
// claim nothing and report true.
func claim(tb assert.TB, dir, contract string) bool {
	c, ok := tb.(cleaner)
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
		Found:          found,
	}
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
// each stored entry whose case now decodes to other values than the entry
// records. runs are the runs of the stored cases in the order of entries,
// up to the one that ended the run.
func (p property) differences(entries []store.Entry, runs []engine.Execution) []error {
	var faults []error
	for i, run := range runs {
		if !alike(entries[i].Counterexample, drawsOf(run.Case.Draws())) {
			faults = append(faults, fault.In(p.op, fault.At(
				fault.New("the stored case decodes to other values than it records, and the run tested those"),
				fault.Field(p.dir), fault.Field(entries[i].Name()))))
		}
	}
	return faults
}

// alike reports whether replayed states the labels of recorded in order,
// and the value of each draw whose value recorded states. Two values are
// alike when they decode to the same JSON value, so that the spelling of a
// number or an escape in a file that another implementation wrote does not
// count.
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

// sameValue reports whether a and b are JSON texts of the same value. A
// stored value a is JSON, because its file parsed. A replayed value b is
// nil for a value that no typed literal states, which is the same as no
// stored value.
func sameValue(a, b json.RawMessage) bool {
	var x, y any
	return b != nil && json.Unmarshal(a, &x) == nil && json.Unmarshal(b, &y) == nil && reflect.DeepEqual(x, y)
}
