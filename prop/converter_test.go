// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"maps"
	bignum "math/big"
	"net/netip"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// everyShape is a record of one field of each shape, with a definition that
// refers to itself.
const everyShape = `{"shape":"record","fields":[` +
	`["i8",{"shape":"int","width":8,"signed":true}],` +
	`["u16",{"shape":"int","width":16,"signed":false}],` +
	`["i128",{"shape":"int","width":128,"signed":true}],` +
	`["f32",{"shape":"float","width":32}],` +
	`["b",{"shape":"bool"}],` +
	`["c",{"shape":"char"}],` +
	`["s",{"shape":"string","max_size":3}],` +
	`["raw",{"shape":"bytes","max_size":3}],` +
	`["list",{"shape":"list","of":{"shape":"bool"},"max_size":3}],` +
	`["fixed",{"shape":"fixed-list","of":{"shape":"bool"},"size":2}],` +
	`["set",{"shape":"set","of":{"shape":"string","max_size":2},"max_size":3}],` +
	`["map",{"shape":"map","key":{"shape":"string","max_size":2},"of":{"shape":"bool"},"max_size":3}],` +
	`["maybe",{"shape":"optional","of":{"shape":"string"}}],` +
	`["enum",{"shape":"enum","variants":[["none",null],["some",{"shape":"bool"}]]}],` +
	`["literal",{"shape":"literal","values":[{"type":"int","value":1},` +
	`{"type":"record","fields":[["a",{"type":"int","value":2}]]}]}],` +
	`["id",{"shape":"uuid"}],` +
	`["ip",{"shape":"ip-address"}],` +
	`["price",{"shape":"decimal","scale":2}],` +
	`["at",{"shape":"instant","unit":"us"}],` +
	`["day",{"shape":"date"}],` +
	`["opens",{"shape":"time-of-day","unit":"ms"}],` +
	`["starts",{"shape":"local-date-time","unit":"s"}],` +
	`["wait",{"shape":"duration","unit":"ms"}],` +
	`["offset",{"shape":"offset"}],` +
	`["zone",{"shape":"zone"}],` +
	`["meets",{"shape":"zoned-date-time","unit":"ms"}],` +
	`["wall",{"shape":"wall-time","unit":"s"}],` +
	`["tree",{"shape":"ref","name":"t"}]],` +
	`"definitions":{"t":{"shape":"list","of":{"shape":"ref","name":"t"},"max_size":2}}}`

// The allocations of a decode of one value through the generator that Of
// returns, measured: the replay of its case, the value that the shape's
// generator boxes, and the value that Of returns. A converter stores an
// integer and a float by its kind, and allocates no converted number.
const (
	int8DecodeAllocs    = 13
	uint64DecodeAllocs  = 12
	float32DecodeAllocs = 14
	float64DecodeAllocs = 14
	charDecodeAllocs    = 18
)

// The types whose generators the converter tests run back.
type (
	// scalars has a field of each scalar type that the reader reads.
	scalars struct {
		B    bool
		I    int
		I8   int8
		I16  int16
		I32  int32
		I64  int64
		U    uint
		U8   uint8
		U16  uint16
		U32  uint32
		U64  uint64
		F32  float32
		F64  float64
		S    string
		R    rune `prop:"char"`
		Raw  []byte
		Key  [4]byte
		Temp celsius
	}
	// domains has a field of each type that reads as a domain shape.
	domains struct {
		At     time.Time
		Day    time.Time     `prop:"date"`
		Starts time.Time     `prop:"local-date-time,unit=us"`
		Meets  time.Time     `prop:"zoned-date-time,unit=ms"`
		Wait   time.Duration `prop:"unit=us"`
		Opens  time.Duration `prop:"time-of-day,unit=s"`
		Offset int32         `prop:"offset"`
		Zone   *time.Location
		ID     uuid.UUID
		Host   netip.Addr
		V4     netip.Addr `prop:"version=4"`
		Big    *bignum.Int
		Price  *bignum.Rat `prop:"scale=2"`
		Wall   prop.WallTime
	}
	// collections has a field of each collection that the reader reads.
	collections struct {
		List     []int16
		Fixed    [3]bool
		Set      map[string]struct{}
		Map      map[uint8]int8
		Maybe    *string
		Nested   [][]uint8
		Octets   [2]octet
		Payments []payment
		Status   status
	}
	// lost is a type of payment that RegisterVariants does not state.
	lost struct{}
	// event is an interface whose variants RegisterVariants states.
	event interface{ isEvent() }
	// voided is a variant of event whose only field the reader does not
	// read, so it has no payload.
	voided struct {
		note string
	}
	// stamped is a variant of event whose payload is a date.
	stamped struct {
		At time.Time `json:"at" prop:"date"`
	}
	// letterOf is a struct of one character.
	letterOf struct {
		V rune `prop:"char"`
	}
)

