package mutex

import (
	"context"
	"fmt"
	"github.com/benbjohnson/clock"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"log"
	"time"
)

const defaultLeaseDuration = 30 * time.Second
const lockScript = `
	if redis.call('exists', KEYS[1]) == 0 then
		redis.call('set', KEYS[1], ARGV[2])
		redis.call('pexpire', KEYS[1],ARGV[1])
		return 0
	end
	return redis.call('pttl',KEYS[1])`
const unlockScript = `
	if redis.call('exists', KEYS[1]) == 0 then
		return 0;
	end;
	redis.call('del', KEYS[1]);
    redis.call('publish', KEYS[2], ARGV[1]);
	return 1;
`
const extendScript = `
	if redis.call('get', KEYS[1]) == ARGV[2] then
		redis.call('pexpire', KEYS[1], ARGV[1]);
		return 1;
	end;
	return 0;
`
const unlockMsg = "unlocked"

type Mutex struct {
	clock         clock.Clock
	client        redis.UniversalClient
	key           string
	leaseDuration time.Duration
	logger        logr.Logger
}

type Options struct {
	clock         clock.Clock
	leaseDuration time.Duration
	logger        logr.Logger
}

func defaultOptions() *Options {
	opts := &Options{}
	WithClock(clock.New())(opts)
	WithLeaseDuration(defaultLeaseDuration)(opts)
	WithLogger(stdr.New(log.Default()))
	return opts
}

type Option func(*Options)

// WithLeaseDuration specifies the TTL on the underlying Redis cache entry. Practically speaking, this is the upper
// bound on how long a lock will appear to be held when its owner abandons it.
func WithLeaseDuration(leaseDuration time.Duration) Option {
	return func(mo *Options) {
		mo.leaseDuration = leaseDuration
	}
}

// WithClock allows a pluggable clock primarily for unit testing.
func WithClock(clock clock.Clock) Option {
	return func(mo *Options) {
		mo.clock = clock
	}
}

// WithLogger allows a pluggable logger implementation.
func WithLogger(logger logr.Logger) Option {
	return func(mo *Options) {
		mo.logger = logger
	}
}

// NewMutex Creates a new Mutex with the provided options.
func NewMutex(client redis.UniversalClient, key string, options ...Option) *Mutex {
	opts := defaultOptions()
	for _, option := range options {
		option(opts)
	}
	return &Mutex{key: key, client: client, leaseDuration: opts.leaseDuration, clock: opts.clock, logger: opts.logger}
}

// Lock blocks until the lock can be acquired. The function intelligently waits for either the lease duration to expire
// or a pub sub notification that the lock has been released before attempting to acquire the lock.
func (m *Mutex) Lock(ctx context.Context) error {
	success, err := m.TryLock(ctx)
	if err != nil {
		return fmt.Errorf("tryign to acquire lock: %w", err)
	}

	// Lock acquired, can return.
	if success {
		return nil
	}

	// Subscribe to pubsub notifications
	pubSub := m.client.Subscribe(ctx, m.getChannelName())
	pubSubCh := pubSub.Channel()
	defer func() {
		if closeErr := pubSub.Close(); closeErr != nil {
			m.logger.Error(closeErr, "failed to close pub sub", "key", m.key)
		}
	}()

	// Loop that tries to acquire the lock, otherwise waits for the lease duration or a pub sub notification
	for {
		ttl, err := m.doTryLock(ctx)
		if err != nil {
			return fmt.Errorf("trying to acquire lock: %w", err)
		}

		if ttl == 0 {
			return nil
		}

	wait:
		select {
		case <-m.clock.After(time.Duration(ttl) * time.Millisecond):
			break wait
		case msg := <-pubSubCh:
			if msg.Payload == unlockMsg {
				break wait
			}
		case <-ctx.Done():
			return fmt.Errorf("cancelled while waiting for lock: %w", err)
		}
	}
}

// TryLock attempts to acquire the lock but does not block. Returns whether the lock was aquired.
func (m *Mutex) TryLock(ctx context.Context) (bool, error) {
	ttl, err := m.doTryLock(ctx)
	if err != nil {
		return false, fmt.Errorf("trylock failed: %w", err)
	}

	return ttl == 0, nil
}

func (m *Mutex) doTryLock(ctx context.Context) (int64, error) {
	// Create an extension ID so that we only extend this lease if we own the lock
	extensionId := uuid.New()
	ttl, err := m.client.Eval(ctx, lockScript, []string{m.getLockName()}, m.leaseDuration.Milliseconds(), extensionId).Int64()
	if err != nil {
		return 0, err
	}

	// If lock was acquired, kick off lease extender
	if ttl == 0 {
		go m.launchLeaseExtender(extensionId)
	}

	return ttl, nil
}

// Unlock releases the lock. It will also publish an unlock notification on the Redis pub sub channel.
func (m *Mutex) Unlock(ctx context.Context) error {
	_, err := m.client.Eval(ctx, unlockScript, []string{m.getLockName(), m.getChannelName()}, unlockMsg).Int64()
	if err != nil {
		return fmt.Errorf("unlocked failed: %w", err)
	}

	return nil
}

func (m *Mutex) launchLeaseExtender(extensionId uuid.UUID) {
	// Use a fresh root context here
	ctx := context.Background()

	ticker := m.clock.Ticker(m.leaseDuration / 3)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			success, err := m.extendLease(ctx, extensionId)
			if err != nil {
				m.logger.Error(err, "failed to extend lease", "key", m.key)
				return
			}

			// we don't own the lock anymore
			if !success {
				return
			}
		}
	}
}

func (m *Mutex) extendLease(ctx context.Context, acquireId uuid.UUID) (bool, error) {
	result, err := m.client.Eval(ctx, extendScript, []string{m.key}, m.leaseDuration.Milliseconds(), acquireId).Int64()

	if err != nil {
		return false, fmt.Errorf("extending lease: %w", err)
	}

	if result == 1 {
		return true, nil
	}

	return false, nil
}

func (m *Mutex) getLockName() string {
	return fmt.Sprintf("go_redisson_lock:%s", m.key)
}

func (m *Mutex) getChannelName() string {
	return fmt.Sprintf("go_redisson_lock_channel:%s", m.key)
}
