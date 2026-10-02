// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

// The vendored definition's files.
const (
	assertionsFile = "spec/assertions.json"
	namingFile     = "spec/naming.json"
	overlayFile    = "spec/overlay.json"
	versionFile    = "spec/VERSION"
)

// goLanguage keys this language's entries in the naming table.
const goLanguage = "go"

// definition is the vendored definition. Each reader of it takes the file
// system as a parameter, so a test can hand a reader a definition whose
// files are missing or malformed.
//
//go:embed spec/assertions.json spec/naming.json spec/overlay.json spec/VERSION spec/corpus
var definition embed.FS

// ID is an assertion's canonical name.
type ID string

// Assertion is one entry in the definition's assertion table.
type Assertion struct {
	// Arity counts required arguments to the function form, excluding
	// the seat and including the trailing message.
	Arity int `json:"arity"`
	// Package names the subpackage that contains the assertion, empty for
	// the root namespace.
	Package string `json:"package"`
	// Summary states what the assertion means.
	Summary string `json:"summary"`
	// DetailFields names the detail fields that a failure's record
	// contains.
	DetailFields []string `json:"detail_fields"`
}

// Assertions reads the definition's assertion table.
func Assertions() (map[ID]Assertion, error) { return assertionsIn(definition) }

// assertionsIn reads the assertion table of the definition in fsys.
func assertionsIn(fsys fs.FS) (map[ID]Assertion, error) {
	var doc struct {
		Version    string           `json:"version"`
		Assertions map[ID]Assertion `json:"assertions"`
	}
	if err := read(fsys, assertionsFile, &doc); err != nil {
		return nil, err
	}
	return doc.Assertions, nil
}

// Names maps each assertion to the name this language exports it
// under, qualified with its subpackage where it has one.
func Names() (map[ID]string, error) { return namesIn(definition) }

// namesIn reads the names of the assertions from the definition in fsys.
func namesIn(fsys fs.FS) (map[ID]string, error) {
	var doc struct {
		Names map[ID]map[string]string `json:"names"`
	}
	if err := read(fsys, namingFile, &doc); err != nil {
		return nil, err
	}

	out := make(map[ID]string, len(doc.Names))
	for id, byLanguage := range doc.Names {
		if name, ok := byLanguage[goLanguage]; ok {
			out[id] = name
		}
	}
	return out, nil
}

// DeclinesSurface reports whether the overlay declines the surface id.
func (o OverlayDoc) DeclinesSurface(id ID) bool {
	for _, d := range o.Surface {
		if d.ID == id {
			return true
		}
	}
	return false
}

// SurfaceNames maps every id the surface table states to the name this
// language gives it, across the types, members and helpers sections. An
// id that the table gives this language no name for maps to the empty
// string, as an id that the overlay declines does.
func SurfaceNames() (map[ID]string, error) { return surfaceNamesIn(definition) }

// surfaceNamesIn reads the surface table of the definition in fsys.
func surfaceNamesIn(fsys fs.FS) (map[ID]string, error) {
	var doc struct {
		Surface map[string]map[ID]map[string]string `json:"surface"`
	}
	if err := read(fsys, namingFile, &doc); err != nil {
		return nil, err
	}

	out := make(map[ID]string)
	for _, section := range doc.Surface {
		for id, byLanguage := range section {
			out[id] = byLanguage[goLanguage]
		}
	}
	return out, nil
}

// RelaxationNames maps each relaxation the definition states to the
// name this language exports it under. A relaxation that the naming
// table gives this language no name for maps to the empty string, as a
// relaxation that the overlay declines does.
func RelaxationNames() (map[ID]string, error) { return relaxationNamesIn(definition) }

// relaxationNamesIn reads the names of the relaxations from the
// definition in fsys.
func relaxationNamesIn(fsys fs.FS) (map[ID]string, error) {
	var spec struct {
		Relaxations map[ID]struct{} `json:"relaxations"`
	}
	if err := read(fsys, assertionsFile, &spec); err != nil {
		return nil, err
	}

	var doc struct {
		Relaxations map[ID]map[string]string `json:"relaxations"`
	}
	if err := read(fsys, namingFile, &doc); err != nil {
		return nil, err
	}

	out := make(map[ID]string, len(spec.Relaxations))
	for id := range spec.Relaxations {
		out[id] = doc.Relaxations[id][goLanguage]
	}
	return out, nil
}

// Version reports the definition version this library implements.
func Version() (string, error) { return versionIn(definition) }

// versionIn reads the version of the definition in fsys.
func versionIn(fsys fs.FS) (string, error) {
	raw, err := fs.ReadFile(fsys, versionFile)
	if err != nil {
		return "", fmt.Errorf("conformance: read %s: %w", versionFile, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

// Divergence is one declared difference from the definition.
type Divergence struct {
	ID     ID     `json:"id"`
	Stance string `json:"stance"`
	Why    string `json:"why"`
	Remedy string `json:"remedy"`
}

// OverlayDoc is this language's declared divergences.
type OverlayDoc struct {
	Extends     string       `json:"extends"`
	Language    string       `json:"language"`
	Diverge     []Divergence `json:"diverge"`
	Relaxations []Declined   `json:"relaxations"`
	Surface     []Declined   `json:"surface"`
}

// Declined is a relaxation this language does not offer, with the
// reason. There is no what, because nothing is partly there.
type Declined struct {
	ID  ID     `json:"id"`
	Why string `json:"why"`
}

// DeclinesRelaxation reports whether the overlay declines id.
func (o OverlayDoc) DeclinesRelaxation(id ID) bool {
	for _, d := range o.Relaxations {
		if d.ID == id {
			return true
		}
	}
	return false
}

// Diverges reports whether the overlay declares a divergence for id.
func (o OverlayDoc) Diverges(id ID) bool {
	for _, d := range o.Diverge {
		if d.ID == id {
			return true
		}
	}
	return false
}

// Overlay reads this language's declared divergences.
func Overlay() (OverlayDoc, error) { return overlayIn(definition) }

// overlayIn reads this language's overlay from the definition in fsys.
func overlayIn(fsys fs.FS) (OverlayDoc, error) {
	var doc OverlayDoc
	if err := read(fsys, overlayFile, &doc); err != nil {
		return OverlayDoc{}, err
	}
	return doc, nil
}

// read decodes the file name of fsys into into.
//
// The definition is authored in YAML and rendered to JSON beside it,
// which is what this reads. A YAML parser would be a dependency in the
// module graph of everything importing this library, to read a file
// that changes when the standard does.
func read(fsys fs.FS, name string, into any) error {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("conformance: read %s: %w", name, err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("conformance: parse %s: %w", name, err)
	}
	return nil
}
