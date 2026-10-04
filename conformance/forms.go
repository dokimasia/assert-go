// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"reflect"
	"strconv"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// formContract is the contract of the run of a forms vector's form.
const formContract = "the form of a forms vector"

// The Go types of the values that a form's assertion takes beside its
// subjects, as a typed literal decodes to them.
var (
	anyValue     = reflect.TypeFor[any]()
	intValue     = reflect.TypeFor[int]()
	floatValue   = reflect.TypeFor[float64]()
	stringValue  = reflect.TypeFor[string]()
	stringsValue = reflect.TypeFor[[]string]()
)

// The shapes of a subject that a form takes, each reporting whether the
// subject s has the shape.
func isFunction(s *Subject) bool { return s.Function != nil }
func isOrder(s *Subject) bool    { return s.Ordered != nil }
func isCall(s *Subject) bool     { return s.Call != nil }
func isRaise(s *Subject) bool    { return s.Raise != nil }
func isHandle(s *Subject) bool   { return s.Ctx != nil }
func isObserved(s *Subject) bool { return s.Call != nil && s.Observe != nil }
func isCompute(s *Subject) bool  { return s.Compute != nil }
func isCombine(s *Subject) bool  { return s.Combine != nil }
func isRender(s *Subject) bool   { return s.Render != nil }

// The subjects of the forms that take one subject or two, of a function of
// the input.
var (
	oneFunction  = []func(*Subject) bool{isFunction}
	twoFunctions = []func(*Subject) bool{isFunction, isFunction}
)

// integers states an integer in the Go type of the integers of a form's
// input, the type that OfShape decodes them to. A Go caller states an
// integer in that type, and Go's equality does not equate an int32 with an
// int64.
type integers struct {
	// typ is the Go type of the input's integers, and nil for an input of no
	// integers, whose run states each integer as it is.
	typ reflect.Type
}

// integersOf returns the integers of a form's input of the shape text,
// which OfShape has read: the type of the value that OfShape decodes for
// the shape, or for the shape of its elements when it states one. OfShape
// reads the shape of the elements on its own, and integersOf returns the
// error of OfShape when that read fails.
func integersOf(text json.RawMessage) (integers, error) {
	var input struct {
		// Of is the shape of the input's elements.
		Of json.RawMessage `json:"of"`
	}
	// OfShape has read text, so text is a JSON object.
	_ = json.Unmarshal(text, &input)
	if input.Of != nil {
		text = input.Of
	}
	g, err := prop.OfShape(string(text))
	if err != nil {
		return integers{}, err
	}
	sample, _ := decodeWith(engine.Generator[any](g), func(body engine.Body) engine.Execution {
		return engine.Generate(body, 0, 0, nil)
	})
	if v := reflect.ValueOf(sample); v.CanInt() {
		return integers{typ: v.Type()}, nil
	}
	return integers{}, nil
}

// bind returns v with each integer in it, the items of a list included, in
// the Go type of the input's integers.
func (n integers) bind(v any) any {
	if items, ok := v.([]any); ok {
		out := make([]any, len(items))
		for i, item := range items {
			out[i] = n.bind(item)
		}
		return out
	}
	if rv := reflect.ValueOf(v); n.typ != nil && rv.CanInt() {
		return rv.Convert(n.typ).Interface()
	}
	return v
}

// formRun is one run of a forms vector's form: the seat, the subjects, the
// values that the assertion takes beside them, the integers of the input,
// and the options of the run.
type formRun struct {
	// tb is the seat.
	tb assert.TB
	// subjects are the subjects, in the order the form takes them, each
	// built for the run.
	subjects []*Subject
	// args are the values that the assertion takes beside the subjects.
	args []any
	// ints are the integers of the input.
	ints integers
	// opts are the options of the run.
	opts []prop.FormOption
}

// function returns the function of the input of subject i, which returns
// its integers in the type of the input's.
func (r formRun) function(i int) func(any) any {
	f := r.subjects[i].Function
	return func(x any) any { return r.ints.bind(f(x)) }
}

// list returns the function of the input of subject i, whose value is a
// list.
func (r formRun) list(i int) func(any) []any {
	f := r.function(i)
	return func(x any) []any { return f(x).([]any) }
}

// condition returns the function of the input of the first subject, whose
// value is a bool.
func (r formRun) condition() func(any) bool {
	f := r.function(0)
	return func(x any) bool { return f(x).(bool) }
}

// fn returns the call of the first subject as a call without a failure,
// for the forms of purity.
func (r formRun) fn() func(any) {
	call := r.subjects[0].Call
	return func(x any) { _ = call(x) }
}

