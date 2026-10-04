// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.dokimi.dev/assert/internal/fault"
)

// Surface is a surface of the library, named by its directory relative
// to this package.
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
	// Prop checks a property over generated inputs.
	Prop Surface = "../prop"
)

// subpackages maps the name that the definition gives a subpackage to
// the surface that contains it, so that a qualified name resolves to a
// directory.
var subpackages = map[string]Surface{
	"golden": Golden,
	"bench":  Bench,
	"prop":   Prop,
}

// Subpackage returns the surface of the subpackage that an assertion's
// package names, and whether the name is a subpackage of the library.
func Subpackage(name string) (Surface, bool) {
	s, ok := subpackages[name]
	return s, ok
}

// The files a surface's members are read from.
const (
	goSuffix   = ".go"
	testSuffix = "_test.go"
)

// seatType is the name of the seat's type, which an arity does not count.
const seatType = "TB"

// Members returns every exported package-level name a surface
// declares, sorted and deduplicated.
//
// Methods are left out. A method belongs to its type, and the types are
// compared by name.
//
// # The source, not the compiled package
//
// The names come from the files, so the list cannot go stale without
// the check failing. Members reads the files with [go/parser]. The
// parser is part of the standard library, so a check that only tests
// run adds no dependency to the modules that import the library.
//
// Build tags are not evaluated. A surface split across tagged files
// reads as one surface, and no surface uses build tags.
func Members(s Surface) ([]string, error) {
	decls, err := declarations(s)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, decl := range decls {
		out = append(out, exported(decl)...)
	}

	slices.Sort(out)
	return slices.Compact(out), nil
}

// Arities returns the arity of every exported function and method that a
// surface declares, keyed by its name, and a method's by Type.Method.
//
// The arity is the one the definition states. It counts the parameters
// after the seat, without a variadic one, and the type parameters that no
// parameter's type names, because a caller states each of those. The
// seat is a first parameter of the type TB, of this package or another.
// It reads the source as [Members] does.
func Arities(s Surface) (map[string]int, error) {
	decls, err := declarations(s)
	if err != nil {
		return nil, err
	}

	out := make(map[string]int)
	for _, decl := range decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() {
			continue
		}
		name := fn.Name.Name
		if fn.Recv != nil {
			owner := receiver(fn.Recv.List[0].Type)
			if !ast.IsExported(owner) {
				continue
			}
			name = owner + "." + name
		}
		out[name] = arity(fn.Type)
	}
	return out, nil
}

// arity counts what a caller states to call a function of type fn: the
// parameters after the seat, without a variadic one, and each type
// parameter that no parameter's type names.
func arity(fn *ast.FuncType) int {
	n := 0
	named := make(map[string]bool)
	for i, field := range fn.Params.List {
		ast.Inspect(field.Type, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok {
				named[id.Name] = true
			}
			return true
		})

		if _, variadic := field.Type.(*ast.Ellipsis); variadic || (i == 0 && isSeat(field.Type)) {
			continue
		}
		n += max(len(field.Names), 1)
	}

	if fn.TypeParams != nil {
		for _, field := range fn.TypeParams.List {
			for _, param := range field.Names {
				if !named[param.Name] {
					n++
				}
			}
		}
	}
	return n
}

// isSeat reports whether a parameter's type is TB, of this package or
// another.
func isSeat(typ ast.Expr) bool {
	switch t := typ.(type) {
	case *ast.Ident:
		return t.Name == seatType
	case *ast.SelectorExpr:
		return t.Sel.Name == seatType
	}
	return false
}

// receiver returns the name of a method's receiver type, without a
// pointer, parentheses and type arguments. It returns the empty string
// for a receiver that names no type, which the parser accepts and the
// compiler refuses.
func receiver(typ ast.Expr) string {
	switch t := typ.(type) {
	case *ast.StarExpr:
		return receiver(t.X)
	case *ast.ParenExpr:
		return receiver(t.X)
	case *ast.IndexExpr:
		return receiver(t.X)
	case *ast.IndexListExpr:
		return receiver(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// declarations returns the top-level declarations of a surface's source
// files, its test files left out. It returns a fault whose cause is the
// error of the file system for a directory that does not read, and one at
// the file's name whose cause is the error of the parser for a file that
// does not parse.
func declarations(s Surface) ([]ast.Decl, error) {
	dir := string(s)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fault.New("the surface does not read").Because(err)
	}

	fset := token.NewFileSet()
	var out []ast.Decl
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, goSuffix) || strings.HasSuffix(name, testSuffix) {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			return nil, fault.At(fault.New("the file does not parse").Because(err), fault.Field(name))
		}
		out = append(out, file.Decls...)
	}
	return out, nil
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
