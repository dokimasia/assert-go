// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package machine

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

type queue struct{ values []int }

func (q *queue) put(v int) { q.values = append(q.values, v) }

func (q *queue) get() int {
	v := q.values[0]
	q.values = q.values[1:]
	return v
}

func next(s []int, op history.Operation) [][]int {
	return [][]int{s}
}

func correct(t *testing.T) {
	prop.ForAll(t, "the queue keeps its values in order", func(c *prop.Case) {
		q := &queue{}
		stateful.Steps(c, stateful.Machine[[]int]{
			Spec: history.Spec[[]int]{Initial: func() []int { return nil }, Next: next},
			Actions: []stateful.Action[[]int]{{
				Name: "put",
				Run: func(c *prop.Case, client int, input any) {
					call := c.History().Invoke(client, "put", []any{input})
					q.put(1)
					call.OK(nil)
				},
			}},
		})
	})
}

func handWritten(t *testing.T) {
	prop.ForAll(t, "the queue keeps its values in order", func(c *prop.Case) { // want `machine: state the check with stateful.Steps`
		q := &queue{}
		var model []int
		for range c.Draw(prop.Integer(1, 20), "steps") {
			switch c.Draw(prop.SampledFrom("put", "get"), "action") {
			case "put":
				q.put(len(model))
				model = append(model, len(model))
			case "get":
				if len(model) > 0 {
					assert.Equal(c, q.get(), model[0], "the queue returns its oldest value")
					model = model[1:]
				}
			}
		}
	})
	prop.ForAll(t, "one action", func(c *prop.Case) {
		switch c.Draw(prop.SampledFrom("put", "get"), "action") {
		case "put":
		}
	})
	prop.ForAll(t, "a fixed plan", func(c *prop.Case) {
		for _, action := range []string{"put", "get"} {
			switch action {
			case "put":
			}
		}
		for range 3 {
			switch {
			case true:
			}
		}
	})
	for range 3 {
		prop.ForAll(t, "an action per case", func(c *prop.Case) {
			switch c.Draw(prop.SampledFrom("put", "get"), "action") {
			case "put":
			}
		})
	}
}
