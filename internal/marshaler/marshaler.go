package marshaler

import (
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/proto"
)

var (
	NotImplProtoMessageError = errors.New("Not implment proto message.")
)

type (
	Marshaler interface {
		Marshal(obj interface{}) ([]byte, error)
		UnMarshal(data []byte, obj interface{}) error
	}

	JsonMarshaler struct {
	}

	ProtoMarshaler struct {
	}
)

func (*JsonMarshaler) Marshal(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}

func (*JsonMarshaler) UnMarshal(data []byte, obj interface{}) error {
	return json.Unmarshal(data, obj)
}

func (*ProtoMarshaler) Marshal(obj interface{}) ([]byte, error) {
	if obj == nil {
		return []byte{}, nil
	}

	body, ok := obj.(proto.Message)
	if !ok {
		return []byte{}, NotImplProtoMessageError
	}

	return proto.Marshal(body)
}

func (*ProtoMarshaler) UnMarshal(data []byte, obj interface{}) error {
	if obj == nil {
		return nil
	}

	body, ok := obj.(proto.Message)
	if !ok {
		return NotImplProtoMessageError
	}

	return proto.Unmarshal(data, body)
}
