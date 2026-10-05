// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful

import "fmt"

// The settings of a run of steps when no option states them, which the
// definition fixes.
const (
	// defaultMean is the mean number of sequential steps.
	defaultMean = 30
	// defaultMax is the most sequential steps, and the most drain steps.
	defaultMax = 100
	// defaultClients is the clients of a machine without a concurrent
	// section.
	defaultClients = 1
	// defaultConcurrent is the most steps of a concurrent section.
	defaultConcurrent = 16
	// defaultRepeat is the runs of a case whose concurrent section runs on
	// threads.
	defaultRepeat = 4
)

// Option configures one run of [Steps]. Each option states one setting, and
// a later option overrides an earlier one. The zero Option changes nothing.
//
// # Allocation contract
//
// An option allocates nothing when its call does not keep it, and the
// closure of its setting when it does. Applying the options of a run
// allocates nothing.
type Option struct {
	// set returns the settings of a run with the option's setting applied.
	set func(config) config
}

// Mean sets the mean number of sequential steps of a case, 30 by default.
// It panics for n below 0.
func Mean(n int) Option {
	if n < 0 {
		panic(fmt.Sprintf("stateful: Mean(%d) is below 0", n))
	}
	return Option{set: func(c config) config {
		c.mean = n
		return c
	}}
}

// Max sets the most sequential steps of a case, and the most steps of its
// drain, 100 by default. It panics for n below 0.
func Max(n int) Option {
	if n < 0 {
		panic(fmt.Sprintf("stateful: Max(%d) is below 0", n))
	}
	return Option{set: func(c config) config {
		c.max = n
		return c
	}}
}

// Swarm sets whether a case first chooses the actions that it keeps, on by
// default. Off, a case keeps every action.
func Swarm(on bool) Option {
	return Option{set: func(c config) config {
		c.swarm = on
		return c
	}}
}

// Clients sets the clients of the concurrent section besides client 0, 1 by
// default. A section runs for n of 2 or more, on the clients 0 to n. It
// panics for n below 1.
func Clients(n int) Option {
	if n < 1 {
		panic(fmt.Sprintf("stateful: Clients(%d) is below 1", n))
	}
	return Option{set: func(c config) config {
		c.clients = n
		return c
	}}
}

// Concurrent sets the most steps of a concurrent section, 16 by default. It
// panics for n below 0.
func Concurrent(n int) Option {
	if n < 0 {
		panic(fmt.Sprintf("stateful: Concurrent(%d) is below 0", n))
	}
	return Option{set: func(c config) config {
		c.concurrent = n
		return c
	}}
}

// Tasks makes the clients of a concurrent section run as tasks of s, which
// replay, instead of on threads. It panics for a nil s.
func Tasks(s *Scheduler) Option {
	if s == nil {
		panic("stateful: Tasks(nil) states no scheduler")
	}
	return Option{set: func(c config) config {
		c.scheduler = s
		return c
	}}
}

// Repeat sets the runs of a case whose concurrent section runs on threads,
// 4 by default. It panics for n below 1.
func Repeat(n int) Option {
	if n < 1 {
		panic(fmt.Sprintf("stateful: Repeat(%d) is below 1", n))
	}
	return Option{set: func(c config) config {
		c.repeat = n
		return c
	}}
}

// config is what the options of one run state.
type config struct {
	// mean is the mean number of sequential steps.
	mean int
	// max is the most sequential steps, and the most drain steps.
	max int
	// swarm reports whether a case chooses the actions that it keeps.
	swarm bool
	// clients is the clients of the concurrent section besides client 0.
	clients int
	// concurrent is the most steps of a concurrent section.
	concurrent int
	// scheduler runs the clients of a section as tasks, and is nil for
	// threads.
	scheduler *Scheduler
	// repeat is the runs of a case whose section runs on threads.
	repeat int
}

// configure returns the defaults with each option of opts applied in order.
func configure(opts []Option) config {
	c := config{
		mean:       defaultMean,
		max:        defaultMax,
		swarm:      true,
		clients:    defaultClients,
		concurrent: defaultConcurrent,
		repeat:     defaultRepeat,
	}
	for _, o := range opts {
		if o.set != nil {
			c = o.set(c)
		}
	}
	return c
}
