// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package literal_test

import (
	"encoding/json"
	"math"
	"math/big"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
)

// The allocations of the decoders: Decode on a record of one integer field,
// Int on a decimal string beyond 2^53 - 1, Float on the name NaN, and
// Objects.Decode on a reference to an integer whose id it decoded before,
// which allocates the copies of the reference's id and value text.
const (
	decodeAllocs        = 17
	intAllocs           = 3
	floatAllocs         = 1
	objectsDecodeAllocs = 2
)

// referenceToOne is the literal of a reference of the id a to the integer 1.
const referenceToOne = `{"type":"reference","id":"a","value":{"type":"int","value":1}}`

// FuzzDecode checks that Decode returns a value or an error for any bytes,
// and does not panic.
func FuzzDecode(f *testing.F) {
	f.Add([]byte(`{"type":"null"}`))
	f.Add([]byte(`{"type":"int","value":"9223372036854775808"}`))
	f.Add([]byte(`{"type":"list","of":"float","value":["NaN",1.5,null]}`))
	f.Add([]byte(`{"type":"map","entries":[[{"type":"list","items":[]},{"type":"bytes","value":"00"}]]}`))
	f.Add([]byte(`{"type":"record","fields":[["id",{"type":"int","value":1}]]}`))
	f.Add([]byte(`{"type":"variant","name":"paid","payload":{"type":"null"}}`))
	f.Add([]byte(referenceToOne))
	f.Fuzz(func(t *testing.T, raw []byte) {
		_, _ = literal.Decode(raw)
	})
}

