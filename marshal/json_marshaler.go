package marshal

import (
	"context"
	"encoding/json"
)

// JsonMarshaler is an implementation of Marshaler that uses JSON.
type JsonMarshaler[T any] struct {
}

// Marshal marshals a go struct into a JSON string.
func (jm *JsonMarshaler[T]) Marshal(ctx context.Context, value T) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// Unmarshal unmarshals a go struct from a JSON string into a go struct.
func (jm *JsonMarshaler[T]) Unmarshal(ctx context.Context, valueString string, value *T) error {
	return json.Unmarshal([]byte(valueString), value)
}