// formDriver is how a forms vector calls one form: the shape of each
// subject that the form takes, the Go types of the values that its
// assertion takes beside them, and the call.
type formDriver struct {
	// subjects report for each subject whether a subject has the shape that
	// the form takes.
	subjects []func(*Subject) bool
	// values are the Go types of the values, and anyValue for a value of
	// any type.
	values []reflect.Type
	// call runs the form on the subjects and the values of r, which match
	// subjects and values.
	call func(r formRun)
}

// build returns the subjects that kinds name, each built for the run, and
// a fault at the member subjects for kinds that are not the subjects that
// d takes.
func (d formDriver) build(kinds []string) ([]*Subject, error) {
	if len(kinds) != len(d.subjects) {
		return nil, fault.At(fault.New("the form takes %d subjects, and the vector states %d",
			len(d.subjects), len(kinds)), fault.Field(subjectsMember))
	}
	out := make([]*Subject, len(kinds))
	for i, is := range d.subjects {
		s := &Subject{}
		if build, ok := Subjects[kinds[i]]; ok {
			s = build()
		}
		if !is(s) {
			return nil, fault.At(fault.New("%q is no subject that the form takes", kinds[i]),
				fault.Field(subjectsMember), fault.Index(i))
		}
		out[i] = s
	}
	return out, nil
}

// check returns a fault at the member args for values that are not the
// values that d takes.
func (d formDriver) check(args []any) error {
	if len(args) != len(d.values) {
		return fault.At(fault.New("the form takes %d values, and the vector states %d", len(d.values), len(args)),
			fault.Field(argsMember))
	}
	for i, t := range d.values {
		if t != anyValue && reflect.TypeOf(args[i]) != t {
			return fault.At(fault.New("the value is a %T, and the form takes a %v", args[i], t),
				fault.Field(argsMember), fault.Index(i))
		}
	}
	return nil
}

