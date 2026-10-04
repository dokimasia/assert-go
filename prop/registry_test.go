// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// The types that the registry tests register.
type (
	// die is an integer whose generator Register states.
	die int8
	// dice is a list of dice.
	dice struct {
		Dice []die `json:"dice"`
	}
	// grade is a string whose values RegisterValues states.
	grade string
	// point is a struct whose values RegisterValues states.
	point struct {
		X int8 `json:"x"`
		Y int8 `json:"y"`
	}
	// parcel is an interface whose variants RegisterVariants states.
	parcel interface{ isParcel() }
	// label is the variant of parcel of a text.
	label string
	// letter is the variant of parcel without a payload.
	letter struct{}
	// crate is the variant of parcel that a pointer implements.
	crate struct {
		Contents string `json:"contents"`
	}
	// again is an integer that a case registers two generators of.
	again int8
	// valued is an integer that a case registers values and a generator of.
	valued int8
	// looked is an integer that a case reads before it registers it.
	looked int8
	// shipping is an interface that a case registers variants and a
	// generator of.
	shipping interface{ isShipping() }
	// courier is the variant of shipping.
	courier struct{}
	// late is an integer that a test registers after a property ran.
	late int8
	// blank is a string that a case registers no value of.
	blank string
	// counted is an integer whose generator Register states.
	counted int8
	// holder contains counted, whose generator states no typed literal.
	holder struct {
		Counts []counted
	}
	// secret is a struct with a field that its shape states no value for.
	secret struct {
		ID     int8
		hidden int
	}
	// raw is a string of which a case registers a value that is no UTF-8.
	raw string
	// twin is a string of which a case registers one value twice.
	twin string
	// plain is an integer, which has no variants.
	plain int8
	// vacant is an interface that a case registers no variant of.
	vacant interface{ isVacant() }
	// withNil is an interface that a case registers a nil variant of.
	withNil interface{ isWithNil() }
	// filled is the variant of withNil.
	filled struct{}
	// duplicated is an interface that a case registers one variant of twice.
	duplicated interface{ isDuplicated() }
	// firstOf is a variant of duplicated.
	firstOf struct{}
	// secondOf is a variant of duplicated.
	secondOf struct{}
	// reading is a float whose values RegisterValues states: +0, -0 and
	// NaN.
	reading float64
)

func (label) isParcel()        {}
func (letter) isParcel()       {}
func (*crate) isParcel()       {}
func (courier) isShipping()    {}
func (filled) isWithNil()      {}
func (firstOf) isDuplicated()  {}
func (secondOf) isDuplicated() {}

// The values of grade, in the order that RegisterValues states them.
const (
	gradeB grade = "b"
	gradeA grade = "a"
)

// registryCase is a registration that init makes in the test process's
// registry before any property runs, as a test process makes one, and the
// value that it panicked with.
type registryCase struct {
	// name is the case's name.
	name string
	// give makes the registration.
	give func()
	// want is the value that give panics with.
	want any
	// got is the value that give panicked with in init, and nil for none.
	got any
}

