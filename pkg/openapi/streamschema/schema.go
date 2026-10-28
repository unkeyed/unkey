package streamschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"

	"github.com/unkeyed/unkey/pkg/fault"
)

// Limits bounds the resources one schema compilation or one validation may
// use. Every field must be positive.
type Limits struct {
	// InputBytesMax is the most bytes Validate reads from its input,
	// whitespace included. At most one byte more is read to detect the excess.
	InputBytesMax int64
	// DepthMax is the most objects and arrays a value may be nested in.
	DepthMax int
	// KeyBytesMax is the longest member name, in decoded UTF-8 bytes.
	KeyBytesMax int
	// NumberBytesMax is the longest number token, in bytes.
	NumberBytesMax int
	// KeysRetainedBytesMax caps the accounted size of member names held for
	// duplicate and required checks across all open objects. Each name costs
	// its decoded length plus a fixed overhead of 64 bytes.
	KeysRetainedBytesMax int

	// SchemaBytesMax is the largest schema document Compile accepts.
	SchemaBytesMax int
	// SchemaDepthMax is the most objects and arrays a value in the schema
	// document may be nested in. Each nested subschema adds at least one level,
	// and two through properties.
	SchemaDepthMax int
	// SchemaNodesMax is the most schemas, including boolean schemas, that a
	// document may contain.
	SchemaNodesMax int
}

// DefaultLimits returns the limits used unless a caller has a reason to differ.
func DefaultLimits() Limits {
	return Limits{
		InputBytesMax:        256 << 20,
		DepthMax:             128,
		KeyBytesMax:          1 << 10,
		NumberBytesMax:       1 << 10,
		KeysRetainedBytesMax: 1 << 20,
		SchemaBytesMax:       1 << 20,
		SchemaDepthMax:       128,
		SchemaNodesMax:       10_000,
	}
}

// Schema is a compiled schema. It is immutable and safe for concurrent use.
type Schema struct {
	root   *node
	limits Limits
}

// Compile parses document, a JSON Schema 2020-12 document in JSON form, and
// compiles it for streaming validation under limits.
//
// The document must be strict JSON with no duplicate keys, and must stay
// within the schema limits. A problem with the document is returned as a
// [*CompileError] whose Kind separates unsupported keywords from invalid
// schemas and exceeded limits. Invalid limits are a programming error and are
// returned as a plain error.
func Compile(document []byte, limits Limits) (*Schema, error) {
	if err := limits.check(); err != nil {
		return nil, err
	}
	if len(document) > limits.SchemaBytesMax {
		return nil, compileError(CompileLimit, "", "schema exceeds SchemaBytesMax")
	}

	documentLimits := limits
	documentLimits.InputBytesMax = int64(limits.SchemaBytesMax)
	documentLimits.DepthMax = limits.SchemaDepthMax
	strict := &Schema{root: trueNode, limits: documentLimits}
	if err := strict.Validate(context.Background(), bytes.NewReader(document)); err != nil {
		var parseErr *ValidationError
		if !errors.As(err, &parseErr) {
			return nil, fault.Wrap(err, fault.Internal("failed to read schema document"))
		}
		kind := CompileInvalid
		if parseErr.Reason == ReasonLimit {
			kind = CompileLimit
		}
		return nil, &CompileError{
			Kind:            kind,
			KeywordLocation: parseErr.InstanceLocation,
			Message:         parseErr.Message,
			Err:             parseErr,
		}
	}

	// The strict pass guarantees encoding/json sees no duplicate keys, invalid
	// UTF-8, or unpaired surrogates, so its tree matches the document exactly.
	// UseNumber keeps numbers as their literal text.
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.UseNumber()
	var tree any
	if err := decoder.Decode(&tree); err != nil {
		return nil, fault.Wrap(err, fault.Internal("schema passed strict parsing but failed to decode"))
	}

	c := compiler{nodesMax: limits.SchemaNodesMax, nodes: 0}
	root, err := c.schema(tree, "")
	if err != nil {
		return nil, err
	}
	return &Schema{root: root, limits: limits}, nil
}

// Validate reads one JSON document from r and checks it against s.
//
// It returns nil only after reading r through EOF and finding exactly one
// valid value followed by optional whitespace. The first failure in the input
// is returned as a [*ValidationError]; Validate then stops reading, so r may be
// partly consumed. Errors from r and from ctx are returned wrapped, and
// [errors.Is] matches them. Validate does not close r or retain it.
func (s *Schema) Validate(ctx context.Context, r io.Reader) error {
	if err := ctx.Err(); err != nil {
		return fault.Wrap(err, fault.Internal("validation canceled"))
	}

	v := validation{
		scanner:    newScanner(ctx, r, s.limits.InputBytesMax),
		limits:     s.limits,
		path:       nil,
		namesets:   nil,
		namesBytes: 0,
		nameBuf:    nil,
		numberBuf:  nil,
		number:     decimal{negative: false, digits: nil, exponent: 0},
	}
	err := v.value(s.root, 0)
	if err == nil {
		err = v.scanner.finish()
	}
	if err == nil {
		return nil
	}

	var failure *ValidationError
	if errors.As(err, &failure) && failure.Reason != ReasonSchema {
		failure.InstanceLocation = v.pointer()
	}
	return err
}

func (l Limits) check() error {
	if l.InputBytesMax <= 0 || l.InputBytesMax == math.MaxInt64 ||
		l.DepthMax <= 0 || l.KeyBytesMax <= 0 || l.NumberBytesMax <= 0 ||
		l.KeysRetainedBytesMax <= 0 || l.SchemaBytesMax <= 0 ||
		l.SchemaDepthMax <= 0 || l.SchemaNodesMax <= 0 {
		return fault.New("streamschema limits must be positive and InputBytesMax below math.MaxInt64")
	}
	return nil
}
