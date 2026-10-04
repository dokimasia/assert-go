// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"encoding/json"
	"math"
	"math/big"
	"reflect"
	"slices"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/matching"
)

// The ids of the structural shapes whose generators record a span of their
// own, labelled with the id.
const (
	optionalID = "optional"
	listID     = "list"
	dictID     = "dict"
	recordID   = "record"
	enumID     = "enum"
	intID      = "int"
)

// The parameters of the structural shapes.
const (
	// pairLength is the length of a field of a record and of a variant of
	// an enum: a name and a shape.
	pairLength = 2
	// wide is the width of an int that is two choices, and half the width
	// of each.
	wide = 128
	half = 64
	// float32Width and float64Width are the widths of a float.
	float32Width = 32
	float64Width = 64
)

// The bounds of the choices that the structural shapes make.
var (
	// bit are the bounds [0, 1] of a presence or a coin.
	bit = choice.MustIntegerBounds(choice.Int{}, choice.UintOf(1))
	// none are the bounds [0, 0] of a forced exit.
	none = choice.IntegerBounds{}
	// lowMask is the largest low half of a 128-bit int.
	lowMask = choice.UintOf(math.MaxUint64)
)

// intWidths are the widths of the int shape.
var intWidths = []int64{8, 16, 32, 64, wide}

// floatWidths are the widths of the float shape.
var floatWidths = map[int64]choice.Width{float32Width: choice.Width32, float64Width: choice.Width64}

// boolShape returns the generator of a bool, true with probability 1/2.
func boolShape(*reader, node) (engine.Generator[any], error) {
	return engine.Erase(engine.Boolean(1, 2)), nil
}

// intShape returns the generator of an int of the stated width and
// signedness, over its whole range unless min and max narrow it.
func intShape(_ *reader, n node) (engine.Generator[any], error) {
	width, err := intParam(n, widthKey)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if !width.IsInt64() || !slices.Contains(intWidths, width.Int64()) {
		return engine.Generator[any]{}, fault.At(unreadable("%s is none of the widths 8, 16, 32, 64 and 128", width),
			fault.Field(widthKey))
	}
	signed, ok := n[signedKey].(bool)
	if !ok {
		return engine.Generator[any]{}, fault.At(unreadable("%s is no boolean", show(n[signedKey])),
			fault.Field(signedKey))
	}
	bits := uint(width.Int64())
	lo, hi := new(big.Int), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), bits), big.NewInt(1))
	if signed {
		hi.Rsh(hi, 1)
		lo.Not(hi)
	}
	least, most, err := bigBounds(n, lo, hi)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if bits == wide {
		return wideInt(least, most), nil
	}
	if signed {
		return engine.Erase(engine.Integer(least.Int64(), most.Int64())), nil
	}
	return engine.Erase(engine.Integer(least.Uint64(), most.Uint64())), nil
}

// bigBounds returns the integers that the min and the max of an int shape
// state, inside [lo, hi], which are the bounds by default.
func bigBounds(n node, lo, hi *big.Int) (*big.Int, *big.Int, error) {
	least, most := lo, hi
	var err error
	if n[minKey] != nil {
		if least, err = intParam(n, minKey); err != nil {
			return nil, nil, err
		}
	}
	if n[maxKey] != nil {
		if most, err = intParam(n, maxKey); err != nil {
			return nil, nil, err
		}
	}
	if least.Cmp(lo) < 0 || least.Cmp(most) > 0 || most.Cmp(hi) > 0 {
		return nil, nil, unreadable("the bounds [%s, %s] are empty or outside [%s, %s]", least, most, lo, hi)
	}
	return least, most, nil
}

