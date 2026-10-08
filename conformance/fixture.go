// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"maps"
	"math/big"
	"net/netip"
	"reflect"
	"slices"
	"strconv"
	"time"
	"uuid"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/prop"
)

// The keys of a shape file that the fixtures runner reads.
const (
	// shapeKey names a shape's id.
	shapeKey = "shape"
	// refShape is the id of a shape that names a definition.
	refShape = "ref"
	// nameKey states the definition that a ref names.
	nameKey = "name"
	// definitionsKey states a shape file's definitions.
	definitionsKey = "definitions"
	// sourceKey states the language and the type a shape file was read from.
	sourceKey = "source"
)

// The fixture types of the definition, each as a fixtures vector states it.
// A field's json tag states its name in the record, and its prop tag the
// constraints of its shape. A type has the name of its fixture, except
// counted and hexDigits, whose fixtures' names this package declares
// already, and the variants of payment have the names of the variants.
type (
	// flag is a record of one boolean field, enabled.
	flag struct {
		Enabled bool `json:"enabled"`
	}
	// counted is the fixture counter: a record of one signed 32-bit integer
	// field, count.
	counted struct {
		Count int32 `json:"count"`
	}
	// ratio is a record of one 64-bit float field, value.
	ratio struct {
		Value float64 `json:"value"`
	}
	// initial is a record of one character field, letter.
	initial struct {
		Letter rune `json:"letter" prop:"char"`
	}
	// person is a record of one text field, name.
	person struct {
		Name string `json:"name"`
	}
	// blob is a record of one byte-string field, payload.
	blob struct {
		Payload []byte `json:"payload"`
	}
	// readings is a record of one field, values, a list of signed 32-bit
	// integers.
	readings struct {
		Values []int32 `json:"values"`
	}
	// channel is one unsigned byte of a colour. It is a type over uint8, so
	// an array of channels reads as a fixed-list and not as bytes.
	channel uint8
	// colour is a record of one field, rgb, an array of exactly three
	// unsigned bytes.
	colour struct {
		RGB [3]channel `json:"rgb"`
	}
	// tags is a record of one field, tags, a set of text.
	tags struct {
		Tags map[string]struct{} `json:"tags"`
	}
	// inventory is a record of one field, counts, a map from text to signed
	// 32-bit integers.
	inventory struct {
		Counts map[string]int32 `json:"counts"`
	}
	// memo is a record of one field, note, an optional text.
	memo struct {
		Note *string `json:"note"`
	}
	// line is one line of an order: sku, text, and qty, a signed 32-bit
	// integer.
	line struct {
		SKU string `json:"sku"`
		Qty int32  `json:"qty"`
	}
	// order is an order: id, an unsigned 32-bit integer, lines, a list of
	// lines, and note, an optional text.
	order struct {
		ID    uint32  `json:"id"`
		Lines []line  `json:"lines"`
		Note  *string `json:"note"`
	}
	// payment is a payment state, whose variants RegisterVariants states in
	// order: pending, paid and refunded.
	payment any
	// pending is the variant of payment without a payload.
	pending struct{}
	// paid is the variant of payment of a signed 64-bit amount.
	paid int64
	// refunded is the variant of payment of a text reason.
	refunded string
	// status is an enumeration of the text values that RegisterValues
	// states: pending, paid and shipped, in that order.
	status string
	// tree is a tree: value, a signed 32-bit integer, and children, a list
	// of trees.
	tree struct {
		Value    int32  `json:"value"`
		Children []tree `json:"children"`
	}
	// account is a record of one UUID field, id.
	account struct {
		ID uuid.UUID `json:"id"`
	}
	// host is a record of one IP address field of either version, address.
	host struct {
		Address netip.Addr `json:"address"`
	}
	// price is a record of one decimal field, amount, with two digits after
	// the point.
	price struct {
		Amount *big.Rat `json:"amount" prop:"scale=2"`
	}
	// event is a record of one instant field, at, at microseconds.
	event struct {
		At time.Time `json:"at" prop:"unit=us"`
	}
	// due is a record of one date field, day.
	due struct {
		Day time.Time `json:"day" prop:"date"`
	}
	// opening is a record of one time-of-day field, opens, at microseconds.
	opening struct {
		Opens time.Duration `json:"opens" prop:"time-of-day,unit=us"`
	}
	// appointment is a record of one date-and-time field without a zone,
	// starts, at microseconds.
	appointment struct {
		Starts time.Time `json:"starts" prop:"local-date-time,unit=us"`
	}
	// timeout is a record of one duration field, after, at microseconds.
	timeout struct {
		After time.Duration `json:"after" prop:"unit=us"`
	}
	// offset is a record of one UTC-offset field, offset.
	offset struct {
		Offset int32 `json:"offset" prop:"offset"`
	}
	// locale is a record of one time-zone field, zone.
	locale struct {
		Zone *time.Location `json:"zone"`
	}
	// meeting is a record of one field, starts, an instant in a time zone,
	// at microseconds.
	meeting struct {
		Starts time.Time `json:"starts" prop:"zoned-date-time,unit=us"`
	}
	// quantity is a record of one signed 32-bit integer field, qty, from 1
	// to 99.
	quantity struct {
		Qty int32 `json:"qty" prop:"min=1,max=99"`
	}
	// batch is a record of one field, lines, a list of 1 to 10 texts.
	batch struct {
		Lines []string `json:"lines" prop:"min_size=1,max_size=10"`
	}
	// sku is a record of one text field, code, three capital letters, a
	// hyphen and four digits.
	sku struct {
		Code string `json:"code" prop:"pattern=[A-Z]{3}-[0-9]{4}"`
	}
	// hexDigits is the fixture hex: a record of one text field, digits, over
	// the alphabet 0123456789abcdef.
	hexDigits struct {
		Digits string `json:"digits" prop:"alphabet=0123456789abcdef"`
	}
	// measurement is a record of one 64-bit float field, value, that admits
	// NaN and the infinities.
	measurement struct {
		Value float64 `json:"value" prop:"allow_nan,allow_infinity"`
	}
	// stamp is a record of one instant field, at, at milliseconds.
	stamp struct {
		At time.Time `json:"at" prop:"unit=ms"`
	}
	// fee is a record of one decimal field, amount, with four digits after
	// the point, from 0.0001.
	fee struct {
		Amount *big.Rat `json:"amount" prop:"scale=4,min=0.0001"`
	}
	// gateway is a record of one IP address field of version 4, address.
	gateway struct {
		Address netip.Addr `json:"address" prop:"version=4"`
	}
)

