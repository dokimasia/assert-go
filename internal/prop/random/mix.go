// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package random

// Mix folds data into one 64-bit value with the stream alone. The value
// starts at 0, and each byte of data replaces it with the first value of
// the stream New(value XOR byte). Mix of the empty string is 0.
func Mix(data string) uint64 {
	var value uint64
	for i := range len(data) {
		s := New(value ^ uint64(data[i]))
		value = s.Next()
	}
	return value
}