// floatShape returns the generator of a float of the stated width, finite
// unless allow_infinity admits the infinities, and without NaN unless
// allow_nan admits it.
func floatShape(_ *reader, n node) (engine.Generator[any], error) {
	width, err := intParam(n, widthKey)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	w, ok := floatWidths[width.Int64()]
	if !width.IsInt64() || !ok {
		return engine.Generator[any]{}, fault.At(unreadable("%s is neither 32 nor 64", width), fault.Field(widthKey))
	}
	allowNaN, err := flag(n, allowNaNKey)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	infinite, err := flag(n, allowInfinityKey)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	edge := math.MaxFloat64
	if w == choice.Width32 {
		edge = math.MaxFloat32
	}
	if infinite {
		edge = math.Inf(1)
	}
	lo, err := floatBound(n, minKey, -edge)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	hi, err := floatBound(n, maxKey, edge)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	if infinite && !math.IsInf(lo, 0) && !math.IsInf(hi, 0) {
		return engine.Generator[any]{}, unreadable("the shape allows the infinities, and its bounds leave both out")
	}
	nan := choice.ExcludeNaN
	if allowNaN {
		nan = choice.AdmitNaN
	}
	if _, err := choice.NewFloatBounds(lo, hi, nan, w); err != nil {
		return engine.Generator[any]{}, unreadable("the bounds admit no float").Because(err)
	}
	if w == choice.Width32 {
		return engine.Erase(engine.Float(float32(lo), float32(hi), nan)), nil
	}
	return engine.Erase(engine.Float(lo, hi, nan)), nil
}

// flag returns the boolean that the parameter key of n states, false when
// n does not state the key, and a fault at the key for a parameter that is
// no boolean, null included.
func flag(n node, key string) (bool, error) {
	stated, present := n[key]
	if !present {
		return false, nil
	}
	v, ok := stated.(bool)
	if !ok {
		return false, fault.At(unreadable("%s is no boolean", show(stated)), fault.Field(key))
	}
	return v, nil
}

// floatBound returns the float that the bound key of n states, and
// otherwise fallback.
func floatBound(n node, key string, fallback float64) (float64, error) {
	if n[key] == nil {
		return fallback, nil
	}
	f, ok := number(n[key])
	if !ok {
		return 0, fault.At(unreadable("%s is no float", show(n[key])), fault.Field(key))
	}
	return f, nil
}

// alphabetOf returns the alphabet that a char or a string shape states, and
// "" for the default alphabet.
func alphabetOf(n node) (string, error) {
	v, present := n[alphabetKey]
	if !present || v == nil {
		return "", nil
	}
	chars, ok := v.(string)
	if !ok || chars == "" || !utf8.ValidString(chars) {
		return "", fault.At(unreadable("%s is no string of characters", show(v)), fault.Field(alphabetKey))
	}
	runes := []rune(chars)
	sorted := slices.Clone(runes)
	slices.Sort(sorted)
	if len(slices.Compact(sorted)) != len(runes) {
		return "", fault.At(unreadable("the alphabet repeats a character"), fault.Field(alphabetKey))
	}
	return chars, nil
}

// text returns the generator of strings over an alphabet, the default one
// for "", with lengths that sizes admits.
func text(chars string, sizes choice.Sizes) engine.Generator[any] {
	if chars == "" {
		return engine.Erase(engine.String(sizes))
	}
	return engine.Erase(engine.StringOver(chars, sizes))
}

