// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/token"
)

// choiceForm is a choice in the corpus form of a vector: a JSON integer, or
// a decimal string beyond 2^53 - 1 in magnitude, for an integer; an object
// with the one key float, of a number or a float's name, for a float; and
// an object with the one key sequence, of a list of integers, for a
// sequence.
type choiceForm struct {
	// Float is the float of a float choice, and nil for another kind.
	Float json.RawMessage `json:"float"`
	// Sequence are the elements of a sequence choice, and nil for another
	// kind.
	Sequence []uint32 `json:"sequence"`
}

// parseChoices returns the choices that a list of corpus forms states. It
// returns the fault of a form that states no choice, at the form's index.
func parseChoices(forms []json.RawMessage) ([]choice.Choice, error) {
	out := make([]choice.Choice, len(forms))
	for i, form := range forms {
		c, err := parseChoice(form)
		if err != nil {
			return nil, fault.At(err, fault.Index(i))
		}
		out[i] = c
	}
	return out, nil
}

// parseChoice returns the choice that one corpus form states.
func parseChoice(form json.RawMessage) (choice.Choice, error) {
	var object choiceForm
	if json.Unmarshal(form, &object) == nil {
		if object.Float != nil {
			f, err := literal.Float(object.Float)
			if err != nil {
				return choice.Choice{}, fault.At(err, fault.Field(floatMember))
			}
			return choice.Choice{Kind: choice.Float, Float: f}, nil
		}
		if object.Sequence != nil {
			return choice.Choice{Kind: choice.Sequence, Sequence: object.Sequence}, nil
		}
		return choice.Choice{}, fault.New("%s is no choice", form)
	}
	i, err := parseInt(form)
	if err != nil {
		return choice.Choice{}, err
	}
	return choice.Choice{Kind: choice.Integer, Integer: i}, nil
}

// parseInt returns the integer that a JSON integer states, or a decimal
// string beyond 2^53 - 1 in magnitude.
func parseInt(raw json.RawMessage) (choice.Int, error) {
	v, err := literal.Int(raw)
	if err != nil {
		return choice.Int{}, err
	}
	switch v := v.(type) {
	case int:
		return choice.IntOf(int64(v)), nil
	case int64:
		return choice.IntOf(v), nil
	}
	return choice.UintOf(v.(uint64)), nil
}

// sameChoices reports whether got are the choices that the corpus forms of
// want state, in order, with floats compared by their bits. It returns the
// fault of a form that states no choice.
func sameChoices(got []choice.Choice, want []json.RawMessage) (bool, error) {
	parsed, err := parseChoices(want)
	if err != nil {
		return false, err
	}
	if len(got) != len(parsed) {
		return false, nil
	}
	for i := range got {
		if !got[i].Equal(parsed[i]) {
			return false, nil
		}
	}
	return true, nil
}

// compareChoices returns a fault of how got differs from the choices that
// the corpus forms of want state, or nil when they match. The fault states
// got as its replay token. It returns the fault of a form that states no
// choice.
func compareChoices(got []choice.Choice, want []json.RawMessage) error {
	same, err := sameChoices(got, want)
	if err != nil {
		return err
	}
	if !same {
		return fault.New("the choices are %s, want %s", token.Encode(got), jsonOf(want))
	}
	return nil
}
