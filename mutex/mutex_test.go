package mutex

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"math/rand"
	"testing"
)

func TestMutex(t *testing.T) {

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
		t.Error(err)
	}
	defer func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err.Error())
		}
	}()

	endpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		t.Error(err)
	}

	client := redis.NewClient(&redis.Options{Addr: endpoint})

	t.Run("Try to take lock when free", func(t *testing.T) {
		mutex := NewMutex(client, RandomLockName())
		success, err := mutex.TryLock(ctx)
		require.NoError(t, err)
		require.True(t, success)
		mutex.Unlock(ctx)
	})

	t.Run("Try to take lock when taken", func(t *testing.T) {
		mutex := NewMutex(client, RandomLockName())
		success, err := mutex.TryLock(ctx)
		require.NoError(t, err)
		require.True(t, success)
		defer mutex.Unlock(ctx)

		success, err = mutex.TryLock(ctx)
		require.NoError(t, err)
		require.False(t, success)
	})

}

func RandomLockName() string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, 20)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
