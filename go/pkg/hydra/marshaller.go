package hydra

import "encoding/json"

type Marshaller interface {
	Marshal(v any) ([]byte, error)

	Unmarshal(data []byte, v any) error
}

type JSONMarshaller struct{}

func NewJSONMarshaller() Marshaller {
	return &JSONMarshaller{}
}

func (j *JSONMarshaller) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (j *JSONMarshaller) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
