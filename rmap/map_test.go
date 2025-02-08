package rmap

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"math/rand"
	"testing"
)

func TestCache(t *testing.T) {

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to create redis container: %v", err)
	}
	defer func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err.Error())
		}
	}()

	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("failed to get container endpoint: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: endpoint})

	t.Run("get, set, delete string key/value", func(t *testing.T) {
		cache := NewMap[string, string](client, RandomNamespace())

		err = cache.Set(ctx, "key", "value")
		require.NoError(t, err)

		value, exists, err := cache.Get(ctx, "key")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, "value", value)

		err = cache.Del(ctx, "key")
		require.NoError(t, err)

		value, exists, err = cache.Get(ctx, "key")
		require.NoError(t, err)
		require.False(t, exists)
		require.Equal(t, "", value)
	})

}

func RandomNamespace() string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, 20)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
