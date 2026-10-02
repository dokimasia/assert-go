// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"io/fs"
	"maps"
	"path"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

// The files of the smallest definition that every reader accepts.
const (
	assertionsFile = "spec/assertions.json"
	namingFile     = "spec/naming.json"
	overlayFile    = "spec/overlay.json"
	versionFile    = "spec/VERSION"
	corpusFile     = "spec/corpus/equal.json"
	vectorsFile    = "spec/corpus/prop/token.json"
)

// The globs of the corpus and of the vectors.
const (
	corpusGlob  = "spec/corpus/*.json"
	vectorsGlob = "spec/corpus/prop/*.json"
)

// smallest returns the smallest definition that every reader accepts.
func smallest() fstest.MapFS {
	return fstest.MapFS{
		assertionsFile: {Data: []byte(`{"assertions": {"equal": {"arity": 3, "detail_fields": ["want", "got"]}},
			"relaxations": {"equate-nans": {}}}`)},
		namingFile: {Data: []byte(`{"names": {"equal": {"go": "Equal"}},
			"surface": {"types": {"seat": {"go": "TB"}}}, "relaxations": {"equate-nans": {"go": "EquateNaNs"}}}`)},
		overlayFile: {Data: []byte(`{"language": "go"}`)},
		versionFile: {Data: []byte("2.0.0\n")},
		corpusFile:  {Data: []byte(`{"assertion": "equal", "cases": [{"id": "equal/same", "expect": "pass"}]}`)},
		vectorsFile: {Data: []byte(`{"kind": "token", "cases": [{"id": "token/none"}]}`)},
	}
}

// with returns the smallest definition with the file at name replaced by
// data, or removed when data is nil.
func with(name string, data []byte) fstest.MapFS {
	fsys := maps.Clone(smallest())
	if data == nil {
		delete(fsys, name)
		return fsys
	}
	fsys[name] = &fstest.MapFile{Data: data}
	return fsys
}

// malformed is a JSON document cut short.
var malformed = []byte(`{`)