// formDrivers states how a forms vector calls each form, by its id. The
// form of max-allocs has none, because no vector states an allocation
// count.
var formDrivers = map[string]formDriver{
	"prop-equal": {subjects: twoFunctions, call: func(r formRun) {
		prop.Equal(r.tb, r.function(0), r.function(1), formContract, r.opts...)
	}},
	"prop-not-equal": {subjects: twoFunctions, call: func(r formRun) {
		prop.NotEqual(r.tb, r.function(0), r.function(1), formContract, r.opts...)
	}},
	"prop-true": {subjects: oneFunction, call: func(r formRun) {
		prop.True(r.tb, r.condition(), formContract, r.opts...)
	}},
	"prop-false": {subjects: oneFunction, call: func(r formRun) {
		prop.False(r.tb, r.condition(), formContract, r.opts...)
	}},
	"prop-nil": {subjects: oneFunction, call: func(r formRun) {
		prop.Nil(r.tb, r.function(0), formContract, r.opts...)
	}},
	"prop-not-nil": {subjects: oneFunction, call: func(r formRun) {
		prop.NotNil(r.tb, r.function(0), formContract, r.opts...)
	}},
	"prop-length": {subjects: oneFunction, values: []reflect.Type{intValue}, call: func(r formRun) {
		prop.Length(r.tb, r.function(0), r.args[0].(int), formContract, r.opts...)
	}},
	"prop-empty": {subjects: oneFunction, call: func(r formRun) {
		prop.Empty(r.tb, r.function(0), formContract, r.opts...)
	}},
	"prop-not-empty": {subjects: oneFunction, call: func(r formRun) {
		prop.NotEmpty(r.tb, r.function(0), formContract, r.opts...)
	}},
	"prop-contains": {subjects: oneFunction, values: []reflect.Type{anyValue}, call: func(r formRun) {
		prop.Contains(r.tb, r.function(0), r.ints.bind(r.args[0]), formContract, r.opts...)
	}},
	"prop-not-contains": {subjects: oneFunction, values: []reflect.Type{anyValue}, call: func(r formRun) {
		prop.NotContains(r.tb, r.function(0), r.ints.bind(r.args[0]), formContract, r.opts...)
	}},
	"prop-contains-in-order": {subjects: oneFunction, values: []reflect.Type{stringsValue}, call: func(r formRun) {
		prop.ContainsInOrder(r.tb, r.function(0), r.args[0].([]string), formContract, r.opts...)
	}},
	"prop-permutation": {subjects: twoFunctions, call: func(r formRun) {
		prop.IsPermutation(r.tb, r.list(0), r.list(1), formContract, r.opts...)
	}},
	"prop-has-prefix": {subjects: oneFunction, values: []reflect.Type{stringValue}, call: func(r formRun) {
		prop.HasPrefix(r.tb, r.function(0), r.args[0].(string), formContract, r.opts...)
	}},
	"prop-has-suffix": {subjects: oneFunction, values: []reflect.Type{stringValue}, call: func(r formRun) {
		prop.HasSuffix(r.tb, r.function(0), r.args[0].(string), formContract, r.opts...)
	}},
	"prop-matches": {subjects: oneFunction, values: []reflect.Type{stringValue}, call: func(r formRun) {
		prop.Matches(r.tb, r.function(0), r.args[0].(string), formContract, r.opts...)
	}},
	"prop-close-to": {subjects: oneFunction, values: []reflect.Type{floatValue, floatValue}, call: func(r formRun) {
		prop.CloseTo(r.tb, r.function(0), r.args[0].(float64), r.args[1].(float64), formContract, r.opts...)
	}},
	"prop-in-range": {subjects: oneFunction, values: []reflect.Type{floatValue, floatValue}, call: func(r formRun) {
		prop.InRange(r.tb, r.function(0), r.args[0].(float64), r.args[1].(float64), formContract, r.opts...)
	}},
	"prop-pairwise": {subjects: []func(*Subject) bool{isFunction, isOrder}, call: func(r formRun) {
		prop.Pairwise(r.tb, r.list(0), r.subjects[1].Ordered, formContract, r.opts...)
	}},
	"prop-err-absent": {subjects: []func(*Subject) bool{isCall}, call: func(r formRun) {
		prop.NoError(r.tb, r.subjects[0].Call, formContract, r.opts...)
	}},
	"prop-err-present": {subjects: []func(*Subject) bool{isCall}, call: func(r formRun) {
		prop.HasError(r.tb, r.subjects[0].Call, formContract, r.opts...)
	}},
	"prop-err-is": {subjects: []func(*Subject) bool{isCall}, call: func(r formRun) {
		prop.ErrorIs(r.tb, r.subjects[0].Call, ErrOwn, formContract, r.opts...)
	}},
	"prop-err-is-not": {subjects: []func(*Subject) bool{isCall}, call: func(r formRun) {
		prop.ErrorIsNot(r.tb, r.subjects[0].Call, ErrOwn, formContract, r.opts...)
	}},
	// The type of ErrOwn is unexported, so the form takes error as the type
	// of the subject's own failure.
	"prop-err-as": {subjects: []func(*Subject) bool{isCall}, call: func(r formRun) {
		prop.ErrorAs[error](r.tb, r.subjects[0].Call, formContract, r.opts...)
	}},
	"prop-throws": {subjects: []func(*Subject) bool{isRaise}, call: func(r formRun) {
		prop.Panics(r.tb, r.subjects[0].Raise, formContract, r.opts...)
	}},
	"prop-not-throws": {subjects: []func(*Subject) bool{isRaise}, call: func(r formRun) {
		prop.NotPanics(r.tb, r.subjects[0].Raise, formContract, r.opts...)
	}},
	"prop-pure": {subjects: []func(*Subject) bool{isObserved}, call: func(r formRun) {
		prop.Pure(r.tb, r.subjects[0].Observe, r.fn(), formContract, r.opts...)
	}},
	"prop-not-pure": {subjects: []func(*Subject) bool{isObserved}, call: func(r formRun) {
		prop.NotPure(r.tb, r.subjects[0].Observe, r.fn(), formContract, r.opts...)
	}},
	"prop-nil-context-safe": {subjects: []func(*Subject) bool{isHandle}, call: func(r formRun) {
		prop.NilContextSafe(r.tb, r.subjects[0].Ctx, formContract, r.opts...)
	}},
	"prop-honours-cancellation": {subjects: []func(*Subject) bool{isHandle}, call: func(r formRun) {
		prop.HonoursCancellation(r.tb, r.subjects[0].Ctx, formContract, r.opts...)
	}},
	"prop-honours-deadline": {subjects: []func(*Subject) bool{isHandle}, call: func(r formRun) {
		prop.HonoursDeadline(r.tb, r.subjects[0].Ctx, formContract, r.opts...)
	}},
	"prop-idempotent": {subjects: []func(*Subject) bool{isObserved}, call: func(r formRun) {
		prop.Idempotent(r.tb, r.subjects[0].Call, r.subjects[0].Observe, formContract, r.opts...)
	}},
	"prop-accumulates": {subjects: []func(*Subject) bool{isObserved}, call: func(r formRun) {
		prop.Accumulates(r.tb, r.subjects[0].Call, r.subjects[0].Observe, formContract, r.opts...)
	}},
	"prop-deterministic": {subjects: []func(*Subject) bool{isCompute}, call: func(r formRun) {
		compute := r.subjects[0].Compute
		prop.Deterministic(r.tb, func(x any) (any, error) { return r.ints.bind(compute(x)), nil }, formContract,
			r.opts...)
	}},
	"prop-commutative": {subjects: []func(*Subject) bool{isCombine}, call: func(r formRun) {
		combine := r.subjects[0].Combine
		prop.Commutative(r.tb, func(a, b any) any { return r.ints.bind(combine(a, b)) }, formContract, r.opts...)
	}},
	"prop-associative": {subjects: []func(*Subject) bool{isCombine}, call: func(r formRun) {
		combine := r.subjects[0].Combine
		prop.Associative(r.tb, func(a, b any) any { return r.ints.bind(combine(a, b)) }, formContract, r.opts...)
	}},
	"prop-round-trip": {subjects: []func(*Subject) bool{isRender}, call: func(r formRun) {
		render := r.subjects[0].Render
		prop.RoundTrip(r.tb, func(x any) (string, error) { return render(x), nil }, func(text string) (any, error) {
			n, err := strconv.ParseInt(text, 10, 64)
			return r.ints.bind(n), err
		}, formContract, r.opts...)
	}},
}

