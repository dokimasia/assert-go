// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"embed"
	"encoding/json"
	"slices"
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

// definition is the vendored definition.
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

// Assertions returns the definition's assertion table.
func Assertions() map[ID]Assertion {
	var doc struct {
		Assertions map[ID]Assertion `json:"assertions"`
	}
	read(assertionsFile, &doc)
	return doc.Assertions
}

// Names maps each assertion to the name this language exports it
// under, qualified with its subpackage where it has one.
func Names() map[ID]string {
	var doc struct {
		Names map[ID]map[string]string `json:"names"`
	}
	read(namingFile, &doc)

	out := make(map[ID]string, len(doc.Names))
	for id, byLanguage := range doc.Names {
		if name, ok := byLanguage[goLanguage]; ok {
			out[id] = name
		}
	}
	return out
}

// SurfaceNames maps every id the surface table states to the name this
// language gives it, across the types, members and helpers sections. An
// id that the table gives this language no name for maps to the empty
// string, as an id that the overlay declines does.
func SurfaceNames() map[ID]string {
	var doc struct {
		Surface map[string]map[ID]map[string]string `json:"surface"`
	}
	read(namingFile, &doc)

	out := make(map[ID]string)
	for _, section := range doc.Surface {
		for id, byLanguage := range section {
			out[id] = byLanguage[goLanguage]
		}
	}
	return out
}

// RelaxationNames maps each relaxation the definition states to the
// name this language exports it under. A relaxation that the naming
// table gives this language no name for maps to the empty string, as a
// relaxation that the overlay declines does.
func RelaxationNames() map[ID]string {
	var spec struct {
		Relaxations map[ID]struct{} `json:"relaxations"`
	}
	read(assertionsFile, &spec)

	var doc struct {
		Relaxations map[ID]map[string]string `json:"relaxations"`
	}
	read(namingFile, &doc)

	out := make(map[ID]string, len(spec.Relaxations))
	for id := range spec.Relaxations {
		out[id] = doc.Relaxations[id][goLanguage]
	}
	return out
}

// Version returns the version of the definition that this library
// implements.
func Version() string {
	raw, _ := definition.ReadFile(versionFile)
	return strings.TrimSpace(string(raw))
}

// Divergence is one declared difference from the definition.
type Divergence struct {
	ID     ID     `json:"id"`
	Stance string `json:"stance"`
	Why    string `json:"why"`
	Remedy string `json:"remedy"`
}

// OverlayDoc is this language's declared divergences, and how it runs the
// concurrent section of a machine.
type OverlayDoc struct {
	Extends     string       `json:"extends"`
	Language    string       `json:"language"`
	Diverge     []Divergence `json:"diverge"`
	Relaxations []Declined   `json:"relaxations"`
	Surface     []Declined   `json:"surface"`
	// Sections are the ways that a concurrent section of a machine runs:
	// tasks of the task scheduler, threads, or both.
	Sections []string `json:"sections"`
}

// Declined is a relaxation this language does not offer, with the
// reason. There is no what, because nothing is partly there.
type Declined struct {
	ID  ID     `json:"id"`
	Why string `json:"why"`
}

// DeclinesRelaxation reports whether the overlay declines id.
func (o OverlayDoc) DeclinesRelaxation(id ID) bool {
	return slices.ContainsFunc(o.Relaxations, func(d Declined) bool { return d.ID == id })
}

// DeclinesSurface reports whether the overlay declines the surface id.
func (o OverlayDoc) DeclinesSurface(id ID) bool {
	return slices.ContainsFunc(o.Surface, func(d Declined) bool { return d.ID == id })
}

// Diverges reports whether the overlay declares a divergence for id.
func (o OverlayDoc) Diverges(id ID) bool {
	return slices.ContainsFunc(o.Diverge, func(d Divergence) bool { return d.ID == id })
}

// Overlay returns this language's declared divergences.
func Overlay() OverlayDoc {
	var doc OverlayDoc
	read(overlayFile, &doc)
	return doc
}

// read decodes the file name of the vendored definition into each of
// into.
//
// It ignores the error of the read, because the embed directive includes
// every file that a reader names. It ignores the error of the decode as
// well: a file that does not decode leaves empty each value that it fails
// to state. The definition is authored in YAML and rendered to JSON beside
// it, so the module does not depend on a YAML parser.
func read(name string, into ...any) {
	raw, _ := definition.ReadFile(name)
	for _, v := range into {
		_ = json.Unmarshal(raw, v)
	}
}
