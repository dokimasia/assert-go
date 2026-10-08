// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// ErrShape reports a shape document that does not read.
var ErrShape = errors.New("shape: the shape does not read")

// The keys of a shape that name it and a shape file's parts.
const (
	// shapeKey names a shape's id.
	shapeKey = "shape"
	// definitionsKey states a shape file's definitions.
	definitionsKey = "definitions"
	// sourceKey states the language and the type a shape file was read
	// from.
	sourceKey = "source"
	// valuesKey states the values of a literal, which contain no shape.
	valuesKey = "values"
	// nameKey states the definition that a ref names.
	nameKey = "name"
	// refKind is the id of a ref.
	refKind = "ref"
)

// The ids of the collection shapes that the reader tells apart from a list.
const (
	// fixedListID is the id of a list of one size.
	fixedListID = "fixed-list"
	// setID is the id of a list of distinct elements.
	setID = "set"
	// mapID is the id of a map.
	mapID = "map"
)

// The keys of the parameters of the shapes.
const (
	widthKey         = "width"
	signedKey        = "signed"
	minKey           = "min"
	maxKey           = "max"
	allowNaNKey      = "allow_nan"
	allowInfinityKey = "allow_infinity"
	alphabetKey      = "alphabet"
	patternKey       = "pattern"
	minSizeKey       = "min_size"
	maxSizeKey       = "max_size"
	sizeKey          = "size"
	ofKey            = "of"
	mapKey           = "key"
	fieldsKey        = "fields"
	variantsKey      = "variants"
	versionKey       = "version"
	scaleKey         = "scale"
	unitKey          = "unit"
)

// node is one shape as its JSON states it, with its numbers as
// json.Number.
type node = map[string]any

// takes are the keys that a shape takes besides "shape": the keys it needs,
// and the keys it may state.
type takes struct {
	// required are the keys that the shape needs.
	required []string
	// optional are the keys that the shape may state.
	optional []string
}

// vocabulary is what each shape of the vocabulary takes, by its id.
var vocabulary = map[string]takes{
	"bool": {},
	intID:  {required: []string{widthKey, signedKey}, optional: []string{minKey, maxKey}},
	"float": {
		required: []string{widthKey},
		optional: []string{minKey, maxKey, allowNaNKey, allowInfinityKey},
	},
	"char":            {optional: []string{alphabetKey}},
	"string":          {optional: []string{minSizeKey, maxSizeKey, alphabetKey, patternKey}},
	"bytes":           {optional: []string{minSizeKey, maxSizeKey}},
	listID:            {required: []string{ofKey}, optional: []string{minSizeKey, maxSizeKey}},
	fixedListID:       {required: []string{ofKey, sizeKey}},
	setID:             {required: []string{ofKey}, optional: []string{minSizeKey, maxSizeKey}},
	mapID:             {required: []string{mapKey, ofKey}, optional: []string{minSizeKey, maxSizeKey}},
	optionalID:        {required: []string{ofKey}},
	recordID:          {required: []string{fieldsKey}},
	enumID:            {required: []string{variantsKey}},
	"literal":         {required: []string{valuesKey}},
	refKind:           {required: []string{nameKey}},
	"uuid":            {},
	"ip-address":      {optional: []string{versionKey}},
	"decimal":         {required: []string{scaleKey}, optional: []string{minKey, maxKey}},
	"instant":         {required: []string{unitKey}, optional: []string{minKey, maxKey}},
	"date":            {optional: []string{minKey, maxKey}},
	"time-of-day":     {required: []string{unitKey}, optional: []string{minKey, maxKey}},
	"local-date-time": {required: []string{unitKey}, optional: []string{minKey, maxKey}},
	"duration":        {required: []string{unitKey}, optional: []string{minKey, maxKey}},
	"offset":          {optional: []string{minKey, maxKey}},
	"zone":            {},
	"zoned-date-time": {required: []string{unitKey}},
	"wall-time":       {required: []string{unitKey}},
}