// formVector is a forms vector: the form, the kinds of the subjects that
// its assertion takes, the values that the assertion takes beside them, the
// shape of what the form generates, the seed, and the detail of the run.
type formVector struct {
	// Form is the form's id.
	Form string `json:"form"`
	// Subjects are the kinds of the subjects.
	Subjects []string `json:"subjects"`
	// Args are the values, as typed literals.
	Args []json.RawMessage `json:"args"`
	// Shape is the shape of each argument that the form generates.
	Shape json.RawMessage `json:"shape"`
	// Seed is the seed in decimal.
	Seed string `json:"seed"`
	// Detail is the detail of the run.
	Detail formDetail `json:"detail"`
}

// formDetail is the detail of a forms vector's run. It states the failure
// of the minimal case as its record's assertion and the fields of the
// record that a typed literal states.
type formDetail struct {
	runDetail
	// Failure is the failure of the minimal case, and nil for a run without
	// one.
	Failure *struct {
		// Assertion is the assertion of the failure's record.
		Assertion string `json:"assertion"`
		// Detail are the fields of the record that the vector states, as
		// typed literals.
		Detail map[string]json.RawMessage `json:"detail"`
	} `json:"failure"`
}

// checkForms runs the form of a forms vector on its subjects and values,
// over the generator that OfShape returns for its shape, with its seed, and
// compares the run with the detail it states. A failing run is compared
// through its record, every detail field of it and each field of the
// failure that the vector states. A passing run reports no record, so it
// is compared through the detail that the form's call record states.
func checkForms(raw json.RawMessage, _ string) error {
	var v formVector
	if err := decode(raw, &v); err != nil {
		return err
	}
	d, ok := formDrivers[v.Form]
	if !ok {
		return fault.At(fault.New("%q is no form that a vector runs", v.Form), fault.Field(formMember))
	}
	opts, err := behaviourSettings{Seed: v.Seed}.resolve("")
	if err != nil {
		return err
	}
	g, err := prop.OfShape(string(v.Shape))
	if err != nil {
		return fault.At(fault.New("the shape does not read").Because(err), fault.Field(shapeMember))
	}
	ints, err := integersOf(v.Shape)
	if err != nil {
		return fault.At(fault.New("the shape of the elements does not read alone").Because(err),
			fault.Field(shapeMember), fault.Field(ofMember))
	}
	subjects, err := d.build(v.Subjects)
	if err != nil {
		return err
	}
	args := make([]any, len(v.Args))
	for i, a := range v.Args {
		if args[i], err = literal.Decode(a); err != nil {
			return fault.At(err, fault.Field(argsMember), fault.Index(i))
		}
	}
	if err := d.check(args); err != nil {
		return err
	}
	r := formRun{subjects: subjects, args: args, ints: ints, opts: []prop.FormOption{prop.Using(g)}}
	for _, o := range opts {
		r.opts = append(r.opts, o)
	}
	rec := assert.NewRecorder()
	r.tb = rec
	d.call(r)
	if records := rec.Failures(); len(records) > 0 {
		return at(v.compare(records[0]), fault.Field(detailMember))
	}
	return at(v.Detail.comparePassed(callsOf(rec)[0]), fault.Field(detailMember))
}

// compare returns how f, the record of a failing run, differs from the one
// that v states, or nil when they match. It returns a fault with a path
// inside the detail of v.
func (v formVector) compare(f assert.Failure) error {
	d := v.Detail.runDetail
	var stated map[string]json.RawMessage
	if failure := v.Detail.Failure; failure != nil {
		d.Failure, stated = &failure.Assertion, failure.Detail
	}
	if err := d.compare(f.Detail); err != nil {
		return err
	}
	failure, _ := f.Detail[failureField].(assert.Failure)
	return compareDetail("the failure", stated, failure.Detail, fault.Field(failureField))
}
