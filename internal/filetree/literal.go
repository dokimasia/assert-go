// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package filetree

import (
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"maps"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"go.dokimi.dev/assert/internal/fault"
)

// treeType is the type of a tree literal.
const treeType = "tree"

// The members of a tree literal and of its entries, which name the
// segments of the path of a fault in the literal.
const (
	typeMember       = "type"
	entriesMember    = "entries"
	pathMember       = "path"
	textMember       = "text"
	bytesMember      = "bytes"
	digestMember     = "digest"
	sizeMember       = "size"
	directoryMember  = "directory"
	linkMember       = "link"
	executableMember = "executable"
	modeMember       = "mode"
)

// kindMembers are the members of an entry that state its kind, of which an
// entry states exactly one. A record states digest, which no input states.
var kindMembers = [...]string{textMember, bytesMember, directoryMember, linkMember, digestMember}

// takes are the members that an entry of each kind may state.
var takes = map[string][]string{
	textMember:      {pathMember, textMember, executableMember, modeMember},
	bytesMember:     {pathMember, bytesMember, executableMember, modeMember},
	directoryMember: {pathMember, directoryMember, modeMember},
	linkMember:      {pathMember, linkMember},
	digestMember:    {pathMember, digestMember, sizeMember, executableMember, modeMember},
}

// treeLiteral is a tree literal as its JSON states it.
type treeLiteral struct {
	Type    string         `json:"type"`
	Entries []entryLiteral `json:"entries"`
}

// entryLiteral is one entry of a tree literal. A member that an entry does
// not state is absent.
type entryLiteral struct {
	Path       string  `json:"path"`
	Text       *string `json:"text,omitempty"`
	Bytes      *string `json:"bytes,omitempty"`
	Digest     string  `json:"digest,omitempty"`
	Size       int     `json:"size,omitempty"`
	Directory  bool    `json:"directory,omitempty"`
	Executable bool    `json:"executable,omitempty"`
	Link       *string `json:"link,omitempty"`
	Mode       *uint32 `json:"mode,omitempty"`
}

// MarshalJSON returns the tree literal of t, as a record states it: its
// entries in path order. A file states its content as text when the
// content is UTF-8 and as bytes otherwise, and by its digest and its size
// when the content is longer than [ContentLimit]. A file that states no
// mode states executable only when its owner may execute it.
//
// # Allocation contract
//
// MarshalJSON allocates the paths in order, the literal of each entry, and
// the JSON that it returns.
func (t Tree) MarshalJSON() ([]byte, error) {
	return t.literal(ContentLimit)
}

// Encode returns the tree literal of t as an input states it, which [Decode]
// reads back: the literal that [Tree.MarshalJSON] returns, with the whole
// content of every file, however long.
//
// # Allocation contract
//
// Encode allocates what [Tree.MarshalJSON] allocates.
func (t Tree) Encode() ([]byte, error) {
	return t.literal(math.MaxInt)
}

// literal returns the tree literal of t, which states the content of a file
// longer than limit by its digest and its size.
func (t Tree) literal(limit int) ([]byte, error) {
	out := treeLiteral{Type: treeType, Entries: make([]entryLiteral, 0, len(t))}
	for _, path := range t.Paths() {
		out.Entries = append(out.Entries, literalOf(path, t[path], limit))
	}
	return json.Marshal(out)
}

// literalOf returns the literal of the entry e at path, which states the
// content of a file longer than limit by its digest and its size.
func literalOf(path string, e Entry, limit int) entryLiteral {
	l := entryLiteral{Path: path}
	if e.Kind == File {
		content := e.Content
		if len(content) > limit {
			l.Digest, l.Size = digest(content), len(content)
		} else if utf8.ValidString(content) {
			l.Text = &content
		} else {
			bytes := hex.EncodeToString([]byte(content))
			l.Bytes = &bytes
		}
		l.Executable = e.Executable() && !e.Stated
	}
	if e.Kind == Dir {
		l.Directory = true
	}
	if e.Kind == Link {
		target := e.Target
		l.Link = &target
	}
	if e.Stated {
		mode := uint32(e.Mode)
		l.Mode = &mode
	}
	return l
}

