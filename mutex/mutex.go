package mutex

import (
	"context"
	"fmt"
	"github.com/benbjohnson/clock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	return 1;
`
const extendScript = `
	if redis.call('get', KEYS[1]) == ARGV[2] then
		redis.call('pexpire', KEYS[1], ARGV[1]);
		return 1;
	end;
	return 0;
`

type Mutex struct {
	clock         clock.Clock
	client        redis.UniversalClient
	key           string
	leaseDuration time.Duration
}

type Options struct {
	clock         clock.Clock
	leaseDuration time.Duration
}

func defaultOptions() *Options {
	opts := &Options{}
	WithClock(clock.New())(opts)
	WithLeaseDuration(defaultLeaseDuration)(opts)
	return opts
}

type Option func(*Options)

func WithLeaseDuration(leaseDuration time.Duration) Option {
	return func(mo *Options) {
		mo.leaseDuration = leaseDuration
	}
}

func WithClock(clock clock.Clock) Option {
	return func(mo *Options) {
		mo.clock = clock
	}
}

func NewMutex(client redis.UniversalClient, key string, options ...Option) *Mutex {
	opts := defaultOptions()
	for _, option := range options {
		option(opts)
	}
	return &Mutex{key: key, client: client, leaseDuration: opts.leaseDuration, clock: opts.clock}
}

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
	ttl, err := m.client.Eval(ctx, lockScript, []string{m.key}, m.leaseDuration.Milliseconds(), extensionId).Int64()
	if err != nil {
		return 0, err
	}

	// If lock was acquired, kick off lease extender
	if ttl == 0 {
		go m.launchLeaseExtender(extensionId)
	}

	return ttl, nil
}

func (m *Mutex) Unlock(ctx context.Context) error {
	_, err := m.client.Eval(ctx, unlockScript, []string{m.key}).Int64()
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
				//FIXME log
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
