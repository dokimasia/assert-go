// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"math"
	"time"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
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
// literals of just and sampled-from. Each generator runs backwards from
// the value that a typed literal of one of its values decodes to.
//
// It returns a fault for a spec that names no generator of the vocabulary
// or misstates a parameter, whose path leads through the spec to the part
// at fault. It returns one for arguments that state no domain, for which
// the engine's constructors panic, with the panic's value as its cause.
func generatorOf(raw json.RawMessage) (g engine.Generator[any], err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fault.New("the generator states no domain").Because(fault.New("%v", r))
		}
	}()
	return build(raw, nil)
}

// build returns the generator of raw, where self is the position of the
// recursive value whose extension contains raw, and nil outside one.
func build(raw json.RawMessage, self *engine.Generator[any]) (engine.Generator[any], error) {
	var spec generatorSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return engine.Generator[any]{}, fault.New("the generator does not parse").Because(err)
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
			return engine.Generator[any]{}, fault.At(
				fault.New("self is outside a recursive extension"),
				fault.Field(genMember),
			)
		}
		return *self, nil
	}
	return engine.Generator[any]{}, fault.At(fault.New("%q names no generator", spec.Gen), fault.Field(genMember))
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
			return engine.Erase(engine.Integer(low, high)), nil
		}
	}
	low, lok := lo.Uint64()
	high, hok := hi.Uint64()
	if !lok || !hok {
		return engine.Generator[any]{}, fault.New("the bounds [%s, %s] are inside no 64-bit range", lo, hi)
	}
	return engine.Erase(engine.Integer(low, high)), nil
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
		return engine.Generator[any]{}, fault.New("the bounds [%s, %s] are no durations", lo, hi)
	}
	return engine.Erase(engine.Duration(time.Duration(low), time.Duration(high))), nil
}

// integerBounds returns the bounds min and max of spec.
func integerBounds(spec generatorSpec) (choice.Int, choice.Int, error) {
	lo, err := parseInt(spec.Min)
	if err != nil {
		return choice.Int{}, choice.Int{}, fault.At(err, fault.Field(minMember))
	}
	hi, err := parseInt(spec.Max)
	if err != nil {
		return choice.Int{}, choice.Int{}, fault.At(err, fault.Field(maxMember))
	}
	return lo, hi, nil
}

// floatOf returns the float generator of spec, of width 32 or 64.
func floatOf(spec generatorSpec) (engine.Generator[any], error) {
	low, err := literal.Float(spec.Min)
	if err != nil {
		return engine.Generator[any]{}, fault.At(err, fault.Field(minMember))
	}
	high, err := literal.Float(spec.Max)
	if err != nil {
		return engine.Generator[any]{}, fault.At(err, fault.Field(maxMember))
	}
	nan := choice.ExcludeNaN
	if spec.AllowNaN {
		nan = choice.AdmitNaN
	}
	if spec.Width == nil || *spec.Width == width64 {
		return engine.Erase(engine.Float(low, high, nan)), nil
	}
	if *spec.Width != width32 {
		return engine.Generator[any]{}, fault.At(
			fault.New("%d is neither 32 nor 64", *spec.Width),
			fault.Field(widthMember),
		)
	}
	if !sameFloat(float64(float32(low)), low) || !sameFloat(float64(float32(high)), high) {
		return engine.Generator[any]{}, fault.New("the bounds [%v, %v] are no floats of width 32", low, high)
	}
	return engine.Erase(engine.Float(float32(low), float32(high), nan)), nil
}

// sameFloat reports whether a and b are one float, NaN included.
func sameFloat(a, b float64) bool {
	return a == b || (math.IsNaN(a) && math.IsNaN(b))
}

// booleanOf returns the boolean generator of spec.
func booleanOf(spec generatorSpec) (engine.Generator[any], error) {
	if spec.P == nil {
		return engine.Erase(engine.Boolean(1, 2)), nil
	}
	if len(spec.P) != probabilityParts {
		return engine.Generator[any]{}, fault.At(
			fault.New("%v is no numerator and denominator", spec.P),
			fault.Field(pMember),
		)
	}
	return engine.Erase(engine.Boolean(spec.P[0], spec.P[1])), nil
}

