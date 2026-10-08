// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package lint reports the checks that a test writes by hand and that an
// assertion of go.dokimi.dev/assert states.
//
// A failed True or False reports its message alone, and an if statement that
// fails a test reports the text that the test formats. The assertion that
// states the check reports the values that differ:
//
//	assert.True(t, len(items) == 3, "the store returns three items")
//	assert.Length(t, items, 3, "the store returns three items")
//
// [Analyzer] has 61 rules, which cover 100 of the 110 assertions of
// go.dokimi.dev/assert and its packages. 21 of the rules suggest a fix, and a
// fix applies only where the rewrite keeps the check's meaning in every case.
// The command assertlint runs the analyzer, and applies the fixes under -fix.
// The module go.dokimi.dev/assert/lint/golangci registers the analyzer with
// golangci-lint as the module plugin assertlint.
//
// # Checks
//
// A rule reads one of three kinds of check:
//
//   - A call of an assertion calls a package-level function of a surface,
//     assert or expect, whose first parameter is assert.TB. The type checker
//     resolves the function, so a renamed import or a dot import names it as
//     well.
//   - An if check is an if statement without else whose body is one call of
//     Error, Errorf, Fatal, Fatalf, Fail or FailNow on a value whose method
//     set has Helper. It states False of its condition.
//   - A statement that a rule names is a loop, a select statement, or a call
//     of a function other than an assertion.
//
// A condition loses its parentheses and its negations, and each negation
// inverts what the check states. False(t, !(x == nil)) and
// if x != nil { t.Fatal() } both state x == nil.
//
// An equality is a call of Equal or NotEqual without options, or a condition
// of ==, !=, reflect.DeepEqual, bytes.Equal, slices.Equal or maps.Equal. The
// origin of a variable is the call that assigns it, as os.ReadFile in
// want, err := os.ReadFile(path), or the call that receives its address, as
// json.Unmarshal in json.Unmarshal(data, &got). An inert statement is a call
// of an assertion, whatever its arguments call, or a declaration whose
// variables take no values. The rules round-trip and after-close count no
// inert statement as a step between two calls. The rules over two readings
// of one call count an assertion as a step where its values make a call that
// can write a variable that the reading reads. Such a call is a method of
// the variable, or receives the variable's address or a value of it that
// shares memory, as a slice, a map or a pointer does. A call of an assertion
// or of a method of a test writes nothing.
//
// A rule that relates a value to the call that made it follows the value
// only through earlier statements of the check's block. A value that a loop
// assigns has no origin there, and a function literal that assigns the value
// ends the search, because the literal gives the value when it runs. These
// rules are round-trip, the context rules, path-absent, after-close,
// has-content and golden-match. Other rules, such as links-to, follow a value
// through at most four conversions and origins anywhere in the package.
//
// # Rules over a check
//
// The rules over a check run in the following order, and the first rule that
// reports a check ends the search. Each diagnostic starts with the name of
// its rule and names the assertion, as in "compare: state the check with
// Equal". A rule that matches calls outside the check names them as well, as
// in "pure: state the check with Pure of s.Snapshot(), around s.Get(k)". The
// rules from commutative to permutation, except not-pure, read an equality
// that states its values equal. The rule not-pure reads one that states them
// different.
//
//   - panics, not-panics: Nil, NotNil, or a comparison with nil, of the
//     result of recover(). Panics states that the result is present, and
//     NotPanics that it is nil.
//   - nil-context-safe: NotPanics of a function literal that passes nil for
//     a parameter whose declared type is context.Context. NilContextSafe
//     states the check.
//   - in-range: True of lo <= x && x <= hi, or False of x < lo || x > hi,
//     over one number x. InRange states the check. Its fix applies where x
//     calls no function and InRange states both bounds exactly. A float64
//     takes a bound of <= or >= by a constant, or by a float64 that calls no
//     function. An integer takes a constant bound of a magnitude up to 2^53,
//     which the fix moves by one for < and >. A literal bound becomes the
//     number of the closed bound. Any other bound keeps its source text, with
//     +1 or -1 after it, and a typed bound gets a conversion to float64, as
//     in float64(slowStart-1).
//   - conjunction: True of a && b, and False of a || b, as a call or as an if
//     check outside a loop that waits for a condition, as eventually reads
//     it. An assertion of each operand states the check. The diagnostic
//     names the assertion that the rules over a check name for each operand
//     alone, as in "NoError for err != nil, False for failed". Its fix writes
//     a True or a False of each operand, for a statement of assert whose test
//     and message neither call a function nor receive from a channel.
//   - honours-cancellation, honours-deadline: a match of an error with
//     context.Canceled or context.DeadlineExceeded, where the call that
//     returns the error receives a context that ended before the call. A
//     context ends with Canceled where an earlier statement of the block calls
//     its cancel function, and with DeadlineExceeded where its deadline had
//     passed when it was made. A function of the package that returns such a
//     context, such as a helper cancelled(t), ends it too. A function literal
//     of an earlier statement that assigns the error, such as the function of
//     MaxAllocs, returns it where each of its assignments of the error is
//     such a call with a context that ended before the literal, and no
//     statement from the literal up to the check assigns the context's
//     variable. HonoursCancellation and HonoursDeadline state the check.
//   - path-absent: True of os.IsNotExist(err), and a match of err with
//     fs.ErrNotExist or os.ErrNotExist, where err comes from os.Stat.
//     files.Absent states the check.
//   - after-close: a match of an error with a sentinel, where the error comes
//     from a method of a value whose Close an earlier statement of the block
//     called, with only inert statements between the two. FailsAfterClose
//     states the check.
//   - errors-is: True or False of errors.Is. ErrorIs and ErrorIsNot state the
//     check, and the fix calls them.
//   - errors-as: True of errors.As(err, &target), or of a variable that a
//     call of errors.As assigns. ErrorAs states the check. Its fix writes
//     target = assert.ErrorAs[T](t, err, msg) for a statement of assert,
//     where the file can write the type T and another statement reads the
//     target.
//   - sentinel: an equality of two errors, exactly one of them a
//     package-level variable. ErrorIs and ErrorIsNot state the check, and also
//     match an error that wraps the variable.
//   - is-dir, is-file: True of IsDir of an fs.FileInfo or an fs.FileMode, and
//     of IsRegular of an fs.FileMode. files.IsDir and files.IsFile state the
//     check.
//   - has-mode: an equality of the value of Perm of an fs.FileMode, and not
//     of a value that the check computes from it, such as one bit.
//     files.HasMode states the check.
//   - links-to: an equality of a value whose origin is os.Readlink, and not
//     of a value that the check computes from it, such as its base name.
//     files.LinksTo states the check.
//   - has-content, golden-match: an equality of the text that os.ReadFile
//     reads. golden.Match states it for a file under testdata/golden,
//     golden.MatchAt for any other file under testdata, and files.HasContent
//     for any other file.
//   - max-allocs: a check whose value under test is a value of
//     testing.AllocsPerRun. MaxAllocs and MaxAllocsWithSetup state the check.
//     A ceiling that an assertion takes is no value under test.
//   - goroutine-leaks: a check whose value under test is a value of
//     runtime.NumGoroutine. NoGoroutineLeaks states the check.
//   - bench-max-allocs, bench-max-bytes, bench-max-mean: a check whose value
//     under test is AllocsPerOp, AllocedBytesPerOp or NsPerOp of a
//     testing.BenchmarkResult, or Elapsed of a *testing.B. MaxAllocs, MaxBytes
//     and MaxMean of a bench.Contract state the check.
//   - commutative: an equality of f(a, b) and f(b, a). Commutative states the
//     check. Its fix applies to Equal, where f is no builtin and has no type
//     parameters, and neither f nor an operand calls a function.
//   - associative: an equality of f(f(a, b), c) and f(a, f(b, c)).
//     Associative states the check. Its fix applies to Equal with the left
//     grouping first, under the conditions of commutative.
//   - round-trip: an equality of x and g(f(x)), where x is no constant and f
//     is no constructor of a fixture: a function of a test file that takes no
//     test. Every other statement from the call of f up to the check is
//     inert, and no statement but the call of g reads the result of f, so
//     RoundTrip, which makes both calls in one step, states the check.
//   - deterministic, stable-order, pure, not-pure, idempotent: an equality of
//     the results of two calls of one function with one input, other than two
//     errors. A conversion and a call of a builtin, such as make, are no such
//     calls, and a result that a later statement assigns is no result of its
//     call. Deterministic states two consecutive calls, with no step between
//     them, the second of which may be in the check. StableOrder states them
//     for a function without parameters that returns a slice other than
//     bytes. Pure and NotPure state two calls with steps between them, and
//     Idempotent states them where one statement repeats before each call.
//     Where a step passes the first result to a call that can write it, Pure
//     and NotPure observe a copy of the result around the steps up to the
//     check, as in "Pure of a copy of buf, around n, err := encode(buf)".
//   - permutation: an equality of two slices that earlier statements of the
//     block sort. Permutation states the check.
//   - rejects: True of X.Failed(), where X is a test other than the check's
//     own, such as an assert.Recorder. Rejects states the check.
//   - equal-func: True or False of bytes.Equal, or of slices.Equal or
//     maps.Equal over booleans, numbers or strings. Equal and NotEqual with
//     EquateEmpty state the check. Its fix applies where both operands have
//     one type.
//   - deep-equal: True or False of reflect.DeepEqual. Equal and NotEqual
//     state the check. Its fix applies where both operands have one type, and
//     the type contains no float, complex number, function, interface or
//     unsafe.Pointer, and no pointer in a map key.
//   - nil, error-nil: True or False of x == nil or x != nil. Nil and NotNil
//     state the check for a pointer, a slice, a map, a channel, a function or
//     an unsafe.Pointer, and NoError and HasError for an interface that
//     implements error. The fix calls them. The rules leave out every other
//     interface, in which == counts a typed nil as present and Nil counts it
//     as nil.
//   - equal-nil: Equal or NotEqual of a value and nil, without options. The
//     fix calls the assertion of nil or error-nil.
//   - length: a comparison of len(x) with a count, Equal of len(x) and a
//     count, NotEqual of len(x) and 0, and Length of x and 0, where len(x) is
//     the value under test: the first value, or the second after a constant.
//     Length, Empty and NotEmpty state the check. The fix calls them. The
//     rule reports no Length of a string, because len counts its bytes and
//     Length its Unicode scalar values.
//   - contains: True or False of strings.Contains, bytes.Contains or
//     slices.Contains, and of the ok of a lookup of a map. Contains and
//     NotContains state the check. The fix applies to each function but
//     slices.Contains over bytes or over values that are no booleans, numbers
//     or strings.
//   - membership: True of x == a || x == b, or False of x != a && x != b,
//     over one variable x. Contains states the check. Its fix writes the
//     members in a slice, where x is a boolean, a number other than a byte or
//     a string, the file can write x's type, and no member calls a function.
//   - contains-in-order: strings.Index(s, a) < strings.Index(s, b).
//     ContainsInOrder states the check.
//   - has-prefix, has-suffix: True of strings.HasPrefix, bytes.HasPrefix and
//     their suffix forms. HasPrefix and HasSuffix state the check, and the fix
//     calls them.
//   - matches: True of regexp.MatchString, or of MatchString or Match of a
//     *regexp.Regexp, where the condition is the match or a variable that it
//     assigns. Matches states the check.
//   - close-to: math.Abs(a-b) <= tol or math.Abs(a-b) < tol. CloseTo states
//     the check. Its fix applies to <=.
//   - pairwise: True of slices.IsSorted, slices.IsSortedFunc, or a function
//     of sort that reports whether values are sorted. Pairwise states the
//     check.
//   - order: <, <=, > or >= between a number and a constant. InRange states
//     the check. Its fix applies to an integer and a bound of a magnitude up
//     to 2^53, which the fix moves by one for < and > and writes as in-range
//     does. The open end of the range is -1<<63 or 1<<63 for a signed
//     integer, and 0 or 1<<64 for an unsigned one.
//   - compare: == or != between booleans, numbers or strings. Equal and
//     NotEqual state the check, and the fix calls them.
//   - condition: an if check that no other rule reports, outside a loop that
//     waits for a condition. True or False of its condition states the check.
//     A guard on a && b fails where both operands are true, so the rule names
//     the assertion of !b under an if of a, as in "NoError for err != nil,
//     where ready".
//
// # Rules over a statement
//
// A rule over a statement reads its statement apart from the rules over a
// check, so a loop and a check inside it can each have a diagnostic.
//
//   - chain: two or more consecutive statements that call assertions of one
//     surface with one test and one variable, where the chain of the surface
//     has a method of each assertion's name. That states them in one
//     statement. Its fix applies where no other fix rewrites one of the calls
//     and no comment is between them.
//   - total: a range over a slice whose body is one NoError of f(element),
//     where f does not read the element. Total states the check. Its fix
//     applies to a statement of assert where f calls no function and the
//     message does not read the element.
//   - poisoned: a for loop with a condition, or a range over an integer,
//     whose body is one HasError of a call. Poisoned states the check.
//   - no-duplicates: a loop that checks seen[x], or the ok of its lookup, and
//     assigns seen[x]. NoDuplicates states the check.
//   - monotonic: a loop that checks an order of two variables and assigns one
//     of them to the other. Monotonic states the check.
//   - eventually: a loop that waits for a condition. It calls time.Sleep, it
//     can end on a condition before a count of rounds, and a test is in
//     scope. A return or a break of its body ends it on a condition, and so
//     does the condition of a for statement that compares no counter, a
//     variable that the loop changes. EventuallyTrue and Eventually state the
//     wait.
//   - for-all: a loop that calls a function or method of math/rand or
//     math/rand/v2 and contains a check, and a call of quick.Check or
//     quick.CheckEqual. prop.ForAll states the property.
//   - completes-within: a select statement with a case that receives from
//     time.After and whose body fails the test. CompletesWithin states the
//     check.
//   - update-flag: a call of flag.Bool or flag.BoolVar in a test file that
//     defines the flag update. golden.MatchAt with golden.ShouldUpdate states
//     the golden file.
//   - property-form: a prop.ForAll whose body assigns the result of a Draw
//     and then calls one assertion that has a property form. The form, such
//     as prop.Equal, states the property where it takes each argument that
//     reads the drawn value. It passes its input to a function that it takes
//     in place of a value or of a function with fewer parameters, as got
//     func(T) U in place of got any. It generates an argument that it leaves
//     out, as the input of RoundTrip. No other argument, such as the observe
//     function of prop.Pure, reads the input.
//   - machine: a prop.ForAll whose body switches, inside a loop, on the
//     result of a Draw. stateful.Steps states the actions as a machine.
//
// # Annotations
//
// A line comment //dokimi:lint-skip, a comma-separated list of rules, a
// colon and a reason leaves out the reports of the listed rules on one line.
// On a line of its own it covers the next line, and after code it covers its
// own line:
//
//	//dokimi:lint-skip for-all: a fixed workload that the machine checks
//	for range rounds {
//
// The reason is required. The run reports, under the name lint-skip, an
// annotation without rules or a reason, and each listed rule that leaves out
// no report. An annotation then fails the run once the check that it excuses
// is gone.
//
// # Fixes
//
// A fix keeps the name under which the file imports the surface. An if check
// gets no fix, because its failure text states no contract to use as the
// message. A call that names its assertion through a dot import or with type
// arguments gets no fix either.
//
// # Dependency position
//
// Imports analysis, analysis/passes/inspect, ast/edge, ast/inspector and
// types/typeutil of golang.org/x/tools, and go/ast, go/constant, go/format,
// go/token, go/types, reflect, slices, strconv and strings of the standard
// library.
package lint