// TestDecode checks Decode, which turns a typed literal into the Go value
// that it states.
func TestDecode(t *testing.T) {
	t.Parallel()

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want any
		}{
			{name: "returns nil for the null literal", give: `{"type":"null"}`, want: nil},
			{name: "returns a bool for a bool literal", give: `{"type":"bool","value":true}`, want: true},
			{name: "returns an int for an int literal", give: `{"type":"int","value":1}`, want: 1},
			{
				name: "returns an int64 for a decimal string below -(2^53 - 1)",
				give: `{"type":"int","value":"-9007199254740992"}`,
				want: int64(-9007199254740992),
			},
			{
				name: "returns an int64 for a decimal string above 2^53 - 1",
				give: `{"type":"int","value":"9007199254740992"}`,
				want: int64(9007199254740992),
			},
			{
				name: "returns a uint64 for a decimal string above the int64 range",
				give: `{"type":"int","value":"9223372036854775808"}`,
				want: uint64(9223372036854775808),
			},
			{
				name: "returns a *big.Int for a decimal string above the uint64 range",
				give: `{"type":"int","value":"18446744073709551616"}`,
				want: bigInt(t, "18446744073709551616"),
			},
			{
				name: "returns a *big.Int for a decimal string below the int64 range",
				give: `{"type":"int","value":"-9223372036854775809"}`,
				want: bigInt(t, "-9223372036854775809"),
			},
			{name: "returns a float64 for a float literal", give: `{"type":"float","value":1.5}`, want: 1.5},
			{
				name: "returns positive infinity for the float literal Inf",
				give: `{"type":"float","value":"Inf"}`,
				want: math.Inf(1),
			},
			{
				name: "returns negative infinity for the float literal -Inf",
				give: `{"type":"float","value":"-Inf"}`,
				want: math.Inf(-1),
			},
			{name: "returns a string for a string literal", give: `{"type":"string","value":"abc"}`, want: "abc"},
			{
				name: "returns the bytes of lowercase hexadecimal",
				give: `{"type":"bytes","value":"00ff"}`,
				want: []byte{0, 0xff},
			},
			{
				name: "returns a non-nil empty slice for an empty list",
				give: `{"type":"list","of":"int","value":[]}`,
				want: []int{},
			},
			{
				name: "returns a slice of bool for a list of bool",
				give: `{"type":"list","of":"bool","value":[true]}`,
				want: []bool{true},
			},
			{
				name: "returns a slice of float64 for a list of float",
				give: `{"type":"list","of":"float","value":[1.5]}`,
				want: []float64{1.5},
			},
			{
				name: "returns a slice of string for a list of string",
				give: `{"type":"list","of":"string","value":["a"]}`,
				want: []string{"a"},
			},
			{
				name: "returns a nil slice of the stated type for a list whose value is null",
				give: `{"type":"list","of":"int","value":null}`,
				want: []int(nil),
			},
			{
				name: "returns the infinities for the named floats of a list of float",
				give: `{"type":"list","of":"float","value":["Inf","-Inf",1.5]}`,
				want: []float64{math.Inf(1), math.Inf(-1), 1.5},
			},
			{
				name: "returns an int for a decimal string of a list of int",
				give: `{"type":"list","of":"int","value":[1,"9007199254740992"]}`,
				want: []int{1, 9007199254740992},
			},
			{
				name: "returns a slice of any for a list of int with a value beyond the range of int",
				give: `{"type":"list","of":"int","value":[1,"9223372036854775808","18446744073709551616"]}`,
				want: []any{1, uint64(9223372036854775808), bigInt(t, "18446744073709551616")},
			},
			{
				name: "returns a slice of any for a list of items",
				give: `{"type":"list","items":[{"type":"int","value":1},{"type":"string","value":"a"}]}`,
				want: []any{1, "a"},
			},
			{
				name: "returns a non-nil empty map for an empty map of string keys",
				give: `{"type":"map","key":"string","of":"int","value":{}}`,
				want: map[string]int{},
			},
			{
				name: "returns a map of int for a map of string keys and int values",
				give: `{"type":"map","key":"string","of":"int","value":{"a":1}}`,
				want: map[string]int{"a": 1},
			},
			{
				name: "returns a map of bool for a map of string keys and bool values",
				give: `{"type":"map","key":"string","of":"bool","value":{"a":true}}`,
				want: map[string]bool{"a": true},
			},
			{
				name: "returns a map of float64 for a map of string keys and float values",
				give: `{"type":"map","key":"string","of":"float","value":{"a":1.5}}`,
				want: map[string]float64{"a": 1.5},
			},
			{
				name: "returns a map of string for a map of string keys and string values",
				give: `{"type":"map","key":"string","of":"string","value":{"a":"b"}}`,
				want: map[string]string{"a": "b"},
			},
			{
				name: "returns a nil map of the stated type for a map whose value is null",
				give: `{"type":"map","key":"string","of":"string","value":null}`,
				want: map[string]string(nil),
			},
			{
				name: "returns the infinities for the named floats of a map of float",
				give: `{"type":"map","key":"string","of":"float","value":{"a":"Inf","b":"-Inf"}}`,
				want: map[string]float64{"a": math.Inf(1), "b": math.Inf(-1)},
			},
			{
				name: "returns an int for a decimal string of a map of int",
				give: `{"type":"map","key":"string","of":"int","value":{"a":"-9007199254740992"}}`,
				want: map[string]int{"a": -9007199254740992},
			},
			{
				name: "returns a map of any for a map of int with a value beyond the range of int",
				give: `{"type":"map","key":"string","of":"int","value":{"a":1,"b":"18446744073709551616"}}`,
				want: map[string]any{"a": 1, "b": bigInt(t, "18446744073709551616")},
			},
			{
				name: "returns a nil map of int for a map of int whose value is null",
				give: `{"type":"map","key":"string","of":"int","value":null}`,
				want: map[string]int(nil),
			},
			{
				name: "returns a map of any keys for a map of entries",
				give: `{"type":"map","entries":[[{"type":"int","value":1},{"type":"string","value":"a"}]]}`,
				want: map[any]any{1: "a"},
			},
			{
				name: "returns a map of a nil key for an entry of a null key",
				give: `{"type":"map","entries":[[{"type":"null"},{"type":"int","value":1}]]}`,
				want: map[any]any{nil: 1},
			},
			{
				name: "returns a record of its fields in order",
				give: `{"type":"record","fields":[["id",{"type":"int","value":1}],["note",{"type":"null"}]]}`,
				want: literal.Record{Fields: []literal.Field{{Name: "id", Value: 1}, {Name: "note"}}},
			},
			{
				name: "returns a record of no field",
				give: `{"type":"record","fields":[]}`,
				want: literal.Record{Fields: []literal.Field{}},
			},
			{
				name: "returns a variant without a payload",
				give: `{"type":"variant","name":"pending"}`,
				want: literal.Variant{Name: "pending"},
			},
			{
				name: "returns a variant whose payload is null apart from one without a payload",
				give: `{"type":"variant","name":"refunded","payload":{"type":"null"}}`,
				want: literal.Variant{Name: "refunded", HasPayload: true},
			},
			{
				name: "returns a variant with its payload",
				give: `{"type":"variant","name":"paid","payload":{"type":"int","value":5}}`,
				want: literal.Variant{Name: "paid", Payload: 5, HasPayload: true},
			},
			{
				name: "returns a pointer to the value of a reference",
				give: referenceToOne,
				want: new(1),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := literal.Decode(json.RawMessage(tt.give))
				assert.NoError(t, err, "the literal decodes")
				assert.Equal(t, got, tt.want, "the literal decodes to the value of its type")
			})
		}

		t.Run("returns NaN for the float literal NaN", func(t *testing.T) {
			t.Parallel()
			got, err := literal.Decode(json.RawMessage(`{"type":"float","value":"NaN"}`))
			assert.NoError(t, err, "the literal decodes")
			assert.True(t, math.IsNaN(got.(float64)), "the literal decodes to NaN")
		})

		t.Run("returns NaN for the named float NaN of a list of float", func(t *testing.T) {
			t.Parallel()
			got, err := literal.Decode(json.RawMessage(`{"type":"list","of":"float","value":["NaN"]}`))
			assert.NoError(t, err, "the literal decodes")
			assert.True(t, math.IsNaN(got.([]float64)[0]), "the element decodes to NaN")
		})

		unknown := []struct {
			name string
			give string
			want fault.Path
		}{
			{
				name: "returns ErrUnknownType at the type for an unknown type",
				give: `{"type":"widget"}`,
				want: fault.Path{fault.Field("type")},
			},
			{
				name: "returns ErrUnknownType at the type for an opaque literal, which no corpus case states",
				give: `{"type":"opaque","text":"func(int) bool"}`,
				want: fault.Path{fault.Field("type")},
			},
			{
				name: "returns ErrUnknownType at of for a list of an unknown element type",
				give: `{"type":"list","of":"widget","value":[]}`,
				want: fault.Path{fault.Field("of")},
			},
			{
				name: "returns ErrUnknownType at of for a null list of an unknown element type",
				give: `{"type":"list","of":"widget","value":null}`,
				want: fault.Path{fault.Field("of")},
			},
			{
				name: "returns ErrUnknownType at the item for a list item of an unknown type",
				give: `{"type":"list","items":[{"type":"widget"}]}`,
				want: fault.Path{fault.Field("items"), fault.Index(0), fault.Field("type")},
			},
			{
				name: "returns ErrUnknownType at the key type for a map of int keys stated as an object",
				give: `{"type":"map","key":"int","of":"int","value":{}}`,
				want: fault.Path{fault.Field("key")},
			},
			{
				name: "returns ErrUnknownType at of for a map of string keys and values of an unknown type",
				give: `{"type":"map","key":"string","of":"widget","value":{}}`,
				want: fault.Path{fault.Field("of")},
			},
			{
				name: "returns ErrUnknownType at the key for a map key of an unknown type",
				give: `{"type":"map","entries":[[{"type":"widget"},{"type":"int","value":1}]]}`,
				want: fault.Path{fault.Field("entries"), fault.Index(0), fault.Index(0), fault.Field("type")},
			},
			{
				name: "returns ErrUnknownType at the value for a map value of an unknown type",
				give: `{"type":"map","entries":[[{"type":"int","value":1},{"type":"widget"}]]}`,
				want: fault.Path{fault.Field("entries"), fault.Index(0), fault.Index(1), fault.Field("type")},
			},
			{
				name: "returns ErrUnknownType at the value for a record field of an unknown type",
				give: `{"type":"record","fields":[["id",{"type":"widget"}]]}`,
				want: fault.Path{fault.Field("fields"), fault.Index(0), fault.Index(1), fault.Field("type")},
			},
			{
				name: "returns ErrUnknownType at the payload for a payload of an unknown type",
				give: `{"type":"variant","name":"paid","payload":{"type":"widget"}}`,
				want: fault.Path{fault.Field("payload"), fault.Field("type")},
			},
			{
				name: "returns ErrUnknownType at the value for a reference to a value of an unknown type",
				give: `{"type":"reference","id":"a","value":{"type":"widget"}}`,
				want: fault.Path{fault.Field("value"), fault.Field("type")},
			},
		}
		for _, tt := range unknown {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := literal.Decode(json.RawMessage(tt.give))
				assert.ErrorIs(t, err, literal.ErrUnknownType, "the literal states a type outside the encoding")
				f := assert.ErrorAs[*fault.Error](t, err, "a fault")
				assert.Equal(t, f.Path, tt.want, "the path to the type")
			})
		}

		refusals := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault for text that is no JSON",
				give:       `{`,
				wantReason: "the text is no typed literal",
			},
			{
				name:       "returns a fault at the value for an int value that is no number",
				give:       `{"type":"int","value":true}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: "the value is no int",
			},
			{
				name:       "returns a fault at the value for the decimal string of -(2^53 - 1)",
				give:       `{"type":"int","value":"-9007199254740991"}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: "-9007199254740991 is within 2^53 - 1, which a JSON number states",
			},
			{
				name:       "returns a fault at the value for the decimal string of 2^53 - 1",
				give:       `{"type":"int","value":"9007199254740991"}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: "9007199254740991 is within 2^53 - 1, which a JSON number states",
			},
			{
				name:       "returns a fault at the value for a decimal string of a plus sign",
				give:       `{"type":"int","value":"+9007199254740992"}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: `"+9007199254740992" is no canonical integer of 64 bits`,
			},
			{
				name:       "returns a fault at the value for a decimal string of a leading zero",
				give:       `{"type":"int","value":"09007199254740992"}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: `"09007199254740992" is no canonical integer of 64 bits`,
			},
			{
				name:       "returns a fault at the value for a decimal string of 2^64 with a leading zero",
				give:       `{"type":"int","value":"018446744073709551616"}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: `"018446744073709551616" is no canonical integer of 64 bits`,
			},
			{
				name:       "returns a fault at the value for a string that is no integer",
				give:       `{"type":"int","value":"12a"}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: `"12a" is no canonical integer of 64 bits`,
			},
			{
				name:       "returns a fault at the value for an unknown float name",
				give:       `{"type":"float","value":"Huge"}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: `"Huge" is none of the names NaN, Inf and -Inf`,
			},
			{
				name:       "returns a fault at the value for a float value that is no number",
				give:       `{"type":"float","value":true}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: "the value is no float",
			},
			{
				name:       "returns a fault at the value for a bytes value that is no string",
				give:       `{"type":"bytes","value":1}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: "the value is no text",
			},
			{
				name:       "returns a fault at the value for uppercase hexadecimal",
				give:       `{"type":"bytes","value":"00FF"}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: `"00FF" is no lowercase hexadecimal`,
			},
			{
				name:       "returns a fault at the value for hexadecimal of an odd length",
				give:       `{"type":"bytes","value":"0"}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: `"0" is no lowercase hexadecimal`,
			},
			{
				name:       "returns a fault at the value for a list value that is no array",
				give:       `{"type":"list","of":"int","value":3}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: "the value is no list",
			},
			{
				name:       "returns a fault at the element for a list element of another type",
				give:       `{"type":"list","of":"bool","value":[1]}`,
				wantPath:   fault.Path{fault.Field("value"), fault.Index(0)},
				wantReason: "the value is no bool",
			},
			{
				name:       "returns a fault at the element for an element of a list of int that is no number",
				give:       `{"type":"list","of":"int","value":[true]}`,
				wantPath:   fault.Path{fault.Field("value"), fault.Index(0)},
				wantReason: "the value is no int",
			},
			{
				name:       "returns a fault at the element for an element of a list of int that is no integer",
				give:       `{"type":"list","of":"int","value":["9a"]}`,
				wantPath:   fault.Path{fault.Field("value"), fault.Index(0)},
				wantReason: `"9a" is no canonical integer of 64 bits`,
			},
			{
				name:       "returns a fault at the key for a value of a map of int that is no number",
				give:       `{"type":"map","key":"string","of":"int","value":{"a":true}}`,
				wantPath:   fault.Path{fault.Field("value"), fault.Key("a")},
				wantReason: "the value is no int",
			},
			{
				name:       "returns a fault at the key for a value of a map of float that is no number",
				give:       `{"type":"map","key":"string","of":"float","value":{"a":true}}`,
				wantPath:   fault.Path{fault.Field("value"), fault.Key("a")},
				wantReason: "the value is no float",
			},
			{
				name:       "returns a fault at the value for a map value that is no object",
				give:       `{"type":"map","key":"string","of":"int","value":[]}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: "the value is no object",
			},
			{
				name:       "returns a fault at the key of the entry for a map key of a list",
				give:       `{"type":"map","entries":[[{"type":"list","of":"int","value":[1]},{"type":"int","value":1}]]}`,
				wantPath:   fault.Path{fault.Field("entries"), fault.Index(0), fault.Index(0)},
				wantReason: "a key of []int is no key of a Go map",
			},
			{
				name:       "returns a fault at the entry for a map entry of three literals",
				give:       `{"type":"map","entries":[[{"type":"int","value":1},{"type":"int","value":2},{"type":"null"}]]}`,
				wantPath:   fault.Path{fault.Field("entries"), fault.Index(0)},
				wantReason: "the entry states 3 literals, not a key and a value",
			},
			{
				name:       "returns a fault at the fields for a record without fields",
				give:       `{"type":"record"}`,
				wantPath:   fault.Path{fault.Field("fields")},
				wantReason: "the record states no fields",
			},
			{
				name:       "returns a fault at the field for a record field of one part",
				give:       `{"type":"record","fields":[["id"]]}`,
				wantPath:   fault.Path{fault.Field("fields"), fault.Index(0)},
				wantReason: "the field is no pair of a name and a literal",
			},
			{
				name:       "returns a fault at the field for a record field whose name is no string",
				give:       `{"type":"record","fields":[[1,{"type":"null"}]]}`,
				wantPath:   fault.Path{fault.Field("fields"), fault.Index(0)},
				wantReason: "the field is no pair of a name and a literal",
			},
			{
				name:       "returns a fault at the field for a record field that is no list",
				give:       `{"type":"record","fields":["id"]}`,
				wantPath:   fault.Path{fault.Field("fields"), fault.Index(0)},
				wantReason: "the field is no pair of a name and a literal",
			},
			{
				name:       "returns a fault at the name for a record field of an empty name",
				give:       `{"type":"record","fields":[["",{"type":"null"}]]}`,
				wantPath:   fault.Path{fault.Field("fields"), fault.Index(0), fault.Index(0)},
				wantReason: `the name "" is empty or named twice`,
			},
			{
				name:       "returns a fault at the second name for a record that names a field twice",
				give:       `{"type":"record","fields":[["id",{"type":"null"}],["id",{"type":"null"}]]}`,
				wantPath:   fault.Path{fault.Field("fields"), fault.Index(1), fault.Index(0)},
				wantReason: `the name "id" is empty or named twice`,
			},
			{
				name:       "returns a fault at the name for a variant without a name",
				give:       `{"type":"variant"}`,
				wantPath:   fault.Path{fault.Field("name")},
				wantReason: "the variant states no name",
			},
			{
				name:       "returns a fault at the id for a reference without an id",
				give:       `{"type":"reference","value":{"type":"int","value":1}}`,
				wantPath:   fault.Path{fault.Field("id")},
				wantReason: "the reference states no id",
			},
			{
				name:       "returns a fault at the value for a reference to null",
				give:       `{"type":"reference","id":"a","value":{"type":"null"}}`,
				wantPath:   fault.Path{fault.Field("value")},
				wantReason: `the reference "a" refers to null, which is no object`,
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := literal.Decode(json.RawMessage(tt.give))
				f := assert.ErrorAs[*fault.Error](t, err, "the literal states no value of its type")
				assert.Equal(t, f.Path, tt.wantPath, "the path to what the literal misstates")
				assert.Equal(t, f.Reason, tt.wantReason, "what the literal misstates")
			})
		}

		t.Run("returns two objects for two references of one id", func(t *testing.T) {
			t.Parallel()
			first, err := literal.Decode(json.RawMessage(referenceToOne))
			assert.NoError(t, err, "the first reference decodes")
			second, err := literal.Decode(json.RawMessage(referenceToOne))
			assert.NoError(t, err, "the second reference decodes")
			assert.NotEqual(t, first, second, "a new object each time", assert.ByIdentity())
		})
	})

	t.Run("Objects.Decode", func(t *testing.T) {
		t.Parallel()

		t.Run("returns one object for every reference of one id, and another for another id", func(t *testing.T) {
			t.Parallel()
			objects := literal.Objects{}
			first, err := objects.Decode(json.RawMessage(referenceToOne))
			assert.NoError(t, err, "the first reference decodes")
			listed, err := objects.Decode(json.RawMessage(`{"type":"list","items":[` + referenceToOne + `]}`))
			assert.NoError(t, err, "the list decodes")
			other, err := objects.Decode(
				json.RawMessage(`{"type":"reference","id":"b","value":{"type":"int","value":1}}`),
			)
			assert.NoError(t, err, "the other reference decodes")
			assert.Equal(t, listed.([]any)[0], first, "the item is the object of a", assert.ByIdentity())
			assert.NotEqual(t, other, first, "b is another object", assert.ByIdentity())
			assert.Equal(t, other, first, "of an equal value")
		})

		t.Run("returns the fault of a literal as Decode does", func(t *testing.T) {
			t.Parallel()
			_, err := literal.Objects{}.Decode(json.RawMessage(`{"type":"widget"}`))
			assert.ErrorIs(t, err, literal.ErrUnknownType, "the literal states a type outside the encoding")
		})
	})
}