// valuesOf returns just, sampled-from or permutation, of the typed
// literals that spec states.
func valuesOf(spec generatorSpec) (engine.Generator[any], error) {
	if spec.Gen == justGen {
		v, err := literal.Decode(spec.Value)
		if err != nil {
			return engine.Generator[any]{}, fault.At(err, fault.Field(valueMember))
		}
		return engine.Just(v), nil
	}
	var literals []json.RawMessage
	if err := json.Unmarshal(spec.Values, &literals); err != nil {
		return engine.Generator[any]{}, fault.At(
			fault.New("the values are no list").Because(err),
			fault.Field(valuesMember),
		)
	}
	values := make([]any, len(literals))
	for i, raw := range literals {
		v, err := literal.Decode(raw)
		if err != nil {
			return engine.Generator[any]{}, fault.At(err, fault.Field(valuesMember), fault.Index(i))
		}
		values[i] = v
	}
	if spec.Gen == sampledFromGen {
		return engine.SampledFrom(values...), nil
	}
	return engine.Erase(engine.Permutation(values...)), nil
}

// composedOf returns one-of, optional, list or filter, over the
// generators that spec states.
func composedOf(spec generatorSpec, self *engine.Generator[any]) (engine.Generator[any], error) {
	if spec.Gen == oneOfGen {
		var specs []json.RawMessage
		if err := json.Unmarshal(spec.Of, &specs); err != nil {
			return engine.Generator[any]{}, fault.At(
				fault.New("the generators are no list").Because(err),
				fault.Field(ofMember),
			)
		}
		gens := make([]engine.Generator[any], len(specs))
		for i, raw := range specs {
			g, err := build(raw, self)
			if err != nil {
				return engine.Generator[any]{}, fault.At(err, fault.Field(ofMember), fault.Index(i))
			}
			gens[i] = g
		}
		return engine.OneOf(gens...), nil
	}
	of, err := build(spec.Of, self)
	if err != nil {
		return engine.Generator[any]{}, fault.At(err, fault.Field(ofMember))
	}
	if spec.Gen == optionalGen {
		return engine.Optional(of).MapBack(func(v *any) any {
			if v == nil {
				return nil
			}
			return *v
		}, func(v any) (*any, error) { return &v, nil }), nil
	}
	if spec.Gen == filterGen {
		return filterOf(of, spec.Keep)
	}
	sizes, err := sizesOf(spec.MinSize, spec.MaxSize)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if spec.Unique {
		return engine.Erase(engine.UniqueList(of, sizes)), nil
	}
	return engine.Erase(engine.List(of, sizes)), nil
}

// filterOf returns the generator of the values of g that the predicate keep
// reports true for.
func filterOf(g engine.Generator[any], keep json.RawMessage) (engine.Generator[any], error) {
	holds, err := predicateOf(keep)
	if err != nil {
		return engine.Generator[any]{}, fault.At(err, fault.Field(keepMember))
	}
	return g.Filter(holds), nil
}

// dictOf returns the dict generator of spec.
func dictOf(spec generatorSpec, self *engine.Generator[any]) (engine.Generator[any], error) {
	keys, err := build(spec.Keys, self)
	if err != nil {
		return engine.Generator[any]{}, fault.At(err, fault.Field(keysMember))
	}
	values, err := build(spec.Values, self)
	if err != nil {
		return engine.Generator[any]{}, fault.At(err, fault.Field(valuesMember))
	}
	sizes, err := sizesOf(spec.MinSize, spec.MaxSize)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return engine.Erase(engine.Dict(keys, values, sizes)), nil
}

// textOf returns string, bytes or string-matching.
func textOf(spec generatorSpec) (engine.Generator[any], error) {
	if spec.Gen == stringMatchingGen {
		g, err := matching.StringMatching(spec.Pattern)
		if err != nil {
			return engine.Generator[any]{}, fault.At(err, fault.Field(patternMember))
		}
		return engine.Erase(g), nil
	}
	sizes, err := sizesOf(spec.MinSize, spec.MaxSize)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if spec.Gen == bytesGen {
		return engine.Erase(engine.Bytes(sizes)), nil
	}
	if spec.Alphabet == nil {
		return engine.Erase(engine.String(sizes)), nil
	}
	return engine.Erase(engine.StringOver(*spec.Alphabet, sizes)), nil
}

// recursiveOf returns the recursive generator of spec. Its base takes no
// self, and each self of its extension is a position of its own value.
func recursiveOf(spec generatorSpec) (engine.Generator[any], error) {
	base, err := build(spec.Base, nil)
	if err != nil {
		return engine.Generator[any]{}, fault.At(err, fault.Field(baseMember))
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
		return engine.Generator[any]{}, fault.At(extendErr, fault.Field(extendMember))
	}
	return g, nil
}

// sizesOf returns the lengths from minSize, 0 when nil, to maxSize,
// unbounded when nil, as a generator spec and the bounds of a sequence
// state them.
func sizesOf(minSize, maxSize *int) (choice.Sizes, error) {
	least := 0
	if minSize != nil {
		least = *minSize
	}
	if maxSize == nil {
		return choice.NewUnboundedSizes(least)
	}
	return choice.NewSizes(least, *maxSize)
}
