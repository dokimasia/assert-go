// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// The values that the machines vectors and their subjects fix.
const (
	// lostWrite is the assertion of the failure of the store's settle
	// check, and durableContract its contract, which every such failure
	// states, so the failures of every lost write are one failure. Its
	// detail states the key under keyField and the lost value under
	// lostField.
	lostWrite       = "lost-write"
	durableContract = "the store keeps every acknowledged put"
	keyField        = "key"
	lostField       = "lost"
	// overflow is the count at which the counter of counter-overflows starts
	// at 0 again.
	overflow = 3
	// setupClients is the clients of a vector's setup that states none.
	setupClients = 2
	// drawsOption is the option of prop at which the path of a refused
	// trace's fault starts.
	drawsOption = "Draws"
)

// The members of a machines vector that the path of a fault names.
const (
	subjectMember  = "subject"
	setupMember    = "setup"
	strategyMember = "strategy"
)

// The draws of the machine subjects.
var (
	// capacities are the capacities of a queue.
	capacities = prop.Integer(1, 8)
	// queued are the values that a queue's put writes.
	queued = prop.Integer(0, 1000)
	// storeKeys are the keys of the store's put.
	storeKeys = prop.Integer(0, 3)
)

// machineSubject runs the steps of one machine subject in a case, with the
// options of a vector's setup.
type machineSubject func(c *prop.Case, o machineOptions)

// machineSubjects are the six machine subjects of the definition, by name.
var machineSubjects = map[string]machineSubject{
	"queue-loses-on-wrap":  func(c *prop.Case, o machineOptions) { queueSubject(c, o, true) },
	"correct-queue":        func(c *prop.Case, o machineOptions) { queueSubject(c, o, false) },
	"counter-overflows":    counterOverflows,
	"store-loses-on-crash": storeLosesOnCrash,
	"racy-counter":         func(c *prop.Case, o machineOptions) { sharedCounter(c, o, true) },
	"correct-counter":      func(c *prop.Case, o machineOptions) { sharedCounter(c, o, false) },
}

// machinesVector is a machines vector: a subject, the options of its steps,
// the settings of the run, a trace, and the detail of the run or the
// refusal of the trace.
type machinesVector struct {
	// Subject names the machine subject.
	Subject string `json:"subject"`
	// Setup states the options of the subject's steps.
	Setup machineSetup `json:"setup"`
	// Settings are the settings of the run.
	Settings behaviourSettings `json:"settings"`
	// Trace is the trace that the run follows first, as the text of Draws,
	// and nil for none.
	Trace json.RawMessage `json:"trace"`
	// Detail is the detail of the run.
	Detail runDetail `json:"detail"`
	// Error is the refusal of an entry of the trace, and nil for a run that
	// takes each.
	Error *traceRefusal `json:"error"`
}

// machineSetup is the setup of a machines vector. A nil field states the
// default of the definition.
type machineSetup struct {
	Mean       *int            `json:"mean"`
	Max        *int            `json:"max"`
	Swarm      *bool           `json:"swarm"`
	Clients    *int            `json:"clients"`
	Concurrent *int            `json:"concurrent"`
	Strategy   json.RawMessage `json:"strategy"`
}

// machineOptions are the options of a subject's steps: those that every
// subject takes, those of a concurrent section, and the strategy of the
// scheduler that a subject with a section makes in each case.
type machineOptions struct {
	// steps are the options that every subject takes.
	steps []stateful.Option
	// section are the options of a concurrent section.
	section []stateful.Option
	// strategy is the strategy of the scheduler.
	strategy stateful.Strategy
}