// The cases of each registration, which init runs.
var (
	// registerTests are the cases of Register.
	registerTests = []*registryCase{
		{
			name: "panics for a type with a registered generator",
			give: func() {
				prop.Register(prop.Integer[again](0, 1))
				prop.Register(prop.Integer[again](0, 9))
			},
			want: "prop: Register[prop_test.again] registers prop_test.again a second time",
		},
		{
			name: "panics for a type with registered values",
			give: func() {
				prop.RegisterValues(valued(1))
				prop.Register(prop.Integer[valued](0, 9))
			},
			want: "prop: Register[prop_test.valued] registers prop_test.valued a second time",
		},
		{
			name: "panics for an interface with registered variants",
			give: func() {
				prop.RegisterVariants[shipping](courier{})
				prop.Register(prop.Just[shipping](courier{}))
			},
			want: "prop: Register[prop_test.shipping] registers prop_test.shipping a second time",
		},
		{
			name: "panics for a type that a read looked up",
			give: func() {
				_, _ = prop.ShapeOf[struct{ L []looked }]()
				prop.Register(prop.Integer[looked](0, 9))
			},
			want: "prop: Register[prop_test.looked] after a read of prop_test.looked, " +
				"whose generators would not see the registration",
		},
	}
	// registerValuesTests are the cases of RegisterValues.
	registerValuesTests = []*registryCase{
		{
			name: "panics for no value",
			give: func() { prop.RegisterValues[blank]() },
			want: "prop: RegisterValues[prop_test.blank] states no value",
		},
		{
			name: "panics for a type that the reader refuses",
			give: func() { prop.RegisterValues(make(chan int)) },
			want: "prop: RegisterValues[chan int]: chan int: chan int is no type that a shape states",
		},
		{
			name: "panics for a type that contains a type with a registered generator",
			give: func() {
				prop.Register(prop.Integer[counted](0, 9))
				prop.RegisterValues(holder{})
			},
			want: "prop: RegisterValues[prop_test.holder]: holder.Counts[]: the type has a registered generator, " +
				"whose values state no typed literal",
		},
		{
			name: "panics for a value that its shape states no value for",
			give: func() { prop.RegisterValues(secret{hidden: 1}) },
			want: "prop: RegisterValues[prop_test.secret]: value 0: hidden: the field is not zero, " +
				"and the shape states no value for it",
		},
		{
			name: "panics for a value that no typed literal states",
			give: func() { prop.RegisterValues(raw("\xff")) },
			want: "prop: RegisterValues[prop_test.raw]: value 0 states no typed literal",
		},
		{
			name: "panics for two values of one typed literal",
			give: func() { prop.RegisterValues(twin("a"), twin("b"), twin("a")) },
			want: "prop: RegisterValues[prop_test.twin]: values 0 and 2 state one typed literal",
		},
	}
	// registerVariantsTests are the cases of RegisterVariants.
	registerVariantsTests = []*registryCase{
		{
			name: "panics for a type that is no interface",
			give: func() { prop.RegisterVariants[plain](plain(1)) },
			want: "prop: RegisterVariants[prop_test.plain]: prop_test.plain is no interface",
		},
		{
			name: "panics for no variant",
			give: func() { prop.RegisterVariants[vacant]() },
			want: "prop: RegisterVariants[prop_test.vacant] states no variant",
		},
		{
			name: "panics for a nil variant",
			give: func() { prop.RegisterVariants[withNil](filled{}, nil) },
			want: "prop: RegisterVariants[prop_test.withNil]: variant 1 is nil",
		},
		{
			name: "panics for a variant of a type without a name",
			give: func() { prop.RegisterVariants[any]([]int{}) },
			want: "prop: RegisterVariants[interface {}]: variant 0 is of []int, whose type has no name",
		},
		{
			name: "panics for two variants of one name",
			give: func() { prop.RegisterVariants[duplicated](firstOf{}, secondOf{}, firstOf{}) },
			want: "prop: RegisterVariants[prop_test.duplicated]: variants 0 and 2 are both named firstOf",
		},
	}
)

// init makes the registrations of the registry tests in the test process's
// registry, as a test process makes them: before any property runs. It
// records the panic of each case.
func init() {
	prop.Register(prop.Integer[die](3, 3))
	prop.RegisterValues(gradeB, gradeA)
	prop.RegisterValues(reading(0), reading(math.Copysign(0, -1)), reading(math.NaN()))
	prop.RegisterValues(point{X: 1, Y: 2}, point{X: 3, Y: 4})
	prop.RegisterVariants[parcel](label(""), letter{}, &crate{})
	for _, tests := range [][]*registryCase{registerTests, registerValuesTests, registerVariantsTests} {
		for _, tt := range tests {
			tt.got = assert.Panics(assert.NewRecorder(), tt.give, "the registration panics")
		}
	}
}

