package proto

import (
	"google.golang.org/protobuf/proto"
)

func Marshal(message proto.Message) ([]byte, error) {
	return proto.Marshal(message)
}

func Unmarshal(data []byte, message proto.Message) error {
	return proto.Unmarshal(data, message)
}
