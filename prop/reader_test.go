// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	bignum "math/big"
	"net/netip"
	"strconv"
	"testing"
	"time"
	"unsafe"
	"uuid"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/prop"
)

// The names of the definitions of the recursive test types.
const (
	// treeName is the definition of tree.
	treeName = "go.dokimi.dev/assert/prop_test.tree"
	// pingName is the definition of ping.
	pingName = "go.dokimi.dev/assert/prop_test.ping"
)

// The shapes of the test types, as JSON.
const (
	// orderShape is the shape of order.
	orderShape = `{"shape":"record","fields":[` +
		`["id",{"shape":"int","width":32,"signed":false}],` +
		`["lines",{"shape":"list","max_size":3,"of":{"shape":"record","fields":[` +
		`["sku",{"shape":"string"}],["qty",{"shape":"int","width":32,"signed":true,"min":1,"max":99}]]}}],` +
		`["note",{"shape":"optional","of":{"shape":"string","max_size":10}}]]}`
	// statusShape is the shape of status.
	statusShape = `{"shape":"literal","values":[{"type":"string","value":"pending"},` +
		`{"type":"string","value":"paid"},{"type":"string","value":"shipped"}]}`
	// paymentShape is the shape of payment.
	paymentShape = `{"shape":"enum","variants":[["pending",null],` +
		`["paid",{"shape":"int","width":64,"signed":true}],["refunded",{"shape":"string"}],` +
		`["cancelled",{"shape":"record","fields":[["reason",{"shape":"string"}]]}]]}`
	// treeDefinition is the definition of tree.
	treeDefinition = `{"shape":"record","fields":[["value",{"shape":"int","width":32,"signed":true}],` +
		`["children",{"shape":"list","of":{"shape":"ref","name":"` + treeName + `"}}]]}`
	// treeShape is the shape of tree.
	treeShape = `{"shape":"ref","name":"` + treeName + `","definitions":{"` + treeName + `":` + treeDefinition + `}}`
	// pingShape is the shape of ping.
	pingShape = `{"shape":"ref","name":"` + pingName + `","definitions":{"` + pingName + `":` +
		`{"shape":"record","fields":[["next",{"shape":"optional","of":{"shape":"record","fields":[` +
		`["next",{"shape":"ref","name":"` + pingName + `"}]]}}]]}}}`
)

// The types that only the reader tests read.
type (
	// ignored is a struct whose only field its tag leaves out.
	ignored struct {
		H int `prop:"-"`
	}
	// keyed is an interface whose variants a Go map accepts as keys.
	keyed interface{ key() }
	// flat is a variant of keyed that does not refer back to keyed.
	flat int
	// nested is a variant of keyed that contains keyed.
	nested struct {
		Inner keyed `json:"inner"`
	}
	// loose is an interface with a variant that no Go map accepts as a key.
	loose interface{ loose() }
	// tight is a variant of loose that a Go map accepts as a key.
	tight int
	// sliced is a variant of loose that no Go map accepts as a key.
	sliced []int
	// pipeline is an interface with a variant whose payload no shape
	// states.
	pipeline interface{ isPipeline() }
	// clogged is the variant of pipeline of a channel.
	clogged struct {
		C chan int
	}
)

func (flat) key()           {}
func (nested) key()         {}
func (tight) loose()        {}
func (sliced) loose()       {}
func (clogged) isPipeline() {}

// init registers the variants of keyed, loose and pipeline in the test
// process's registry, before any property runs.
func init() {
	prop.RegisterVariants[keyed](flat(0), nested{})
	prop.RegisterVariants[loose](tight(0), sliced(nil))
	prop.RegisterVariants[pipeline](clogged{})
}

