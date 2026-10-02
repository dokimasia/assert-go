// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/matching"
)

// The ids of the generators that the corpus states, and of the position of
// a recursive value inside its extension.
const (
	integerGen        = "integer"
	durationGen       = "duration"
	floatGen          = "float"
	booleanGen        = "boolean"
	justGen           = "just"
	sampledFromGen    = "sampled-from"
	oneOfGen          = "one-of"
	optionalGen       = "optional"
	listGen           = "list"
	dictGen           = "dict"
	stringGen         = "string"
	bytesGen          = "bytes"
	permutationGen    = "permutation"
	stringMatchingGen = "string-matching"
	recursiveGen      = "recursive"
	filterGen         = "filter"
	selfGen           = "self"
)

// The widths of a float generator, and the parts of a probability.
const (
	// width32 is the width of a float32.
	width32 = 32
	// width64 is the width of a float64, the default.
	width64 = 64
	// probabilityParts are a probability's numerator and denominator.
	probabilityParts = 2
)

// generatorSpec is a generator as the corpus states it. Which fields a
// spec states depends on its id.
type generatorSpec struct {
	// Gen is the generator's id.
	Gen string `json:"gen"`
	// Min and Max are the bounds of integer, duration and float.
	Min json.RawMessage `json:"min"`
	Max json.RawMessage `json:"max"`
	// AllowNaN makes NaN a value of float.
	AllowNaN bool `json:"allow_nan"`
	// Width is the width of float, 64 when nil.
	Width *int `json:"width"`
	// P is the probability of true of boolean, as [numerator,
	// denominator], 1/2 when nil.
	P []uint64 `json:"p"`
	// Value is the typed literal of just.
	Value json.RawMessage `json:"value"`
	// Values are the typed literals of sampled-from and permutation, and
	// the generator of a dict's values.
	Values json.RawMessage `json:"values"`
	// Of is the generator of optional, list and filter, and the list of
	// generators of one-of.
	Of json.RawMessage `json:"of"`
	// Keys is the generator of a dict's keys.
	Keys json.RawMessage `json:"keys"`
	// MinSize is the shortest length, 0 when nil.
	MinSize *int `json:"min_size"`
	// MaxSize is the longest length, unbounded when nil.
	MaxSize *int `json:"max_size"`
	// Unique makes a list discard an element equal to an earlier one.
	Unique bool `json:"unique"`
	// Alphabet are the characters of string, the default alphabet when
	// nil.
	Alphabet *string `json:"alphabet"`
	// Pattern is the regular expression of string-matching.
	Pattern string `json:"pattern"`
	// Base and Extend are the base and the extension of recursive.
	Base   json.RawMessage `json:"base"`
	Extend json.RawMessage `json:"extend"`
	// MaxLeaves bounds the base values of recursive, 100 when nil.
	MaxLeaves *int `json:"max_leaves"`
	// Keep is the predicate of filter.
	Keep json.RawMessage `json:"keep"`
}

// generatorOf returns the generator that a corpus spec states, as a
// generator of any whose values are the Go values of the typed
// generators: int64 or uint64 for an integer, time.Duration, float32 or
// float64, bool, string, []byte, []any for a list and a permutation,
// map[any]any for a dict, nil for an absent optional, and the decoded
// literals of just and sampled-from.
//
// It returns an error for a spec that names no generator of the
// vocabulary or misstates a parameter, and for arguments that state no
// domain, for which the engine's constructors panic.
func generatorOf(raw json.RawMessage) (g engine.Generator[any], err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("conformance: %s states no domain: %v", raw, r)
		}
	}()
	return build(raw, nil)
}

// build returns the generator of raw, where self is the position of the
// recursive value whose extension contains raw, and nil outside one.
func build(raw json.RawMessage, self *engine.Generator[any]) (engine.Generator[any], error) {
	var spec generatorSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return engine.Generator[any]{}, fmt.Errorf("conformance: parse generator: %w", err)
	}
	switch spec.Gen {
	case integerGen:
		return integerOf(spec)
	case durationGen:
		return durationOf(spec)
	case floatGen:
		return floatOf(spec)
	case booleanGen:
		return booleanOf(spec)
	case justGen, sampledFromGen, permutationGen:
		return valuesOf(spec)
	case oneOfGen, optionalGen, listGen, filterGen:
		return composedOf(spec, self)
	case dictGen:
		return dictOf(spec, self)
	case stringGen, bytesGen, stringMatchingGen:
		return textOf(spec)
	case recursiveGen:
		return recursiveOf(spec)
	case selfGen:
		if self == nil {
			return engine.Generator[any]{}, fmt.Errorf("conformance: %s is outside a recursive extension", selfGen)
		}
		return *self, nil
	}
	return engine.Generator[any]{}, fmt.Errorf("conformance: %q names no generator", spec.Gen)
}

