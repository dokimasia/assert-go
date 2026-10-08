// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package tree records the choice sequences of a run as a trie: the case
// tree.
//
// A node is one choice request, recorded with its bounds, and an edge is
// the value the request took. A case that ends marks its last node as a
// leaf. A [Walker] follows one case down the tree and finds repeated
// cases, diverging bodies and exhausted domains:
//
//   - A repeated case. A body is a function of its choices, so a choice
//     that arrives at a leaf would end the body as that leaf's case ended.
//     [Walker.Step] returns [ErrRepeated] there, and the case does not
//     count.
//   - A diverging body. A request whose bounds differ from the request that
//     the tree recorded at the same position, a request where an earlier
//     case ended, and an end where an earlier case made a request each
//     return a [*DivergenceError].
//   - An exhausted domain. A node is exhausted when it is a leaf, or when
//     every value its bounds admit leads to an exhausted node. A float
//     choice never exhausts, and neither does a sequence without a maximum
//     length.
//
// A tree stops growing at its node limit, [NodeLimit] for a run. A walk
// that would add a node past the limit stops checking for the rest of its
// case, and the tree no longer reports an exhausted domain.
//
// # Concurrency
//
// A [Tree] and its walkers are not safe for concurrent use. The runner
// enters each case into the tree in the order that one worker produces
// them.
//
// # Allocation contract
//
// [New] allocates the tree, and [Tree.Walk] the walker and its path.
// [Walker.Step] allocates when it adds a node, and the key of every
// sequence choice. Every other method allocates nothing.
//
// # Dependency position
//
// Imports the choice package of this module and the standard library.
package tree
