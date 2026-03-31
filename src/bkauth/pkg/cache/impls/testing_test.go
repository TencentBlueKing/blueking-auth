package impls

import (
	"github.com/alicebob/miniredis"
	goredis "github.com/go-redis/redis/v8"
)

func newTestRedisClient() *goredis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}
