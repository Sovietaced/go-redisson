package redisson

import (
	"github.com/redis/go-redis/v9"
	"go-redisson/mutex"
)

type Redisson struct {
	redisClient *redis.Client
}

func New(redisClient *redis.Client) *Redisson {
	return &Redisson{redisClient: redisClient}
}

func (r *Redisson) NewMutex(key string, options ...mutex.Option) *mutex.Mutex {
	return mutex.NewMutex(r.redisClient, key, options...)
}
