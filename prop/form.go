// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"errors"
	"reflect"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// The names of the options that state the values of cases, at the front of
// the path of the fault of an example.
const (
	// exampleOption states the values of one case.
	exampleOption = "Example"
	// examplesOption states one case per value.
	examplesOption = "Examples"
)

// FormOption configures a property form: an [Option] of the run, a
// relaxation of the form's assertion, such as [assert.EquateNaNs], or what
// [Using] and [Example] return. Its method takes a type of an internal
// package of this module, so no type outside the module is a FormOption.
type FormOption interface {
	// FormOption marks the type as an option of a property form.
	FormOption(matcher.FormSeal)
}

// FormOption makes an Option an option of a property form, which applies
// it to the form's run.
func (Option) FormOption(matcher.FormSeal) {}

// The options of a run and the relaxations are options of a property form.
var (
	_ FormOption = Option{}
	_ FormOption = assert.Option(nil)
)

// using is the option that [Using] returns: the generator of one type.
type using struct {
	// typ is the type.
	typ reflect.Type
	// erased is the generator with its type erased.
	erased engine.Generator[any]
	// typed is the Generator of the type.
	typed any
}

// FormOption makes the generator of one type an option of a property form.
func (using) FormOption(matcher.FormSeal) {}

// Using makes g the generator of T in the property form that it is passed
// to: of the form's input when the input is a T, and of each T that the
// input contains. It takes precedence over the generator that [Register]
// states and over T's shape. A second Using of one type replaces the
// first.
func Using[T any](g Generator[T]) FormOption {
	return using{typ: reflect.TypeFor[T](), erased: engine.Erase(engine.Generator[T](g)), typed: g}
}

// example is the option that [Example] or [Examples] returns: the values of
// one case, or one case per value.
type example struct {
	// typ is the type of the values.
	typ reflect.Type
	// values are the values: one for each generated argument of one case, or
	// for each reports one case per value.
	values []any
	// each reports whether each value is a case of its own, as Examples
	// states them.
	each bool
}

// FormOption makes the values of cases an option of a property form.
func (example) FormOption(matcher.FormSeal) {}

// Example makes the property form that it is passed to run a case of
// values before its stored cases: one value for each argument that the form
// generates, in order. That is one value for a form over a function, two
// for [Commutative] and three for [Associative]. The form computes the
// case's choices from the values when it is built, so a failing example
// shrinks as a generated case does. Each Example states one case.
//
// An input built with [Generator.Map], [Generator.Bind] or [Composite] has
// no inverse, so the form computes no choices. The case's draws then take
// the values as they are, and a failing case is reported as found, without
// a replay token and without an entry in the store.
//
// The form fails the test before any case runs for an example of another
// number of values, of values of another type than the form's input, and
// of a value that an input with an inverse does not produce.
func Example[T any](values ...T) FormOption {
	return example{typ: reflect.TypeFor[T](), values: anyValues(values)}
}

// Examples makes the property form over one input that it is passed to run
// a case of each value before its stored cases, in order, as Example of that
// one value does. Example and Examples run in the order of the options.
//
// A form over two or three inputs, [Commutative] and [Associative], fails
// the test before any case runs when it is passed Examples.
func Examples[T any](values ...T) FormOption {
	return example{typ: reflect.TypeFor[T](), values: anyValues(values), each: true}
}

// anyValues returns values as a slice of any.
func anyValues[T any](values []T) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

// The labels of the arguments that the forms generate.
var (
	// inputLabel labels the input of a form over a function and of a
	// relation of one input.
	inputLabel = []string{"input"}
	// pairLabels label the two inputs of a commutative form.
	pairLabels = []string{"a", "b"}
	// tripleLabels label the three inputs of an associative form.
	tripleLabels = []string{"a", "b", "c"}
)

// form is what a property form states: its operation, its id, whether its
// assertion takes relaxations, whether its cases run one at a time, and the
// labels of the arguments that it generates.
type form struct {
	// op is the form's operation, the package and its Go name, which names
	// its faults.
	op string
	// id is the form's id, the assertion of its record.
	id string
	// relaxed reports whether the form's assertion takes relaxations.
	relaxed bool
	// serial reports whether the form's cases run one at a time whatever
	// Workers states.
	serial bool
	// labels label the arguments that each case generates, in order.
	labels []string
}

// formOptions are the options of one call of a property form, by kind.
type formOptions struct {
	// run are the options of the run.
	run []Option
	// relaxations are the relaxations of the form's assertion.
	relaxations []matcher.Option
	// using are the generators that Using states, by type.
	using map[reflect.Type]using
	// examples are the cases that Example and Examples state, in the order
	// of the options.
	examples []example
}

// sortOptions returns the options of opts by kind.
func sortOptions(opts []FormOption) formOptions {
	var out formOptions
	for _, o := range opts {
		switch o := o.(type) {
		case Option:
			out.run = append(out.run, o)
		case matcher.Option:
			out.relaxations = append(out.relaxations, o)
		case using:
			if out.using == nil {
				out.using = make(map[reflect.Type]using)
			}
			out.using[o.typ] = o
		case example:
			out.examples = append(out.examples, o)
		}
	}
	return out
}

