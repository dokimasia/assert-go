// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"io/fs"

	"go.dokimi.dev/assert/internal/prop/engine"
)

// The readers of the definition, over a file system that a test supplies.
var (
	AssertionsIn      = assertionsIn
	NamesIn           = namesIn
	SurfaceNamesIn    = surfaceNamesIn
	RelaxationNamesIn = relaxationNamesIn
	VersionIn         = versionIn
	OverlayIn         = overlayIn
	CasesIn           = casesIn
	VectorsIn         = vectorsIn
)

// StoreIn writes the stored cases of a behaviour vector that states none
// to dir, with the version of the definition in fsys.
func StoreIn(fsys fs.FS, dir string) error {
	return behaviourSettings{}.storeIn(fsys, dir, &engine.Settings{})
}