// ids are the ids of the shape vocabulary, sorted.
var ids = slices.Sorted(maps.Keys(vocabulary))

// builder returns the generator of one shape of its kind, and a fault whose
// path starts at the shape.
type builder func(r *reader, n node) (engine.Generator[any], error)

// builders are the builder of each shape, by its id.
var builders map[string]builder

// init fills builders. A builder of a shape that nests shapes reads
// builders, so the map cannot be the initializer of the variable.
func init() {
	builders = map[string]builder{
		"bool":            boolShape,
		intID:             intShape,
		"float":           floatShape,
		"char":            charShape,
		"string":          stringShape,
		"bytes":           bytesShape,
		listID:            listShape,
		fixedListID:       listShape,
		setID:             listShape,
		mapID:             mapShape,
		optionalID:        optionalShape,
		recordID:          recordShape,
		enumID:            enumShape,
		"literal":         literalShape,
		refKind:           refShape,
		"uuid":            uuidShape,
		"ip-address":      ipShape,
		"decimal":         decimalShape,
		"instant":         instantShape,
		"date":            dateShape,
		"time-of-day":     timeOfDayShape,
		"local-date-time": localShape,
		"duration":        durationShape,
		"offset":          offsetShape,
		"zone":            zoneShape,
		"zoned-date-time": zonedShape,
		"wall-time":       wallShape,
	}
}

// Shapes returns the ids of the shape vocabulary, sorted.
func Shapes() []string {
	return slices.Clone(ids)
}

// Read returns the generator of the shape file document, a JSON object.
//
// A shape file without a definition that refers back to itself reads as the
// generator its root maps to, so the root of an int shape is a draw from
// integer, which the explain phase steps as one. A recursive file reads as
// its root inside a count of each value's nodes.
//
// # Errors
//
// Read returns a fault of the kind [ErrShape] for a document that is no
// shape file. Its path leads through the document to the part that does not
// read, as fields[0][1].max does to the max of a record's first field.
func Read(document []byte) (engine.Generator[any], error) {
	return ReadWith(document, nil)
}

// ReadWith returns the generator of the shape file document, as [Read]
// does, where a ref to a name of externals is the generator that externals
// states for it. Such a ref needs no definition and never refers back. A
// language places a generator that has no shape, such as a registered one,
// inside a shape this way.
//
// # Errors
//
// ReadWith returns the faults that [Read] returns.
func ReadWith(document []byte, externals map[string]engine.Generator[any]) (engine.Generator[any], error) {
	tree, err := parse(document)
	if err != nil {
		return engine.Generator[any]{}, err
	}
	root, ok := tree.(node)
	if !ok {
		return engine.Generator[any]{}, unreadable("%s is no shape", show(tree))
	}
	r := &reader{
		document: root,
		budget:   &budget{limit: limit},
		built:    make(map[string]engine.Generator[any]),
		sources:  make(map[string]node),
	}
	maps.Copy(r.built, externals)
	return r.root()
}

// parse returns the JSON value of document, with its numbers as
// json.Number. The document states one JSON value, and nothing but space
// after it.
func parse(document []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(document))
	d.UseNumber()
	var tree any
	if err := d.Decode(&tree); err != nil {
		return nil, unreadable("the document is no JSON").Because(err)
	}
	if rest := document[d.InputOffset():]; len(bytes.TrimLeft(rest, " \t\r\n")) > 0 {
		return nil, unreadable("the document states more than its JSON value")
	}
	return tree, nil
}

// reader is one shape file being read: its definitions, the definitions
// that refer back to themselves, and the budget of its values.
type reader struct {
	// document is the shape file.
	document node
	// budget is the limit on the nodes of each value.
	budget *budget
	// built are the generators of the definitions, by name.
	built map[string]engine.Generator[any]
	// sources are the shapes of the definitions, by name.
	sources map[string]node
	// cyclic are the definitions that refer back to themselves through a
	// chain of refs.
	cyclic map[string]bool
}

