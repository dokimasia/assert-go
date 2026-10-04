// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package equality compares two values as the definition's equal assertion
// compares them, for every assertion of this module that compares values,
// and finds the places where two unequal values differ.
//
// Two values are equal when they have the same type and the same value:
//
//   - A value of an interface type compares by the value inside it.
//   - Every field of a struct takes part, exported and unexported.
//   - A float compares by value, so -0 equals +0, and a NaN equals nothing,
//     itself included.
//   - A nil slice or map differs from an empty one.
//   - A map key compares as any value compares, so a map lookup that Go's
//     == would miss, such as a pointer key equal to the needle's target,
//     still finds it.
//   - A function compares by its code pointer, so two closures of one
//     function literal compare equal whatever they capture.
//   - A pointer, a map or a slice that the walk meets a second time inside
//     itself compares equal, so a value that contains itself compares in
//     finite time.
//
// [Rules] relax the comparison of NaNs and of empty containers for one
// call. No method of a value runs, so a type's Equal or String method
// cannot change a verdict, or panic inside one. [Diff] walks two unequal
// values under the same rules, and descends only into the parts that
// [Equal] reports unequal.
//
// # Dependency position
//
// Imports the standard library and internal/align, which aligns the
// elements of two slices. internal/matcher imports it for every assertion
// that compares values, and for the text of a failure of one.
package equality
