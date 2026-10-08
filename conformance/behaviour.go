// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/prop"
)

// The property of a behaviour, a recording or a machines vector, and the
// identity of the entries that keep its stored cases.
const (
	// behaviourContract is the contract of the property, whose store entries
	// resolve writes.
	behaviourContract = "the behaviour vector is true"
	// storedAssertion is the assertion of a stored case's entry.
	storedAssertion = "stored"
	// storedContract is the contract of a stored case's entry.
	storedContract = "a stored case of the vector"
)

// firstStored is the date of discovery of a behaviour vector's first
// stored case. Each later case was found a day later, so the run tries
// them in the order the vector states.
var firstStored = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// behaviourSettings are the settings of a behaviour vector's run. A nil
// field states the default.
type behaviourSettings struct {
	Seed         string               `json:"seed"`
	Cases        *int                 `json:"cases"`
	MaxChoices   *int                 `json:"max-choices"`
	Requirements []engine.Requirement `json:"requirements"`
	Stored       [][]json.RawMessage  `json:"stored"`
	Shrink       *int                 `json:"shrink"`
	Replay       *string              `json:"replay"`
	Workers      *int                 `json:"workers"`
}

// checkBehaviour runs the body of a behaviour vector under its settings,
// and compares the run with the detail it states. A failing run is
// compared through the record that ForAll reports to a recorder, every
// detail field of it. A passing run reports no record, so it is compared
// through the detail that the property's call record states.
func checkBehaviour(raw json.RawMessage, dir string) error {
	var v struct {
		// Body is the body spec.
		Body json.RawMessage `json:"body"`
		// Settings are the settings of the run.
		Settings behaviourSettings `json:"settings"`
		// Detail is the detail of the run.
		Detail runDetail `json:"detail"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	opts, err := v.Settings.resolve(dir)
	if err != nil {
		return fault.At(err, fault.Field(settingsMember))
	}
	body, err := bodyOf(v.Body, failing)
	if err != nil {
		return fault.At(err, fault.Field(bodyMember))
	}
	rec := assert.NewRecorder()
	prop.ForAll(rec, behaviourContract, body, opts...)
	if records := rec.Failures(); len(records) > 0 {
		return at(v.Detail.compare(records[0].Detail), fault.Field(detailMember))
	}
	return at(v.Detail.comparePassed(callsOf(rec)[0]), fault.Field(detailMember))
}

// resolve returns the options of a run under s, with the stored cases
// written to a store in dir. It returns a fault with a path inside the
// settings.
func (s behaviourSettings) resolve(dir string) ([]prop.Option, error) {
	seed, err := parseSeed(s.Seed)
	if err != nil {
		return nil, err
	}
	opts := []prop.Option{prop.Seed(seed), prop.ShrinkTime(0), prop.Store(dir)}
	if s.Cases != nil {
		opts = append(opts, prop.Cases(*s.Cases))
	}
	if s.MaxChoices != nil {
		opts = append(opts, prop.MaxChoices(*s.MaxChoices))
	}
	if s.Shrink != nil {
		opts = append(opts, prop.Shrink(*s.Shrink))
	}
	if s.Workers != nil {
		opts = append(opts, prop.Workers(*s.Workers))
	}
	for _, r := range s.Requirements {
		opts = append(opts, prop.Require(r.Label, r.Share))
	}
	if err := s.store(dir); err != nil {
		return nil, err
	}
	if s.Replay == nil {
		return opts, nil
	}
	if _, err := token.Decode(*s.Replay); err != nil {
		return nil, fault.At(err, fault.Field(replayMember))
	}
	return append(opts, prop.Replay(*s.Replay)), nil
}

// store writes each stored case of s to the store dir as an entry of the
// vector's property. It returns the fault of a stored case at its index.
func (s behaviourSettings) store(dir string) error {
	for i, forms := range s.Stored {
		choices, err := parseChoices(forms)
		if err != nil {
			return fault.At(err, fault.Field(storedMember), fault.Index(i))
		}
		entry := store.Entry{
			Definition: Version(),
			Property:   behaviourContract,
			Identity:   store.Identity{Assertion: storedAssertion, Contract: storedContract},
			Choices:    choices,
			Found:      firstStored.AddDate(0, 0, i),
		}
		if _, err := store.Save(dir, entry); err != nil {
			return fault.At(err, fault.Field(storedMember), fault.Index(i))
		}
	}
	return nil
}
