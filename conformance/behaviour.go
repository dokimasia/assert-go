// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
	"go.dokimi.dev/assert/internal/prop/token"
	"go.dokimi.dev/assert/prop"
)

// The property of a behaviour vector, and the identity of the entries that
// keep its stored cases.
const (
	// behaviourContract is the contract of the property.
	behaviourContract = "the behaviour vector holds"
	// storedAssertion is the assertion of a stored case's entry.
	storedAssertion = "stored"
	// storedContract is the contract of a stored case's entry.
	storedContract = "a stored case of the vector"
)

// The names of the fields of a record's detail, as the definition states
// them.
const (
	outcomeField        = "outcome"
	casesField          = "cases"
	rejectedField       = "rejected"
	seedField           = "seed"
	counterexampleField = "counterexample"
	failureField        = "failure"
	choicesField        = "choices"
	othersField         = "others"
	divergenceField     = "divergence"
	coverageField       = "coverage"
)

// The field names of a draw, a divergence and a coverage requirement in a
// record's detail.
const (
	labelField          = "label"
	valueField          = "value"
	anyValueFailsField  = "any-value-fails"
	nearestPassingField = "nearest-passing"
	whatField           = "what"
	indexField          = "index"
	recordedField       = "recorded"
	replayedField       = "replayed"
	shareField          = "share"
	countedField        = "counted"
	validField          = "valid"
	verdictField        = "verdict"
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

// behaviourDetail is the detail of a behaviour vector's run, as the vector
// states it. A null field is nil.
type behaviourDetail struct {
	Outcome        string          `json:"outcome"`
	Cases          int             `json:"cases"`
	Rejected       int             `json:"rejected"`
	Seed           string          `json:"seed"`
	Counterexample []drawnSpec     `json:"counterexample"`
	Failure        *string         `json:"failure"`
	Choices        *string         `json:"choices"`
	Others         []otherSpec     `json:"others"`
	Divergence     *divergenceSpec `json:"divergence"`
	Coverage       *shortfallSpec  `json:"coverage"`
}

// drawnSpec is one draw of a counterexample, as a vector states it.
type drawnSpec struct {
	Label          string          `json:"label"`
	Value          json.RawMessage `json:"value"`
	AnyValueFails  *bool           `json:"any-value-fails"`
	NearestPassing json.RawMessage `json:"nearest-passing"`
}

// otherSpec is another failure of a run, as a vector states it.
type otherSpec struct {
	Failure        string      `json:"failure"`
	Counterexample []drawnSpec `json:"counterexample"`
	Choices        string      `json:"choices"`
}

// divergenceSpec is a divergence, as a vector states it: a side is a
// request's bounds, a fingerprint, an identity, or null.
type divergenceSpec struct {
	What     string          `json:"what"`
	Index    int             `json:"index"`
	Recorded json.RawMessage `json:"recorded"`
	Replayed json.RawMessage `json:"replayed"`
}

// shortfallSpec is a coverage requirement that a run missed, as a vector
// states it.
type shortfallSpec struct {
	Label   string  `json:"label"`
	Share   float64 `json:"share"`
	Counted int     `json:"counted"`
	Valid   int     `json:"valid"`
	Verdict string  `json:"verdict"`
}

// boundsSpec is a request's bounds, as a vector states them.
type boundsSpec struct {
	Kind     string          `json:"kind"`
	Min      json.RawMessage `json:"min"`
	Max      json.RawMessage `json:"max"`
	AllowNaN bool            `json:"allow_nan"`
	Width    int             `json:"width"`
	K        uint32          `json:"k"`
	MinSize  int             `json:"min_size"`
	MaxSize  *int            `json:"max_size"`
}

// checkBehaviour runs the body of a behaviour vector under its settings,
// and compares the run with the detail it states. A failing run is
// compared through the record that ForAll reports to a recorder, every
// detail field of it. A passing run reports no record, so its counts are
// compared through the engine's run of the same settings.
func checkBehaviour(raw json.RawMessage, dir string) error {
	var v struct {
		// Body is the body spec.
		Body json.RawMessage `json:"body"`
		// Settings are the settings of the run.
		Settings behaviourSettings `json:"settings"`
		// Detail is the detail of the run.
		Detail behaviourDetail `json:"detail"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	opts, s, replay, err := v.Settings.resolve(dir)
	if err != nil {
		return err
	}
	body, err := bodyOf(v.Body)
	if err != nil {
		return err
	}
	rec := assert.NewRecorder()
	prop.ForAll(rec, behaviourContract, body, opts...)
	if records := rec.Failures(); len(records) > 0 {
		return v.Detail.compare(records[0].Detail)
	}
	if v.Detail.Outcome != prop.Passed.String() {
		return fmt.Errorf("the run passes, want %s: %s", v.Detail.Outcome, rec.Message())
	}
	fresh, _ := bodyOf(v.Body)
	var r engine.Result
	if replay != nil {
		r = engine.RunReplay(engineBody(fresh), s, replay)
	} else {
		r = engine.Run(engineBody(fresh), s)
	}
	got := []any{r.Outcome.String(), r.Cases, r.Rejected, strconv.FormatUint(r.Seed, 10)}
	want := []any{v.Detail.Outcome, v.Detail.Cases, v.Detail.Rejected, v.Detail.Seed}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("the run ends as %v, want %v", got, want)
	}
	return nil
}

// resolve returns the options of a run under s, with the stored cases
// written to a store in dir, the engine's settings of the same run, and
// the choices to replay, nil for a run without them.
func (s behaviourSettings) resolve(dir string) ([]prop.Option, engine.Settings, []choice.Choice, error) {
	seed, err := strconv.ParseUint(s.Seed, 10, 64)
	if err != nil {
		return nil, engine.Settings{}, nil, err
	}
	opts := []prop.Option{prop.Seed(seed), prop.ShrinkTime(0), prop.Store(dir)}
	settings := engine.Settings{
		Seed:         seed,
		Cases:        engine.DefaultCases,
		MaxChoices:   engine.MaxChoices,
		Requirements: s.Requirements,
		Shrink:       engine.DefaultShrink,
		Explain:      true,
		Workers:      1,
	}
	if s.Cases != nil {
		opts, settings.Cases = append(opts, prop.Cases(*s.Cases)), *s.Cases
	}
	if s.MaxChoices != nil {
		opts, settings.MaxChoices = append(opts, prop.MaxChoices(*s.MaxChoices)), *s.MaxChoices
	}
	if s.Shrink != nil {
		opts, settings.Shrink = append(opts, prop.Shrink(*s.Shrink)), *s.Shrink
	}
	if s.Workers != nil {
		opts, settings.Workers = append(opts, prop.Workers(*s.Workers)), *s.Workers
	}
	for _, r := range s.Requirements {
		opts = append(opts, prop.Require(r.Label, r.Share))
	}
	err = s.store(dir, &settings)
	if err != nil {
		return nil, engine.Settings{}, nil, err
	}
	if s.Replay == nil {
		return opts, settings, nil, nil
	}
	replay, err := token.Decode(*s.Replay)
	if err != nil {
		return nil, engine.Settings{}, nil, err
	}
	return append(opts, prop.Replay(*s.Replay)), settings, replay, nil
}

// store writes each stored case of s to the store dir as an entry of the
// behaviour vector's property, and adds its choices to settings.
func (s behaviourSettings) store(dir string, settings *engine.Settings) error {
	version, err := Version()
	if err != nil {
		return err
	}
	for i, forms := range s.Stored {
		choices, err := parseChoices(forms)
		if err != nil {
			return err
		}
		entry := store.Entry{
			Definition: version,
			Property:   behaviourContract,
			Identity:   store.Identity{Assertion: storedAssertion, Contract: storedContract},
			Choices:    choices,
			Found:      firstStored.AddDate(0, 0, i),
		}
		if _, err := store.Save(dir, entry); err != nil {
			return err
		}
		settings.Stored = append(settings.Stored, choices)
	}
	return nil
}

// compare returns how the detail of a record differs from d, or nil when
// they match.
func (d behaviourDetail) compare(detail map[string]any) error {
	want, err := d.normal()
	if err != nil {
		return err
	}
	got := normalRecord(detail)
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("the record's detail is %s, want %s", jsonOf(got), jsonOf(want))
	}
	return nil
}

// normal returns d in the form that normalRecord returns a record's
// detail in: a typed literal as its canonical text, the bounds of a
// request as their text, and an absent field as nil.
func (d behaviourDetail) normal() (map[string]any, error) {
	out := map[string]any{
		outcomeField:        d.Outcome,
		casesField:          d.Cases,
		rejectedField:       d.Rejected,
		seedField:           d.Seed,
		counterexampleField: nil,
		failureField:        nil,
		choicesField:        nil,
		othersField:         nil,
		divergenceField:     nil,
		coverageField:       nil,
	}
	if d.Counterexample != nil {
		draws, err := normalDraws(d.Counterexample, true)
		if err != nil {
			return nil, err
		}
		out[counterexampleField] = draws
	}
	if d.Failure != nil {
		out[failureField] = *d.Failure
	}
	if d.Choices != nil {
		out[choicesField] = *d.Choices
	}
	if d.Others != nil {
		others := make([]any, len(d.Others))
		for i, o := range d.Others {
			draws, err := normalDraws(o.Counterexample, false)
			if err != nil {
				return nil, err
			}
			others[i] = map[string]any{failureField: o.Failure, counterexampleField: draws, choicesField: o.Choices}
		}
		out[othersField] = others
	}
	if d.Divergence != nil {
		recorded, err := normalSide(d.Divergence.Recorded)
		if err != nil {
			return nil, err
		}
		replayed, err := normalSide(d.Divergence.Replayed)
		if err != nil {
			return nil, err
		}
		out[divergenceField] = map[string]any{
			whatField:     d.Divergence.What,
			indexField:    d.Divergence.Index,
			recordedField: recorded,
			replayedField: replayed,
		}
	}
	if c := d.Coverage; c != nil {
		out[coverageField] = normalShortfall(c.Label, c.Share, c.Counted, c.Valid, c.Verdict)
	}
	return out, nil
}

// normalDraws returns the draws of a vector's counterexample in normal
// form, with what the explain phase found where explained is set.
func normalDraws(draws []drawnSpec, explained bool) ([]any, error) {
	out := make([]any, len(draws))
	for i, d := range draws {
		value, err := Decode(d.Value)
		if err != nil {
			return nil, err
		}
		draw := map[string]any{labelField: d.Label, valueField: canonical(value)}
		if explained {
			nearest, err := normalLiteral(d.NearestPassing)
			if err != nil {
				return nil, err
			}
			draw[anyValueFailsField], draw[nearestPassingField] = nil, nearest
			if d.AnyValueFails != nil {
				draw[anyValueFailsField] = *d.AnyValueFails
			}
		}
		out[i] = draw
	}
	return out, nil
}

// normalLiteral returns the canonical text of a typed literal, and nil for
// JSON null.
func normalLiteral(raw json.RawMessage) (any, error) {
	if string(raw) == jsonNull {
		return nil, nil
	}
	v, err := Decode(raw)
	if err != nil {
		return nil, err
	}
	return canonical(v), nil
}

// normalSide returns one side of a vector's divergence in the form that
// prop's record states it: a request's bounds as their text, an identity
// as itself, a fingerprint as a uint64, and nil for null.
func normalSide(raw json.RawMessage) (any, error) {
	if string(raw) == jsonNull {
		return nil, nil
	}
	var identity string
	if json.Unmarshal(raw, &identity) == nil {
		return identity, nil
	}
	var fingerprint uint64
	if json.Unmarshal(raw, &fingerprint) == nil {
		return fingerprint, nil
	}
	var bounds boundsSpec
	if err := json.Unmarshal(raw, &bounds); err != nil {
		return nil, err
	}
	return bounds.text()
}

// text returns the bounds as prop's record states them: "integer in [lo,
// hi]", "float in [lo, hi] of width w", with " or NaN" when NaN is a
// value, and "sequence of min to max values below k" or "sequence of min
// or more values below k".
func (b boundsSpec) text() (string, error) {
	if b.Kind == integerGen {
		lo, err := parseInt(b.Min)
		if err != nil {
			return "", err
		}
		hi, err := parseInt(b.Max)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("integer in [%s, %s]", lo, hi), nil
	}
	if b.Kind == floatGen {
		lo, err := decodeFloat(b.Min)
		if err != nil {
			return "", err
		}
		hi, err := decodeFloat(b.Max)
		if err != nil {
			return "", err
		}
		text := fmt.Sprintf("float in [%v, %v] of width %d", lo, hi, b.Width)
		if b.AllowNaN {
			text += " or NaN"
		}
		return text, nil
	}
	if b.MaxSize == nil {
		return fmt.Sprintf("sequence of %d or more values below %d", b.MinSize, b.K), nil
	}
	return fmt.Sprintf("sequence of %d to %d values below %d", b.MinSize, *b.MaxSize, b.K), nil
}

// normalRecord returns the detail of a record in normal form.
func normalRecord(detail map[string]any) map[string]any {
	out := map[string]any{
		outcomeField:        fmt.Sprint(detail[outcomeField]),
		casesField:          detail[casesField],
		rejectedField:       detail[rejectedField],
		seedField:           detail[seedField],
		counterexampleField: nil,
		failureField:        nil,
		choicesField:        detail[choicesField],
		othersField:         nil,
		divergenceField:     nil,
		coverageField:       nil,
	}
	if drawn, ok := detail[counterexampleField].([]prop.Drawn); ok {
		out[counterexampleField] = normalDrawn(drawn, true)
	}
	if f, ok := detail[failureField].(assert.Failure); ok {
		out[failureField] = f.Assertion
	}
	if others, ok := detail[othersField].([]prop.Other); ok {
		normal := make([]any, len(others))
		for i, o := range others {
			normal[i] = map[string]any{
				failureField:        o.Failure.Assertion,
				counterexampleField: normalDrawn(o.Counterexample, false),
				choicesField:        o.Choices,
			}
		}
		out[othersField] = normal
	}
	if d, ok := detail[divergenceField].(*prop.Divergence); ok {
		out[divergenceField] = map[string]any{
			whatField:     d.What.String(),
			indexField:    d.Index,
			recordedField: d.Recorded,
			replayedField: d.Replayed,
		}
	}
	if s, ok := detail[coverageField].(*prop.Shortfall); ok {
		out[coverageField] = normalShortfall(s.Label, s.Share, s.Counted, s.Valid, s.Verdict.String())
	}
	return out
}

// normalDrawn returns the draws of a record in normal form, with what the
// explain phase found where explained is set.
func normalDrawn(drawn []prop.Drawn, explained bool) []any {
	out := make([]any, len(drawn))
	for i, d := range drawn {
		draw := map[string]any{labelField: d.Label, valueField: canonical(d.Value)}
		if explained {
			draw[anyValueFailsField], draw[nearestPassingField] = nil, nil
			if d.Relevance != prop.Untested {
				draw[anyValueFailsField] = d.Relevance == prop.AnyValueFails
			}
			if d.NearestPassing != nil {
				draw[nearestPassingField] = canonical(d.NearestPassing)
			}
		}
		out[i] = draw
	}
	return out
}

// normalShortfall returns a coverage requirement in normal form.
func normalShortfall(label string, share float64, counted, valid int, verdict string) map[string]any {
	return map[string]any{
		labelField:   label,
		shareField:   share,
		countedField: counted,
		validField:   valid,
		verdictField: verdict,
	}
}
