package marshal

import (
	"context"
)

type Marshaler[T any] interface {
	Marshal(ctx context.Context, value T) (string, error)
	Unmarshal(ctx context.Context, valueString string, value *T) error
}
