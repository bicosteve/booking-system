package connections

import (
	"context"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/redis/go-redis/v9"
)

type Redisdb struct {
	Client *redis.Client
	ctx    context.Context
	Config entities.RedisConfig
}

func NewRedisDB(ctx context.Context, config entities.RedisConfig) (Redisdb, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         config.Address + ":" + config.Port,
		Password:     config.Password,
		DB:           config.Database,
		ClientName:   config.Name,
		PoolSize:     1000,
		PoolTimeout:  time.Second * 5,
		MinIdleConns: 32,
	})

	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return Redisdb{}, err
	}

	entities.MessageLogs.InfoLog.Printf("REDIS: %v", pong)

	redis := Redisdb{Client: client, ctx: ctx, Config: config}

	return redis, nil
}
