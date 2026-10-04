// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"reflect"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/literal"
)

// TestModels steps each named model of the definition through its
// operations. Written with testing rather than with this library, because a
// model decides the verdict of a vector.
func TestModels(t *testing.T) {
	t.Parallel()

	t.Run("Models", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			model string
			state any
			op    history.Op
			want  []any
		}{
			{
				name:  "leaves the value of a write in the register",
				model: "register",
				op:    opOf("write", 1),
				want:  []any{1},
			},
			{
				name: "accepts a read of the register's value", model: "register", state: 1,
				op: known(opOf("read"), 1), want: []any{1},
			},
			{
				name:  "rejects a read of another value of the register",
				model: "register",
				state: 1,
				op:    known(opOf("read"), 2),
			},
			{
				name: "accepts a read of the register whose outcome is unknown", model: "register", state: 1,
				op: opOf("read"), want: []any{1},
			},
			{
				name: "stores to for a cas that outputs true while the value equals from", model: "cas-register",
				state: 1, op: known(opOf("cas", 1, 2), true), want: []any{2},
			},
			{
				name:  "rejects a cas that outputs true while the value differs from the expected one",
				model: "cas-register",
				state: 1,
				op:    known(opOf("cas", 3, 2), true),
			},
			{
				name: "rejects a cas that outputs false while the value equals from", model: "cas-register",
				state: 1, op: known(opOf("cas", 1, 2), false),
			},
			{
				name: "leaves the value for a cas that outputs false while the value differs", model: "cas-register",
				state: 1, op: known(opOf("cas", 3, 2), false), want: []any{1},
			},
			{
				name:  "takes effect for a cas whose outcome is unknown while the value equals from",
				model: "cas-register", state: 1, op: opOf("cas", 1, 2), want: []any{2},
			},
			{
				name:  "leaves the value for a cas whose outcome is unknown while the value differs",
				model: "cas-register", state: 1, op: opOf("cas", 3, 2), want: []any{1},
			},
			{
				name: "leaves the value of a write in the cas register", model: "cas-register", state: 1,
				op: opOf("write", 5), want: []any{5},
			},
			{
				name: "rejects a read of another value of the cas register", model: "cas-register", state: 1,
				op: known(opOf("read"), 2),
			},
			{
				name: "leaves the new value and then the old one for a lossy write", model: "lossy-register", state: 1,
				op: opOf("write", 2), want: []any{2, 1},
			},
			{
				name: "accepts a read of the lossy register's value", model: "lossy-register", state: 1,
				op: known(opOf("read"), 1), want: []any{1},
			},
			{
				name: "reads the empty value of a key that the map does not store", model: "key-value",
				state: pairs(), op: known(opOf("get", "k"), ""), want: []any{pairs()},
			},
			{
				name: "rejects a get of another value", model: "key-value", state: pairs("k", "a"),
				op: known(opOf("get", "k"), "b"),
			},
			{
				name: "accepts a get whose outcome is unknown", model: "key-value", state: pairs("k", "a"),
				op: opOf("get", "k"), want: []any{pairs("k", "a")},
			},
			{
				name: "stores a put of a new key last", model: "key-value", state: pairs("a", "1"),
				op: opOf("put", "b", "2"), want: []any{pairs("a", "1", "b", "2")},
			},
			{
				name:  "keeps the place of a key whose value changes",
				model: "key-value",
				state: pairs("a", "1", "b", "2"),
				op:    opOf("put", "a", "3"),
				want:  []any{pairs("a", "3", "b", "2")},
			},
			{
				name: "removes a key whose value becomes empty", model: "key-value", state: pairs("a", "1", "b", "2"),
				op: opOf("put", "a", ""), want: []any{pairs("b", "2")},
			},
			{
				name: "stores no empty value under a key that the map does not store", model: "key-value",
				state: pairs("a", "1"), op: opOf("put", "b", ""), want: []any{pairs("a", "1")},
			},
			{
				name: "appends to the value under a key", model: "key-value", state: pairs("a", "x"),
				op: opOf("append", "a", "y"), want: []any{pairs("a", "xy")},
			},
			{
				name: "adds an enqueued value at the tail", model: "queue", state: []any{1},
				op: opOf("enqueue", 2), want: []any{[]any{1, 2}},
			},
			{
				name: "removes and outputs the head for a dequeue", model: "queue", state: []any{1, 2},
				op: known(opOf("dequeue"), 1), want: []any{[]any{2}},
			},
			{
				name: "rejects a dequeue of another value than the head", model: "queue", state: []any{1, 2},
				op: known(opOf("dequeue"), 2),
			},
			{
				name: "removes the head for a dequeue whose outcome is unknown", model: "queue", state: []any{1, 2},
				op: opOf("dequeue"), want: []any{[]any{2}},
			},
			{
				name: "outputs null for a dequeue of the empty queue", model: "queue", state: []any{},
				op: known(opOf("dequeue"), nil), want: []any{[]any{}},
			},
			{
				name: "rejects a dequeue of a value from the empty queue", model: "queue", state: []any{},
				op: known(opOf("dequeue"), 1),
			},
			{
				name: "leaves the empty queue for a dequeue whose outcome is unknown", model: "queue", state: []any{},
				op: opOf("dequeue"), want: []any{[]any{}},
			},
			{
				name: "adds an absent value to the set last", model: "set", state: []any{1},
				op: opOf("add", 2), want: []any{[]any{1, 2}},
			},
			{
				name: "leaves the set for an add of a present value", model: "set", state: []any{1},
				op: opOf("add", 1), want: []any{[]any{1}},
			},
			{
				name: "accepts a contains that outputs whether the value is present", model: "set", state: []any{1},
				op: known(opOf("contains", 2), false), want: []any{[]any{1}},
			},
			{
				name: "rejects a contains that outputs another presence", model: "set", state: []any{1},
				op: known(opOf("contains", 1), false),
			},
			{
				name: "accepts a contains whose outcome is unknown", model: "set", state: []any{1},
				op: opOf("contains", 2), want: []any{[]any{1}},
			},
			{
				name: "removes a present value that a remove outputs as present", model: "set", state: []any{1, 2},
				op: known(opOf("remove", 1), true), want: []any{[]any{2}},
			},
			{
				name: "rejects a remove that outputs another presence", model: "set", state: []any{1, 2},
				op: known(opOf("remove", 3), true),
			},
			{
				name: "leaves the set for a remove of an absent value whose outcome is unknown", model: "set",
				state: []any{1}, op: opOf("remove", 2), want: []any{[]any{1}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				if got := conformance.Models[tt.model].Step(tt.state, tt.op); !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("the %s model steps %v from %v to %#v, want %#v", tt.model, tt.op, tt.state, got, tt.want)
				}
			})
		}

		undefined := []struct {
			model string
			state any
			want  string
		}{
			{model: "register", want: `conformance: the register model has no operation "drop"`},
			{model: "cas-register", want: `conformance: the cas-register model has no operation "drop"`},
			{model: "lossy-register", want: `conformance: the lossy-register model has no operation "drop"`},
			{model: "key-value", state: pairs(), want: `conformance: the key-value model has no operation "drop"`},
			{model: "queue", state: []any{}, want: `conformance: the queue model has no operation "drop"`},
			{model: "set", state: []any{}, want: `conformance: the set model has no operation "drop"`},
		}
		for _, tt := range undefined {
			t.Run("panics on an operation that the "+tt.model+" model does not define", func(t *testing.T) {
				t.Parallel()
				defer func() {
					if got := recover(); got != tt.want {
						t.Fatalf("the step panics with %v, want %q", got, tt.want)
					}
				}()
				conformance.Models[tt.model].Step(tt.state, opOf("drop", 1))
			})
		}

		t.Run("leaves the state of an enqueue unchanged", func(t *testing.T) {
			t.Parallel()
			state := make([]any, 1, 2)
			state[0] = 1
			queue := conformance.Models["queue"]
			first := queue.Step(state, opOf("enqueue", 2))
			queue.Step(state, opOf("enqueue", 3))
			if want := []any{[]any{1, 2}}; !reflect.DeepEqual(first, want) {
				t.Fatalf("a second enqueue from the state changes the first to %#v, want %#v", first, want)
			}
		})

		sameStates := []struct {
			name  string
			model string
			a, b  any
			want  bool
		}{
			{
				name: "reports two maps of one entry in two orders equal", model: "key-value",
				a: pairs("a", "1", "b", "2"), b: pairs("b", "2", "a", "1"), want: true,
			},
			{
				name:  "reports two maps of other values unequal",
				model: "key-value",
				a:     pairs("a", "1"),
				b:     pairs("a", "2"),
			},
			{
				name: "reports two sets of one value in two orders equal", model: "set",
				a: []any{1, 2}, b: []any{2, 1}, want: true,
			},
			{name: "reports two sets of other values unequal", model: "set", a: []any{1}, b: []any{2}},
			{name: "reports two sets of other sizes unequal", model: "set", a: []any{1}, b: []any{1, 2}},
		}
		for _, tt := range sameStates {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				m := conformance.Models[tt.model]
				if got := m.Equal(tt.a, tt.b); got != tt.want {
					t.Fatalf("Equal(%v, %v) of the %s model is %t, want %t", tt.a, tt.b, tt.model, got, tt.want)
				}
				if tt.want && m.Hash(tt.a) != m.Hash(tt.b) {
					t.Fatalf("the %s model hashes the equal states %v and %v apart", tt.model, tt.a, tt.b)
				}
			})
		}
	})
}

// opOf returns the op of operation with args, whose outcome is unknown.
func opOf(operation string, args ...any) history.Op {
	return history.Op{Operation: operation, Args: args}
}

// known returns op completed with output.
func known(op history.Op, output any) history.Op {
	op.Known, op.Output = true, output
	return op
}

// pairs returns the state of the key-value map of the keys and values of
// kv, in order.
func pairs(kv ...string) literal.Pairs {
	var p literal.Pairs
	for i := 0; i < len(kv); i += 2 {
		p.Entries = append(p.Entries, literal.Entry{Key: kv[i], Value: kv[i+1]})
	}
	return p
}
