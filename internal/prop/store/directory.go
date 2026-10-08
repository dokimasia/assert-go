// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.dokimi.dev/assert/internal/fault"
)

// The modes of a store's directory and of an entry's file, before the
// process's umask, as a golden file has them.
const (
	// dirMode is the mode of a directory that Save creates.
	dirMode = 0o755
	// fileMode is the mode of an entry's file.
	fileMode = 0o644
)

// ErrInvalid reports an entry that [Save] refuses, because [Read] would
// not replay it.
var ErrInvalid = errors.New("store: not an entry that a reader replays")

// Stored is what a run finds in its store for one property.
type Stored struct {
	// Entries are the property's entries, oldest first by their date, and
	// by the name of their file among the entries of one date.
	Entries []Entry
	// Skipped are the faults of the files that the run passes over, one per
	// file, each of the kind [ErrLater] at a path that starts at the file's
	// name.
	Skipped []error
}

// Load reads every file of dir whose name ends in ".json", and returns the
// entries of the property contract with the fault of each file that it
// skips. A dir that does not exist is an empty store. It reads every file,
// also after an error.
//
// # Errors
//
// Load returns, joined, a fault of the kind [ErrDamaged] for each damaged
// file, whose path starts at the file's name, and a fault whose cause is
// the error of the file system for dir or a file that cannot be read.
func Load(dir, contract string) (Stored, error) {
	files, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return Stored{}, nil
	}
	if err != nil {
		return Stored{}, fault.New("the store cannot be listed").Because(err)
	}
	var stored Stored
	var errs []error
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), nameSuffix) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			errs = append(errs, fault.At(fault.New("the file cannot be read").Because(err), fault.Field(file.Name())))
			continue
		}
		e, verdict, err := Read(data, contract)
		switch verdict {
		case Replay:
			stored.Entries = append(stored.Entries, e)
		case Other:
			// The entry belongs to another property of the test.
		case Skip:
			stored.Skipped = append(stored.Skipped, fault.At(err, fault.Field(file.Name())))
		case Damaged:
			errs = append(errs, fault.At(err, fault.Field(file.Name())))
		}
	}
	slices.SortStableFunc(stored.Entries, func(a, b Entry) int { return a.Found.Compare(b.Found) })
	return stored, errors.Join(errs...)
}

// Save writes e to dir as the file named [Entry.Name], creating dir and
// its parents with mode 0o755 when they are missing, and reports whether
// it wrote the file. The file is the entry as JSON indented by two spaces,
// with mode 0o644. The process's umask applies to both modes, as it does
// to a golden file.
//
// Save never overwrites a file: when one of the name exists, it writes
// nothing and returns false. A write that fails once the file exists
// leaves the file, which the next run reports as damaged.
//
// # Errors
//
// Save returns a fault of the kind [ErrInvalid] for an entry that [Read]
// would not replay, whose cause is the fault of [Read] or the error of the
// JSON encoder. It returns a fault whose cause is the error of the file
// system when dir or the file cannot be written.
func Save(dir string, e Entry) (bool, error) {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return false, fault.Of(ErrInvalid, "the entry does not encode as JSON").Because(err)
	}
	data = append(data, '\n')
	if _, verdict, damage := Read(data, e.Property); verdict != Replay {
		return false, fault.Of(ErrInvalid, "a reader would not replay the entry").Because(damage)
	}
	if err = os.MkdirAll(dir, dirMode); err != nil {
		return false, fault.New("the store cannot be created").Because(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, e.Name()), os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err == nil {
		_, err = f.Write(data)
		err = errors.Join(err, f.Close())
	}
	if err != nil {
		return false, fault.New("the entry cannot be written").Because(err)
	}
	return true, nil
}
