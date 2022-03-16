package redis

import (
	"BlueBell2/settings"
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// 声明一个全局rdb变量
var rdb *redis.Client

func Init(conf *settings.RedisConfig) error {
	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d",
			conf.Host,
			conf.Port,
		),
		Password: conf.Password,
		DB:       conf.Db,
		PoolSize: conf.PoolSize, // 连接池大小
	})
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	_, err := rdb.Ping(ctx).Result()
	return err
}

func Close() {
	_ = rdb.Close()
}
