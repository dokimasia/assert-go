// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package record_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/record"
)

// TestCall checks the line of JSON that a record of each kind of call is
// encoded as.
func TestCall(t *testing.T) {
	t.Parallel()

	t.Run("Call", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give record.Call
			want string
		}{
			{
				name: "writes a passing call with the base name of its file",
				give: record.Call{
					Assertion: "equal", Contract: "the count is right", Verdict: record.Pass, Aborting: true,
					Where: record.Where{File: "/src/shop/store_test.go", Line: 42},
				},
				want: `{"definition":"7.1.0","seq":1,"assertion":"equal","contract":"the count is right",` +
					`"verdict":"pass","aborting":true,"where":{"file":"store_test.go","line":42}}`,
			},
			{
				name: "writes a failing call on the recording surface with its detail",
				give: record.Call{
					Assertion: "equal", Contract: "the name is kept", Verdict: record.Fail,
					Detail: json.RawMessage(`{"got":{"type":"string","value":"ada"}}`),
				},
				want: `{"definition":"7.1.0","seq":1,"assertion":"equal","contract":"the name is kept",` +
					`"verdict":"fail","aborting":false,"detail":{"got":{"type":"string","value":"ada"}}}`,
			},
			{
				name: "writes the empty detail of a failing call of an assertion without fields",
				give: record.Call{
					Assertion: "true",
					Contract:  "the flag is set",
					Verdict:   record.Fail,
					Detail:    json.RawMessage(`{}`),
				},
				want: `{"definition":"7.1.0","seq":1,"assertion":"true","contract":"the flag is set",` +
					`"verdict":"fail","aborting":false,"detail":{}}`,
			},
			{
				name: "writes the text of the fault of an error",
				give: record.Call{
					Assertion: "prop-for-all",
					Contract:  "c",
					Verdict:   record.Error,
					Aborting:  true,
					Error:     "prop.ForAll: no",
				},
				want: `{"definition":"7.1.0","seq":1,"assertion":"prop-for-all","contract":"c",` +
					`"verdict":"error","aborting":true,"error":"prop.ForAll: no"}`,
			},
			{
				name: "writes the characters of HTML unescaped",
				give: call("equal", "a <b> & c"),
				want: `{"definition":"7.1.0","seq":1,"assertion":"equal","contract":"a <b> & c",` +
					`"verdict":"pass","aborting":true}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				k := newKeeping()
				record.Add(&k.calls, tt.give)
				assert.Equal(t, record.Lines(&k.calls), []string{tt.want}, "the record")
			})
		}
	})

	t.Run("Detail", func(t *testing.T) {
		t.Parallel()

		t.Run("panics when the detail is no JSON", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			bad := record.Call{Assertion: "equal", Verdict: record.Fail, Detail: json.RawMessage(`{`)}
			got := assert.Panics(t, func() { record.Add(&k.calls, bad) }, "the detail is no JSON")
			assert.HasPrefix(t, got, "record: the detail of a call of equal is no JSON: ", "the message")
		})
	})
}