func (lost) isPayment()  {}
func (voided) isEvent()  {}
func (stamped) isEvent() {}

// init registers the variants of event in the test process's registry,
// before any property runs.
func init() {
	prop.RegisterVariants[event](voided{}, stamped{})
}

// TestConverter checks the Go values that a derived generator decodes its
// choices to and runs back to them: those of Of for each kind of Go type,
// and those of OfShape for each shape. It checks each value that the
// inverse refuses, with the path to the part of the value.
func TestConverter(t *testing.T) {
	t.Parallel()

	t.Run("Of", func(t *testing.T) {
		t.Parallel()

		t.Run("decodes a struct's fields from its record's choices in order", func(t *testing.T) {
			t.Parallel()
			got := decodedBy(prop.Of[order](), integer(7), integer(1), sequence(10, 11), integer(5), integer(0),
				integer(0))
			assert.Equal(t, got, order{ID: 7, Lines: []line{{SKU: "ab", Qty: 5}}}, "an order of one line and no note")
		})

		t.Run("decodes a list of no element to a nil slice", func(t *testing.T) {
			t.Parallel()
			got := decodedBy(prop.Of[order](), integer(7), integer(0), integer(0))
			assert.Nil(t, got.Lines, "no lines")
		})

		t.Run("decodes a present optional to a pointer to its value", func(t *testing.T) {
			t.Parallel()
			got := decodedBy(prop.Of[*string](), integer(1), sequence(10))
			assert.Equal(t, got, new("a"), "a pointer to the string")
		})

		t.Run("decodes a set to a map to empty structs", func(t *testing.T) {
			t.Parallel()
			got := decodedBy(prop.Of[map[string]struct{}](), integer(1), sequence(10), integer(0))
			assert.Equal(t, got, map[string]struct{}{"a": {}}, "the set of one string")
		})

		t.Run("decodes a fixed-list to an array", func(t *testing.T) {
			t.Parallel()
			got := decodedBy(prop.Of[[2]bool](), integer(1), integer(1), integer(1), integer(0), integer(0))
			assert.Equal(t, got, [2]bool{true, false}, "the array of two values")
		})

		t.Run("decodes each registered variant to a value of its type", func(t *testing.T) {
			t.Parallel()
			g := prop.Of[payment]()
			assert.Equal(t, decodedBy(g, integer(0)), payment(pending{}), "the variant without a payload")
			assert.Equal(t, decodedBy(g, integer(1), integer(42)), payment(paid(42)), "the variant of an amount")
			assert.Equal(t, decodedBy(g, integer(3), sequence(10)), payment(&cancelled{Reason: "a"}),
				"the variant that a pointer implements")
		})

		t.Run("decodes a variant without a payload to the zero value of its type", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, decodedBy(prop.Of[event](), integer(0)), event(voided{}), "the zero voided")
		})

		t.Run("decodes a registered value", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, decodedBy(prop.Of[status](), integer(2)), statusShipped, "the third value")
		})

		t.Run("runs every generated value of every type back to choices that decode to it", func(t *testing.T) {
			t.Parallel()
			roundTrips(t, prop.Of[scalars](), sameValue)
			roundTrips(t, prop.Of[domains](), sameValue)
			roundTrips(t, prop.Of[collections](), sameValue)
			roundTrips(t, prop.Of[order](), sameValue)
			roundTrips(t, prop.Of[tree](), sameValue)
			roundTrips(t, prop.Of[ping](), sameValue)
			roundTrips(t, prop.Of[payment](), sameValue)
		})

		refusals := []struct {
			name       string
			give       func() error
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "refuses a struct whose unexported field is not zero",
				give:       func() error { return refusal(order{hidden: 1}) },
				wantPath:   fault.Path{fault.Field("hidden")},
				wantReason: "the field is not zero, and the shape states no value for it",
			},
			{
				name:       "refuses a struct whose field that the tag leaves out is not zero",
				give:       func() error { return refusal(order{Skipped: "x"}) },
				wantPath:   fault.Path{fault.Field("Skipped")},
				wantReason: "the field is not zero, and the shape states no value for it",
			},
			{
				name:       "refuses a nil *big.Int",
				give:       func() error { return refusal(struct{ V *bignum.Int }{}) },
				wantPath:   fault.Path{fault.Field("V")},
				wantReason: "a nil *big.Int is no integer",
			},
			{
				name:       "refuses a pointer to an absent pointer",
				give:       func() error { return refusal(struct{ V **int8 }{V: new(*int8)}) },
				wantPath:   fault.Path{fault.Field("V")},
				wantReason: "**int8 points to an absent value, which an optional states as absent",
			},
			{
				name:       "refuses a nil interface",
				give:       func() error { return refusal(struct{ V payment }{}) },
				wantPath:   fault.Path{fault.Field("V")},
				wantReason: "a nil prop_test.payment is no variant",
			},
			{
				name:       "refuses a value of a type that is no registered variant",
				give:       func() error { return refusal(struct{ V payment }{V: lost{}}) },
				wantPath:   fault.Path{fault.Field("V")},
				wantReason: "prop_test.lost is no variant of prop_test.payment that RegisterVariants states",
			},
			{
				name:       "refuses a nil pointer of a variant that a pointer implements",
				give:       func() error { return refusal(struct{ V payment }{V: (*cancelled)(nil)}) },
				wantPath:   fault.Path{fault.Field("V")},
				wantReason: "a nil *prop_test.cancelled is no variant",
			},
			{
				name:       "refuses a variant without a payload that is not zero",
				give:       func() error { return refusal(event(voided{note: "x"})) },
				wantPath:   nil,
				wantReason: "prop_test.voided is not zero, and the variant voided has no payload",
			},
			{
				name: "refuses a payload and states its variant",
				give: func() error {
					return refusal(event(stamped{At: time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)}))
				},
				wantPath:   fault.Path{fault.Variant("stamped"), fault.Field("at")},
				wantReason: "2020-01-01 12:00:00 +0000 UTC is no midnight in UTC, which a date is",
			},
			{
				name: "refuses a list and states the index of its element",
				give: func() error {
					return refusal(struct {
						V []rune `prop:"char"`
					}{V: []rune{'a', 0xD800}})
				},
				wantPath:   fault.Path{fault.Field("V"), fault.Index(1)},
				wantReason: "U+D800 is no character",
			},
			{
				name: "refuses a set and states its element",
				give: func() error {
					return refusal(struct{ V map[*bignum.Int]struct{} }{V: map[*bignum.Int]struct{}{nil: {}}})
				},
				wantPath:   fault.Path{fault.Field("V"), fault.Key((*bignum.Int)(nil))},
				wantReason: "a nil *big.Int is no integer",
			},
			{
				name: "refuses a map whose key converts to no key of the shape, and states the key",
				give: func() error {
					return refusal(struct{ V map[*bignum.Int]int8 }{V: map[*bignum.Int]int8{nil: 1}})
				},
				wantPath:   fault.Path{fault.Field("V"), fault.Key((*bignum.Int)(nil))},
				wantReason: "the key converts to no key of the shape",
			},
			{
				name: "refuses a map and states the key of its value",
				give: func() error {
					return refusal(struct{ V map[int8]*bignum.Int }{V: map[int8]*bignum.Int{1: nil}})
				},
				wantPath:   fault.Path{fault.Field("V"), fault.Key(int8(1))},
				wantReason: "a nil *big.Int is no integer",
			},
			{
				name:       "refuses a value that is none of the registered values",
				give:       func() error { return refusal(struct{ V status }{V: "lost"}) },
				wantPath:   fault.Path{fault.Field("V")},
				wantReason: "lost is none of the 3 values",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, tt.give(), fault.Error{
					Path: tt.wantPath, Kind: engine.ErrCannotInvert, Reason: tt.wantReason,
				})
			})
		}
	})

	t.Run("OfShape", func(t *testing.T) {
		t.Parallel()

		t.Run("runs every generated value back to choices that decode to it, its set in any order", func(t *testing.T) {
			t.Parallel()
			g, err := prop.OfShape(everyShape)
			assert.NoError(t, err, "the shape reads")
			roundTrips(t, g, sameRecord)
		})

		literals := []struct {
			name  string
			shape string
			give  any
		}{
			{
				name:  "runs a record back from its typed literal",
				shape: `{"shape":"record","fields":[["on",{"shape":"bool"}]]}`,
				give:  literal.Record{Fields: []literal.Field{{Name: "on", Value: true}}},
			},
			{
				name:  "runs a map back from the map that its typed literal decodes to",
				shape: `{"shape":"map","key":{"shape":"string"},"of":{"shape":"int","width":8,"signed":true}}`,
				give:  map[string]int{"a": 1},
			},
			{
				name:  "runs a map back from its entries",
				shape: `{"shape":"map","key":{"shape":"string"},"of":{"shape":"bool"}}`,
				give:  literal.Pairs{Entries: []literal.Entry{{Key: "a", Value: true}}},
			},
			{
				name:  "runs a variant back from its typed literal",
				shape: `{"shape":"enum","variants":[["none",null],["some",{"shape":"char"}]]}`,
				give:  literal.Variant{Name: "some", Payload: "a", HasPayload: true},
			},
			{
				name:  "runs a variant without a payload back from its typed literal",
				shape: `{"shape":"enum","variants":[["none",null],["some",{"shape":"char"}]]}`,
				give:  literal.Variant{Name: "none"},
			},
			{
				name:  "runs a literal back from the value that its typed literal decodes to",
				shape: `{"shape":"literal","values":[{"type":"record","fields":[["a",{"type":"int","value":1}]]}]}`,
				give:  literal.Record{Fields: []literal.Field{{Name: "a", Value: 1}}},
			},
			{
				name:  "runs a list back from a list of ints",
				shape: `{"shape":"list","of":{"shape":"int","width":8,"signed":true}}`,
				give:  []int{1, 2},
			},
			{
				name:  "runs an absent optional back from a nil list",
				shape: `{"shape":"optional","of":{"shape":"list","of":{"shape":"bool"}}}`,
				give:  []bool(nil),
			},
		}
		for _, tt := range literals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				g, err := prop.OfShape(tt.shape)
				assert.NoError(t, err, "the shape reads")
				choices, err := engine.Invert(engine.Generator[any](g), tt.give)
				assert.NoError(t, err, "the value runs back to its choices")
				assert.NotNil(t, choices, "the choices of the value")
			})
		}

		refusals := []struct {
			name       string
			shape      string
			give       any
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "refuses a list and states the index of its element",
				shape:      `{"shape":"list","of":{"shape":"char"}}`,
				give:       []any{'a', rune(0xD800)},
				wantPath:   fault.Path{fault.Index(1)},
				wantReason: "U+D800 is no character",
			},
			{
				name:       "refuses a map whose key converts to no key of the shape, and states the key",
				shape:      `{"shape":"map","key":{"shape":"char"},"of":{"shape":"bool"}}`,
				give:       map[any]any{rune(0xD800): true},
				wantPath:   fault.Path{fault.Key(rune(0xD800))},
				wantReason: "the key converts to no key of the shape",
			},
			{
				name:       "refuses a map and states the key of its value",
				shape:      `{"shape":"map","key":{"shape":"string"},"of":{"shape":"char"}}`,
				give:       map[any]any{"a": rune(0xD800)},
				wantPath:   fault.Path{fault.Key("a")},
				wantReason: "U+D800 is no character",
			},
			{
				name:       "refuses a record and states its field",
				shape:      `{"shape":"record","fields":[["c",{"shape":"char"}]]}`,
				give:       map[string]any{"c": rune(0xD800)},
				wantPath:   fault.Path{fault.Field("c")},
				wantReason: "U+D800 is no character",
			},
			{
				name:       "refuses a variant and states its payload",
				shape:      `{"shape":"enum","variants":[["some",{"shape":"char"}]]}`,
				give:       prop.Variant{Name: "some", Payload: rune(0xD800), HasPayload: true},
				wantPath:   fault.Path{fault.Variant("some")},
				wantReason: "U+D800 is no character",
			},
			{
				name:       "refuses a record of other fields",
				shape:      `{"shape":"record","fields":[["c",{"shape":"char"}]]}`,
				give:       map[string]any{"d": 'a'},
				wantReason: "map[d:97] is no record of the fields [c]",
			},
			{
				name:       "refuses a record of more fields",
				shape:      `{"shape":"record","fields":[["c",{"shape":"char"}]]}`,
				give:       map[string]any{"c": 'a', "d": 'b'},
				wantReason: "map[c:97 d:98] is no record of the fields [c]",
			},
			{
				name:       "refuses the typed literal of a record of other fields",
				shape:      `{"shape":"record","fields":[["c",{"shape":"char"}]]}`,
				give:       literal.Record{Fields: []literal.Field{{Name: "d", Value: "a"}}},
				wantReason: "{[{d a}]} is no record of the fields [c]",
			},
			{
				name:       "refuses the typed literal of a record of more fields",
				shape:      `{"shape":"record","fields":[["c",{"shape":"char"}]]}`,
				give:       literal.Record{Fields: []literal.Field{{Name: "c", Value: "a"}, {Name: "d", Value: "b"}}},
				wantReason: "{[{c a} {d b}]} is no record of the fields [c]",
			},
			{
				name:       "refuses a list for a record",
				shape:      `{"shape":"record","fields":[["c",{"shape":"char"}]]}`,
				give:       []any{'a'},
				wantReason: "[97] is no record of the fields [c]",
			},
			{
				name:       "refuses a variant that the enum does not have",
				shape:      `{"shape":"enum","variants":[["some",{"shape":"char"}]]}`,
				give:       prop.Variant{Name: "other"},
				wantReason: "{other <nil> false} is no variant of [some]",
			},
			{
				name:       "refuses a value of no shape's Go value for an enum",
				shape:      `{"shape":"enum","variants":[["some",{"shape":"char"}]]}`,
				give:       3,
				wantReason: "3 is no variant of [some]",
			},
			{
				name:       "passes a nil field on to the shape, which refuses it at the field",
				shape:      `{"shape":"record","fields":[["on",{"shape":"bool"}]]}`,
				give:       map[string]any{"on": nil},
				wantPath:   fault.Path{fault.Field("on")},
				wantReason: "<nil> is no bool",
			},
			{
				name:       "refuses a value that is no byte string for bytes",
				shape:      `{"shape":"bytes"}`,
				give:       3,
				wantReason: "3 is no []byte",
			},
			{
				name:       "refuses a value that is no list for a list",
				shape:      `{"shape":"list","of":{"shape":"char"}}`,
				give:       3,
				wantReason: "3 is no list",
			},
			{
				name:       "refuses a value that is no map for a map",
				shape:      `{"shape":"map","key":{"shape":"string"},"of":{"shape":"bool"}}`,
				give:       3,
				wantReason: "3 is no map",
			},
			{
				name:       "refuses a value that is none of a literal's values",
				shape:      `{"shape":"literal","values":[{"type":"int","value":1}]}`,
				give:       2,
				wantReason: "2 is none of the 1 values",
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, shapeRefusal(t, tt.shape, tt.give), fault.Error{
					Path: tt.wantPath, Kind: engine.ErrCannotInvert, Reason: tt.wantReason,
				})
			})
		}
	})
}