// TestReaders drives each reader of the definition over a file system whose
// files are missing or malformed.
func TestReaders(t *testing.T) {
	t.Parallel()

	t.Run("returns the contents of the smallest definition", func(t *testing.T) {
		t.Parallel()

		fsys := smallest()
		assertions, err := conformance.AssertionsIn(fsys)
		assert.NoError(t, err, "the assertion table can be read")
		assert.Equal(t, assertions["equal"].DetailFields, []string{"want", "got"}, "the fields of equal")
		names, err := conformance.NamesIn(fsys)
		assert.NoError(t, err, "the naming table can be read")
		assert.Equal(t, names, map[conformance.ID]string{"equal": "Equal"}, "the Go name of equal")
		surface, err := conformance.SurfaceNamesIn(fsys)
		assert.NoError(t, err, "the surface table can be read")
		assert.Equal(t, surface, map[conformance.ID]string{"seat": "TB"}, "the Go name of the seat")
		relaxations, err := conformance.RelaxationNamesIn(fsys)
		assert.NoError(t, err, "the relaxations can be read")
		assert.Equal(t, relaxations, map[conformance.ID]string{"equate-nans": "EquateNaNs"},
			"the Go name of the relaxation")
		version, err := conformance.VersionIn(fsys)
		assert.NoError(t, err, "the version can be read")
		assert.Equal(t, version, "2.0.0", "the version without its line break")
		overlay, err := conformance.OverlayIn(fsys)
		assert.NoError(t, err, "the overlay can be read")
		assert.Equal(t, overlay.Language, "go", "the language of the overlay")
		cases, err := conformance.CasesIn(fsys, corpusGlob)
		assert.NoError(t, err, "the corpus can be read")
		assert.Length(t, cases["equal"], 1, "the one case of equal")
		vectors, err := conformance.VectorsIn(fsys, vectorsGlob)
		assert.NoError(t, err, "the vectors can be read")
		assert.Length(t, vectors, 1, "the one vector")
		assert.NoError(t, conformance.StoreIn(fsys, t.TempDir()), "a vector that stores no case")
	})

	missing := []struct {
		name string
		file string
		read func(t *testing.T, f fs.FS) error
	}{
		{name: "Assertions", file: assertionsFile, read: func(_ *testing.T, f fs.FS) error {
			_, err := conformance.AssertionsIn(f)
			return err
		}},
		{name: "Names", file: namingFile, read: func(_ *testing.T, f fs.FS) error {
			_, err := conformance.NamesIn(f)
			return err
		}},
		{name: "SurfaceNames", file: namingFile, read: func(_ *testing.T, f fs.FS) error {
			_, err := conformance.SurfaceNamesIn(f)
			return err
		}},
		{name: "RelaxationNames of the assertion table", file: assertionsFile, read: func(_ *testing.T, f fs.FS) error {
			_, err := conformance.RelaxationNamesIn(f)
			return err
		}},
		{name: "RelaxationNames of the naming table", file: namingFile, read: func(_ *testing.T, f fs.FS) error {
			_, err := conformance.RelaxationNamesIn(f)
			return err
		}},
		{name: "Version", file: versionFile, read: func(_ *testing.T, f fs.FS) error {
			_, err := conformance.VersionIn(f)
			return err
		}},
		{name: "Overlay", file: overlayFile, read: func(_ *testing.T, f fs.FS) error {
			_, err := conformance.OverlayIn(f)
			return err
		}},
		{name: "Cases", file: assertionsFile, read: func(_ *testing.T, f fs.FS) error {
			_, err := conformance.CasesIn(f, corpusGlob)
			return err
		}},
		{name: "the store of a behaviour vector", file: versionFile, read: func(t *testing.T, f fs.FS) error {
			t.Helper()
			return conformance.StoreIn(f, t.TempDir())
		}},
	}
	for _, tc := range missing {
		t.Run(tc.name+" returns an error for a missing "+path.Base(tc.file), func(t *testing.T) {
			t.Parallel()

			err := tc.read(t, with(tc.file, nil))
			assert.ErrorIs(t, err, fs.ErrNotExist, "the file does not exist")
			assert.Contains(t, err.Error(), "conformance: read "+tc.file, "the error names the file")
		})
	}

	for _, tc := range missing {
		if tc.file == versionFile {
			continue
		}
		t.Run(tc.name+" returns an error for a malformed "+path.Base(tc.file), func(t *testing.T) {
			t.Parallel()

			err := tc.read(t, with(tc.file, malformed))
			assert.HasError(t, err, "the file does not parse")
			assert.Contains(t, err.Error(), "conformance: parse "+tc.file, "the error names the file")
		})
	}

	t.Run("Cases returns an error for a glob that does not parse", func(t *testing.T) {
		t.Parallel()

		_, err := conformance.CasesIn(smallest(), "[")
		assert.ErrorIs(t, err, path.ErrBadPattern, "the glob does not parse")
	})

	t.Run("Cases returns an error for a directory that the glob matches", func(t *testing.T) {
		t.Parallel()

		_, err := conformance.CasesIn(with("spec/corpus/listed.json/inside", []byte("{}")), corpusGlob)
		assert.HasError(t, err, "a directory does not read as a file")
		assert.Contains(t, err.Error(), "conformance: read spec/corpus/listed.json", "the error names the directory")
	})

	t.Run("Cases returns an error for a malformed corpus file", func(t *testing.T) {
		t.Parallel()

		_, err := conformance.CasesIn(with(corpusFile, malformed), corpusGlob)
		assert.HasError(t, err, "the corpus file does not parse")
		assert.Contains(t, err.Error(), "conformance: parse "+corpusFile, "the error names the file")
	})

	t.Run("Cases returns an error for a corpus file of an unstated assertion", func(t *testing.T) {
		t.Parallel()

		_, err := conformance.CasesIn(with(corpusFile, []byte(`{"assertion": "unstated", "cases": []}`)), corpusGlob)
		assert.HasError(t, err, "the assertion is not in the table")
		assert.Contains(t, err.Error(), `covers "unstated", which the definition does not state`,
			"the error names the assertion")
	})

	t.Run("Vectors returns an error for a glob that does not parse", func(t *testing.T) {
		t.Parallel()

		_, err := conformance.VectorsIn(smallest(), "[")
		assert.ErrorIs(t, err, path.ErrBadPattern, "the glob does not parse")
	})

	t.Run("Vectors returns an error for a directory that the glob matches", func(t *testing.T) {
		t.Parallel()

		_, err := conformance.VectorsIn(with("spec/corpus/prop/listed.json/inside", []byte("{}")), vectorsGlob)
		assert.HasError(t, err, "a directory does not read as a file")
		assert.Contains(t, err.Error(), "conformance: read spec/corpus/prop/listed.json",
			"the error names the directory")
	})

	t.Run("Vectors returns an error for a malformed vector file", func(t *testing.T) {
		t.Parallel()

		_, err := conformance.VectorsIn(with(vectorsFile, malformed), vectorsGlob)
		assert.HasError(t, err, "the vector file does not parse")
		assert.Contains(t, err.Error(), "conformance: parse vectors", "the error names the parse")
	})

	t.Run("Vectors returns an error for a vector whose id is no string", func(t *testing.T) {
		t.Parallel()

		_, err := conformance.VectorsIn(with(vectorsFile, []byte(`{"kind": "token", "cases": [{"id": 5}]}`)),
			vectorsGlob)
		assert.HasError(t, err, "the id does not parse")
		assert.Contains(t, err.Error(), "conformance: parse the ids of the vectors", "the error names the parse")
	})
}
