# go-redisson

[![Test](https://github.com/sovietaced/go-redisson/actions/workflows/ci.yml/badge.svg)](https://github.com/sovietaced/go-redisson/actions/workflows/ci.yml)
[![GoDoc](https://godoc.org/github.com/sovietaced/go-redisson?status.png)](http://godoc.org/github.com/sovietaced/go-redisson)

 
Distributed data structures backed by Redis. Heavily inspired by [Redisson](https://github.com/redisson/redisson).

## Examples

### Distributed Lock

The `Mutex` struct aims to provide identical semantics to the [sync.Mutex](https://pkg.go.dev/sync#Mutex) package.

```go
ctx := context.Background()
client := redis.NewClient(&redis.Options{Addr: endpoint})
mutex := mutex.NewMutex(client, "test")
err := mutex.Lock(ctx)
err = mutex.Unlock(ctx)
```

### Distributed Map

The `Mapp` struct aims to provide similar semantics to native Go map.

```go
ctx := context.Background()
client := redis.NewClient(&redis.Options{Addr: endpoint})
m := mapp.NewMapp(client)
err = m.Set(ctx, "key", "value")
value, exists, err := m.Get(ctx, "key")
```

The `Mapp` struct supports generics so you can use any struct you'd like for the key/value. By default, structs will be
marshalled using json but can be configured with key/value marshalers. 

```go

m := mapp.NewMapp(client, mapp.WithKeyMarshaler(...), mapp.WithValueMarshaller(...))
```
