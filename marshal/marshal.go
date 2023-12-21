package marshal

import (
	"context"
)

// Marshaler is an interface contract for marshalling values from Go structs into strings that can be stored in Redis.
type Marshaler[T any] interface {
	// Marshal marshals a go struct into a string.
	Marshal(ctx context.Context, value T) (string, error)
	// Unmarshal unmarshals a string into a go struct.
	Unmarshal(ctx context.Context, valueString string, value *T) error
}