// root reads the definitions and the source, checks that every ref names a
// definition and that each definition has a finite value, and builds the
// definitions and then the root. The root counts each value's nodes only
// when a definition refers back to itself.
func (r *reader) root() (engine.Generator[any], error) {
	if err := r.readDefinitions(); err != nil {
		return engine.Generator[any]{}, err
	}
	if err := r.readSource(); err != nil {
		return engine.Generator[any]{}, err
	}
	if err := r.findCycles(); err != nil {
		return engine.Generator[any]{}, err
	}
	if err := r.checkFinite(); err != nil {
		return engine.Generator[any]{}, err
	}
	for _, name := range slices.Sorted(maps.Keys(r.sources)) {
		g, err := r.build(r.sources[name], false)
		if err != nil {
			return engine.Generator[any]{}, fault.At(err, fault.Field(definitionsKey), fault.Key(name))
		}
		r.built[name] = g
	}
	g, err := r.build(r.document, true)
	if err != nil || len(r.cyclic) == 0 {
		return g, err
	}
	return rootOf(g, r.budget), nil
}

// readDefinitions keeps the shape of each definition by its name.
func (r *reader) readDefinitions() error {
	stated, present := r.document[definitionsKey]
	if !present {
		return nil
	}
	definitions, ok := stated.(node)
	if !ok {
		return fault.At(unreadable("%s is no map of names to shapes", show(stated)), fault.Field(definitionsKey))
	}
	for name, v := range definitions {
		n, ok := v.(node)
		if !ok {
			return fault.At(unreadable("%s is no shape", show(v)), fault.Field(definitionsKey), fault.Key(name))
		}
		r.sources[name] = n
	}
	return nil
}

// readSource checks the source that the root states, when it states one:
// a language and a type.
func (r *reader) readSource() error {
	source := r.document[sourceKey]
	if source == nil {
		return nil
	}
	n, ok := source.(node)
	if ok {
		_, language := n["language"].(string)
		_, typ := n["type"].(string)
		ok = language && typ
	}
	if !ok {
		return fault.At(unreadable("%s is no language and type", show(source)), fault.Field(sourceKey))
	}
	return nil
}

// refs calls found with the name of every ref inside v, without following
// refs. With stopAtExits, a container that can always exit is not entered,
// so only the refs that every value of v expands are found.
func (r *reader) refs(v any, stopAtExits bool, found func(string)) {
	switch v := v.(type) {
	case []any:
		for _, item := range v {
			r.refs(item, stopAtExits, found)
		}
	case node:
		if v[shapeKey] == refKind {
			found(nameOf(v[nameKey]))
			return
		}
		if stopAtExits && r.exits(v) {
			return
		}
		for key, value := range v {
			if key != definitionsKey && key != sourceKey && key != valuesKey {
				r.refs(value, stopAtExits, found)
			}
		}
	}
}

// nameOf returns the name that a ref states: its text, or the JSON of a
// name that is no string.
func nameOf(v any) string {
	if name, ok := v.(string); ok {
		return name
	}
	return show(v)
}

// refsOf returns the names of the refs inside v, as refs finds them.
func (r *reader) refsOf(v any, stopAtExits bool) map[string]bool {
	names := make(map[string]bool)
	r.refs(v, stopAtExits, func(name string) { names[name] = true })
	return names
}

// findCycles keeps the definitions that refer back to themselves through a
// chain of refs. It returns a fault for a ref that names no definition.
func (r *reader) findCycles() error {
	reach := make(map[string]map[string]bool, len(r.sources))
	targets := r.refsOf(r.document, false)
	for name, n := range r.sources {
		reach[name] = r.refsOf(n, false)
		maps.Copy(targets, reach[name])
	}
	for _, target := range slices.Sorted(maps.Keys(targets)) {
		_, defined := r.sources[target]
		_, external := r.built[target]
		if !defined && !external {
			return unreadable("a ref names %q, which is no definition", target)
		}
	}
	for changed := true; changed; {
		changed = false
		for _, targets := range reach {
			for target := range targets {
				for further := range reach[target] {
					if !targets[further] {
						targets[further], changed = true, true
					}
				}
			}
		}
	}
	r.cyclic = make(map[string]bool)
	for name, targets := range reach {
		if targets[name] {
			r.cyclic[name] = true
		}
	}
	return nil
}

