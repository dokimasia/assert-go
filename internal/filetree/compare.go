// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package filetree

import (
	"io/fs"
	"slices"
)

// Differing returns the paths at which got, a tree read from a directory,
// differs from want, in path order. want implies the parents of its
// entries. A path differs when one of the two trees has it and the other
// does not, or when the entries at it differ. With contains set, a path
// that want lacks is no difference.
//
// Two entries at one path differ when their kinds differ, when two files
// have other content or two links other targets, when the wanted entry
// states a mode and the entry read has other permission bits, and when the
// wanted file states no mode and the owner's execute bits differ. A
// platform that records no permission bits compares neither.
//
// # Allocation contract
//
// Differing allocates the wanted tree with its implied directories, the
// paths that differ, and what their sort allocates.
func Differing(want, got Tree, contains bool) []string {
	wanted := want.Full()
	var out []string
	for path, w := range wanted {
		if differs(w, got[path]) {
			out = append(out, path)
		}
	}
	if !contains {
		for path := range got {
			if _, stated := wanted[path]; !stated {
				out = append(out, path)
			}
		}
	}
	slices.Sort(out)
	return out
}

// differs reports whether g, the entry read at a path, differs from w, the
// wanted entry there. A zero Entry is no entry.
func differs(w, g Entry) bool {
	if w.Kind != g.Kind {
		return true
	}
	return w.Content != g.Content || w.Target != g.Target || (w.Mode^g.Mode)&compared(w) != 0
}

// compared returns the permission bits that a comparison reads of an entry
// read where the wanted entry w is: all nine where w states its mode, the
// execute bit of a file where it states none, and the bits that the
// platform records of either.
func compared(w Entry) fs.FileMode {
	if w.Stated {
		return fs.ModePerm & recordedBits
	}
	return executeBit(w.Kind) & recordedBits
}
