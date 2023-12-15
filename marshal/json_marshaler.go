package marshal

import (
	"context"
	"encoding/json"
)

type JsonMarshaler[T any] struct {
}

func (jm *JsonMarshaler[T]) Marshal(ctx context.Context, value T) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (jm *JsonMarshaler[T]) Unmarshal(ctx context.Context, valueString string, value *T) error {
	return json.Unmarshal([]byte(valueString), value)
}