// erased returns g as a generator of any, which makes g's choices and adds
// no span.
func erased[T any](g engine.Generator[T]) engine.Generator[any] {
	return g.Map(func(v T) any { return v })
}

// integerOf returns the integer generator of spec: over int64 for bounds
// inside the signed range, and over uint64 for bounds inside the unsigned
// range alone.
func integerOf(spec generatorSpec) (engine.Generator[any], error) {
	lo, hi, err := integerBounds(spec)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if low, ok := lo.Int64(); ok {
		if high, ok := hi.Int64(); ok {
			return erased(engine.Integer(low, high)), nil
		}
	}
	low, lok := lo.Uint64()
	high, hok := hi.Uint64()
	if !lok || !hok {
		return engine.Generator[any]{}, fmt.Errorf("conformance: [%s, %s] lies inside no 64-bit range", lo, hi)
	}
	return erased(engine.Integer(low, high)), nil
}

// durationOf returns the duration generator of spec, whose bounds are
// nanoseconds inside the signed range.
func durationOf(spec generatorSpec) (engine.Generator[any], error) {
	lo, hi, err := integerBounds(spec)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	low, lok := lo.Int64()
	high, hok := hi.Int64()
	if !lok || !hok {
		return engine.Generator[any]{}, fmt.Errorf("conformance: [%s, %s] are no durations", lo, hi)
	}
	return erased(engine.Duration(time.Duration(low), time.Duration(high))), nil
}

// integerBounds returns the bounds min and max of spec.
func integerBounds(spec generatorSpec) (choice.Int, choice.Int, error) {
	lo, err := parseInt(spec.Min)
	if err != nil {
		return choice.Int{}, choice.Int{}, fmt.Errorf("conformance: %s min: %w", spec.Gen, err)
	}
	hi, err := parseInt(spec.Max)
	if err != nil {
		return choice.Int{}, choice.Int{}, fmt.Errorf("conformance: %s max: %w", spec.Gen, err)
	}
	return lo, hi, nil
}

// floatOf returns the float generator of spec, of width 32 or 64.
func floatOf(spec generatorSpec) (engine.Generator[any], error) {
	lo, err := decodeFloat(spec.Min)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	hi, err := decodeFloat(spec.Max)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	nan := choice.ExcludeNaN
	if spec.AllowNaN {
		nan = choice.AdmitNaN
	}
	low, high := lo.(float64), hi.(float64)
	if spec.Width == nil || *spec.Width == width64 {
		return erased(engine.Float(low, high, nan)), nil
	}
	if *spec.Width != width32 {
		return engine.Generator[any]{}, fmt.Errorf("conformance: a float of width %d", *spec.Width)
	}
	if !sameFloat(float64(float32(low)), low) || !sameFloat(float64(float32(high)), high) {
		return engine.Generator[any]{}, fmt.Errorf("conformance: [%v, %v] are no floats of width 32", low, high)
	}
	return erased(engine.Float(float32(low), float32(high), nan)), nil
}

// sameFloat reports whether a and b are one float, NaN included.
func sameFloat(a, b float64) bool {
	return a == b || (math.IsNaN(a) && math.IsNaN(b))
}

// booleanOf returns the boolean generator of spec.
func booleanOf(spec generatorSpec) (engine.Generator[any], error) {
	if spec.P == nil {
		return erased(engine.Boolean(1, 2)), nil
	}
	if len(spec.P) != probabilityParts {
		return engine.Generator[any]{}, fmt.Errorf("conformance: p is %v, not [numerator, denominator]", spec.P)
	}
	return erased(engine.Boolean(spec.P[0], spec.P[1])), nil
}

