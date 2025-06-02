package connections

import (
	"context"
	"fmt"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/redis/go-redis/v9"
)

func NewRedisDB(ctx context.Context, config entities.RedisConfig) (*redis.Client, error) {
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
		return nil, err
	}

	utils.LogInfo(fmt.Sprintf("REDIS: %v", pong), entities.InfoLog)

	return client, nil
}
