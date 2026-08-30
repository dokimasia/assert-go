// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Surface names one of the two the standard requires, by the directory
// holding it relative to this package.
type Surface string

const (
	// Aborting stops the test at the first failure.
	Aborting Surface = ".."
	// Recording records a failure and lets the test continue.
	Recording Surface = "../expect"
	// Golden compares output against a recorded file.
	Golden Surface = "../golden"
	// Bench fails a benchmark that exceeds a ceiling.
	Bench Surface = "../bench"
)

// subpackages maps the name the definition gives a subpackage to the
// surface holding it, so a qualified name resolves to somewhere to
// look.
var subpackages = map[string]Surface{
	"golden": Golden,
	"bench":  Bench,
}

// Subpackage answers where an assertion's package name says to look.
func Subpackage(name string) (Surface, bool) {
	s, ok := subpackages[name]
	return s, ok
}

// The files a surface's members are read from.
const (
	goSuffix   = ".go"
	testSuffix = "_test.go"
)

// Members returns every exported package-level name a surface
// declares, sorted and deduplicated.
//
// Methods are left out: a method belongs to its type, and the types
// are compared by name.
//
// # Why the source rather than the compiled package
//
// A hand-maintained list would go stale without failing, which is the
// failure this guards against, so the answer comes from the files. It
// reads them with [go/parser], which needs nothing outside the
// standard library: a test-only check has no business adding a
// dependency to everything that imports the library.
//
// The cost is that build tags are not evaluated, so a surface split
// across tagged files would be read as one. Neither surface uses
// them, and one that started to would need this reconsidered.
func Members(s Surface) ([]string, error) {
	dir := string(s)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("conformance: read %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	var out []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, goSuffix) || strings.HasSuffix(name, testSuffix) {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			return nil, fmt.Errorf("conformance: parse %s: %w", name, err)
		}
		for _, decl := range file.Decls {
			out = append(out, exported(decl)...)
		}
	}

	slices.Sort(out)
	return slices.Compact(out), nil
}

// exported names what one declaration exports.
func exported(decl ast.Decl) []string {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil && d.Name.IsExported() {
			return []string{d.Name.Name}
		}
	case *ast.GenDecl:
		var out []string
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if s.Name.IsExported() {
					out = append(out, s.Name.Name)
				}
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if n.IsExported() {
						out = append(out, n.Name)
					}
				}
			}
		}
		return out
	}
	return nil
}