// bigInt returns the integer of the decimal text, failing the test when the
// text states none.
func bigInt(tb testing.TB, text string) *big.Int {
	tb.Helper()
	n, ok := new(big.Int).SetString(text, 10)
	assert.True(tb, ok, "the text states an integer")
	return n
}

// BenchmarkDecode measures Decode on a record of one integer field.
func BenchmarkDecode(b *testing.B) {
	raw := json.RawMessage(`{"type":"record","fields":[["id",{"type":"int","value":1}]]}`)
	got, _ := literal.Decode(raw)
	c := bench.Start(b).MaxAllocs(decodeAllocs)
	defer c.End()
	for c.Loop() {
		got, _ = literal.Decode(raw)
	}
	assert.Equal(b, got, any(literal.Record{Fields: []literal.Field{{Name: "id", Value: 1}}}), "the record")
}

// BenchmarkObjectsDecode measures Objects.Decode on a reference whose id it
// decoded before, which returns the object without decoding the value.
func BenchmarkObjectsDecode(b *testing.B) {
	raw := json.RawMessage(referenceToOne)
	objects := literal.Objects{}
	first, _ := objects.Decode(raw)
	got := first
	c := bench.Start(b).MaxAllocs(objectsDecodeAllocs)
	defer c.End()
	for c.Loop() {
		got, _ = objects.Decode(raw)
	}
	assert.Equal(b, got, first, "the object of the id", assert.ByIdentity())
}

// BenchmarkInt measures Int on a decimal string beyond 2^53 - 1.
func BenchmarkInt(b *testing.B) {
	raw := json.RawMessage(`"9007199254740992"`)
	got, _ := literal.Int(raw)
	c := bench.Start(b).MaxAllocs(intAllocs)
	defer c.End()
	for c.Loop() {
		got, _ = literal.Int(raw)
	}
	assert.Equal(b, got, any(int64(9007199254740992)), "the integer")
}

// BenchmarkFloat measures Float on the name NaN.
func BenchmarkFloat(b *testing.B) {
	raw := json.RawMessage(`"NaN"`)
	got, _ := literal.Float(raw)
	c := bench.Start(b).MaxAllocs(floatAllocs)
	defer c.End()
	for c.Loop() {
		got, _ = literal.Float(raw)
	}
	assert.True(b, math.IsNaN(got), "the float is NaN")
}
