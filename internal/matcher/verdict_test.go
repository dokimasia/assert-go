// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
	"go.dokimi.dev/assert/internal/record"
)

// textSeat is a seat without Report, which receives the writer's text of
// every failure.
type textSeat struct {
	mu     sync.Mutex
	fatals []string
	errs   []string
}

// Helper marks nothing: the seat states no location.
func (*textSeat) Helper() {}

// Fatalf keeps the message of an aborting failure, and returns.
func (s *textSeat) Fatalf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fatals = append(s.fatals, fmt.Sprintf(format, args...))
}

// Errorf keeps the message of a recording failure.
func (s *textSeat) Errorf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs = append(s.errs, fmt.Sprintf(format, args...))
}

// unencodable is the detail of a run whose JSON cannot be encoded.
type unencodable struct{}

// MarshalJSON returns an error.
func (unencodable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("matcher_test: no JSON")
}

// runSeat is the seat of one run of a body, whose calls go to the call
// that ran the body.
type runSeat struct {
	matchertest.Seat
	calls
}

// TestVerdict checks the three verdicts of a call: the call record that
// each writes, and what each reports to the seat.
func TestVerdict(t *testing.T) {
	t.Parallel()

	t.Run("Pass", func(t *testing.T) {
		t.Parallel()

		t.Run("reports nothing to a seat whose calls are not recorded", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.Pass(seat, matcher.Fatal, "true", "the claim is true")
			if seat.Failed() || len(seat.Records()) != 0 {
				t.Fatalf("reported %q and %v, want nothing", seat.First(), seat.Records())
			}
		})
		t.Run("marks no frame as a helper of the seat", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.Pass(seat, matcher.Fatal, "true", "the claim is true")
			matcher.Begin(seat).Pass(matcher.Fatal, "eventually", "it converges")
			if got := seat.HelperCalls(); got != 0 {
				t.Fatalf("marked %d frames, want none for a call that sends the seat nothing", got)
			}
		})
		t.Run("writes the record of the call on the aborting surface", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			_, _, line, _ := runtime.Caller(0)
			matcher.Pass(seat, matcher.Fatal, "true", "the claim is true")
			lines := seat.lines(t)
			if len(lines) != 1 {
				t.Fatalf("wrote %d records, want 1", len(lines))
			}
			got := lines[0]
			if got["assertion"] != "true" || got["contract"] != "the claim is true" || got["verdict"] != "pass" ||
				got["aborting"] != true || got["seq"] != 1.0 {
				t.Fatalf("wrote %v, want a passing call of true on the aborting surface", got)
			}
			where := got["where"].(map[string]any)
			if where["file"] != "verdict_test.go" || where["line"] != float64(line+1) {
				t.Fatalf("the record states the site %v, want verdict_test.go:%d", where, line+1)
			}
			if _, ok := got["detail"]; ok {
				t.Fatalf("the record of a pass states the detail %v", got["detail"])
			}
		})
		t.Run("writes a call of the recording surface as not aborting", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Pass(seat, matcher.Soft, "true", "the claim is true")
			if got := seat.lines(t)[0]["aborting"]; got != false {
				t.Fatalf("the record states aborting %v, want false", got)
			}
		})
	})

	t.Run("Fail", func(t *testing.T) {
		t.Parallel()

		t.Run("reports a record of the aborting mode to a Reporter", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.Fail(seat, matcher.Fatal, "equal", "the values match", map[string]any{"want": 2, "got": 1})
			if records := seat.Records(); len(records) != 1 || records[0].Assertion != "equal" ||
				records[0].Contract != "the values match" {
				t.Fatalf("reported %v, want one record of equal", records)
			}
			if fatals, errs := len(seat.Fatals()), len(seat.Errs()); fatals != 1 || errs != 0 {
				t.Fatalf("reported %d through Fatalf and %d through Errorf, want 1 and 0", fatals, errs)
			}
		})
		t.Run("sends the writer's text through Fatalf to a seat without Report", func(t *testing.T) {
			t.Parallel()

			seat := &textSeat{}
			f := matcher.Failure{
				Assertion: "length", Contract: "every item comes back", Detail: map[string]any{"want": 3, "got": 2},
			}
			matcher.Fail(seat, matcher.Fatal, f.Assertion, f.Contract, f.Detail)
			if want := []string{matcher.Render(f)}; fmt.Sprint(seat.fatals) != fmt.Sprint(want) || len(seat.errs) != 0 {
				t.Fatalf("Fatalf received %q and Errorf %q, want %q through Fatalf", seat.fatals, seat.errs, want)
			}
		})
		t.Run("sends the writer's text through Errorf under Soft", func(t *testing.T) {
			t.Parallel()

			seat := &textSeat{}
			matcher.Fail(seat, matcher.Soft, "true", "the flag is set", nil)
			if len(seat.fatals) != 0 || len(seat.errs) != 1 || seat.errs[0] != "the flag is set" {
				t.Fatalf("Fatalf received %q and Errorf %q, want the contract through Errorf", seat.fatals, seat.errs)
			}
		})
		t.Run("names the line of the test that called the assertion", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			_, file, line, _ := runtime.Caller(0)
			matcher.Equal(seat, matcher.Fatal, 1, 2, "the values match")
			records := seat.Records()
			if len(records) != 1 {
				t.Fatalf("reported %d records, want 1", len(records))
			}
			if want := (matcher.Where{File: file, Line: line + 1}); records[0].Where != want {
				t.Fatalf("the record names %+v, want %+v", records[0].Where, want)
			}
		})
		t.Run("writes the detail of the record as typed literals", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Equal(seat, matcher.Soft, 1, 2, "the values match")
			got := seat.lines(t)[0]
			want := map[string]any{
				"want": map[string]any{"type": "int", "value": 2.0},
				"got":  map[string]any{"type": "int", "value": 1.0},
			}
			if got["verdict"] != "fail" || fmt.Sprint(got["detail"]) != fmt.Sprint(want) {
				t.Fatalf("wrote %v, want a failing call with the detail %v", got, want)
			}
		})
		t.Run("writes an absent list apart from an empty one", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Equal(seat, matcher.Soft, []int(nil), []int{}, "the list is empty")
			got := seat.lines(t)[0]
			want := map[string]any{
				"want": map[string]any{"type": "list", "items": []any{}},
				"got":  map[string]any{"type": "list", "of": "int", "value": nil},
			}
			if got["verdict"] != "fail" || fmt.Sprint(got["detail"]) != fmt.Sprint(want) {
				t.Fatalf("wrote %v, want a failing call with the detail %v", got, want)
			}
		})
		t.Run("writes an empty detail for an assertion that declares no field", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.True(seat, matcher.Soft, false, "the flag is set")
			if got := seat.lines(t)[0]["detail"]; fmt.Sprint(got) != "map[]" {
				t.Fatalf("wrote the detail %v, want an empty object", got)
			}
		})
		t.Run("writes a failure of the recording surface as not aborting", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Fail(seat, matcher.Soft, "true", "the flag is set", nil)
			if got := seat.lines(t)[0]["aborting"]; got != false {
				t.Fatalf("the record states aborting %v, want false", got)
			}
		})
	})

	t.Run("Fault", func(t *testing.T) {
		t.Parallel()

		t.Run("sends the writer's text of the fault through Fatalf under Soft", func(t *testing.T) {
			t.Parallel()

			seat := &textSeat{}
			err := fault.In("prop.ForAll", fault.New("the seed is no number"))
			matcher.Fault(seat, matcher.Soft, "prop-for-all", "the claim is true", err)
			want := []string{matcher.RenderFault(err)}
			if fmt.Sprint(seat.fatals) != fmt.Sprint(want) || len(seat.errs) != 0 {
				t.Fatalf("Fatalf received %q and Errorf %q, want %q through Fatalf", seat.fatals, seat.errs, want)
			}
		})
		t.Run("passes the fault to a FaultReporter as a fault that ends the call", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			err := fault.In("prop.ForAll", fault.New("the seed is no number"))
			matcher.Fault(seat, matcher.Soft, "prop-for-all", "the claim is true", err)
			if faults := seat.Faults(); len(faults) != 1 || !errors.Is(faults[0], err) {
				t.Fatalf("received the faults %v, want %v", faults, err)
			}
			if fatals := seat.Fatals(); len(fatals) != 1 || len(seat.Records()) != 0 {
				t.Fatalf("Fatalf received %q and Report %v, want the one text that ReportFault sends", fatals,
					seat.Records())
			}
		})
		t.Run("writes the record of an error with the fault's text", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			err := fault.New("the seed is no number")
			matcher.Fault(seat, matcher.Fatal, "prop-for-all", "the claim is true", err)
			got := seat.lines(t)[0]
			if want := matcher.RenderFault(err); got["verdict"] != "error" || got["error"] != want {
				t.Fatalf("wrote %v, want an error with the writer's text %q", got, want)
			}
		})
		t.Run("writes an error of the aborting surface as aborting", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Fault(seat, matcher.Fatal, "prop-for-all", "the claim is true", fault.New("the seed is no number"))
			if got := seat.lines(t)[0]["aborting"]; got != true {
				t.Fatalf("the record states aborting %v, want true", got)
			}
		})
	})

	t.Run("Begin", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a call without a slot for a seat whose calls are not recorded", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			run := matcher.Begin(seat)
			if run.Slot() != nil {
				t.Fatalf("Slot() = %v, want nil", run.Slot())
			}
			run.Pass(matcher.Fatal, "eventually", "it converges")
			if seat.Failed() {
				t.Fatalf("reported %q for a pass", seat.First())
			}
		})
		t.Run("numbers the call before the calls of its body", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			run := matcher.Begin(seat)
			body := &runSeat{}
			record.Run(&body.calls, run.Slot(), nil)
			matcher.True(body, matcher.Fatal, true, "in the body")
			run.Slot().Take(&body.calls, record.NoPhase)
			run.Pass(matcher.Fatal, "eventually", "it converges")
			lines := seat.lines(t)
			if len(lines) != 2 {
				t.Fatalf("wrote %d records, want 2", len(lines))
			}
			if lines[0]["assertion"] != "eventually" || lines[0]["seq"] != 1.0 {
				t.Fatalf("wrote %v first, want the call that ran the body as 1", lines[0])
			}
			if lines[1]["seq"] != 2.0 || lines[1]["parent"] != 1.0 || lines[1]["run"] != 1.0 {
				t.Fatalf("wrote %v second, want the body's call as 2 under 1 in run 1", lines[1])
			}
		})
		t.Run("reports the failure of a call that ran a body", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Begin(seat).Fail(matcher.Fatal, "eventually", "it converges", map[string]any{"attempts": 1})
			if records := seat.Records(); len(records) != 1 || records[0].Assertion != "eventually" {
				t.Fatalf("reported %v, want the failure of eventually", records)
			}
			if got := seat.lines(t)[0]["verdict"]; got != "fail" {
				t.Fatalf("wrote the verdict %v, want fail", got)
			}
		})
		t.Run("reports the fault of a call that ran a body", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			err := fault.New("the body cannot run")
			matcher.Begin(seat).Fault(matcher.Fatal, "rejects", "the check fails", err)
			if faults := seat.Faults(); len(faults) != 1 || !errors.Is(faults[0], err) || len(seat.Fatals()) != 1 {
				t.Fatalf("received the faults %v and the texts %q, want %v as a fault that ends the call", faults,
					seat.Fatals(), err)
			}
			if got := seat.lines(t)[0]["verdict"]; got != "error" {
				t.Fatalf("wrote the verdict %v, want error", got)
			}
		})
	})

	t.Run("PassRun", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the detail of the run into the record of a pass at the stated site", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Begin(seat).PassRun(matcher.Fatal, "prop-for-all", "the claim is true",
				matcher.Where{File: "/src/codec_test.go", Line: 18}, json.RawMessage(`{"outcome":"passed"}`))
			got := seat.lines(t)[0]
			if got["verdict"] != "pass" || fmt.Sprint(got["detail"]) != "map[outcome:passed]" {
				t.Fatalf("wrote %v, want a pass that states the detail of the run", got)
			}
			if where := got["where"].(map[string]any); where["file"] != "codec_test.go" || where["line"] != 18.0 {
				t.Fatalf("the record states the site %v, want codec_test.go:18", where)
			}
			if seat.Failed() {
				t.Fatalf("reported %q for a pass", seat.First())
			}
		})
		t.Run("writes the opaque literal of the error for a detail that cannot be encoded", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Begin(seat).
				PassRun(matcher.Fatal, "prop-for-all", "the claim is true", matcher.Where{}, unencodable{})
			if got := seat.lines(t)[0]["detail"]; fmt.Sprint(got) != "map[text:matcher_test: no JSON type:opaque]" {
				t.Fatalf("wrote the detail %v, want the opaque literal of the error", got)
			}
		})
		t.Run("writes the pass of a run on the aborting surface as aborting", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			matcher.Begin(seat).PassRun(matcher.Fatal, "prop-for-all", "the claim is true", matcher.Where{},
				json.RawMessage(`{"outcome":"passed"}`))
			if got := seat.lines(t)[0]["aborting"]; got != true {
				t.Fatalf("the record states aborting %v, want true", got)
			}
		})
	})

	t.Run("FailRun", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the detail of the run into the record of a failure and reports the failure", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			f := matcher.Failure{
				Assertion: "prop-for-all", Contract: "the claim is true", Detail: map[string]any{"outcome": "flaky"},
				Where: matcher.Where{File: "/src/codec_test.go", Line: 18},
			}
			matcher.Begin(seat).FailRun(matcher.Fatal, f, json.RawMessage(`{"outcome":"flaky"}`))
			if records := seat.Records(); len(records) != 1 || records[0].Assertion != "prop-for-all" {
				t.Fatalf("reported %v, want the failure of prop-for-all", records)
			}
			got := seat.lines(t)[0]
			if got["verdict"] != "fail" || fmt.Sprint(got["detail"]) != "map[outcome:flaky]" {
				t.Fatalf("wrote %v, want a failure that states the detail of the run as it is", got)
			}
		})
		t.Run("writes the failure of a run on the recording surface as not aborting", func(t *testing.T) {
			t.Parallel()

			seat := newKeepingSeat()
			f := matcher.Failure{Assertion: "prop-for-all", Contract: "the claim is true"}
			matcher.Begin(seat).FailRun(matcher.Soft, f, json.RawMessage(`{"outcome":"falsified"}`))
			if got := seat.lines(t)[0]["aborting"]; got != false {
				t.Fatalf("the record states aborting %v, want false", got)
			}
		})
	})

	t.Run("switch", func(t *testing.T) {
		t.Parallel()

		t.Run(
			"reports the switch's fault through Fatalf for every call while the switch states another value",
			func(t *testing.T) {
				t.Parallel()
				if !childtest.InChild(t) {
					runChild(t, record.Variable+"=yes")
					return
				}
				_, err := record.On()
				seat := &matchertest.Seat{}
				matcher.True(seat, matcher.Fatal, true, "a pass")
				matcher.True(seat, matcher.Soft, false, "a failure")
				matcher.Fault(seat, matcher.Soft, "true", "a fault", fault.New("the call cannot run"))
				matcher.Begin(seat).
					PassRun(matcher.Fatal, "prop-for-all", "a run", matcher.Where{}, json.RawMessage(`{}`))
				matcher.Begin(seat).
					FailRun(matcher.Fatal, matcher.Failure{Assertion: "prop-for-all"}, json.RawMessage(`{}`))
				faults := seat.Faults()
				if err == nil || len(faults) != 5 {
					t.Fatalf("received the faults %v, want the switch's fault %v five times", faults, err)
				}
				for _, got := range faults {
					if !errors.Is(got, err) {
						t.Fatalf("received the fault %v, want the switch's fault %v", got, err)
					}
				}
				if len(seat.Records()) != 0 || len(seat.Errs()) != 0 {
					t.Fatalf("reported %v and %q, want no verdict", seat.Records(), seat.Errs())
				}
			},
		)
		t.Run("reports the switch's fault of a pass at the line of the call", func(t *testing.T) {
			t.Parallel()
			_, _, line, _ := runtime.Caller(0)
			if childtest.InChild(t) {
				matcher.Pass(t, matcher.Fatal, "true", "the child's call")
				return
			}
			out, err := childtest.Run(t, t.Name(), record.Variable+"=yes")
			want := fmt.Sprintf("verdict_test.go:%d: record.On: ", line+2)
			if err == nil || !strings.Contains(out, want) {
				t.Fatalf("the child exits with %v and writes %q, want a failure that names %q", err, out, want)
			}
		})
		t.Run(
			"writes the record of a call on a test's seat through its Attr while the switch is on",
			func(t *testing.T) {
				t.Parallel()
				if !childtest.InChild(t) {
					out := runChild(t, record.Variable+"=1")
					want := "\x16=== ATTR  " + t.Name() + ` dokimi.assert.1 {"definition":"3.1.0","seq":1,"assertion":"true",` +
						`"contract":"the child's call","verdict":"pass","aborting":true,"where":{"file":"verdict_test.go"`
					if !strings.Contains(out, want) {
						t.Fatalf("the child wrote %q, want the attribute %q", out, want)
					}
					return
				}
				matcher.True(t, matcher.Fatal, true, "the child's call")
			},
		)
	})
}

