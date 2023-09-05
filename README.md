# go-redisson

[![Test](https://github.com/sovietaced/go-redisson/actions/workflows/ci.yml/badge.svg)](https://github.com/sovietaced/go-redisson/actions/workflows/ci.yml)
 
Distributed data structures backed by Redis. Heavily inspired by [Redisson](https://github.com/redisson/redisson).

## Examples

### Distributed Lock

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
