// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

// box gives the nil cases a pointer target that is not itself nil.
type box struct{ n int }

// NilCases are the cases every surface's nil assertion must produce.
// Drive them with [RunOne].
//
// The typed-nil cases are the point. A (*T)(nil) in an interface is
// not equal to a plain nil under ==, and a surface that tests with ==
// passes the plain case and fails these.
func NilCases() []Case {
	var (
		nilPtr   *box
		nilSlice []int
		nilMap   map[string]int
		nilFn    func()
	)

	return []Case{
		{Name: "a plain nil passes", Args: []any{nil}},
		{Name: "a typed nil pointer passes", Args: []any{nilPtr}},
		{Name: "a nil slice passes", Args: []any{nilSlice}},
		{Name: "a nil map passes", Args: []any{nilMap}},
		{Name: "a nil func passes", Args: []any{nilFn}},
		{Name: "a nil defined slice type passes", Args: []any{ids(nil)}},
		{
			Name:     "a present pointer reports",
			Args:     []any{&box{n: 1}},
			Fails:    true,
			Contains: []string{"nil"},
		},
		{
			Name:     "zero is not nil",
			Args:     []any{0},
			Fails:    true,
			Contains: []string{"nil"},
		},
		{
			Name:     "an empty string is not nil",
			Args:     []any{""},
			Fails:    true,
			Contains: []string{"nil"},
		},
	}
}

// NotNilCases are the cases every surface's not-nil assertion must
// produce. Drive them with [RunOne].
func NotNilCases() []Case {
	var nilPtr *box

	return []Case{
		{Name: "a present pointer passes", Args: []any{&box{n: 1}}},
		{Name: "zero passes", Args: []any{0}},
		{
			Name:     "a plain nil reports",
			Args:     []any{nil},
			Fails:    true,
			Contains: []string{"nil"},
		},
		{
			Name:     "a typed nil pointer reports",
			Args:     []any{nilPtr},
			Fails:    true,
			Contains: []string{"nil"},
		},
	}
}