// refersBack reports whether v contains a ref to a definition that refers
// back to itself.
func (r *reader) refersBack(v any) bool {
	for name := range r.refsOf(v, false) {
		if r.cyclic[name] {
			return true
		}
	}
	return false
}

// exits reports whether a container can always take an exit, whatever it
// contains: an optional can be absent, a list, a set or a map without a
// minimum size can be empty, and an enum can take a variant that does not
// refer back.
func (r *reader) exits(n node) bool {
	switch n[shapeKey] {
	case optionalID:
		return true
	case listID, setID, mapID:
		least := integer(n[minSizeKey])
		return n[minSizeKey] == nil || least != nil && least.Sign() == 0
	case enumID:
		_, ok := r.exitVariant(n)
		return ok
	}
	return false
}

// exitVariant returns the index of an enum's first variant that does not
// refer back, and false when every variant refers back.
func (r *reader) exitVariant(n node) (int, bool) {
	variants, _ := n[variantsKey].([]any)
	for index, variant := range variants {
		var payload any
		if pair, ok := variant.([]any); ok && len(pair) == pairLength {
			payload = pair[1]
		}
		if !r.refersBack(payload) {
			return index, true
		}
	}
	return 0, false
}

// checkFinite refuses a definition that refers back to itself through refs
// that every one of its values expands.
func (r *reader) checkFinite() error {
	must := make(map[string]map[string]bool, len(r.sources))
	for name, n := range r.sources {
		must[name] = r.refsOf(n, true)
	}
	for _, start := range slices.Sorted(maps.Keys(r.cyclic)) {
		seen := make(map[string]bool)
		frontier := maps.Clone(must[start])
		for len(frontier) > 0 {
			if frontier[start] {
				infinite := unreadable("the definition has no finite value, because it refers back without an " +
					"optional, a list, a set, a map or an enum that can exit")
				return fault.At(infinite, fault.Field(definitionsKey), fault.Key(start))
			}
			maps.Copy(seen, frontier)
			next := make(map[string]bool)
			for name := range frontier {
				for target := range must[name] {
					if !seen[target] {
						next[target] = true
					}
				}
			}
			frontier = next
		}
	}
	return nil
}

// build returns the generator of one shape, and a fault whose path starts
// at the shape. A root may state the definitions and the source besides
// its own keys.
func (r *reader) build(n node, root bool) (engine.Generator[any], error) {
	kind, _ := n[shapeKey].(string)
	t, known := vocabulary[kind]
	if !known {
		return engine.Generator[any]{}, fault.At(unreadable("%s is no shape of the vocabulary", show(n[shapeKey])),
			fault.Field(shapeKey))
	}
	stated := make(map[string]bool, len(n))
	for key := range n {
		ofRoot := root && (key == definitionsKey || key == sourceKey)
		if key != shapeKey && !ofRoot {
			stated[key] = true
		}
	}
	for _, key := range t.required {
		if !stated[key] {
			return engine.Generator[any]{}, unreadable("the %s shape needs the key %s", kind, key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(stated)) {
		if !slices.Contains(t.required, key) && !slices.Contains(t.optional, key) {
			return engine.Generator[any]{}, fault.At(unreadable("the key does not apply to the %s shape", kind),
				fault.Field(key))
		}
	}
	return builders[kind](r, n)
}

// child returns the generator of the shape v nested in another at segs,
// relative to the other, and a fault whose path starts at the other.
func (r *reader) child(v any, segs ...fault.Segment) (engine.Generator[any], error) {
	n, ok := v.(node)
	if !ok {
		return engine.Generator[any]{}, fault.At(unreadable("%s is no shape", show(v)), segs...)
	}
	g, err := r.build(n, false)
	if err != nil {
		return engine.Generator[any]{}, fault.At(err, segs...)
	}
	return g, nil
}
