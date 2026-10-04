// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// The label of a body's one draw, as the definition names it.
const drawnLabel = "value"

// The bodies that the corpus names, because no predicate states them.
const (
	// drawsNothing requests no input.
	drawsNothing = "draws-nothing"
	// diverges draws an integer in [0, 9] on its first call and a boolean
	// on every later call.
	diverges = "diverges"
	// failsOnce draws an integer in [0, 10^9], and fails with the identity
	// once the first time it draws a value above 1,000.
	failsOnce = "fails-once"
)

// The facts of the bodies that the corpus names.
const (
	// onceIdentity is the identity of the failure of failsOnce.
	onceIdentity = "once"
	// onceAbove is the value above which failsOnce fails.
	onceAbove = 1000
	// onceMax is the upper bound of the integer of failsOnce.
	onceMax = 1000000000
	// digitMax is the upper bound of the first draw of diverges.
	digitMax = 9
)

// bodySpec is a body as the corpus states it: one draw, then the labels it
// classifies the case under, the predicate that rejects the case, and the
// failures in order, or the kind of a named body.
type bodySpec struct {
	// Kind names a named body, and is empty for a drawing body.
	Kind string `json:"kind"`
	// Draw is the generator of the one draw.
	Draw json.RawMessage `json:"draw"`
	// Classify are the predicates under whose labels the case counts.
	Classify map[string]json.RawMessage `json:"classify"`
	// RejectsWhen is the predicate that rejects the case, and nil for none.
	RejectsWhen json.RawMessage `json:"rejects-when"`
	// Fails are the failures of the case, the first whose predicate is
	// true.
	Fails []failSpec `json:"fails"`
}

// failSpec is one failure of a body: its identity and its predicate.
type failSpec struct {
	// Identity is the assertion of the failure's record.
	Identity string `json:"identity"`
	// When is the predicate under which the case fails.
	When json.RawMessage `json:"when"`
}

// ender ends the case of a body that ran to its end. failure is the
// identity of the case's failure, and empty for a case that does not fail.
type ender func(c *prop.Case, failure string)

// failing ends a case that fails with an aborting record whose assertion is
// the failure's identity, without a contract or a location, so the identity
// that the engine keeps is the one that the corpus names.
func failing(c *prop.Case, failure string) {
	if failure != "" {
		fail(c, failure)
	}
}

// bodyOf returns a fresh body that a corpus spec states, which end ends. A
// named body keeps its own state, so each run takes a fresh one.
//
// It returns a fault for a spec that names no body of the vocabulary, or
// states a draw or a predicate that the vocabulary does not, whose path
// leads through the spec to the part at fault.
func bodyOf(raw json.RawMessage, end ender) (func(*prop.Case), error) {
	var spec bodySpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fault.New("the body does not parse").Because(err)
	}
	switch spec.Kind {
	case "":
		return drawing(spec, end)
	case drawsNothing:
		return func(c *prop.Case) { end(c, "") }, nil
	case diverges:
		return diverging(end), nil
	case failsOnce:
		return failingOnce(end), nil
	}
	return nil, fault.At(fault.New("%q names no body", spec.Kind), fault.Field(kindMember))
}

// drawing returns the body that draws one value, classifies, rejects and
// fails by the predicates of spec, in that order. Its failure is the first
// whose predicate is true.
func drawing(spec bodySpec, end ender) (func(*prop.Case), error) {
	g, err := generatorOf(spec.Draw)
	if err != nil {
		return nil, fault.At(err, fault.Field(drawMember))
	}
	classify := make(map[string]func(any) bool, len(spec.Classify))
	for label, when := range spec.Classify {
		if classify[label], err = predicateOf(when); err != nil {
			return nil, fault.At(err, fault.Field(classifyMember), fault.Key(label))
		}
	}
	rejects := func(any) bool { return false }
	if spec.RejectsWhen != nil {
		if rejects, err = predicateOf(spec.RejectsWhen); err != nil {
			return nil, fault.At(err, fault.Field(rejectsWhenMember))
		}
	}
	fails := make([]func(any) bool, len(spec.Fails))
	for i, f := range spec.Fails {
		if fails[i], err = predicateOf(f.When); err != nil {
			return nil, fault.At(err, fault.Field(failsMember), fault.Index(i), fault.Field(whenMember))
		}
	}
	draw := prop.Generator[any](g)
	return func(c *prop.Case) {
		v := c.Draw(draw, drawnLabel)
		for label, matches := range classify {
			if matches(v) {
				c.Classify(label)
			}
		}
		c.Assume(!rejects(v))
		failure := ""
		for i, matches := range fails {
			if matches(v) {
				failure = spec.Fails[i].Identity
				break
			}
		}
		end(c, failure)
	}, nil
}

// diverging returns the body of diverges, which end ends. A call counts
// once its draw returns, as in the executable reference.
func diverging(end ender) func(*prop.Case) {
	digit, bit := prop.Integer(0, digitMax), prop.Boolean()
	drawn := false
	return func(c *prop.Case) {
		if drawn {
			c.Draw(bit, drawnLabel)
		} else {
			c.Draw(digit, drawnLabel)
		}
		drawn = true
		end(c, "")
	}
}

// failingOnce returns the body of fails-once, which end ends.
func failingOnce(end ender) func(*prop.Case) {
	wide := prop.Integer(0, onceMax)
	failed := false
	return func(c *prop.Case) {
		failure := ""
		if c.Draw(wide, drawnLabel) > onceAbove && !failed {
			failed, failure = true, onceIdentity
		}
		end(c, failure)
	}
}

// fail ends the case with an aborting record of identity.
func fail(c *prop.Case, identity string) {
	c.Report(assert.Failure{Assertion: identity}, true)
}

// engineBody returns body as a body of the engine.
func engineBody(body func(*prop.Case)) engine.Body {
	return func(c *engine.Case) { body((*prop.Case)(c)) }
}
