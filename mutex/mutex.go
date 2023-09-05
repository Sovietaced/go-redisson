package mutex

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

const defaultLeaseDuration = 30 * time.Second
const lockScript = `
	if redis.call('exists',KEYS[1]) == 0 then
		redis.call('set',KEYS[1], 1)
		redis.call('pexpire',KEYS[1],ARGV[1])
		return 0
	end
	return redis.call('pttl',KEYS[1])`
const unlockScript = `
	if (redis.call('exists', KEYS[1]) == 0) then
		return 0;
	end;
	redis.call('del', KEYS[1]);
	return 1;
`
const extendScript = `
	if (redis.call('exists', KEYS[1]) == 1) then
		redis.call('pexpire', KEYS[1], ARGV[1]);
		return 1;
	end;
	return 0;
`

type Mutex struct {
	client                  redis.UniversalClient
	key                     string
	leaseDuration           time.Duration
	leaseExtenderCancelFunc context.CancelFunc
}

type Options struct {
	leaseDuration time.Duration
}

func defaultOptions() *Options {
	opts := &Options{}
	WithLeaseDuration(defaultLeaseDuration)(opts)
	return opts
}

type Option func(*Options)

func WithLeaseDuration(leaseDuration time.Duration) Option {
	return func(mo *Options) {
		mo.leaseDuration = leaseDuration
	}
}

func NewMutex(client redis.UniversalClient, key string, options ...Option) *Mutex {
	opts := defaultOptions()
	for _, option := range options {
		option(opts)
	}
	return &Mutex{key: key, client: client, leaseDuration: opts.leaseDuration}
}

func (m *Mutex) TryLock(ctx context.Context) (bool, error) {
	ttl, err := m.doTryLock(ctx)
	if err != nil {
		return false, fmt.Errorf("trylock failed: %w", err)
	}

	return ttl == 0, nil
}

func (m *Mutex) doTryLock(ctx context.Context) (int64, error) {
	ttl, err := m.client.Eval(ctx, lockScript, []string{m.key}, m.leaseDuration.Milliseconds()).Int64()
	if err != nil {
		return 0, err
	}

	// If lock was acquired, kick off lease extender
	if ttl == 0 {
		go m.launchLeaseExtender()
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

func (m *Mutex) launchLeaseExtender() {
	// Use a fresh root context here so that we explicitly manage the cancellation
	leaseExtenderCtx, cancelFunc := context.WithCancel(context.Background())
	m.leaseExtenderCancelFunc = cancelFunc
	m.leaseExtensionLoop(leaseExtenderCtx)
}

func (m *Mutex) leaseExtensionLoop(ctx context.Context) {
	ticker := time.NewTicker(m.leaseDuration / 3)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			success, err := m.extendLease(ctx)
			if err != nil {
				//FIXME log
				return
			}

			// lock no longer exists so shut down the goroutine
			if !success {
				if m.leaseExtenderCancelFunc != nil {
					m.leaseExtenderCancelFunc()
				}
				return
			}
		case <-ctx.Done():
			// Context has been cancelled. Nothing to do...
			return
		}
	}
}

func (m *Mutex) extendLease(ctx context.Context) (bool, error) {
	result, err := m.client.Eval(ctx, extendScript, []string{m.key}, m.leaseDuration.Milliseconds()).Int64()

	if err != nil {
		return false, fmt.Errorf("extending lease: %w", err)
	}

	if result == 1 {
		return true, nil
	}

	return false, nil
}

func (m *Mutex) stopLeaseExtender() {
	if m.leaseExtenderCancelFunc != nil {
		m.leaseExtenderCancelFunc()
	}
}
