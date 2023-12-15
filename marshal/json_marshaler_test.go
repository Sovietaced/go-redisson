package marshal

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestJsonMarshaler(t *testing.T) {

	ctx := context.Background()

	t.Run("test marshal and unmarshal", func(t *testing.T) {
		marshaler := JsonMarshaler[string]{}

		marshalled, err := marshaler.Marshal(ctx, "test")
		require.NoError(t, err)
		require.Equal(t, "\"test\"", marshalled)

		var unmarshalled string
		err = marshaler.Unmarshal(ctx, marshalled, &unmarshalled)
		require.NoError(t, err)

		require.Equal(t, "test", unmarshalled)
	})
}
