package mapp

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/sovietaced/go-redisson/marshal"
)

type Options[K any, V any] struct {
	namespace      string
	keyMarshaler   marshal.Marshaler[K]
	valueMarshaler marshal.Marshaler[V]
}

func defaultOptions[K any, V any]() *Options[K, V] {
	opts := &Options[K, V]{}
	WithKeyMarshaler[K, V](&marshal.JsonMarshaler[K]{})(opts)
	WithValueMarshaler[K, V](&marshal.JsonMarshaler[V]{})(opts)
	return opts
}

type Option[K any, V any] func(*Options[K, V])

func WithNamespace[K any, V any](namespace string) Option[K, V] {
	return func(mo *Options[K, V]) {
		mo.namespace = namespace
	}
}

func WithKeyMarshaler[K any, V any](marshaler marshal.Marshaler[K]) Option[K, V] {
	return func(mo *Options[K, V]) {
		mo.keyMarshaler = marshaler
	}
}

func WithValueMarshaler[K any, V any](marshaler marshal.Marshaler[V]) Option[K, V] {
	return func(mo *Options[K, V]) {
		mo.valueMarshaler = marshaler
	}
}

type Mapp[K any, V any] struct {
	namespace      string
	client         redis.UniversalClient
	keyMarshaler   marshal.Marshaler[K]
	valueMarshaler marshal.Marshaler[V]
}

func NewMapp[K any, V any](client redis.UniversalClient, options ...Option[K, V]) *Mapp[K, V] {
	opts := defaultOptions[K, V]()
	for _, option := range options {
		option(opts)
	}

	return &Mapp[K, V]{client: client, keyMarshaler: opts.keyMarshaler, valueMarshaler: opts.valueMarshaler, namespace: opts.namespace}
}

func (c *Mapp[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	keyString, err := c.computeKey(ctx, key)
	if err != nil {
		return *new(V), false, fmt.Errorf("computing key: %w", err)
	}

	result := c.client.Get(ctx, keyString)
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

func (c *Mapp[K, V]) Set(ctx context.Context, key K, value V) error {
	keyString, err := c.computeKey(ctx, key)
	if err != nil {
		return fmt.Errorf("computing key: %w", err)
	}

	marshaledValue, err := c.valueMarshaler.Marshal(ctx, value)
	if err != nil {
		return fmt.Errorf("marshalling value: %w", err)
	}

	result := c.client.Set(ctx, keyString, marshaledValue, 0)
	if result.Err() != nil {
		return fmt.Errorf("setting key=%s: %w", keyString, err)
	}

	return nil
}

func (c *Mapp[K, V]) Del(ctx context.Context, key K) error {
	keyString, err := c.computeKey(ctx, key)
	if err != nil {
		return fmt.Errorf("computing key: %w", err)
	}

	result := c.client.Del(ctx, keyString)
	if result.Err() != nil {
		return fmt.Errorf("deleting key=%s: %w", keyString, err)
	}

	return nil
}

func (c *Mapp[K, V]) computeKey(ctx context.Context, key K) (string, error) {
	marshaledKey, err := c.keyMarshaler.Marshal(ctx, key)
	if err != nil {
		return "", fmt.Errorf("marshalling key: %w", err)
	}

	if len(c.namespace) > 0 {
		return fmt.Sprintf("%s:%s", c.namespace, marshaledKey), nil
	}

	return marshaledKey, nil
}
