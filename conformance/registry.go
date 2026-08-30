// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import "go.dokimi.dev/assert"

// Invoker drives one assertion against a case's decoded arguments.
//
// Go cannot call a function by name at run time, so the corpus
// dispatches through this table. The completeness gate requires an
// entry for every assertion the corpus covers, which is what stops the
// table falling behind the definition.
type Invoker func(r *assert.Recorder, args []any, msg string)

// Registry maps an assertion to the call that drives it.
//
// Every entry drives the aborting surface. The recording surface calls
// the same comparison with a different mode, and the shared suites
// already hold the two to the same cases, so driving both here would
// establish nothing the suites do not.
//
// An assertion whose arguments are not expressible as typed literals
// has no entry: a callable, a context, a predicate or a golden file
// cannot cross a language boundary as data. Those are covered by the
// completeness gate for presence, and by this language's own tests for
// behaviour.
var Registry = map[ID]Invoker{
	"equal": func(r *assert.Recorder, args []any, msg string) {
		assert.Equal(r, args[0], args[1], msg)
	},
	"not-equal": func(r *assert.Recorder, args []any, msg string) {
		assert.NotEqual(r, args[0], args[1], msg)
	},
	"true": func(r *assert.Recorder, args []any, msg string) {
		assert.True(r, args[0].(bool), msg)
	},
	"false": func(r *assert.Recorder, args []any, msg string) {
		assert.False(r, args[0].(bool), msg)
	},
	"nil": func(r *assert.Recorder, args []any, msg string) {
		assert.Nil(r, args[0], msg)
	},
	"not-nil": func(r *assert.Recorder, args []any, msg string) {
		assert.NotNil(r, args[0], msg)
	},
	"length": func(r *assert.Recorder, args []any, msg string) {
		assert.Length(r, args[0], args[1].(int), msg)
	},
	"empty": func(r *assert.Recorder, args []any, msg string) {
		assert.Empty(r, args[0], msg)
	},
	"not-empty": func(r *assert.Recorder, args []any, msg string) {
		assert.NotEmpty(r, args[0], msg)
	},
	"contains": func(r *assert.Recorder, args []any, msg string) {
		assert.Contains(r, args[0], args[1], msg)
	},
	"not-contains": func(r *assert.Recorder, args []any, msg string) {
		assert.NotContains(r, args[0], args[1], msg)
	},
	"contains-in-order": func(r *assert.Recorder, args []any, msg string) {
		assert.ContainsInOrder(r, args[0], args[1].([]string), msg)
	},
	"has-prefix": func(r *assert.Recorder, args []any, msg string) {
		assert.HasPrefix(r, args[0], args[1].(string), msg)
	},
	"has-suffix": func(r *assert.Recorder, args []any, msg string) {
		assert.HasSuffix(r, args[0], args[1].(string), msg)
	},
	"matches": func(r *assert.Recorder, args []any, msg string) {
		assert.Matches(r, args[0], args[1].(string), msg)
	},
	"close-to": func(r *assert.Recorder, args []any, msg string) {
		assert.CloseTo(r, args[0], args[1].(float64), args[2].(float64), msg)
	},
	"in-range": func(r *assert.Recorder, args []any, msg string) {
		assert.InRange(r, args[0], args[1].(float64), args[2].(float64), msg)
	},
}