// runForm runs the property form f, whose input is a T, on tb at the call
// pc, as [ForAll] runs a body. Each case draws the arguments that f
// generates, under f's labels, and check runs f's assertion on them with
// the case as its seat. The run's record is of f's id, with msg as its
// contract.
//
// It ends the call with a fault of f's operation, without a run, for a
// relaxation of a form whose assertion takes none, for an input type that
// the reader refuses, for a wrong example, and for each fault for which
// ForAll ends its call without a run.
func runForm[T any](tb assert.TB, f form, msg string, pc uintptr, opts []FormOption,
	check func(c *Case, args []T, relaxations []matcher.Option),
) {
	tb.Helper()
	run := matcher.Begin(tb)
	o := sortOptions(opts)
	if len(o.relaxations) > 0 && !f.relaxed {
		run.Fault(matcher.Fatal, f.id, msg,
			fault.In(f.op, fault.New("the form takes no relaxation, because its assertion takes none")))
		return
	}
	g, err := inputOf[T](o.using)
	if err != nil {
		run.Fault(matcher.Fatal, f.id, msg, fault.In(f.op, err))
		return
	}
	examples, err := examplesOf(f, g, o.examples)
	if err != nil {
		run.Fault(matcher.Fatal, f.id, msg, fault.In(f.op, err))
		return
	}
	settings := configure(o.run)
	if f.serial {
		settings.workers = 1
	}
	p, err := newProperty(tb, f.op, msg, pc, settings)
	if err != nil {
		run.Fault(matcher.Fatal, f.id, msg, err)
		return
	}
	p.assertion = f.id
	p.settings.Examples = examples
	p.run(tb, run, func(c *Case) {
		args := make([]T, len(f.labels))
		for i, label := range f.labels {
			args[i] = c.Draw(g, label)
		}
		check(c, args, o.relaxations)
	})
}

// inputOf returns the generator of a form's input, a T: the one that using
// states for T, and otherwise the one that [Of] returns, with the
// generators of using in place at each type they state.
func inputOf[T any](generators map[reflect.Type]using) (Generator[T], error) {
	if u, ok := generators[reflect.TypeFor[T]()]; ok {
		return u.typed.(Generator[T]), nil
	}
	if len(generators) == 0 {
		return cachedOf[T]()
	}
	erased := make(map[reflect.Type]engine.Generator[any], len(generators))
	for t, u := range generators {
		erased[t] = u.erased
	}
	return of[T](registrations, erased)
}

// examplesOf returns the cases of examples under g, the generator of the
// input of the form f, in order: the choices of each case, or its values
// where g has no inverse for one of them. It returns a fault at the option,
// by its index among the examples, for values of another type than the
// input, for an Example of another number of values than f generates, and
// for Examples of a form over more than one input. It returns a fault at the
// value for a value that g does not produce.
func examplesOf[T any](f form, g Generator[T], examples []example) ([]engine.Example, error) {
	input := reflect.TypeFor[T]()
	var out []engine.Example
	for i, ex := range examples {
		at := []fault.Segment{fault.Field(exampleOption), fault.Index(i)}
		if ex.each {
			at[0] = fault.Field(examplesOption)
		}
		if ex.typ != input {
			return nil, fault.At(fault.New("the example states values of %v, and the input is of %v", ex.typ, input),
				at...)
		}
		if !ex.each {
			if len(ex.values) != len(f.labels) {
				return nil, fault.At(fault.New("the example states %d values, and the form generates %d",
					len(ex.values), len(f.labels)), at...)
			}
			c, j, err := caseOf(g, ex.values)
			if err != nil {
				return nil, fault.At(err, append(at, fault.Index(j))...)
			}
			out = append(out, c)
			continue
		}
		if len(f.labels) != 1 {
			return nil, fault.At(fault.New("the examples state one value per case, and the form generates %d",
				len(f.labels)), at...)
		}
		for j, v := range ex.values {
			c, _, err := caseOf(g, []any{v})
			if err != nil {
				return nil, fault.At(err, append(at, fault.Index(j))...)
			}
			out = append(out, c)
		}
	}
	return out, nil
}

// caseOf returns the case of values, one for each generated argument: the
// choices that decode to them under g, or the values themselves when g has
// no inverse for one of them. It returns the fault of a value that g does
// not produce, with the value's index.
func caseOf[T any](g Generator[T], values []any) (engine.Example, int, error) {
	var choices []choice.Choice
	for j, v := range values {
		// The type of the values is T, so the assertion fails only for a nil
		// interface value, whose T is the zero value.
		value, _ := v.(T)
		c, err := engine.Invert(engine.Generator[T](g), value)
		if errors.Is(err, engine.ErrNoInverse) {
			return engine.Example{Values: values}, 0, nil
		}
		if err != nil {
			return engine.Example{}, j, err
		}
		choices = append(choices, c...)
	}
	return engine.Example{Choices: choices}, 0, nil
}
