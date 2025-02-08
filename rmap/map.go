package rmap

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sovietaced/go-redisson/marshal"
)

// Options for the Map
type Options[K any, V any] struct {
	keyMarshaler   marshal.Marshaler[K]
	valueMarshaler marshal.Marshaler[V]
}

func defaultOptions[K any, V any]() *Options[K, V] {
	opts := &Options[K, V]{}
	WithKeyMarshaler[K, V](&marshal.JsonMarshaler[K]{})(opts)
	WithValueMarshaler[K, V](&marshal.JsonMarshaler[V]{})(opts)
	return opts
}

// Option for the Map
type Option[K any, V any] func(*Options[K, V])

// WithKeyMarshaler allows you to configure how keys are marshaled into strings.
func WithKeyMarshaler[K any, V any](marshaler marshal.Marshaler[K]) Option[K, V] {
	return func(mo *Options[K, V]) {
		mo.keyMarshaler = marshaler
	}
}

// WithValueMarshaler allows you to configure how values are mashaled into strings.
func WithValueMarshaler[K any, V any](marshaler marshal.Marshaler[V]) Option[K, V] {
	return func(mo *Options[K, V]) {
		mo.valueMarshaler = marshaler
	}
}

// Map data structure backed by Redis.
type Map[K any, V any] struct {
	namespace      string
	client         redis.UniversalClient
	keyMarshaler   marshal.Marshaler[K]
	valueMarshaler marshal.Marshaler[V]
}

// NewMap creates a new generic map. Uses JSON key/value marshalers by default.
func NewMap[K any, V any](client redis.UniversalClient, namespace string, options ...Option[K, V]) *Map[K, V] {
	opts := defaultOptions[K, V]()
	for _, option := range options {
		option(opts)
	}

	return &Map[K, V]{client: client, namespace: namespace, keyMarshaler: opts.keyMarshaler, valueMarshaler: opts.valueMarshaler}
}

// Get retrieves a key/value from the map. Returns the potential value, whether it exists, and any errors that occurred.
func (c *Map[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	keyString, err := c.computeKey(ctx, key)
	if err != nil {
		return *new(V), false, fmt.Errorf("computing key: %w", err)
	}

	result := c.client.HGet(ctx, c.namespace, keyString)
	if result.Err() != nil {
		// Missing key
		if result.Err() == redis.Nil {
			return *new(V), false, nil
		}
		return *new(V), false, fmt.Errorf("getting value: %w", err)
	}

	value := new(V)
	if err = c.valueMarshaler.Unmarshal(ctx, result.Val(), value); err != nil {
		return *value, true, fmt.Errorf("unmarshalling value: %w", err)
	}

	return *value, true, nil
}

// Set inserts a key/value into the map. Returns any errors that occurred.
func (c *Map[K, V]) Set(ctx context.Context, key K, value V) error {
	keyString, err := c.computeKey(ctx, key)
	if err != nil {
		return fmt.Errorf("computing key: %w", err)
	}

	marshaledValue, err := c.valueMarshaler.Marshal(ctx, value)
	if err != nil {
		return fmt.Errorf("marshalling value: %w", err)
	}

	result := c.client.HSet(ctx, c.namespace, keyString, marshaledValue)
	if result.Err() != nil {
		return fmt.Errorf("setting key=%s: %w", keyString, err)
	}

	return nil
}

// Del removes a key/value from the map. Returns any errors that occurred. If the key/value does not exist, this is
// a no-op.
func (c *Map[K, V]) Del(ctx context.Context, key K) error {
	keyString, err := c.computeKey(ctx, key)
	if err != nil {
		return fmt.Errorf("computing key: %w", err)
	}

	result := c.client.HDel(ctx, c.namespace, keyString)
	if result.Err() != nil {
		return fmt.Errorf("deleting key=%s: %w", keyString, err)
	}

	return nil
}

// computeKey computes the key to use for a key/value.
func (c *Map[K, V]) computeKey(ctx context.Context, key K) (string, error) {
	marshaledKey, err := c.keyMarshaler.Marshal(ctx, key)
	if err != nil {
		return "", fmt.Errorf("marshalling key: %w", err)
	}

	return marshaledKey, nil
}
