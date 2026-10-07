// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package checks

import (
	"log"
	"testing"
	"time"

	"example.test/lint/checks/fake"
	a "go.dokimi.dev/assert"
	. "go.dokimi.dev/assert/expect"
)

type logger struct{}

func (logger) Fatal(args ...any) {}

func calls(t *testing.T, p *int, xs []int) {
	a.True(t, p == nil, "the pointer is nil")                    // want `nil: state the check with Nil`
	True(t, p == nil, "the pointer is nil")                      // want `nil: state the check with Nil`
	a.Equal[int](t, len(xs), 3, "the store returns three items") // want `length: state the check with Length`
	fake.True(t, p == nil, "the pointer is nil")
	check := a.True
	check(t, p == nil, "the pointer is nil")
	a.That(t, p == nil).Equal(true, "the pointer is nil")
	_ = a.NewRecorder()
	_ = a.EquateEmpty()
	_ = a.NewControlled(time.Unix(0, 0))
	t.Run("nested", func(t *testing.T) { t.Log(string(rune(len("x")))) })
}

func ifChecks(t *testing.T, ok bool, ch chan int, l logger) {
	if !ok { // want `condition: state the check with True`
		t.Error("the flag is not set")
	}
	if !ok { // want `condition: state the check with True`
		t.Errorf("the flag is %v", ok)
	}
	if !ok { // want `condition: state the check with True`
		t.Fail()
	}
	if ok { // want `condition: state the check with False`
		t.FailNow()
	}
	if !ok {
		t.Fatal("the flag is not set")
	} else {
		t.Log("the flag is set")
	}
	if !ok {
		t.Log("the flag is not set")
		t.FailNow()
	}
	if !ok {
		return
	}
	if !ok {
		<-ch
	}
	if !ok {
		panic("the flag is not set")
	}
	if !ok {
		t.Log("the flag is not set")
	}
	if !ok {
		log.Fatal("the flag is not set")
	}
	if !ok {
		l.Fatal("the flag is not set")
	}
	if ok {
		t.Log("the flag is set")
	} else if !ok { // want `condition: state the check with True`
		t.Fatal("the flag is not set")
	}
}
