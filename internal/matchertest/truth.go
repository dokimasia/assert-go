// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

// TrueCases are the cases every surface's true assertion must produce.
// Drive them with [RunOne].
func TrueCases() []Case {
	return []Case{
		{Name: "a true condition passes", Args: []any{true}},
		{
			Name:     "a false condition reports",
			Args:     []any{false},
			Fails:    true,
			Contains: []string{"true"},
		},
	}
}

// FalseCases are the cases every surface's false assertion must
// produce. Drive them with [RunOne].
func FalseCases() []Case {
	return []Case{
		{Name: "a false condition passes", Args: []any{false}},
		{
			Name:     "a true condition reports",
			Args:     []any{true},
			Fails:    true,
			Contains: []string{"false"},
		},
	}
}