// valuesOf returns just, sampled-from or permutation, of the typed
// literals that spec states.
func valuesOf(spec generatorSpec) (engine.Generator[any], error) {
	if spec.Gen == justGen {
		v, err := Decode(spec.Value)
		if err != nil {
			return engine.Generator[any]{}, err
		}
		return engine.Just(v), nil
	}
	var literals []json.RawMessage
	if err := json.Unmarshal(spec.Values, &literals); err != nil {
		return engine.Generator[any]{}, fmt.Errorf("conformance: %s values: %w", spec.Gen, err)
	}
	values := make([]any, len(literals))
	for i, raw := range literals {
		v, err := Decode(raw)
		if err != nil {
			return engine.Generator[any]{}, err
		}
		values[i] = v
	}
	if spec.Gen == sampledFromGen {
		return engine.SampledFrom(values...), nil
	}
	return erased(engine.Permutation(values...)), nil
}

// composedOf returns one-of, optional, list or filter, over the
// generators that spec states.
func composedOf(spec generatorSpec, self *engine.Generator[any]) (engine.Generator[any], error) {
	if spec.Gen == oneOfGen {
		var specs []json.RawMessage
		if err := json.Unmarshal(spec.Of, &specs); err != nil {
			return engine.Generator[any]{}, fmt.Errorf("conformance: one-of of: %w", err)
		}
		gens := make([]engine.Generator[any], len(specs))
		for i, raw := range specs {
			g, err := build(raw, self)
			if err != nil {
				return engine.Generator[any]{}, err
			}
			gens[i] = g
		}
		return engine.OneOf(gens...), nil
	}
	of, err := build(spec.Of, self)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if spec.Gen == optionalGen {
		return engine.Optional(of).Map(func(v *any) any {
			if v == nil {
				return nil
			}
			return *v
		}), nil
	}
	if spec.Gen == filterGen {
		return filterOf(of, spec.Keep)
	}
	sizes, err := sizesOf(spec)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if spec.Unique {
		return erased(engine.UniqueList(of, sizes)), nil
	}
	return erased(engine.List(of, sizes)), nil
}

// filterOf returns the generator of the values of g that the predicate keep
// reports true for.
func filterOf(g engine.Generator[any], keep json.RawMessage) (engine.Generator[any], error) {
	holds, err := predicateOf(keep)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return g.Filter(holds), nil
}

// dictOf returns the dict generator of spec.
func dictOf(spec generatorSpec, self *engine.Generator[any]) (engine.Generator[any], error) {
	keys, err := build(spec.Keys, self)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	values, err := build(spec.Values, self)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	sizes, err := sizesOf(spec)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return erased(engine.Dict(keys, values, sizes)), nil
}

// textOf returns string, bytes or string-matching.
func textOf(spec generatorSpec) (engine.Generator[any], error) {
	if spec.Gen == stringMatchingGen {
		g, err := matching.StringMatching(spec.Pattern)
		if err != nil {
			return engine.Generator[any]{}, err
		}
		return erased(g), nil
	}
	sizes, err := sizesOf(spec)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if spec.Gen == bytesGen {
		return erased(engine.Bytes(sizes)), nil
	}
	if spec.Alphabet == nil {
		return erased(engine.String(sizes)), nil
	}
	return erased(engine.StringOver(*spec.Alphabet, sizes)), nil
}

// recursiveOf returns the recursive generator of spec. Its base takes no
// self, and each self of its extension is a position of its own value.
func recursiveOf(spec generatorSpec) (engine.Generator[any], error) {
	base, err := build(spec.Base, nil)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	maxLeaves := engine.DefaultMaxLeaves
	if spec.MaxLeaves != nil {
		maxLeaves = *spec.MaxLeaves
	}
	var extendErr error
	g := engine.Recursive(base, func(self engine.Generator[any]) engine.Generator[any] {
		extension, err := build(spec.Extend, &self)
		extendErr = err
		return extension
	}, maxLeaves)
	if extendErr != nil {
		return engine.Generator[any]{}, extendErr
	}
	return g, nil
}

// sizesOf returns the lengths that spec states: from min_size, 0 when
// nil, to max_size, unbounded when nil.
func sizesOf(spec generatorSpec) (choice.Sizes, error) {
	least := 0
	if spec.MinSize != nil {
		least = *spec.MinSize
	}
	if spec.MaxSize == nil {
		return choice.NewUnboundedSizes(least)
	}
	return choice.NewSizes(least, *spec.MaxSize)
}
