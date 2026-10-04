// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"
	"reflect"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/prop"
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

// runDetail is the detail of the record of a property's run, as a vector
// states it. A null field is nil.
type runDetail struct {
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
	Width    uint8           `json:"width"`
	K        uint32          `json:"k"`
	MinSize  *int            `json:"min_size"`
	MaxSize  *int            `json:"max_size"`
}

// compare returns a fault at the first field of the detail of a record
// that differs from d, in the order that the definition states the fields,
// or nil when they match. It returns the fault of a value that d
// misstates, with a path inside d.
func (d runDetail) compare(detail map[string]any) error {
	want, err := d.normal()
	if err != nil {
		return err
	}
	return firstDifference(normalRecord(detail), want, outcomeField, casesField, rejectedField, seedField,
		counterexampleField, failureField, choicesField, othersField, divergenceField, coverageField)
}

// comparePassed returns a fault at the first field of the detail of c that
// differs from d, in the order that the definition states the fields, or
// nil when they match. c is the call record of a property whose run
// reported no failure. The call record states the detail of a run in the
// form that a vector states it. It returns the fault of a value that d
// misstates, with a path inside d.
func (d runDetail) comparePassed(c call) error {
	want, err := d.normal()
	if err != nil {
		return err
	}
	var stated runDetail
	// A run that reports no failure states null for its failure, so its
	// detail decodes as a vector's does, and normal decodes no literal of it.
	_ = json.Unmarshal(c.Detail, &stated)
	got, _ := stated.normal()
	return firstDifference(got, want, outcomeField, casesField, rejectedField, seedField,
		counterexampleField, failureField, choicesField, othersField, divergenceField, coverageField)
}

// firstDifference returns a fault at the first of fields whose normal form
// in got differs from the one in want, which states both as JSON, or nil
// when every field matches.
func firstDifference(got, want map[string]any, fields ...string) error {
	for _, field := range fields {
		if !reflect.DeepEqual(got[field], want[field]) {
			return fault.At(fault.New("the run states %s, want %s", jsonOf(got[field]), jsonOf(want[field])),
				fault.Field(field))
		}
	}
	return nil
}

// normal returns d in the form that normalRecord returns a record's
// detail in: a typed literal as its canonical text, the bounds of a
// request as their text, and an absent field as nil.
func (d runDetail) normal() (map[string]any, error) {
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
			return nil, fault.At(err, fault.Field(counterexampleField))
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
				return nil, fault.At(err, fault.Field(othersField), fault.Index(i), fault.Field(counterexampleField))
			}
			others[i] = map[string]any{failureField: o.Failure, counterexampleField: draws, choicesField: o.Choices}
		}
		out[othersField] = others
	}
	if v := d.Divergence; v != nil {
		recorded, err := normalSide(v.Recorded)
		if err != nil {
			return nil, fault.At(err, fault.Field(divergenceField), fault.Field(recordedField))
		}
		replayed, err := normalSide(v.Replayed)
		if err != nil {
			return nil, fault.At(err, fault.Field(divergenceField), fault.Field(replayedField))
		}
		out[divergenceField] = map[string]any{
			whatField:     v.What,
			indexField:    v.Index,
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
		value, err := literal.Decode(d.Value)
		if err != nil {
			return nil, fault.At(err, fault.Index(i), fault.Field(valueField))
		}
		draw := map[string]any{labelField: d.Label, valueField: literal.Canonical(value)}
		if explained {
			nearest, err := normalLiteral(d.NearestPassing)
			if err != nil {
				return nil, fault.At(err, fault.Index(i), fault.Field(nearestPassingField))
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
	v, err := literal.Decode(raw)
	if err != nil {
		return nil, err
	}
	return literal.Canonical(v), nil
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
	var spec boundsSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fault.New("the side is no bounds, fingerprint or identity").Because(err)
	}
	bounds, err := spec.bounds()
	if err != nil {
		return nil, err
	}
	return bounds.String(), nil
}

// bounds returns the bounds that b states, of the kind integer, float or
// sequence.
func (b boundsSpec) bounds() (choice.Bounds, error) {
	switch b.Kind {
	case choice.Integer.String():
		lo, err := parseInt(b.Min)
		if err != nil {
			return choice.Bounds{}, fault.At(err, fault.Field(minMember))
		}
		hi, err := parseInt(b.Max)
		if err != nil {
			return choice.Bounds{}, fault.At(err, fault.Field(maxMember))
		}
		bounds, err := choice.NewIntegerBounds(lo, hi)
		return choice.OfInteger(bounds), err
	case choice.Float.String():
		lo, err := literal.Float(b.Min)
		if err != nil {
			return choice.Bounds{}, fault.At(err, fault.Field(minMember))
		}
		hi, err := literal.Float(b.Max)
		if err != nil {
			return choice.Bounds{}, fault.At(err, fault.Field(maxMember))
		}
		nan := choice.ExcludeNaN
		if b.AllowNaN {
			nan = choice.AdmitNaN
		}
		bounds, err := choice.NewFloatBounds(lo, hi, nan, choice.Width(b.Width))
		return choice.OfFloat(bounds), err
	case choice.Sequence.String():
		sizes, err := sizesOf(b.MinSize, b.MaxSize)
		if err != nil {
			return choice.Bounds{}, err
		}
		bounds, err := choice.NewSequenceBounds(b.K, sizes)
		return choice.OfSequence(bounds), err
	}
	return choice.Bounds{}, fault.At(fault.New("%q is no kind of bounds", b.Kind), fault.Field(kindMember))
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
		draw := map[string]any{labelField: d.Label, valueField: literal.Canonical(d.Value)}
		if explained {
			draw[anyValueFailsField], draw[nearestPassingField] = nil, nil
			if d.Relevance != prop.Untested {
				draw[anyValueFailsField] = d.Relevance == prop.AnyValueFails
			}
			if d.NearestPassing != nil {
				draw[nearestPassingField] = literal.Canonical(d.NearestPassing)
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
