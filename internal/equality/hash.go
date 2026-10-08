// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package equality

import (
	"math"
	"reflect"
	"strconv"
)

// The bounds of the walk of [Hash].
const (
	// hashDepth is the most levels of a value that Hash reads.
	hashDepth = 8
	// hashParts is the most parts of a value that Hash reads, each value one
	// part.
	hashParts = 256
	// hashBytes is the most bytes of a string that Hash reads after its
	// length.
	hashBytes = 256
)

// The constants of 64-bit FNV-1a, by which Hash mixes each word into the
// hash.
const (
	// offset64 is the hash of no word.
	offset64 = 14695981039346656037
	// prime64 is the multiplier of each step.
	prime64 = 1099511628211
)

// The words that Hash mixes in for a value that states nothing else.
const (
	// invalidWord is the word of the invalid value, which a nil interface is.
	invalidWord = 1
	// nilWord is the word of a nil pointer, slice or map.
	nilWord = 2
)

// Hash returns a hash of v that two values that [Equal] reports equal under
// no relaxation share. A value of an interface type hashes by the value
// inside it, and the invalid value hashes as a nil interface.
//
// -0 and +0 hash alike, a map's entries hash in any order, a pointer hashes
// by its target, and a channel, a function and an unsafe pointer hash by
// their address. Two values that differ may share a hash.
//
// The walk reads at most 8 levels and 256 parts of a value, each value one
// part, and the length and at most 256 bytes of a string. A part past a
// bound adds nothing, so the walk ends on a value that contains itself. A
// map divides the parts left among its entries, and hashes by its length
// alone when it has more entries than parts left, because a map lists its
// entries in no fixed order.
//
// # Allocation contract
//
// Hash allocates nothing for a value without a map. A map whose entries it
// reads allocates its iterator and the copies of keys and values that the
// iterator returns: 9 allocations for a map of three ints.
func Hash(v reflect.Value) uint64 {
	h := hasher{sum: offset64, parts: hashParts}
	h.value(v, 0)
	return h.sum
}

// hasher is the state of one walk of [Hash].
type hasher struct {
	// sum is the hash so far.
	sum uint64
	// parts is the number of parts that the walk may still read.
	parts int
}

// mix mixes the word x into the hash.
func (h *hasher) mix(x uint64) {
	h.sum = (h.sum ^ x) * prime64
}

// value mixes v, at depth levels below the value that Hash was given, into
// the hash.
func (h *hasher) value(v reflect.Value, depth int) {
	v = inside(v)
	if h.parts == 0 || depth == hashDepth {
		return
	}
	h.parts--
	if !v.IsValid() {
		h.mix(invalidWord)
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		h.text(strconv.FormatBool(v.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		h.mix(uint64(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		h.mix(v.Uint())
	case reflect.Float32, reflect.Float64:
		h.float(v.Float())
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		h.float(real(c))
		h.float(imag(c))
	case reflect.String:
		h.text(v.String())
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		h.mix(uint64(v.Pointer()))
	case reflect.Pointer:
		h.pointer(v, depth)
	case reflect.Array:
		h.each(v, v.Len(), depth)
	case reflect.Struct:
		h.each(v, v.NumField(), depth)
	case reflect.Slice:
		h.slice(v, depth)
	case reflect.Map:
		h.entries(v, depth)
	default:
		// inside unwraps every interface, so no other kind reaches the switch.
	}
}

// float mixes f into the hash. Adding 0 turns -0 into +0, so the two zeros,
// which [Equal] reports equal, hash alike.
func (h *hasher) float(f float64) {
	h.mix(math.Float64bits(f + 0))
}

// text mixes the length of s and at most its first 256 bytes into the hash.
func (h *hasher) text(s string) {
	h.mix(uint64(len(s)))
	for i := range min(len(s), hashBytes) {
		h.mix(uint64(s[i]))
	}
}

// pointer mixes a pointer into the hash: a nil pointer as itself, and any
// other by its target, one level down.
func (h *hasher) pointer(v reflect.Value, depth int) {
	if v.IsNil() {
		h.mix(nilWord)
		return
	}
	h.value(v.Elem(), depth+1)
}

// slice mixes a slice into the hash: a nil slice as itself, and any other
// by its length and its elements.
func (h *hasher) slice(v reflect.Value, depth int) {
	if v.IsNil() {
		h.mix(nilWord)
		return
	}
	h.mix(uint64(v.Len()))
	h.each(v, v.Len(), depth)
}

// each mixes the n parts of an array, a slice or a struct into the hash,
// one level down, until the walk may read no more parts.
func (h *hasher) each(v reflect.Value, n, depth int) {
	for i := range n {
		if h.parts == 0 {
			return
		}
		h.value(part(v, i), depth+1)
	}
}

// entries mixes a map into the hash: a nil map as itself, and any other by
// its length and, when its entries fit the parts left, the sum of the
// hashes of its entries. Each entry hashes on a share of the parts left of
// its own, so the order of the entries changes neither which parts are read
// nor the sum.
func (h *hasher) entries(v reflect.Value, depth int) {
	if v.IsNil() {
		h.mix(nilWord)
		return
	}
	n := v.Len()
	h.mix(uint64(n))
	if n == 0 || h.parts < n {
		return
	}
	share := h.parts / n
	h.parts -= share * n
	var sum uint64
	for it := v.MapRange(); it.Next(); {
		entry := hasher{sum: offset64, parts: share}
		entry.value(it.Key(), depth+1)
		entry.value(it.Value(), depth+1)
		sum += entry.sum
	}
	h.mix(sum)
}
