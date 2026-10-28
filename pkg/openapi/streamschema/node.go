package streamschema

import (
	"strings"
)

// node is one compiled schema. Every field is set by the compiler and never
// changed afterwards, so nodes are shared freely across concurrent validations.
// Keywords that do not apply to an instance's type are ignored for it.
type node struct {
	// location is the RFC 6901 pointer of this schema in the document.
	location string
	// never is true for the false schema, which rejects every instance.
	never bool
	types typeSet

	// lengthMin and lengthMax bound string length in code points; lengthMax
	// is -1 when absent.
	lengthMin int64
	lengthMax int64

	// itemsMin and itemsMax bound array length; itemsMax is -1 when absent.
	itemsMin int64
	itemsMax int64

	// Numeric bounds are nil when absent.
	minimum          *decimal
	maximum          *decimal
	exclusiveMinimum *decimal
	exclusiveMaximum *decimal

	properties map[string]*node
	required   []string
	// additional applies to members not named in properties; nil accepts them.
	additional *node
	// items applies to every array element; nil accepts them.
	items *node
}

// trueNode accepts every instance. It is shared and must never be modified.
var trueNode = newNode("")

func newNode(location string) *node {
	return &node{
		location:         location,
		never:            false,
		types:            typeAll,
		lengthMin:        0,
		lengthMax:        -1,
		itemsMin:         0,
		itemsMax:         -1,
		minimum:          nil,
		maximum:          nil,
		exclusiveMinimum: nil,
		exclusiveMaximum: nil,
		properties:       nil,
		required:         nil,
		additional:       nil,
		items:            nil,
	}
}

// typeSet is a bit set of JSON Schema type names. typeNumber admits every
// number; typeInteger admits numbers without a fractional part.
type typeSet uint8

const (
	typeNull typeSet = 1 << iota
	typeBoolean
	typeObject
	typeArray
	typeNumber
	typeInteger
	typeString

	typeAll = typeNull | typeBoolean | typeObject | typeArray | typeNumber | typeInteger | typeString
)

// typeNames maps JSON Schema type names to their bits, in the order used for
// messages.
var typeNames = []struct {
	name string
	bit  typeSet
}{
	{name: "null", bit: typeNull},
	{name: "boolean", bit: typeBoolean},
	{name: "object", bit: typeObject},
	{name: "array", bit: typeArray},
	{name: "number", bit: typeNumber},
	{name: "integer", bit: typeInteger},
	{name: "string", bit: typeString},
}

func (t typeSet) has(bit typeSet) bool {
	return t&bit == bit
}

func (t typeSet) String() string {
	names := make([]string, 0, len(typeNames))
	for _, entry := range typeNames {
		if t.has(entry.bit) {
			names = append(names, entry.name)
		}
	}
	return strings.Join(names, ", ")
}