// The values of status, in the order that the fixture states them.
const (
	statusPending status = "pending"
	statusPaid    status = "paid"
	statusShipped status = "shipped"
)

func init() {
	prop.RegisterValues(statusPending, statusPaid, statusShipped)
	prop.RegisterVariants[payment](pending{}, paid(0), refunded(""))
}

// fixtures maps the name of each fixture type of the definition to the
// shape file that [prop.ShapeOf] writes for its Go type.
var fixtures = map[string]func() (string, error){
	"flag":        prop.ShapeOf[flag],
	"counter":     prop.ShapeOf[counted],
	"ratio":       prop.ShapeOf[ratio],
	"initial":     prop.ShapeOf[initial],
	"person":      prop.ShapeOf[person],
	"blob":        prop.ShapeOf[blob],
	"readings":    prop.ShapeOf[readings],
	"colour":      prop.ShapeOf[colour],
	"tags":        prop.ShapeOf[tags],
	"inventory":   prop.ShapeOf[inventory],
	"memo":        prop.ShapeOf[memo],
	"order":       prop.ShapeOf[order],
	"payment":     prop.ShapeOf[payment],
	"status":      prop.ShapeOf[status],
	"tree":        prop.ShapeOf[tree],
	"account":     prop.ShapeOf[account],
	"host":        prop.ShapeOf[host],
	"price":       prop.ShapeOf[price],
	"event":       prop.ShapeOf[event],
	"due":         prop.ShapeOf[due],
	"opening":     prop.ShapeOf[opening],
	"appointment": prop.ShapeOf[appointment],
	"timeout":     prop.ShapeOf[timeout],
	"offset":      prop.ShapeOf[offset],
	"locale":      prop.ShapeOf[locale],
	"meeting":     prop.ShapeOf[meeting],
	"quantity":    prop.ShapeOf[quantity],
	"batch":       prop.ShapeOf[batch],
	"sku":         prop.ShapeOf[sku],
	"hex":         prop.ShapeOf[hexDigits],
	"measurement": prop.ShapeOf[measurement],
	"stamp":       prop.ShapeOf[stamp],
	"fee":         prop.ShapeOf[fee],
	"gateway":     prop.ShapeOf[gateway],
}

// checkFixtures reads the shape file of the fixture type that a fixtures
// vector names, and compares the shape with the one that the vector states.
// The comparison leaves out the source of the shape file, and names each
// definition by the order in which the shape first refers to it, because Go
// names a definition by its type's package path.
func checkFixtures(raw json.RawMessage, _ string) error {
	var v struct {
		// Fixture names the fixture type.
		Fixture string `json:"fixture"`
		// Shape is the shape of the fixture type.
		Shape map[string]any `json:"shape"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	shapeOf, ok := fixtures[v.Fixture]
	if !ok {
		return fault.At(fault.New("%q is no fixture type", v.Fixture), fault.Field(fixtureMember))
	}
	// ShapeOf reads every fixture type, and writes a JSON object.
	text, _ := shapeOf()
	var read map[string]any
	_ = json.Unmarshal([]byte(text), &read)
	delete(read, sourceKey)
	if got, want := normalShape(read), normalShape(v.Shape); !reflect.DeepEqual(got, want) {
		return fault.At(fault.New("the fixture type reads as %s, want %s", jsonOf(got), jsonOf(want)),
			fault.Field(shapeMember))
	}
	return nil
}

// normalShape returns doc, a shape file as encoding/json decodes it, with
// each definition that a ref names renamed #0, #1 and on, in the order in
// which a walk of the shape first refers to it. The walk visits the keys of
// an object in sorted order and the items of a list in order, and a
// definition when a ref first names it. It keeps the name of a definition
// that no ref names, and renames doc in place.
func normalShape(doc map[string]any) map[string]any {
	definitions, _ := doc[definitionsKey].(map[string]any)
	names := make(map[string]string)
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if name, ok := v[nameKey].(string); ok && v[shapeKey] == refShape {
				renamed, seen := names[name]
				if !seen {
					renamed = "#" + strconv.Itoa(len(names))
					names[name] = renamed
					walk(definitions[name])
				}
				v[nameKey] = renamed
			}
			for _, key := range slices.Sorted(maps.Keys(v)) {
				if key != definitionsKey {
					walk(v[key])
				}
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		}
	}
	walk(doc)
	if definitions != nil {
		renamed := make(map[string]any, len(definitions))
		for name, definition := range definitions {
			if n, ok := names[name]; ok {
				name = n
			}
			renamed[name] = definition
		}
		doc[definitionsKey] = renamed
	}
	return doc
}