// TestConverterAllocs checks the allocation ceiling of a decode of an
// integer, a float and a character through the generator that Of returns.
func TestConverterAllocs(t *testing.T) {
	tests := []struct {
		name string
		give func()
		want uint64
	}{
		{name: "int8", give: decodes(prop.Of[int8](), signed(-5)), want: int8DecodeAllocs},
		{name: "uint64", give: decodes(prop.Of[uint64](), integer(7)), want: uint64DecodeAllocs},
		{name: "float32", give: decodes(prop.Of[float32](), floating(1.5)), want: float32DecodeAllocs},
		{name: "float64", give: decodes(prop.Of[float64](), floating(1.5)), want: float64DecodeAllocs},
		{name: "char", give: decodes(prop.Of[letterOf](), sequence(10)), want: charDecodeAllocs},
	}
	for _, tt := range tests {
		assert.MaxAllocs(t, tt.give, tt.want, "a decode of a "+tt.name+" allocates no converted number")
	}
}

// decodes returns the function that decodes choices with g, so that an
// allocation ceiling counts the decode alone.
func decodes[T any](g prop.Generator[T], choices ...choice.Choice) func() {
	return func() { _ = decodedBy(g, choices...) }
}

// roundTrips checks that each of the first 40 values of seed 7 that g
// generates runs back to choices that decode to a value that same reports
// equal to it.
func roundTrips[T any](tb testing.TB, g prop.Generator[T], same func(a, b T) bool) {
	tb.Helper()
	inner := engine.Generator[T](g)
	for index := range uint64(40) {
		var v T
		engine.Generate(func(c *engine.Case) { v = engine.Draw(c, inner, drawn) }, 7, index, nil)
		choices, err := engine.Invert(inner, v)
		assert.NoError(tb, err, "the value runs back to its choices")
		assert.True(tb, same(decodedBy(g, choices...), v), "the choices decode to the value")
	}
}

// sameValue reports whether a and b are equal as reflect.DeepEqual compares
// them, which costs linear time on a tree of 100 nodes.
func sameValue[T any](a, b T) bool {
	return reflect.DeepEqual(a, b)
}

// sameRecord reports whether a and b, values of everyShape, are equal with
// the elements of their sets in any order. A set runs back in the shortlex
// order of its elements' choices, so its []any can change order.
func sameRecord(a, b any) bool {
	return reflect.DeepEqual(sortedSet(a), sortedSet(b))
}

// sortedSet returns a copy of v, a value of everyShape, with its set's
// strings sorted.
func sortedSet(v any) map[string]any {
	m := maps.Clone(v.(map[string]any))
	set := slices.Clone(m["set"].([]any))
	slices.SortFunc(set, func(x, y any) int { return strings.Compare(x.(string), y.(string)) })
	m["set"] = set
	return m
}