// slot keeps the slot that a call of Running.Slot returns.
var slot *record.Slot

// TestVerdictAllocs checks the allocation ceiling of each function and
// method of verdict.go on a seat whose calls are not recorded.
func TestVerdictAllocs(t *testing.T) {
	checkAllocs(t, verdictCases())
}

// BenchmarkVerdict measures each function and method of verdict.go.
func BenchmarkVerdict(b *testing.B) {
	benchAllocs(b, verdictCases())
}

// verdictCases returns a call of each function and method of verdict.go on
// a seat whose calls are not recorded, with its allocation ceiling,
// measured. A failure states no detail, and a fault no path.
func verdictCases() []allocCase {
	err := fault.New("the seed is no number")
	var run json.Marshaler = json.RawMessage(`{}`)
	f := matcher.Failure{Assertion: "prop-for-all", Contract: allocContract}
	return []allocCase{
		{name: "Pass", call: func(seat matcher.Seat) { matcher.Pass(seat, matcher.Fatal, "true", allocContract) }},
		{name: "Fail", allocs: 4, fails: true, call: func(seat matcher.Seat) {
			matcher.Fail(seat, matcher.Fatal, "true", allocContract, nil)
		}},
		{name: "Fault", allocs: 4, fails: true, call: func(seat matcher.Seat) {
			matcher.Fault(seat, matcher.Fatal, "true", allocContract, err)
		}},
		{name: "Begin", call: func(seat matcher.Seat) { _ = matcher.Begin(seat) }},
		{name: "Running.Slot", call: func(seat matcher.Seat) { slot = matcher.Begin(seat).Slot() }},
		{name: "Running.Pass", call: func(seat matcher.Seat) {
			matcher.Begin(seat).Pass(matcher.Fatal, "eventually", allocContract)
		}},
		{name: "Running.Fail", allocs: 4, fails: true, call: func(seat matcher.Seat) {
			matcher.Begin(seat).Fail(matcher.Fatal, "eventually", allocContract, nil)
		}},
		{name: "Running.PassRun", call: func(seat matcher.Seat) {
			matcher.Begin(seat).PassRun(matcher.Fatal, "prop-for-all", allocContract, matcher.Where{}, run)
		}},
		{name: "Running.FailRun", allocs: 2, fails: true, call: func(seat matcher.Seat) {
			matcher.Begin(seat).FailRun(matcher.Fatal, f, run)
		}},
		{name: "Running.Fault", allocs: 4, fails: true, call: func(seat matcher.Seat) {
			matcher.Begin(seat).Fault(matcher.Fatal, "eventually", allocContract, err)
		}},
	}
}

// runChild runs the test again in a child process, with the variables of
// env, and returns the child's output. It fails the test when the child
// fails or does not run the test.
func runChild(t *testing.T, env ...string) string {
	t.Helper()
	out, err := childtest.Run(t, t.Name(), env...)
	if err != nil || !strings.Contains(out, "--- PASS: "+t.Name()+" ") {
		t.Fatalf("the child exits with %v, want a pass of %s:\n%s", err, t.Name(), out)
	}
	return out
}
