package hydra

import (
	"encoding/json"
)

// Marshaller defines the interface for serializing workflow payloads and step results.
//
// The marshaller is responsible for converting Go values to and from byte arrays
// for storage in the database. Custom marshallers can be implemented to support
// different serialization formats like Protocol Buffers, MessagePack, or custom
// binary formats.
//
// Implementations must ensure that:
// - Marshal and Unmarshal are inverse operations
// - The same input always produces the same output (deterministic)
// - All workflow payload types are supported
// - Error handling is consistent and informative
type Marshaller interface {
	// Marshal converts a Go value to bytes for storage.
	// The value may be any type used in workflow payloads or step results.
	Marshal(v any) ([]byte, error)

	// Unmarshal converts stored bytes back to a Go value.
	// The target value should be a pointer to the desired type.
	Unmarshal(data []byte, v any) error
}

// JSONMarshaller implements Marshaller using standard Go JSON encoding.
//
// This is the default marshaller used by Hydra engines. It provides
// good compatibility with most Go types and is human-readable for
// debugging purposes.
//
// Limitations:
// - Cannot handle circular references
// - Maps with non-string keys are not supported
// - Precision may be lost for large integers
// - Custom types need JSON tags for proper serialization
type JSONMarshaller struct{}

// NewJSONMarshaller creates a new JSON-based marshaller.
//
// This is the default marshaller used when no custom marshaller
// is provided to the engine configuration.
func NewJSONMarshaller() Marshaller {
	return &JSONMarshaller{}
}

// Marshal implements Marshaller.Marshal using encoding/json.
func (j *JSONMarshaller) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal implements Marshaller.Unmarshal using encoding/json.
func (j *JSONMarshaller) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
