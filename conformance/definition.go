// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"embed"
	"encoding/json"
	"fmt"
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

//go:embed spec/assertions.json spec/naming.json spec/overlay.json spec/VERSION spec/corpus
var definition embed.FS

// ID is an assertion's canonical name.
type ID string

// Assertion is one entry in the definition's assertion table.
type Assertion struct {
	// Arity counts required arguments to the function form, excluding
	// the seat and including the trailing message.
	Arity int `json:"arity"`
	// Package names the subpackage holding the assertion, empty for
	// the root namespace.
	Package string `json:"package"`
	// Summary states what the assertion means.
	Summary string `json:"summary"`
	// MessageFields names what a failure must report.
	MessageFields []string `json:"message_fields"`
}

// Assertions reads the definition's assertion table.
func Assertions() (map[ID]Assertion, error) {
	var doc struct {
		Version    string           `json:"version"`
		Assertions map[ID]Assertion `json:"assertions"`
	}
	if err := read(assertionsFile, &doc); err != nil {
		return nil, err
	}
	return doc.Assertions, nil
}

// Names maps each assertion to the name this language exports it
// under, qualified with its subpackage where it has one.
func Names() (map[ID]string, error) {
	var doc struct {
		Names map[ID]map[string]string `json:"names"`
	}
	if err := read(namingFile, &doc); err != nil {
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

// Version reports the definition version this library implements.
func Version() (string, error) {
	raw, err := definition.ReadFile(versionFile)
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
	Extends  string       `json:"extends"`
	Language string       `json:"language"`
	Diverge  []Divergence `json:"diverge"`
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
func Overlay() (OverlayDoc, error) {
	raw, err := definition.ReadFile(overlayFile)
	if err != nil {
		return OverlayDoc{}, fmt.Errorf("conformance: read %s: %w", overlayFile, err)
	}

	var doc OverlayDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return OverlayDoc{}, fmt.Errorf("conformance: parse %s: %w", overlayFile, err)
	}
	return doc, nil
}

// read decodes one embedded file.
//
// The definition is authored in YAML and rendered to JSON beside it,
// which is what this reads. A YAML parser would be a dependency in the
// module graph of everything importing this library, to read a file
// that changes when the standard does.
func read(name string, into any) error {
	raw, err := definition.ReadFile(name)
	if err != nil {
		return fmt.Errorf("conformance: read %s: %w", name, err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("conformance: parse %s: %w", name, err)
	}
	return nil
}
