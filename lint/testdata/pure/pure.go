// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pure

import (
	"bytes"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type status struct{ count int }

type store struct{}

func (store) Get(key string) string { return "" }

func (store) Snapshot() map[string]string { return nil }

func (store) Status() status { return status{} }

type reader struct{}

func (reader) Read(p []byte) (int, error) { return 0, nil }

func classify(err error) string { return "" }

func correct(t *testing.T, s store) {
	assert.Pure(t, s.Snapshot, func() { s.Get("key") }, "Get leaves the store as it was")
}

func observed(t *testing.T, s store) {
	before := s.Snapshot()
	s.Get("key")
	assert.Equal(t, s.Snapshot(), before, "Get leaves the store as it was") // want `pure: state the check with Pure of s\.Snapshot\(\), around s\.Get\("key"\)`
	first := s.Snapshot()
	s.Get("other")
	t.Log("read the other key")
	second := s.Snapshot()
	expect.Equal(t, first, second, "Get leaves the store as it was") // want `pure: state the check with Pure of s\.Snapshot\(\), around s\.Get\("other"\); t\.Log\("read the other key"\)`
}

func described(t *testing.T, s store) {
	first := s.Snapshot()
	description := "the store after a read of the key " + s.Get("key") + " and nothing else"
	second := s.Snapshot()
	assert.Equal(t, second, first, description) // want `pure: state the check with Pure of s\.Snapshot\(\), around description := "the store after a read of the key " \+ s\.Get…$`
}

func changed(t *testing.T, s store) {
	snapshot := s.Snapshot()
	snapshot["key"] = "value"
	s.Get("key")
	assert.Equal(t, s.Snapshot(), snapshot, "Get leaves the store as it was")
	state := s.Status()
	state.count = 2
	s.Get("key")
	assert.Equal(t, s.Status(), state, "Get leaves the status as it was")
}

func refused(t *testing.T, r reader) {
	_, first := r.Read(make([]byte, 64))
	assert.Equal(t, classify(first), "integrity", "the first read returns the verdict")
	n, again := r.Read(make([]byte, 64))
	expect.Equal(t, n, 0, "a reader after its verdict yields no byte")
	expect.Equal(t, again, first, "every read after the verdict returns the same error")
}

func built(t *testing.T, s store) {
	a := make([]int, 3)
	s.Get("key")
	b := make([]int, 3)
	assert.Equal(t, a, b, "the buffers start alike")
	x := []byte("key")
	s.Get("key")
	y := []byte("key")
	assert.Equal(t, x, y, "the keys start alike")
}

func logged(t *testing.T, s store) {
	before := s.Snapshot()
	t.Logf("the store holds %v", before)
	s.Get("key")
	assert.Equal(t, s.Snapshot(), before, "Get leaves the store as it was") // want `pure: state the check with Pure of s\.Snapshot\(\), around t\.Logf\("the store holds %v", before\); s\.Get\("key"\)$`
}

type tally struct{ n int }

func (c *tally) Total() int { return c.n }

func (c *tally) Put(n int) error {
	c.n = n
	return nil
}

func (c *tally) Repair() error {
	c.n = 0
	return nil
}

type config struct{ limit int }

func sum(cfg config) int { return cfg.limit }

func load(cfg *config) error { return nil }

func audit(c *tally) {}

func repaired(t *testing.T, c *tally, cfg config) {
	truth := c.Total()
	assert.NoError(t, c.Put(1), "the total drifts")
	err := c.Repair()
	assert.NoError(t, err, "the repair runs")
	assert.Equal(t, c.Total(), truth, "the repair restores the total") // want `pure: state the check with Pure of c\.Total\(\), around assert\.NoError\(t, c\.Put\(1\), "the total drifts"\); err := c\.Repair\(\)$`
	again := c.Total()
	assert.NoError(t, c.Put(2), "the total drifts")
	assert.NoError(t, c.Repair(), "the repair runs")
	expect.Equal(t, c.Total(), again, "the repair restores the total") // want `pure: state the check with Pure of c\.Total\(\), around assert\.NoError\(t, c\.Put\(2\), "the total drifts"\); assert\.NoError\(t, c\.Repair\(\), "the repair runs"\)$`
	before := c.Total()
	assert.MaxAllocs(t, func() { audit(c) }, 0, "an audit of the tally allocates nothing")
	assert.Equal(t, c.Total(), before, "an audit keeps the total") // want `pure: state the check with Pure of c\.Total\(\), around assert\.MaxAllocs\(…\)$`
	first := sum(cfg)
	assert.NoError(t, load(&cfg), "the configuration loads")
	assert.Equal(t, sum(cfg), first, "a load keeps the limit") // want `pure: state the check with Pure of sum\(cfg\), around assert\.NoError\(t, load\(&cfg\), "the configuration loads"\)$`
}

func encode(buf []byte) (int, error) { return 0, nil }

type counter struct{ n int }

func newCounter() counter { return counter{} }

func (c *counter) Inc() { c.n++ }

type items []int

func list() items { return nil }

func (l items) Zero() {
	if len(l) > 0 {
		l[0] = 0
	}
}

func written(t *testing.T) {
	buf := bytes.Repeat([]byte{0xAA}, 3)
	n, err := encode(buf)
	assert.NoError(t, err, "encode returns no error")
	assert.Equal(t, n, 0, "encode writes no byte")
	assert.Equal(t, buf, bytes.Repeat([]byte{0xAA}, 3), "encode writes nothing into a short buffer") // want `pure: state the check with Pure of a copy of buf, around n, err := encode\(buf\)$`
	kept := bytes.Repeat([]byte{0xAA}, 3)
	fresh := bytes.Repeat([]byte{0xAA}, 3)
	_, _ = encode(kept)
	expect.Equal(t, fresh, kept, "encode writes nothing into a short buffer") // want `pure: state the check with Pure of a copy of kept, around encode\(kept\)$`
	c := newCounter()
	c.Inc()
	assert.Equal(t, c, newCounter(), "a counter starts again") // want `pure: state the check with Pure of a copy of c, around c\.Inc\(\)$`
	l := list()
	l.Zero()
	assert.Equal(t, l, list(), "the first item is zero") // want `pure: state the check with Pure of a copy of l, around l\.Zero\(\)$`
	reread := bytes.Repeat([]byte{0xAA}, 3)
	other := bytes.Repeat([]byte{0xAA}, 3)
	_, _ = encode(other)
	_, _ = encode(reread)
	assert.Equal(t, reread, other, "both buffers stay as they were")
}

type frame struct {
	seq  int
	data []byte
}

func newFrame() frame { return frame{} }

func fill(f frame) {}

type window [2][]int

func newWindow() window { return window{} }

func slide(w window) {}

func show(c counter) {}

type hooks struct{ run func() }

func newHooks() hooks { return hooks{} }

func defaults() config { return config{} }

func count() int { return 0 }

func show2(n int) {}

func scrub(buf []byte) error { return nil }

func clean() error { return nil }

func forget(m map[string]string) {}

func describe(parts []any) error { return nil }

func passedForms(t *testing.T, s store) {
	scratch := bytes.Repeat([]byte{0xAA}, 3)
	_, _ = encode(scratch)
	assert.True(t, bytes.Equal(scratch, bytes.Repeat([]byte{0xAA}, 3)), "encode writes nothing") // want `pure: state the check with Pure of a copy of scratch, around encode\(scratch\)$`
	cfg := defaults()
	_ = load(&cfg)
	assert.Equal(t, cfg, defaults(), "load writes nothing into the defaults") // want `pure: state the check with Pure of a copy of cfg, around load\(&cfg\)$`
	n := count()
	show2(-n)
	assert.Equal(t, n, count(), "the count stays") // want `pure: state the check with Pure of count\(\), around show2\(-n\)$`
	sized := bytes.Repeat([]byte{0xAA}, 3)
	_ = len(sized)
	assert.Equal(t, sized, bytes.Repeat([]byte{0xAA}, 3), "a length reads the buffer") // want `pure: state the check with Pure of bytes\.Repeat\(\[\]byte\{0xAA\}, 3\), around len\(sized\)$`
	scrubbed := bytes.Repeat([]byte{0xAA}, 3)
	_ = []error{scrub(scrubbed), clean()}
	assert.Equal(t, scrubbed, bytes.Repeat([]byte{0xAA}, 3), "scrub writes nothing") // want `pure: state the check with Pure of a copy of scrubbed, around \[\]error\{scrub\(scrubbed\), clean\(\)\}$`
	after := s.Snapshot()
	s.Get("key")
	assert.Equal(t, s.Snapshot(), after, "Get leaves the store as it was") // want `pure: state the check with Pure of s\.Snapshot\(\), around s\.Get\("key"\)$`
	forget(after)
}

func checkedSteps(t *testing.T, c *tally, limit int) {
	first := c.Total()
	assert.NotEmpty(t, []error{c.Put(2), clean()}, "pair")
	assert.Equal(t, c.Total(), first, "the put keeps the total") // want `pure: state the check with Pure of c\.Total\(\), around assert\.NotEmpty\(t, \[\]error\{c\.Put\(2\), clean\(\)\}, "pair"\)$`
	second := c.Total()
	assert.NoError(t, describe([]any{c, limit}), "the tally describes itself")
	assert.Equal(t, c.Total(), second, "a description keeps the total") // want `pure: state the check with Pure of c\.Total\(\), around assert\.NoError\(…\)$`
}

func passedParts(t *testing.T) {
	f := newFrame()
	fill(f)
	assert.Equal(t, f, newFrame(), "fill writes no frame") // want `pure: state the check with Pure of a copy of f, around fill\(f\)$`
	w := newWindow()
	slide(w)
	assert.Equal(t, w, newWindow(), "slide writes no window") // want `pure: state the check with Pure of a copy of w, around slide\(w\)$`
	c := newCounter()
	show(c)
	assert.Equal(t, c, newCounter(), "show writes no counter") // want `pure: state the check with Pure of newCounter\(\), around show\(c\)$`
	h := newHooks()
	h.run()
	assert.Equal(t, h, newHooks(), "the hook leaves the hooks") // want `pure: state the check with Pure of newHooks\(\), around h\.run\(\)$`
}
