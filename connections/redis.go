package connections

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/redis/go-redis/v9"
)

func NewRedisDB(ctx context.Context, config entities.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         config.Address + ":" + config.Port,
		Username:     config.Name,
		Password:     config.Password,
		DB:           config.Database,
		TLSConfig:    &tls.Config{},
		ClientName:   config.Name,
		PoolSize:     100,
		PoolTimeout:  time.Second * 20,
		MinIdleConns: 32,
	})

	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	utils.LogInfo(fmt.Sprintf("REDIS: %v", pong), entities.InfoLog)

	return client, nil
}
