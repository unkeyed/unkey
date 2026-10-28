package streamschema

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// nameOverheadBytes is charged against KeysRetainedBytesMax for every
	// retained member name in addition to its length. It approximates the map
	// entry and string header, so objects with many empty or short names still
	// exhaust the budget.
	nameOverheadBytes = 64
	// namesetReuseMax is the largest member-name set kept for reuse after its
	// object closes. Larger sets are released so that sets reused at different
	// depths cannot together outgrow KeysRetainedBytesMax.
	namesetReuseMax = 64
)

type validation struct {
	scanner scanner
	limits  Limits
	// path holds one segment per open container, for InstanceLocation.
	path []segment
	// namesets holds the member names of the open object at each depth.
	namesets []nameset
	// namesBytes is the accounted size of all names in namesets.
	namesBytes int
	nameBuf    []byte
	numberBuf  []byte
	number     decimal
}

type nameset struct {
	names map[string]struct{}
	// bytes is this object's share of validation.namesBytes.
	bytes int
}

// segment is one step of an instance location: a member name or array index.
type segment struct {
	name   string
	index  int64
	isName bool
}

// value validates the value at the cursor against n. depth counts the
// containers enclosing the value.
func (v *validation) value(n *node, depth int) error {
	c, err := v.scanner.next()
	if err != nil {
		return err
	}
	offset := v.scanner.offset()

	if n.never {
		return v.schemaError(n.location, offset, "schema false rejects every value")
	}

	switch {
	case c == '{':
		if !n.types.has(typeObject) {
			return v.typeError(n, offset, "object")
		}
		return v.object(n, depth)
	case c == '[':
		if !n.types.has(typeArray) {
			return v.typeError(n, offset, "array")
		}
		return v.array(n, depth)
	case c == '"':
		if !n.types.has(typeString) {
			return v.typeError(n, offset, "string")
		}
		return v.stringValue(n, offset)
	case c == 't' || c == 'f':
		if !n.types.has(typeBoolean) {
			return v.typeError(n, offset, "boolean")
		}
		if c == 't' {
			return v.scanner.literal("true")
		}
		return v.scanner.literal("false")
	case c == 'n':
		if !n.types.has(typeNull) {
			return v.typeError(n, offset, "null")
		}
		return v.scanner.literal("null")
	case c == '-' || isDigit(c):
		return v.numberValue(n, offset)
	default:
		return syntaxError(offset, "unexpected character")
	}
}

func (v *validation) object(n *node, depth int) error {
	start := v.scanner.offset()
	if depth >= v.limits.DepthMax {
		return limitError(start, "nesting exceeds DepthMax")
	}
	v.scanner.start++
	for len(v.namesets) <= depth {
		v.namesets = append(v.namesets, nameset{names: nil, bytes: 0})
	}

	c, err := v.scanner.next()
	if err != nil {
		return err
	}
	if c == '}' {
		v.scanner.start++
		return v.closeObject(n, depth, start)
	}

	for {
		if c != '"' {
			return syntaxError(v.scanner.offset(), "expected a member name")
		}
		name, err := v.memberName(depth)
		if err != nil {
			return err
		}

		c, err = v.scanner.next()
		if err != nil {
			return err
		}
		if c != ':' {
			return syntaxError(v.scanner.offset(), "expected ':' after a member name")
		}
		v.scanner.start++

		child := n.properties[name]
		if child == nil {
			child = n.additional
		}
		if child == nil {
			child = trueNode
		}
		v.path = append(v.path, segment{name: name, index: 0, isName: true})
		if err := v.value(child, depth+1); err != nil {
			return err
		}
		v.path = v.path[:len(v.path)-1]

		c, err = v.scanner.next()
		if err != nil {
			return err
		}
		switch c {
		case ',':
			v.scanner.start++
			c, err = v.scanner.next()
			if err != nil {
				return err
			}
		case '}':
			v.scanner.start++
			return v.closeObject(n, depth, start)
		default:
			return syntaxError(v.scanner.offset(), "expected ',' or '}' in object")
		}
	}
}

func (v *validation) memberName(depth int) (string, error) {
	offset := v.scanner.offset()
	decoded, err := v.scanner.name(v.nameBuf[:0], v.limits.KeyBytesMax)
	v.nameBuf = decoded[:0]
	if err != nil {
		return "", err
	}

	set := &v.namesets[depth]
	if _, duplicate := set.names[string(decoded)]; duplicate {
		return "", syntaxError(offset, "duplicate member name")
	}
	cost := len(decoded) + nameOverheadBytes
	if cost > v.limits.KeysRetainedBytesMax-v.namesBytes {
		return "", limitError(offset, "member names exceed KeysRetainedBytesMax")
	}
	v.namesBytes += cost
	set.bytes += cost

	if set.names == nil {
		set.names = make(map[string]struct{})
	}
	name := string(decoded)
	set.names[name] = struct{}{}
	return name, nil
}