// TestReader checks the shape that the reader reads from each Go type, the
// keys of the prop tag that each type takes, and each part of a type that
// it refuses, with the path to the part.
func TestReader(t *testing.T) {
	t.Parallel()

	t.Run("ShapeOf", func(t *testing.T) {
		t.Parallel()

		word := strconv.Itoa(strconv.IntSize)
		reads := []struct {
			name string
			give func() (string, error)
			want string
		}{
			{name: "reads a bool as a bool", give: prop.ShapeOf[bool], want: `{"shape":"bool"}`},
			{
				name: "reads an int as a signed int of the platform's word",
				give: prop.ShapeOf[int],
				want: `{"shape":"int","width":` + word + `,"signed":true}`,
			},
			{name: "reads an int8 as a signed int of 8 bits", give: prop.ShapeOf[int8], want: intShape(8, true)},
			{name: "reads an int16 as a signed int of 16 bits", give: prop.ShapeOf[int16], want: intShape(16, true)},
			{name: "reads an int32 as a signed int of 32 bits", give: prop.ShapeOf[int32], want: intShape(32, true)},
			{name: "reads an int64 as a signed int of 64 bits", give: prop.ShapeOf[int64], want: intShape(64, true)},
			{
				name: "reads a uint as an unsigned int of the platform's word",
				give: prop.ShapeOf[uint],
				want: `{"shape":"int","width":` + word + `,"signed":false}`,
			},
			{name: "reads a uint8 as an unsigned int of 8 bits", give: prop.ShapeOf[uint8], want: intShape(8, false)},
			{
				name: "reads a uint16 as an unsigned int of 16 bits",
				give: prop.ShapeOf[uint16],
				want: intShape(16, false),
			},
			{
				name: "reads a uint32 as an unsigned int of 32 bits",
				give: prop.ShapeOf[uint32],
				want: intShape(32, false),
			},
			{
				name: "reads a uint64 as an unsigned int of 64 bits",
				give: prop.ShapeOf[uint64],
				want: intShape(64, false),
			},
			{
				name: "reads a float32 as a float of 32 bits",
				give: prop.ShapeOf[float32],
				want: `{"shape":"float","width":32}`,
			},
			{
				name: "reads a float64 as a float of 64 bits",
				give: prop.ShapeOf[float64],
				want: `{"shape":"float","width":64}`,
			},
			{name: "reads a string as a string", give: prop.ShapeOf[string], want: `{"shape":"string"}`},
			{name: "reads a []byte as bytes", give: prop.ShapeOf[[]byte], want: `{"shape":"bytes"}`},
			{
				name: "reads a [4]byte as bytes of exactly 4",
				give: prop.ShapeOf[[4]byte],
				want: `{"shape":"bytes","min_size":4,"max_size":4}`,
			},
			{
				name: "reads a slice of a type over uint8 as a list of integers",
				give: prop.ShapeOf[[]octet],
				want: `{"shape":"list","of":` + intShape(8, false) + `}`,
			},
			{
				name: "reads an array of a type over uint8 as a fixed-list of integers",
				give: prop.ShapeOf[[3]octet],
				want: `{"shape":"fixed-list","size":3,"of":` + intShape(8, false) + `}`,
			},
			{name: "reads a uuid.UUID as a uuid", give: prop.ShapeOf[uuid.UUID], want: `{"shape":"uuid"}`},
			{
				name: "reads a netip.Addr as an ip-address of either version",
				give: prop.ShapeOf[netip.Addr],
				want: `{"shape":"ip-address"}`,
			},
			{
				name: "reads a *big.Int as a signed int of 128 bits",
				give: prop.ShapeOf[*bignum.Int],
				want: intShape(128, true),
			},
			{
				name: "reads a time.Time as an instant at nanoseconds",
				give: prop.ShapeOf[time.Time],
				want: `{"shape":"instant","unit":"ns"}`,
			},
			{
				name: "reads a time.Duration as a duration at nanoseconds",
				give: prop.ShapeOf[time.Duration],
				want: `{"shape":"duration","unit":"ns"}`,
			},
			{name: "reads a *time.Location as a zone", give: prop.ShapeOf[*time.Location], want: `{"shape":"zone"}`},
			{
				name: "reads a WallTime as a wall-time at nanoseconds",
				give: prop.ShapeOf[prop.WallTime],
				want: `{"shape":"wall-time","unit":"ns"}`,
			},
			{
				name: "reads a slice as a list of its elements",
				give: prop.ShapeOf[[]int16],
				want: `{"shape":"list","of":` + intShape(16, true) + `}`,
			},
			{
				name: "reads an array as a fixed-list of its length",
				give: prop.ShapeOf[[3]bool],
				want: `{"shape":"fixed-list","size":3,"of":{"shape":"bool"}}`,
			},
			{
				name: "reads a map to an empty struct as a set of its keys",
				give: prop.ShapeOf[map[string]struct{}],
				want: `{"shape":"set","of":{"shape":"string"}}`,
			},
			{
				name: "reads a map as a map of its keys and values",
				give: prop.ShapeOf[map[string]int8],
				want: `{"shape":"map","key":{"shape":"string"},"of":` + intShape(8, true) + `}`,
			},
			{
				name: "reads a pointer as an optional",
				give: prop.ShapeOf[*string],
				want: `{"shape":"optional","of":{"shape":"string"}}`,
			},
			{
				name: "reads a type over a basic kind as that kind over its whole range",
				give: prop.ShapeOf[celsius],
				want: `{"shape":"float","width":64}`,
			},
			{
				name: "reads a struct as a record of its read fields, each named by its json tag",
				give: prop.ShapeOf[order],
				want: orderShape,
			},
			{
				name: "reads a type of registered values as a literal of their typed literals",
				give: prop.ShapeOf[status],
				want: statusShape,
			},
			{
				name: "reads an interface of registered variants as an enum of their shapes",
				give: prop.ShapeOf[payment],
				want: paymentShape,
			},
			{
				name: "reads a type that refers to itself as a ref to its definition",
				give: prop.ShapeOf[tree],
				want: treeShape,
			},
			{
				name: "reads two types that refer to each other as the definition of the first",
				give: prop.ShapeOf[ping],
				want: pingShape,
			},
			{
				name: "reads a recursive type that a type contains twice as one definition",
				give: prop.ShapeOf[struct{ A, B tree }],
				want: `{"shape":"record","fields":[["A",{"shape":"ref","name":"` + treeName + `"}],` +
					`["B",{"shape":"ref","name":"` + treeName + `"}]],"definitions":{"` + treeName + `":` +
					treeDefinition + `}}`,
			},
		}
		for _, tt := range reads {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, shapeTree(t, tt.give), jsonTree(t, tt.want), "the shape of the type")
			})
		}

		tagged := []struct {
			name string
			give func() (string, error)
			want string
		}{
			{
				name: "reads a rune with the key char as a char",
				give: prop.ShapeOf[struct {
					V rune `prop:"char"`
				}],
				want: `{"shape":"char"}`,
			},
			{
				name: "reads min and max of an int",
				give: prop.ShapeOf[struct {
					V int32 `prop:"min=1,max=99"`
				}],
				want: `{"shape":"int","width":32,"signed":true,"min":1,"max":99}`,
			},
			{
				name: "reads min and max of a uint",
				give: prop.ShapeOf[struct {
					V uint16 `prop:"min=1,max=99"`
				}],
				want: `{"shape":"int","width":16,"signed":false,"min":1,"max":99}`,
			},
			{
				name: "reads allow_nan and allow_infinity of a float as true",
				give: prop.ShapeOf[struct {
					V float64 `prop:"allow_nan,allow_infinity"`
				}],
				want: `{"shape":"float","width":64,"allow_nan":true,"allow_infinity":true}`,
			},
			{
				name: "reads min and max of a float",
				give: prop.ShapeOf[struct {
					V float32 `prop:"min=-1.5,max=2.5"`
				}],
				want: `{"shape":"float","width":32,"min":-1.5,"max":2.5}`,
			},
			{
				name: "reads min_size and max_size of a string",
				give: prop.ShapeOf[struct {
					V string `prop:"min_size=1,max_size=8"`
				}],
				want: `{"shape":"string","min_size":1,"max_size":8}`,
			},
			{
				name: "reads min_size of a []byte",
				give: prop.ShapeOf[struct {
					V []byte `prop:"min_size=2"`
				}],
				want: `{"shape":"bytes","min_size":2}`,
			},
			{
				name: "applies a key that a list does not take to its elements",
				give: prop.ShapeOf[struct {
					V []string `prop:"max_size=3,alphabet=ab"`
				}],
				want: `{"shape":"list","max_size":3,"of":{"shape":"string","alphabet":"ab"}}`,
			},
			{
				name: "applies a key that an array does not take to its elements",
				give: prop.ShapeOf[struct {
					V [2]int8 `prop:"min=0"`
				}],
				want: `{"shape":"fixed-list","size":2,"of":{"shape":"int","width":8,"signed":true,"min":0}}`,
			},
			{
				name: "applies a key of a pointer to the shape it points to",
				give: prop.ShapeOf[struct {
					V *string `prop:"max_size=10"`
				}],
				want: `{"shape":"optional","of":{"shape":"string","max_size":10}}`,
			},
			{
				name: "applies a key that a set does not take to its elements",
				give: prop.ShapeOf[struct {
					V map[string]struct{} `prop:"max_size=2,alphabet=xy"`
				}],
				want: `{"shape":"set","max_size":2,"of":{"shape":"string","alphabet":"xy"}}`,
			},
			{
				name: "reads min_size of a map",
				give: prop.ShapeOf[struct {
					V map[string]int8 `prop:"min_size=1"`
				}],
				want: `{"shape":"map","min_size":1,"key":{"shape":"string"},"of":` + intShape(8, true) + `}`,
			},
			{
				name: "reads unit, min and max of a time.Time",
				give: prop.ShapeOf[struct {
					V time.Time `prop:"unit=ms,min=2020-01-01T00:00:00Z"`
				}],
				want: `{"shape":"instant","unit":"ms","min":"2020-01-01T00:00:00Z"}`,
			},
			{
				name: "reads a time.Time with the key date as a date",
				give: prop.ShapeOf[struct {
					V time.Time `prop:"date,max=2020-01-01"`
				}],
				want: `{"shape":"date","max":"2020-01-01"}`,
			},
			{
				name: "reads a time.Time with the key local-date-time as a local date and time",
				give: prop.ShapeOf[struct {
					V time.Time `prop:"local-date-time,unit=us,max=2020-01-01T00:00:00"`
				}],
				want: `{"shape":"local-date-time","unit":"us","max":"2020-01-01T00:00:00"}`,
			},
			{
				name: "reads a time.Time with the key zoned-date-time as a zoned date and time",
				give: prop.ShapeOf[struct {
					V time.Time `prop:"zoned-date-time,unit=s"`
				}],
				want: `{"shape":"zoned-date-time","unit":"s"}`,
			},
			{
				name: "reads unit, min and max of a time.Duration as units",
				give: prop.ShapeOf[struct {
					V time.Duration `prop:"unit=us,min=-5,max=5"`
				}],
				want: `{"shape":"duration","unit":"us","min":-5,"max":5}`,
			},
			{
				name: "reads a time.Duration with the key time-of-day as a time of day",
				give: prop.ShapeOf[struct {
					V time.Duration `prop:"time-of-day,unit=s,max=12:00:00"`
				}],
				want: `{"shape":"time-of-day","unit":"s","max":"12:00:00"}`,
			},
			{
				name: "reads an integer with the key offset as an offset",
				give: prop.ShapeOf[struct {
					V int32 `prop:"offset,min=-3600"`
				}],
				want: `{"shape":"offset","min":-3600}`,
			},
			{
				name: "reads min of a *big.Int",
				give: prop.ShapeOf[struct {
					V *bignum.Int `prop:"min=0"`
				}],
				want: `{"shape":"int","width":128,"signed":true,"min":0}`,
			},
			{
				name: "reads a *big.Rat as a decimal of the scale that its tag states",
				give: prop.ShapeOf[struct {
					V *bignum.Rat `prop:"scale=2,min=0.01"`
				}],
				want: `{"shape":"decimal","scale":2,"min":"0.01"}`,
			},
			{
				name: "reads version of a netip.Addr",
				give: prop.ShapeOf[struct {
					V netip.Addr `prop:"version=4"`
				}],
				want: `{"shape":"ip-address","version":4}`,
			},
			{
				name: "reads unit of a WallTime",
				give: prop.ShapeOf[struct {
					V prop.WallTime `prop:"unit=s"`
				}],
				want: `{"shape":"wall-time","unit":"s"}`,
			},
		}
		for _, tt := range tagged {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, shapeTree(t, tt.give), jsonTree(t, `{"shape":"record","fields":[["V",`+tt.want+`]]}`),
					"the shape of the field")
			})
		}

		t.Run("names a field by its json tag, and by its Go name for a tag that names none", func(t *testing.T) {
			t.Parallel()
			got := shapeTree(t, prop.ShapeOf[struct {
				A bool `json:"a,omitempty"`
				B bool `json:"-"`
				C bool `json:",omitempty"`
			}])
			want := `{"shape":"record","fields":[["a",{"shape":"bool"}],["B",{"shape":"bool"}],["C",{"shape":"bool"}]]}`
			assert.Equal(t, got, jsonTree(t, want), "the names of the fields")
		})

		t.Run("reads a map keyed by an interface whose variants a Go map accepts as keys", func(t *testing.T) {
			t.Parallel()
			_, err := prop.ShapeOf[map[keyed]int8]()
			assert.NoError(t, err, "an integer, and a struct that contains the interface")
		})

		refusals := []struct {
			name       string
			give       func() (string, error)
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault at a field of a channel",
				give:       prop.ShapeOf[struct{ C chan int }],
				wantPath:   fault.Path{fault.Field("struct { C chan int }"), fault.Field("C")},
				wantReason: "chan int is no type that a shape states",
			},
			{
				name:       "returns a fault at a field of a function",
				give:       prop.ShapeOf[struct{ F func() }],
				wantPath:   fault.Path{fault.Field("struct { F func() }"), fault.Field("F")},
				wantReason: "func() is no type that a shape states",
			},
			{
				name:       "returns a fault at a field of a complex number",
				give:       prop.ShapeOf[struct{ X complex128 }],
				wantPath:   fault.Path{fault.Field("struct { X complex128 }"), fault.Field("X")},
				wantReason: "complex128 is no type that a shape states",
			},
			{
				name:       "returns a fault at a field of a uintptr",
				give:       prop.ShapeOf[struct{ U uintptr }],
				wantPath:   fault.Path{fault.Field("struct { U uintptr }"), fault.Field("U")},
				wantReason: "uintptr is no type that a shape states",
			},
			{
				name:       "returns a fault at a field of an unsafe.Pointer",
				give:       prop.ShapeOf[struct{ P unsafe.Pointer }],
				wantPath:   fault.Path{fault.Field("struct { P unsafe.Pointer }"), fault.Field("P")},
				wantReason: "unsafe.Pointer is no type that a shape states",
			},
			{
				name:       "returns a fault at a field of an interface without registered variants",
				give:       prop.ShapeOf[struct{ A any }],
				wantPath:   fault.Path{fault.Field("struct { A interface {} }"), fault.Field("A")},
				wantReason: "interface {} has no variants, which RegisterVariants states",
			},
			{
				name:       "returns a fault at a named type of no field that the reader reads",
				give:       prop.ShapeOf[ignored],
				wantPath:   fault.Path{fault.Field("ignored")},
				wantReason: "prop_test.ignored has no exported field to read",
			},
			{
				name:       "returns a fault at a type without a name, as its literal states it",
				give:       prop.ShapeOf[chan int],
				wantPath:   fault.Path{fault.Field("chan int")},
				wantReason: "chan int is no type that a shape states",
			},
			{
				name: "returns a fault on the path through a list and a pointer",
				give: prop.ShapeOf[struct{ Lines []struct{ Note *chan int } }],
				wantPath: fault.Path{
					fault.Field("struct { Lines []struct { Note *chan int } }"),
					fault.Field("Lines"), fault.Element(), fault.Field("Note"),
				},
				wantReason: "chan int is no type that a shape states",
			},
			{
				name: "returns a fault at the key of a map",
				give: prop.ShapeOf[struct{ M map[chan int]bool }],
				wantPath: fault.Path{
					fault.Field("struct { M map[chan int]bool }"),
					fault.Field("M"),
					fault.Field("key"),
				},
				wantReason: "chan int is no type that a shape states",
			},
			{
				name: "returns a fault at the value of a map",
				give: prop.ShapeOf[struct{ M map[string]chan int }],
				wantPath: fault.Path{
					fault.Field("struct { M map[string]chan int }"),
					fault.Field("M"),
					fault.Element(),
				},
				wantReason: "chan int is no type that a shape states",
			},
			{
				name: "returns a fault at the element of a set",
				give: prop.ShapeOf[struct{ S map[chan int]struct{} }],
				wantPath: fault.Path{
					fault.Field("struct { S map[chan int]struct {} }"),
					fault.Field("S"),
					fault.Element(),
				},
				wantReason: "chan int is no type that a shape states",
			},
			{
				name:       "returns a fault at the element of an array",
				give:       prop.ShapeOf[struct{ A [2]chan int }],
				wantPath:   fault.Path{fault.Field("struct { A [2]chan int }"), fault.Field("A"), fault.Element()},
				wantReason: "chan int is no type that a shape states",
			},
			{
				name:       "returns a fault at the payload of a variant",
				give:       prop.ShapeOf[pipeline],
				wantPath:   fault.Path{fault.Field("pipeline"), fault.Variant("clogged"), fault.Field("C")},
				wantReason: "chan int is no type that a shape states",
			},
			{
				name:       "returns a fault at the key of a map keyed by an interface with a variant that is no key",
				give:       prop.ShapeOf[map[loose]int8],
				wantPath:   fault.Path{fault.Field("map[prop_test.loose]int8"), fault.Field("key")},
				wantReason: "prop_test.loose has a variant that no Go map accepts as a key",
			},
			{
				name:       "returns a fault at the element of a set of structs that contain such an interface",
				give:       prop.ShapeOf[map[struct{ K loose }]struct{}],
				wantPath:   fault.Path{fault.Field("map[struct { K prop_test.loose }]struct {}"), fault.Element()},
				wantReason: "struct { K prop_test.loose } has a variant that no Go map accepts as a key",
			},
			{
				name:       "returns a fault at the key of a map keyed by arrays of such an interface",
				give:       prop.ShapeOf[map[[1]loose]int8],
				wantPath:   fault.Path{fault.Field("map[[1]prop_test.loose]int8"), fault.Field("key")},
				wantReason: "[1]prop_test.loose has a variant that no Go map accepts as a key",
			},
			{
				name: "returns a fault at a field whose tag states a key that applies to no part of its type",
				give: prop.ShapeOf[struct {
					V bool `prop:"min=1"`
				}],
				wantPath:   fieldV("bool", "min=1"),
				wantReason: "the tag key min applies to no part of bool",
			},
			{
				name: "returns a fault at a map whose tag states a key of its keys",
				give: prop.ShapeOf[struct {
					V map[string]int8 `prop:"alphabet=ab"`
				}],
				wantPath:   fieldV("map[string]int8", "alphabet=ab"),
				wantReason: "the tag key alphabet applies to no part of map[string]int8",
			},
			{
				name: "returns a fault at an offset of fewer than 32 bits",
				give: prop.ShapeOf[struct {
					V int16 `prop:"offset"`
				}],
				wantPath:   fieldV("int16", "offset"),
				wantReason: "an offset needs an integer of 32 bits or more, and int16 has 16",
			},
			{
				name: "returns a fault at a char that is no int32",
				give: prop.ShapeOf[struct {
					V int64 `prop:"char"`
				}],
				wantPath:   fieldV("int64", "char"),
				wantReason: "a char is a rune, and int64 is no int32",
			},
			{
				name: "returns a fault at a unit of a time.Time that the definition does not have",
				give: prop.ShapeOf[struct {
					V time.Time `prop:"unit=min"`
				}],
				wantPath:   fieldV("time.Time", "unit=min"),
				wantReason: `the unit "min" is none of s, ms, us and ns`,
			},
			{
				name: "returns a fault at a unit of a time.Duration that the definition does not have",
				give: prop.ShapeOf[struct {
					V time.Duration `prop:"unit=h"`
				}],
				wantPath:   fieldV("time.Duration", "unit=h"),
				wantReason: `the unit "h" is none of s, ms, us and ns`,
			},
			{
				name: "returns a fault at a unit of a WallTime that the definition does not have",
				give: prop.ShapeOf[struct {
					V prop.WallTime `prop:"unit=d"`
				}],
				wantPath:   fieldV("prop.WallTime", "unit=d"),
				wantReason: `the unit "d" is none of s, ms, us and ns`,
			},
			{
				name:       "returns a fault at a *big.Rat without a scale",
				give:       prop.ShapeOf[struct{ V *bignum.Rat }],
				wantPath:   fault.Path{fault.Field("struct { V *big.Rat }"), fault.Field("V")},
				wantReason: "a *big.Rat reads as a decimal, which needs the tag key scale",
			},
			{
				name: "returns a fault at a scale that is no count of digits",
				give: prop.ShapeOf[struct {
					V *bignum.Rat `prop:"scale=x"`
				}],
				wantPath:   fieldV("*big.Rat", "scale=x"),
				wantReason: `the scale "x" is no count of digits`,
			},
			{
				name: "returns a fault at a negative scale",
				give: prop.ShapeOf[struct {
					V *bignum.Rat `prop:"scale=-1"`
				}],
				wantPath:   fieldV("*big.Rat", "scale=-1"),
				wantReason: `the scale "-1" is no count of digits`,
			},
			{
				name: "returns a fault at a struct of two fields of one name",
				give: prop.ShapeOf[struct {
					A int8 `json:"B"`
					B int8
				}],
				wantPath:   fault.Path{fault.Field(`struct { A int8 "json:\"B\""; B int8 }`)},
				wantReason: `the fields A and B are both named "B"`,
			},
		}
		for _, tt := range refusals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := tt.give()
				expectFault(t, err, fault.Error{Op: shapeOfOp, Path: tt.wantPath, Reason: tt.wantReason})
			})
		}
	})
}

// intShape returns the JSON of an int shape of width bits.
func intShape(width int, signed bool) string {
	return `{"shape":"int","width":` + strconv.Itoa(width) + `,"signed":` + strconv.FormatBool(signed) + `}`
}