// TestRegistry checks what each registration states, and each panic of a
// registration. The registrations are those of init in the test process's
// registry, and two tests run a property, which closes it.
func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("makes Of return the registered generator", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, decodedBy(prop.Of[die]()), die(3), "the registered generator's one value")
		})

		t.Run("places the registered generator in the shape of a type that contains the type", func(t *testing.T) {
			t.Parallel()
			got := decodedBy(prop.Of[dice](), integer(1), integer(0))
			assert.Equal(t, got, dice{Dice: []die{3}}, "a die of the registered generator")
		})

		t.Run("runs a value back through the registered generator of a type it contains", func(t *testing.T) {
			t.Parallel()
			g := prop.Of[dice]()
			choices, err := engine.Invert(engine.Generator[dice](g), dice{Dice: []die{3, 3}})
			assert.NoError(t, err, "the dice run back through the registered generator")
			assert.Equal(t, decodedBy(g, choices...), dice{Dice: []die{3, 3}}, "the choices decode to the value")
		})

		t.Run("reads a type after a property closed the registry", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(assert.NewRecorder(), contract, draws(prop.Integer(0, 9)), prop.Seed(7))
			assert.Equal(t, decodedBy(prop.Of[[]late](), integer(1), integer(4), integer(0)), []late{4},
				"a list of the type's own shape")
		})

		t.Run("panics after the first run of a property, which closed the registry", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(assert.NewRecorder(), contract, draws(prop.Integer(0, 9)), prop.Seed(7))
			got := assert.Panics(t, func() { prop.Register(prop.Integer[late](0, 9)) }, "the registration")
			assert.Equal(t, got, any("prop: Register[prop_test.late] after the first run of a property, "+
				"which closed the registry"), "the panic names the registration and the reason")
		})

		for _, tt := range registerTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.got, tt.want, "the panic of the registration in init")
			})
		}
	})

	t.Run("RegisterValues", func(t *testing.T) {
		t.Parallel()

		t.Run("makes the shape of a type a literal of its values in order", func(t *testing.T) {
			t.Parallel()
			want := `{"shape":"literal","values":[{"type":"string","value":"b"},{"type":"string","value":"a"}]}`
			assert.Equal(t, shapeTree(t, prop.ShapeOf[grade]), jsonTree(t, want), "the literal of the values")
		})

		t.Run("makes the first value the simplest", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, decodedBy(prop.Of[grade](), integer(0)), gradeB, "the first value")
		})

		t.Run("states a struct's value as the record of its shape", func(t *testing.T) {
			t.Parallel()
			want := `{"shape":"literal","values":[` +
				`{"type":"record","fields":[["x",{"type":"int","value":1}],["y",{"type":"int","value":2}]]},` +
				`{"type":"record","fields":[["x",{"type":"int","value":3}],["y",{"type":"int","value":4}]]}]}`
			assert.Equal(t, shapeTree(t, prop.ShapeOf[point]), jsonTree(t, want), "the records of the values")
		})

		t.Run("runs a registered value back to the choice of its index", func(t *testing.T) {
			t.Parallel()
			g := prop.Of[point]()
			assert.Equal(t, decodedBy(g, integer(1)), point{X: 3, Y: 4}, "the second value")
			choices, err := engine.Invert(engine.Generator[point](g), point{X: 3, Y: 4})
			assert.NoError(t, err, "the value runs back")
			assert.Equal(t, decodedBy(g, choices...), point{X: 3, Y: 4}, "the choices decode to the value")
		})

		t.Run("runs -0 back to its own choice, apart from +0", func(t *testing.T) {
			t.Parallel()
			g := prop.Of[reading]()
			choices, err := engine.Invert(engine.Generator[reading](g), reading(math.Copysign(0, -1)))
			assert.NoError(t, err, "-0 runs back")
			assert.True(t, math.Signbit(float64(decodedBy(g, choices...))), "the choices decode to -0")
		})

		for _, tt := range registerValuesTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.got, tt.want, "the panic of the registration in init")
			})
		}
	})

	t.Run("RegisterVariants", func(t *testing.T) {
		t.Parallel()

		t.Run("makes the shape of an interface an enum of its variants in order", func(t *testing.T) {
			t.Parallel()
			want := `{"shape":"enum","variants":[["label",{"shape":"string"}],["letter",null],` +
				`["crate",{"shape":"record","fields":[["contents",{"shape":"string"}]]}]]}`
			assert.Equal(t, shapeTree(t, prop.ShapeOf[parcel]), jsonTree(t, want), "the enum of the variants")
		})

		t.Run("names a variant that a pointer implements by the type it points to", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, decodedBy(prop.Of[parcel](), integer(2), sequence(10)), parcel(&crate{Contents: "a"}),
				"a pointer to a crate")
		})

		for _, tt := range registerVariantsTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.got, tt.want, "the panic of the registration in init")
			})
		}
	})
}