// charShape returns the generator of one character of the alphabet, as a
// string.
func charShape(_ *reader, n node) (engine.Generator[any], error) {
	chars, err := alphabetOf(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	one, _ := choice.NewSizes(1, 1)
	return text(chars, one), nil
}

// stringShape returns the generator of a string over an alphabet, or of
// the strings that a pattern matches.
func stringShape(_ *reader, n node) (engine.Generator[any], error) {
	if p, present := n[patternKey]; present {
		_, alphabet := n[alphabetKey]
		_, least := n[minSizeKey]
		_, most := n[maxSizeKey]
		if alphabet || least || most {
			return engine.Generator[any]{}, unreadable("the shape states a pattern and an alphabet or a size")
		}
		pattern, ok := p.(string)
		if !ok {
			return engine.Generator[any]{}, fault.At(unreadable("%s is no pattern", show(p)), fault.Field(patternKey))
		}
		g, err := matching.StringMatching(pattern)
		if err != nil {
			outside := unreadable("the pattern is outside the portable subset").Because(err)
			return engine.Generator[any]{}, fault.At(outside, fault.Field(patternKey))
		}
		return engine.Erase(g), nil
	}
	chars, err := alphabetOf(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	s, err := sizes(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return text(chars, s), nil
}

// bytesShape returns the generator of a byte string.
func bytesShape(_ *reader, n node) (engine.Generator[any], error) {
	s, err := sizes(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return engine.Erase(engine.Bytes(s)), nil
}

// listShape returns the generator of a list, a fixed-list or a set.
func listShape(r *reader, n node) (engine.Generator[any], error) {
	kind := n[shapeKey]
	of, err := r.child(n[ofKey], fault.Field(ofKey))
	if err != nil {
		return engine.Generator[any]{}, err
	}
	s, err := listSizes(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	exits := kind != "fixed-list" && r.exits(n) && r.refersBack(n)
	return listOf(of, s, kind == "set", exits, r.budget), nil
}

// listSizes returns the sizes of a list or a set, and a fixed-list's one
// size.
func listSizes(n node) (choice.Sizes, error) {
	if n[shapeKey] != "fixed-list" {
		return sizes(n)
	}
	size, stated, err := count(n, sizeKey)
	if err != nil {
		return choice.Sizes{}, err
	}
	if !stated {
		return choice.Sizes{}, fault.At(unreadable("null is no size"), fault.Field(sizeKey))
	}
	exactly, _ := choice.NewSizes(size, size)
	return exactly, nil
}

// listOf returns the generator of a list of of's values with lengths that
// sizes admits, as a []any: per element a continue flag, then the element,
// in a span labelled list. A set discards an element equal to an earlier
// one. A list that exits takes no further element once the value has used
// its budget.
//
// It runs backwards from a slice or an array, through each element: a
// list's in their order, and a set's in the shortlex order of their own
// choices. The fault of an element is at its index.
func listOf(of engine.Generator[any], s choice.Sizes, unique, exits bool, b *budget) engine.Generator[any] {
	var key func(any) string
	if unique {
		key = literal.Canonical
	}
	decode := func(c *engine.Case) any {
		var items []any
		c.Span(listID, func() {
			items = engine.Elements(c, s, engine.ElementLabel, of.Decode, key, b.stop(c, exits))
		})
		return items
	}
	return engine.NewInvertible(listID, decode, func(v any) ([]engine.Step, any, error) {
		items, ok := engine.ListItems(v)
		if !ok {
			return nil, nil, uninvertible("%v is no list", v)
		}
		parts := make([]part, len(items))
		seen := make(map[string]int, len(items))
		for i, item := range items {
			steps, value, err := of.Inverse(item)
			if err != nil {
				return nil, nil, fault.At(err, fault.Index(i))
			}
			parts[i] = part{steps: steps, value: value}
			if !unique {
				continue
			}
			k := literal.Canonical(value)
			if earlier, repeated := seen[k]; repeated {
				return nil, nil, fault.At(uninvertible("the element repeats element %d", earlier), fault.Index(i))
			}
			seen[k] = i
		}
		if unique {
			slices.SortStableFunc(parts, part.compare)
		}
		steps, err := engine.CollectionSteps(s, stepsOf(parts))
		values := make([]any, len(parts))
		for i, p := range parts {
			values[i] = p.value
		}
		return steps, values, err
	})
}

// part is one element or entry of a collection run backwards: its steps,
// and the value they decode to.
type part struct {
	// steps are the steps of the element or the entry.
	steps []engine.Step
	// value is the element, or the entry as a [literal.Entry].
	value any
}

// compare orders two parts in the shortlex order of their steps.
func (p part) compare(q part) int {
	return engine.CompareSteps(p.steps, q.steps)
}

// stepsOf returns the steps of each part, in order.
func stepsOf(parts []part) [][]engine.Step {
	out := make([][]engine.Step, len(parts))
	for i, p := range parts {
		out[i] = p.steps
	}
	return out
}

// mapShape returns the generator of a map from keys to values.
func mapShape(r *reader, n node) (engine.Generator[any], error) {
	key, err := r.child(n[mapKey], fault.Field(mapKey))
	if err != nil {
		return engine.Generator[any]{}, err
	}
	of, err := r.child(n[ofKey], fault.Field(ofKey))
	if err != nil {
		return engine.Generator[any]{}, err
	}
	s, err := sizes(n)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return mapOf(key, of, s, r.exits(n) && r.refersBack(n), r.budget), nil
}

// mapOf returns the generator of maps of key's keys and of's values with
// lengths that sizes admits, as a [literal.Pairs], decoded as a dict is in a
// span labelled dict. A map that exits takes no further entry once the
// value has used its budget.
//
// It runs backwards from a [literal.Pairs] or a map, through each entry's
// key and then its value, the entries in the shortlex order of their own
// choices. The fault of a value is at its key, and so is the fault of a key
// that key does not produce, whose cause is the fault of key.
func mapOf(key, of engine.Generator[any], s choice.Sizes, exits bool, b *budget) engine.Generator[any] {
	entry := func(c *engine.Case) literal.Entry {
		k := key.Decode(c)
		return literal.Entry{Key: k, Value: of.Decode(c)}
	}
	unique := func(e literal.Entry) string { return literal.Canonical(e.Key) }
	decode := func(c *engine.Case) any {
		var entries []literal.Entry
		c.Span(dictID, func() {
			entries = engine.Elements(c, s, engine.EntryLabel, entry, unique, b.stop(c, exits))
		})
		return literal.Pairs{Entries: entries}
	}
	return engine.NewInvertible(dictID, decode, func(v any) ([]engine.Step, any, error) {
		entries, ok := entriesOf(v)
		if !ok {
			return nil, nil, uninvertible("%v is no map", v)
		}
		parts := make([]part, len(entries))
		seen := make(map[string]struct{}, len(entries))
		for i, e := range entries {
			keySteps, k, err := key.Inverse(e.Key)
			if err != nil {
				return nil, nil, fault.At(uninvertible("the generator of keys produces no such key").Because(err),
					fault.Key(e.Key))
			}
			valueSteps, value, err := of.Inverse(e.Value)
			if err != nil {
				return nil, nil, fault.At(err, fault.Key(e.Key))
			}
			text := literal.Canonical(k)
			if _, repeated := seen[text]; repeated {
				return nil, nil, uninvertible("two keys decode to the key %v", k)
			}
			seen[text] = struct{}{}
			parts[i] = part{steps: slices.Concat(keySteps, valueSteps), value: literal.Entry{Key: k, Value: value}}
		}
		slices.SortStableFunc(parts, part.compare)
		steps, err := engine.CollectionSteps(s, stepsOf(parts))
		pairs := literal.Pairs{Entries: make([]literal.Entry, len(parts))}
		for i, p := range parts {
			pairs.Entries[i] = p.value.(literal.Entry)
		}
		return steps, pairs, err
	})
}

// entriesOf returns the entries of a [literal.Pairs] or a map, and false
// for any other value.
func entriesOf(v any) ([]literal.Entry, bool) {
	if p, ok := v.(literal.Pairs); ok {
		return p.Entries, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return nil, false
	}
	entries := make([]literal.Entry, 0, rv.Len())
	for k, value := range rv.Seq2() {
		entries = append(entries, literal.Entry{Key: k.Interface(), Value: value.Interface()})
	}
	return entries, true
}

// optionalShape returns the generator of an optional value.
func optionalShape(r *reader, n node) (engine.Generator[any], error) {
	of, err := r.child(n[ofKey], fault.Field(ofKey))
	if err != nil {
		return engine.Generator[any]{}, err
	}
	return optionalOf(of, r.refersBack(n), r.budget), nil
}

// optionalOf returns the generator of nil or a value of of: a presence
// choice that decides structure, in a span labelled optional, then the
// value when present. An optional that exits is absent once the value has
// used its budget: the choice's bounds admit only 0.
//
// It runs backwards from [engine.Absent] values as absent, and from any
// other value as the present value itself.
func optionalOf(of engine.Generator[any], exits bool, b *budget) engine.Generator[any] {
	decode := func(c *engine.Case) any {
		presence := bit
		if exits && b.exhausted(c) {
			presence = none
		}
		var v any
		c.Span(optionalID, func() {
			if c.Structure(presence, 1) == choice.UintOf(1) {
				v = of.Decode(c)
			}
		})
		return v
	}
	return engine.NewInvertible(optionalID, decode, func(v any) ([]engine.Step, any, error) {
		if engine.Absent(v) {
			return []engine.Step{bitStep(0)}, nil, nil
		}
		steps, value, err := of.Inverse(v)
		if err != nil {
			return nil, nil, err
		}
		return append([]engine.Step{bitStep(1)}, steps...), value, nil
	})
}

// bitStep returns the step of a choice in [0, 1] of the value v.
func bitStep(v uint64) engine.Step {
	return engine.Step{Bounds: choice.OfInteger(bit), Value: integerChoice(choice.UintOf(v))}
}

// indexStep returns the step of index i under b, which admits it.
func indexStep(b choice.IntegerBounds, i int) engine.Step {
	return engine.Step{Bounds: choice.OfInteger(b), Value: integerChoice(choice.UintOf(uint64(i)))}
}

// integerChoice returns the integer choice of i.
func integerChoice(i choice.Int) choice.Choice {
	return choice.Choice{Kind: choice.Integer, Integer: i}
}

// pairs returns the name and the value of each item of the list that the
// parameter key of n states: a non-empty list of distinct, non-empty
// names, each with a value.
func pairs(n node, key string) ([]string, []any, error) {
	items, ok := n[key].([]any)
	if !ok || len(items) == 0 {
		return nil, nil, fault.At(unreadable("%s is no list of pairs", show(n[key])), fault.Field(key))
	}
	names, values := make([]string, len(items)), make([]any, len(items))
	for i, item := range items {
		p, ok := item.([]any)
		var name string
		if ok && len(p) == pairLength {
			name, ok = p[0].(string)
		}
		if !ok || len(p) != pairLength {
			return nil, nil, fault.At(unreadable("%s is no name and shape", show(item)),
				fault.Field(key), fault.Index(i))
		}
		if name == "" || slices.Contains(names[:i], name) {
			return nil, nil, fault.At(unreadable("the name %q is empty or named twice", name),
				fault.Field(key), fault.Index(i), fault.Index(0))
		}
		names[i], values[i] = name, p[1]
	}
	return names, values, nil
}

// recordShape returns the generator of a record of named fields.
func recordShape(r *reader, n node) (engine.Generator[any], error) {
	names, shapes, err := pairs(n, fieldsKey)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	fields := make([]engine.Generator[any], len(names))
	for i := range names {
		if fields[i], err = r.child(shapes[i], fault.Field(fieldsKey), fault.Index(i), fault.Index(1)); err != nil {
			return engine.Generator[any]{}, err
		}
	}
	return recordOf(names, fields), nil
}

// recordOf returns the generator of a [literal.Record] of the named fields,
// each field's value decoded in declaration order, in a span labelled
// record. It runs backwards from a record with the fields in that order, and
// the fault of a field's value is at the field.
func recordOf(names []string, fields []engine.Generator[any]) engine.Generator[any] {
	decode := func(c *engine.Case) any {
		out := literal.Record{Fields: make([]literal.Field, len(names))}
		c.Span(recordID, func() {
			for i, name := range names {
				out.Fields[i] = literal.Field{Name: name, Value: fields[i].Decode(c)}
			}
		})
		return out
	}
	return engine.NewInvertible(recordID, decode, func(v any) ([]engine.Step, any, error) {
		values, err := recordValues(v, names)
		if err != nil {
			return nil, nil, err
		}
		var steps []engine.Step
		out := literal.Record{Fields: make([]literal.Field, len(names))}
		for i, name := range names {
			fieldSteps, value, err := fields[i].Inverse(values[i])
			if err != nil {
				return nil, nil, fault.At(err, fault.Field(name))
			}
			steps = append(steps, fieldSteps...)
			out.Fields[i] = literal.Field{Name: name, Value: value}
		}
		return steps, out, nil
	})
}

// recordValues returns the value of each field of v, a [literal.Record]
// whose fields have the names, in order.
func recordValues(v any, names []string) ([]any, error) {
	r, ok := v.(literal.Record)
	ok = ok && len(r.Fields) == len(names)
	values := make([]any, len(names))
	for i := 0; ok && i < len(names); i++ {
		ok, values[i] = r.Fields[i].Name == names[i], r.Fields[i].Value
	}
	if !ok {
		return nil, uninvertible("%v is no record of the fields %v", v, names)
	}
	return values, nil
}

// enumShape returns the generator of an enum of named variants with
// optional payloads.
func enumShape(r *reader, n node) (engine.Generator[any], error) {
	names, shapes, err := pairs(n, variantsKey)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	payloads := make([]*engine.Generator[any], len(names))
	for i := range names {
		if shapes[i] == nil {
			continue
		}
		g, err := r.child(shapes[i], fault.Field(variantsKey), fault.Index(i), fault.Index(1))
		if err != nil {
			return engine.Generator[any]{}, err
		}
		payloads[i] = &g
	}
	exit, exits := r.exitVariant(n)
	return enumOf(names, payloads, exit, exits && r.refersBack(n), r.budget), nil
}

// enumOf returns the generator of a [literal.Variant] of the named
// variants: an index that decides structure, in a span labelled enum, then
// the chosen variant's payload when it has one. An enum that exits takes
// the variant at exit once the value has used its budget.
//
// It runs backwards from a variant of the enum, through its index and then
// its payload. The fault of a payload is at its variant.
func enumOf(names []string, payloads []*engine.Generator[any], exit int, exits bool, b *budget) engine.Generator[any] {
	all := choice.MustIntegerBounds(choice.Int{}, choice.UintOf(uint64(len(names)-1)))
	only := choice.MustIntegerBounds(choice.UintOf(uint64(exit)), choice.UintOf(uint64(exit)))
	decode := func(c *engine.Case) any {
		indices := all
		if exits && b.exhausted(c) {
			indices = only
		}
		var out literal.Variant
		c.Span(enumID, func() {
			index := c.Structure(indices, 0).Magnitude()
			out.Name = names[index]
			if payload := payloads[index]; payload != nil {
				out.Payload, out.HasPayload = payload.Decode(c), true
			}
		})
		return out
	}
	return engine.NewInvertible(enumID, decode, func(v any) ([]engine.Step, any, error) {
		variant, ok := v.(literal.Variant)
		index := slices.Index(names, variant.Name)
		if !ok || index < 0 {
			return nil, nil, uninvertible("%v is no variant of %v", v, names)
		}
		head := []engine.Step{indexStep(all, index)}
		payload := payloads[index]
		at := fault.Variant(variant.Name)
		switch {
		case payload == nil && variant.HasPayload:
			return nil, nil, fault.At(uninvertible("the variant has no payload, and the value states one"), at)
		case payload != nil && !variant.HasPayload:
			return nil, nil, fault.At(uninvertible("the variant has a payload, and the value states none"), at)
		case payload == nil:
			return head, literal.Variant{Name: variant.Name}, nil
		}
		steps, value, err := payload.Inverse(variant.Payload)
		if err != nil {
			return nil, nil, fault.At(err, at)
		}
		return append(head, steps...), literal.Variant{Name: variant.Name, Payload: value, HasPayload: true}, nil
	})
}

// literalShape returns the generator of one of the stated values, as a
// typed literal states each.
func literalShape(_ *reader, n node) (engine.Generator[any], error) {
	items, ok := n[valuesKey].([]any)
	if !ok || len(items) == 0 {
		return engine.Generator[any]{}, fault.At(unreadable("%s is no list of literals", show(n[valuesKey])),
			fault.Field(valuesKey))
	}
	values := make([]any, len(items))
	for i, item := range items {
		raw, _ := json.Marshal(item)
		v, err := literal.Decode(raw)
		if err != nil {
			return engine.Generator[any]{}, fault.At(unreadable("the value is no typed literal").Because(err),
				fault.Field(valuesKey), fault.Index(i))
		}
		values[i] = v
	}
	return engine.SampledFrom(values...), nil
}

// refShape returns the generator of the definition that a ref names. Every
// ref names a definition: the reader refuses any other before it builds a
// shape.
func refShape(r *reader, n node) (engine.Generator[any], error) {
	name := nameOf(n[nameKey])
	return refOf(name, r.built, r.cyclic[name], r.budget), nil
}