// Decode returns the tree that a tree literal states as an input: its
// entries in path order, each a file of its content, a directory or a link,
// and no file stated by its digest, which only a record states. The tree
// keeps the rules of [Tree.Check].
//
// # Errors
//
// It returns a fault whose path leads through the literal's JSON to what it
// misstates, such as entries[2].mode, and for a tree that breaks a rule of
// a tree, the fault of [Tree.Check].
//
// # Allocation contract
//
// Decode allocates what the JSON decoder allocates for the literal, and the
// tree.
func Decode(raw []byte) (Tree, error) {
	var lit struct {
		Type    string            `json:"type"`
		Entries []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(raw, &lit); err != nil {
		return nil, fault.New("the text is no tree literal").Because(err)
	}
	if lit.Type != treeType {
		return nil, fault.At(fault.New("the type %q is not tree", lit.Type), fault.Field(typeMember))
	}
	if lit.Entries == nil {
		return nil, fault.At(fault.New("the literal states no list of entries"), fault.Field(entriesMember))
	}
	t := make(Tree, len(lit.Entries))
	last := ""
	for i, raw := range lit.Entries {
		path, e, err := decodeEntry(raw)
		if err == nil && i > 0 && path <= last {
			err = fault.New("the path %q is not after the path before it", path)
		}
		if err != nil {
			return nil, fault.At(err, fault.Field(entriesMember), fault.Index(i))
		}
		last, t[path] = path, e
	}
	if err := t.Check(); err != nil {
		return nil, err
	}
	return t, nil
}

// decodeEntry returns the path and the entry that one entry of a tree
// literal states.
func decodeEntry(raw json.RawMessage) (string, Entry, error) {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return "", Entry{}, fault.New("the entry is no object").Because(err)
	}
	var kinds []string
	for _, member := range kindMembers {
		if _, ok := members[member]; ok {
			kinds = append(kinds, member)
		}
	}
	if len(kinds) != 1 {
		return "", Entry{}, fault.New("the entry states %v, and an entry states one of text, bytes, directory and link",
			kinds)
	}
	for _, member := range slices.Sorted(maps.Keys(members)) {
		if !slices.Contains(takes[kinds[0]], member) {
			return "", Entry{}, fault.At(fault.New("a %s entry takes no %s", kinds[0], member), fault.Field(member))
		}
	}
	var l struct {
		Path       string  `json:"path"`
		Directory  bool    `json:"directory"`
		Executable bool    `json:"executable"`
		Mode       *uint32 `json:"mode"`
	}
	if err := json.Unmarshal(raw, &l); err != nil {
		return "", Entry{}, fault.New("the entry does not parse").Because(err)
	}
	if _, executable := members[executableMember]; executable && l.Mode != nil {
		return "", Entry{}, fault.New("the entry states a mode and executable, and the mode states the execute bit")
	}
	e := Entry{Kind: File}
	if l.Executable {
		e.Mode = OwnerExecute
	}
	if l.Mode != nil {
		e.Mode, e.Stated = fs.FileMode(*l.Mode), true
	}
	err := e.stateKind(kinds[0], members, l.Directory)
	return l.Path, e, err
}

// stateKind makes e an entry of the kind that the member kind of an entry's
// members states: a file of the text or of the bytes, a directory or a link.
func (e *Entry) stateKind(kind string, members map[string]json.RawMessage, directory bool) error {
	if kind == directoryMember {
		e.Kind = Dir
		if !directory {
			return fault.At(fault.New("directory is not true"), fault.Field(directoryMember))
		}
		return nil
	}
	if kind == digestMember {
		return fault.At(
			fault.New("a digest states no content, and only a record states one"),
			fault.Field(digestMember),
		)
	}
	var value *string
	if err := json.Unmarshal(members[kind], &value); err != nil || value == nil {
		return fault.At(fault.New("the %s is no string", kind), fault.Field(kind))
	}
	e.Content = *value
	if kind == linkMember {
		e.Kind, e.Content, e.Target = Link, "", *value
	}
	if kind != bytesMember {
		return nil
	}
	content, err := hex.DecodeString(*value)
	if err != nil || *value != strings.ToLower(*value) {
		return fault.At(fault.New("%q is no lowercase hexadecimal", *value), fault.Field(bytesMember))
	}
	e.Content = string(content)
	return nil
}
