// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"
	"math/big"
	"regexp"
	"slices"
	"strings"

	"go.dokimi.dev/assert/internal/fault"
)

// The keys of the prop tag. Each states a constraint of the definition or
// chooses the shape that a type reads as.
const (
	minKey           = "min"
	maxKey           = "max"
	minSizeKey       = "min_size"
	maxSizeKey       = "max_size"
	patternKey       = "pattern"
	alphabetKey      = "alphabet"
	allowNaNKey      = "allow_nan"
	allowInfinityKey = "allow_infinity"
	unitKey          = "unit"
	scaleKey         = "scale"
	versionKey       = "version"
	charKey          = "char"
	dateKey          = "date"
	localKey         = "local-date-time"
	zonedKey         = "zoned-date-time"
	timeOfDayKey     = "time-of-day"
	offsetKey        = "offset"
)

// valued reports for each key of the prop tag whether it states a value
// after =. A key that states none is a flag.
var valued = map[string]bool{
	minKey: true, maxKey: true, minSizeKey: true, maxSizeKey: true, patternKey: true, alphabetKey: true,
	unitKey: true, scaleKey: true, versionKey: true,
	allowNaNKey: false, allowInfinityKey: false, charKey: false, dateKey: false, localKey: false,
	zonedKey: false, timeOfDayKey: false, offsetKey: false,
}

// safeInteger is 2^53 - 1, the largest integer that a shape file states as
// a JSON number.
var safeInteger = big.NewInt(1<<53 - 1)

// jsonNumber matches the text of a JSON number.
var jsonNumber = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

// tag is one key of a prop tag, and its value.
type tag struct {
	// key is the key.
	key string
	// value is the text after =, and empty for a flag.
	value string
}

// tags are the keys of one field's prop tag that no part of the field's
// type has taken. A nil *tags states no key.
type tags struct {
	// keys are the keys, in the order the tag states them.
	keys []tag
}

// parseTags returns the keys of the prop tag text. A pattern and an
// alphabet take the rest of the tag after =, so each is the last key. It
// returns a fault for a key that the tag does not have, a key stated twice,
// a flag with a value, a key without one, and a tag that ends in a comma.
func parseTags(text string) (*tags, error) {
	k := &tags{}
	for rest := text; rest != ""; {
		item, after, more := strings.Cut(rest, ",")
		key, value, stated := strings.Cut(item, "=")
		if stated && (key == patternKey || key == alphabetKey) {
			value, after, more = rest[len(key)+1:], "", false
		}
		if err := k.add(key, value, stated); err != nil {
			return nil, err
		}
		if more && after == "" {
			return nil, fault.New("the tag %q ends in a comma", text)
		}
		rest = after
	}
	return k, nil
}

// add adds key with value, which states one when stated is set.
func (k *tags) add(key, value string, stated bool) error {
	takes, known := valued[key]
	switch {
	case !known:
		return fault.New("the key %q is no key of the prop tag", key)
	case takes && !stated:
		return fault.New("the key %s states no value", key)
	case !takes && stated:
		return fault.New("the key %s takes no value", key)
	case slices.ContainsFunc(k.keys, func(t tag) bool { return t.key == key }):
		return fault.New("the tag states the key %s twice", key)
	}
	k.keys = append(k.keys, tag{key: key, value: value})
	return nil
}

// take removes key from k and returns its value, and false when k does not
// state it.
func (k *tags) take(key string) (string, bool) {
	if k == nil {
		return "", false
	}
	i := slices.IndexFunc(k.keys, func(t tag) bool { return t.key == key })
	if i < 0 {
		return "", false
	}
	value := k.keys[i].value
	k.keys = slices.Delete(k.keys, i, i+1)
	return value, true
}

// flag removes key from k and reports whether k stated it.
func (k *tags) flag(key string) bool {
	_, ok := k.take(key)
	return ok
}

// choice removes the keys from k and returns the one that k stated, and ""
// for none. It returns a fault when k states two, which choose two shapes.
func (k *tags) choice(keys ...string) (string, error) {
	var chosen []string
	for _, key := range keys {
		if k.flag(key) {
			chosen = append(chosen, key)
		}
	}
	if len(chosen) > 1 {
		return "", fault.New("the tag states %s and %s, which choose two shapes", chosen[0], chosen[1])
	}
	if len(chosen) == 0 {
		return "", nil
	}
	return chosen[0], nil
}

// copy moves the keys from k into n, each value as form writes it.
func (k *tags) copy(n node, form func(string) any, keys ...string) {
	for _, key := range keys {
		if value, ok := k.take(key); ok {
			n[key] = form(value)
		}
	}
}

// left returns the first key that k still states, and "" for none.
func (k *tags) left() string {
	if len(k.keys) == 0 {
		return ""
	}
	return k.keys[0].key
}

// numeric returns the JSON value that a shape file states the number text
// as: a JSON number, and a string for an integer beyond 2^53 - 1 in
// magnitude, which a typed literal states as a decimal string, and for
// text that is no JSON number, which the shape refuses.
func numeric(text string) any {
	if !jsonNumber.MatchString(text) {
		return text
	}
	if n, ok := new(big.Int).SetString(text, decimalBase); ok && n.CmpAbs(safeInteger) > 0 {
		return text
	}
	return json.Number(text)
}

// verbatim returns text as a JSON string, the form of a bound of a decimal
// and of a date and time shape.
func verbatim(text string) any {
	return text
}
