package proto

import (
	"google.golang.org/protobuf/proto"
)

// Marshal serializes a protobuf message into its binary wire format.
// Returns the encoded bytes or an error if the message cannot be serialized.
func Marshal(message proto.Message) ([]byte, error) {
	return proto.Marshal(message)
}

// Unmarshal parses a protobuf binary wire format into the given message.
// The message must be a non-nil pointer to a proto.Message of the appropriate type.
func Unmarshal(data []byte, message proto.Message) error {
	return proto.Unmarshal(data, message)
}