// traceRefusal is the refusal of a trace, as a machines vector states it:
// the entry, the name of its draw or its step, and the member of the entry
// that the run refused.
type traceRefusal struct {
	Entry  int    `json:"entry"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// faulting is a recorder that keeps the fault that ends a call, as the
// error that it is.
type faulting struct {
	*assert.Recorder
	// fault is the fault that ended the call, and nil for none.
	fault error
}

// ReportFault keeps err.
func (f *faulting) ReportFault(err error, _ bool) {
	f.fault = err
}

// checkMachines runs the subject of a machines vector under its setup and
// its settings, after its trace, and compares the run with the detail that
// the vector states, as checkBehaviour compares it, or the refusal of the
// trace with the vector's error. The run is the property of a behaviour
// vector, whose store entries are the vector's stored cases.
func checkMachines(raw json.RawMessage, dir string) error {
	var v machinesVector
	if err := decode(raw, &v); err != nil {
		return err
	}
	subject, ok := machineSubjects[v.Subject]
	if !ok {
		return fault.At(fault.New("%q names no machine subject", v.Subject), fault.Field(subjectMember))
	}
	o, err := v.Setup.options()
	if err != nil {
		return fault.At(err, fault.Field(setupMember))
	}
	opts, err := v.Settings.resolve(dir)
	if err != nil {
		return fault.At(err, fault.Field(settingsMember))
	}
	if v.Trace != nil {
		opts = append(opts, prop.Draws(string(v.Trace)))
	}
	seat := &faulting{Recorder: assert.NewRecorder()}
	prop.ForAll(seat, behaviourContract, func(c *prop.Case) { subject(c, o) }, opts...)
	if v.Error != nil || seat.fault != nil {
		return at(v.Error.compare(seat.fault), fault.Field(errorMember))
	}
	if records := seat.Failures(); len(records) > 0 {
		return at(v.Detail.compare(records[0].Detail), fault.Field(detailMember))
	}
	return at(v.Detail.comparePassed(callsOf(seat.Recorder)[0]), fault.Field(detailMember))
}

// options returns the options that s states, with the clients of the
// definition's setup for a setup that states none.
func (s machineSetup) options() (machineOptions, error) {
	var o machineOptions
	if s.Mean != nil {
		o.steps = append(o.steps, stateful.Mean(*s.Mean))
	}
	if s.Max != nil {
		o.steps = append(o.steps, stateful.Max(*s.Max))
	}
	if s.Swarm != nil {
		o.steps = append(o.steps, stateful.Swarm(*s.Swarm))
	}
	o.section = []stateful.Option{stateful.Clients(setupClients)}
	if s.Clients != nil {
		o.section[0] = stateful.Clients(*s.Clients)
	}
	if s.Concurrent != nil {
		o.section = append(o.section, stateful.Concurrent(*s.Concurrent))
	}
	var err error
	o.strategy, err = strategyOf(s.Strategy)
	return o, err
}

// strategyOf returns the strategy that a setup states: uniform when it
// states none or "uniform", and PCT of a depth that {"pct": depth} states.
func strategyOf(raw json.RawMessage) (stateful.Strategy, error) {
	var uniform string
	if raw == nil || json.Unmarshal(raw, &uniform) == nil && uniform == "uniform" {
		return stateful.Uniform(), nil
	}
	var pct struct {
		Depth *int `json:"pct"`
	}
	if json.Unmarshal(raw, &pct) != nil || pct.Depth == nil {
		return stateful.Strategy{}, fault.At(fault.New("%s is no strategy", raw), fault.Field(strategyMember))
	}
	return stateful.PCT(*pct.Depth), nil
}

// compare returns how the fault of a run differs from the refusal r, or nil
// when they match. The fault matches when its path is the member that r's
// reason names of the entry of Draws at r's index.
func (r *traceRefusal) compare(err error) error {
	if r == nil {
		return fault.New("the run refuses an entry, and the vector states none").Because(err)
	}
	f, ok := errors.AsType[*fault.Error](err)
	want := fault.Path{fault.Field(drawsOption), fault.Index(r.Entry), fault.Field(r.Reason)}
	if !ok || !slices.Equal(f.Path, want) {
		return fault.New("the run refuses no %s of entry %d, the %s of %q", r.Reason, r.Entry, r.Reason, r.Name).
			Because(err)
	}
	return nil
}

// ring is a bounded queue in a ring of slots. A ring that loses on wrap
// writes the value of a put at the last slot into the first slot.
type ring struct {
	// slots are the slots of the ring, one per value of its capacity.
	slots []int
	// head is the slot of the oldest value, tail the slot of the next put,
	// and size the number of values.
	head, tail, size int
	// losesOnWrap reports a ring that writes its last slot's value into the
	// first slot.
	losesOnWrap bool
}

// put adds v, and reports whether the ring had room for it.
func (r *ring) put(v int) bool {
	if r.size == len(r.slots) {
		return false
	}
	at := r.tail
	r.tail = (r.tail + 1) % len(r.slots)
	if r.losesOnWrap && r.tail == 0 {
		at = 0
	}
	r.slots[at] = v
	r.size++
	return true
}

// get removes and returns the oldest value, and nil for an empty ring.
func (r *ring) get() any {
	if r.size == 0 {
		return nil
	}
	v := r.slots[r.head]
	r.head = (r.head + 1) % len(r.slots)
	r.size--
	return v
}

// queueSubject runs the queue machine over a ring of a drawn capacity: put
// of a drawn value, and get, over the model of a bounded queue.
func queueSubject(c *prop.Case, o machineOptions, losesOnWrap bool) {
	capacity := c.Draw(capacities, "capacity")
	q := &ring{slots: make([]int, capacity), losesOnWrap: losesOnWrap}
	stateful.Steps(c, stateful.Machine[[]int]{
		Model: boundedQueue(capacity),
		Actions: []stateful.Action[[]int]{{
			Name:  "put",
			Input: func(c *prop.Case, _ []int) any { return c.Draw(queued, "v") },
			Run: func(c *prop.Case, client int, v any) {
				call := c.History().Invoke(client, "put", []any{v})
				call.OK(q.put(v.(int)))
			},
		}, {
			Name: "get",
			Run: func(c *prop.Case, client int, _ any) {
				call := c.History().Invoke(client, "get", nil)
				call.OK(q.get())
			},
		}},
	}, o.steps...)
}

// boundedQueue returns the model of a queue of at most capacity values: put
// appends its value while the queue has room, and get returns
// the oldest value and removes it, and leaves an empty queue empty. It
// models the calls of the queue subjects, each of which completes, whose put
// reports the room that the model counts, and whose get of an empty queue
// returns nil.
func boundedQueue(capacity int) history.Model[[]int] {
	return history.Model[[]int]{
		Init: func() []int { return []int{} },
		Step: func(state []int, op history.Op) [][]int {
			if op.Operation == "put" && len(state) == capacity {
				return [][]int{state}
			}
			if op.Operation == "put" {
				return [][]int{append(slices.Clip(state), op.Args[0].(int))}
			}
			if len(state) == 0 {
				return [][]int{state}
			}
			if op.Output != any(state[0]) {
				return nil
			}
			return [][]int{state[1:]}
		},
	}
}

// counterModel is the model of a counter whose increment returns the new
// count, and whose reset sets it to 0. It models the calls of the counter
// subjects, each of which completes.
var counterModel = history.Model[int]{
	Init: func() int { return 0 },
	Step: func(state int, op history.Op) []int {
		if op.Operation == "reset" {
			return []int{0}
		}
		if op.Output != any(state+1) {
			return nil
		}
		return []int{state + 1}
	},
}

// counterOverflows runs the counter machine over a counter whose increment
// to overflow sets the count to 0: increment and reset, over the model of a
// counter.
func counterOverflows(c *prop.Case, o machineOptions) {
	count := 0
	stateful.Steps(c, stateful.Machine[int]{
		Model: counterModel,
		Actions: []stateful.Action[int]{{
			Name: "increment",
			Run: func(c *prop.Case, client int, _ any) {
				call := c.History().Invoke(client, "increment", nil)
				count = (count + 1) % overflow
				call.OK(count)
			},
		}, {
			Name: "reset",
			Run: func(c *prop.Case, client int, _ any) {
				call := c.History().Invoke(client, "reset", nil)
				count = 0
				call.OK(nil)
			},
		}},
	}, o.steps...)
}

// storedPut is the input of the store's put: a drawn key, and a value that
// differs from the value of every earlier put of the case, the next of a
// count from 1.
type storedPut struct {
	key, value int
}

// storeLosesOnCrash runs the store machine without a model: put into a
// buffer, flush of the buffer into the durable values as a drain action
// enabled while the buffer is not empty, and crash, which empties the
// buffer. Its settle check fails with the identity lost-write for a key
// whose durable value is not the last that a put acknowledged.
func storeLosesOnCrash(c *prop.Case, o machineOptions) {
	durable, buffer, acknowledged := map[int]int{}, map[int]int{}, map[int]int{}
	written := 0
	stateful.Steps(c, stateful.Machine[struct{}]{
		Actions: []stateful.Action[struct{}]{{
			Name: "put",
			Input: func(c *prop.Case, _ struct{}) any {
				key := c.Draw(storeKeys, "key")
				written += 1
				return storedPut{key: key, value: written}
			},
			Run: func(c *prop.Case, client int, input any) {
				p := input.(storedPut)
				call := c.History().Invoke(client, "put", []any{p.key, p.value})
				buffer[p.key] = p.value
				call.OK(nil)
				acknowledged[p.key] = p.value
			},
		}, {
			Name:    "flush",
			Enabled: func(struct{}) bool { return len(buffer) > 0 },
			Drain:   true,
			Run: func(*prop.Case, int, any) {
				maps.Copy(durable, buffer)
				clear(buffer)
			},
		}, {
			Name: "crash",
			Run:  func(*prop.Case, int, any) { clear(buffer) },
		}},
		Settle: func(c *prop.Case, _ struct{}) {
			for _, key := range slices.Sorted(maps.Keys(acknowledged)) {
				if durable[key] != acknowledged[key] {
					c.Report(assert.Failure{
						Assertion: lostWrite, Contract: durableContract,
						Detail: map[string]any{keyField: key, lostField: acknowledged[key]},
					}, true)
				}
			}
		},
	}, o.steps...)
}

// sharedCounter runs the counter machine on clients, as tasks of a scheduler
// of the setup's strategy: increment, over the model of a counter. The racy
// increment yields between its read of the count and its write.
func sharedCounter(c *prop.Case, o machineOptions, racy bool) {
	s := stateful.NewScheduler(c, o.strategy)
	count := 0
	stateful.Steps(c, stateful.Machine[int]{
		Model: counterModel,
		Actions: []stateful.Action[int]{{
			Name: "increment",
			Run: func(c *prop.Case, client int, _ any) {
				call := c.History().Invoke(client, "increment", nil)
				read := count
				if racy {
					s.Yield()
				}
				count = read + 1
				call.OK(count)
			},
		}},
	}, slices.Concat(o.steps, o.section, []stateful.Option{stateful.Tasks(s)})...)
}
