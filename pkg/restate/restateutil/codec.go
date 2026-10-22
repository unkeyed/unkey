package restateutil

import (
	"fmt"

	"buf.build/go/protovalidate"
	"github.com/restatedev/sdk-go/encoding"
	"google.golang.org/protobuf/proto"
)

type validatedProtoJSON[T proto.Message] struct {
	encoding.Codec
	encoding.CodecMetadata
}

// ValidatedProtoJSON enforces protobuf validation rules before a Restate handler
// runs. T must match the handler's request type; invalid requests fail with 400.
func ValidatedProtoJSON[T proto.Message]() encoding.Codec {
	return validatedProtoJSON[T]{
		Codec:         encoding.ProtoJSONCodec,
		CodecMetadata: encoding.ProtoJSONCodec.(encoding.CodecMetadata),
	}
}

func (c validatedProtoJSON[T]) Unmarshal(data []byte, dst any) error {
	if err := c.Codec.Unmarshal(data, dst); err != nil {
		return err
	}
	message, ok := dst.(*T)
	if !ok {
		return fmt.Errorf("unexpected protobuf decoding target %T", dst)
	}
	return protovalidate.Validate(*message)
}