func (v *validation) closeObject(n *node, depth int, start int64) error {
	set := &v.namesets[depth]
	for _, name := range n.required {
		if _, ok := set.names[name]; !ok {
			return v.schemaError(n.location+"/required", start, fmt.Sprintf("missing required property %q", name))
		}
	}

	v.namesBytes -= set.bytes
	set.bytes = 0
	if len(set.names) > namesetReuseMax {
		set.names = nil
	} else {
		clear(set.names)
	}
	return nil
}

func (v *validation) array(n *node, depth int) error {
	start := v.scanner.offset()
	if depth >= v.limits.DepthMax {
		return limitError(start, "nesting exceeds DepthMax")
	}
	v.scanner.start++

	child := n.items
	if child == nil {
		child = trueNode
	}

	count := int64(0)
	c, err := v.scanner.next()
	if err != nil {
		return err
	}
	if c == ']' {
		v.scanner.start++
		return v.closeArray(n, start, count)
	}

	for {
		if n.itemsMax >= 0 && count >= n.itemsMax {
			return v.schemaError(n.location+"/maxItems", v.scanner.offset(), fmt.Sprintf("array has more than %d items", n.itemsMax))
		}
		v.path = append(v.path, segment{name: "", index: count, isName: false})
		if err := v.value(child, depth+1); err != nil {
			return err
		}
		v.path = v.path[:len(v.path)-1]
		count++

		c, err = v.scanner.next()
		if err != nil {
			return err
		}
		switch c {
		case ',':
			v.scanner.start++
			if _, err := v.scanner.next(); err != nil {
				return err
			}
		case ']':
			v.scanner.start++
			return v.closeArray(n, start, count)
		default:
			return syntaxError(v.scanner.offset(), "expected ',' or ']' in array")
		}
	}
}

func (v *validation) closeArray(n *node, start int64, count int64) error {
	if count < n.itemsMin {
		return v.schemaError(n.location+"/minItems", start, fmt.Sprintf("array has fewer than %d items", n.itemsMin))
	}
	return nil
}

func (v *validation) stringValue(n *node, offset int64) error {
	runes, err := v.scanner.stringRunes()
	if err != nil {
		return err
	}
	if runes < n.lengthMin {
		return v.schemaError(n.location+"/minLength", offset, fmt.Sprintf("string is shorter than %d characters", n.lengthMin))
	}
	if n.lengthMax >= 0 && runes > n.lengthMax {
		return v.schemaError(n.location+"/maxLength", offset, fmt.Sprintf("string is longer than %d characters", n.lengthMax))
	}
	return nil
}

func (v *validation) numberValue(n *node, offset int64) error {
	token, err := v.scanner.number(v.numberBuf, v.limits.NumberBytesMax, &v.number)
	v.numberBuf = token[:0]
	if err != nil {
		return err
	}

	integer := v.number.isInteger()
	if !n.types.has(typeNumber) && !(integer && n.types.has(typeInteger)) {
		if integer {
			return v.typeError(n, offset, "integer")
		}
		return v.typeError(n, offset, "number")
	}

	switch {
	case n.minimum != nil && v.number.compare(n.minimum) < 0:
		return v.schemaError(n.location+"/minimum", offset, "number is less than minimum")
	case n.exclusiveMinimum != nil && v.number.compare(n.exclusiveMinimum) <= 0:
		return v.schemaError(n.location+"/exclusiveMinimum", offset, "number is not greater than exclusiveMinimum")
	case n.maximum != nil && v.number.compare(n.maximum) > 0:
		return v.schemaError(n.location+"/maximum", offset, "number is greater than maximum")
	case n.exclusiveMaximum != nil && v.number.compare(n.exclusiveMaximum) >= 0:
		return v.schemaError(n.location+"/exclusiveMaximum", offset, "number is not less than exclusiveMaximum")
	default:
		return nil
	}
}

func (v *validation) typeError(n *node, offset int64, actual string) error {
	return v.schemaError(n.location+"/type", offset, fmt.Sprintf("%s is not allowed; expected %s", actual, n.types))
}

func (v *validation) schemaError(keywordLocation string, offset int64, message string) error {
	return &ValidationError{
		Reason:           ReasonSchema,
		InstanceLocation: v.pointer(),
		KeywordLocation:  keywordLocation,
		Offset:           offset,
		Message:          message,
	}
}

func (v *validation) pointer() string {
	var b strings.Builder
	for _, s := range v.path {
		b.WriteByte('/')
		if s.isName {
			b.WriteString(escapePointer(s.name))
		} else {
			b.WriteString(strconv.FormatInt(s.index, 10))
		}
	}
	return b.String()
}

var pointerEscaper = strings.NewReplacer("~", "~0", "/", "~1")

func escapePointer(token string) string {
	return pointerEscaper.Replace(token)
}
