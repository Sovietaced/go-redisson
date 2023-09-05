package mutex

import (
	"context"
	"github.com/benbjohnson/clock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"math/rand"
	"testing"
	"time"
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

	t.Run("Try to take lock when free", func(t *testing.T) {
		mutex := NewMutex(client, RandomLockName())
		success, err := mutex.TryLock(ctx)
		require.NoError(t, err)
		require.True(t, success)
	})

	t.Run("Try to take lock when taken", func(t *testing.T) {
		mutex := NewMutex(client, RandomLockName())
		success, err := mutex.TryLock(ctx)
		require.NoError(t, err)
		require.True(t, success)

		success, err = mutex.TryLock(ctx)
		require.NoError(t, err)
		require.False(t, success)
	})

	t.Run("Try to free lock when taken", func(t *testing.T) {
		mutex := NewMutex(client, RandomLockName())
		success, err := mutex.TryLock(ctx)
		require.NoError(t, err)
		require.True(t, success)
		err = mutex.Unlock(ctx)
		require.NoError(t, err)
	})

	t.Run("Try to free lock when free", func(t *testing.T) {
		mutex := NewMutex(client, RandomLockName())
		err = mutex.Unlock(ctx)
		require.NoError(t, err)
	})

	t.Run("Ensure that lease is extended while lock is held", func(t *testing.T) {
		fakeClock := clock.NewMock()
		mutex := NewMutex(client, RandomLockName(), WithClock(fakeClock))
		success, err := mutex.TryLock(ctx)
		require.NoError(t, err)
		require.True(t, success)

		// Sleep past the lease duration
		fakeClock.Add(time.Minute)

		// Should not be able to acquire the lock again
		success, err = mutex.TryLock(ctx)
		require.NoError(t, err)
		require.False(t, success)
	})

	t.Run("Ensure that lease is not extended if lock is unlocked", func(t *testing.T) {
		fakeClock := clock.NewMock()
		mutex := NewMutex(client, RandomLockName(), WithClock(fakeClock))
		success, err := mutex.TryLock(ctx)
		require.NoError(t, err)
		require.True(t, success)

		// Immediately unlock
		err = mutex.Unlock(ctx)
		require.NoError(t, err)

		// Sleep past the lease duration
		fakeClock.Add(time.Minute)

		// Should be able to acquire the lock again
		success, err = mutex.TryLock(ctx)
		require.NoError(t, err)
		require.True(t, success)
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
