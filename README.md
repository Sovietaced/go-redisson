# go-redisson

[![Test](https://github.com/sovietaced/go-redisson/actions/workflows/ci.yml/badge.svg)](https://github.com/sovietaced/go-redisson/actions/workflows/ci.yml)
 
Distributed data structures backed by Redis. Heavily inspired by [Redisson](https://github.com/redisson/redisson).

## Examples

### Distributed Lock

The `Mutex` struct aims to provide identical semantics to the [sync.Mutex](https://pkg.go.dev/sync#Mutex) package.

```go
import (
	"github.com/redis/go-redis/v9"
	"github.com/sovietaced/go-redisson/mutex"
)

func main() {
	client := redis.NewClient(&redis.Options{Addr: endpoint})
	mutex := mutex.NewMutex(client, "test")
	err := mutex.Lock(ctx)
	err = mutex.Unlock(ctx)
}
```
