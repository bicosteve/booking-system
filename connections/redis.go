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
	options := &redis.Options{
		Addr:         config.Address + ":" + config.Port,
		Password:     config.Password,
		DB:           config.Database,
		ClientName:   config.Name,
		PoolSize:     100,
		PoolTimeout:  time.Second * 20,
		MinIdleConns: 32,
	}

	// Only enable TLS for managed/remote Redis  when a password is set.
	// A local dev Redis speaks plaintext, so forcing TLS makes the handshake
	// hang until the dial timeout fires ("context deadline exceeded").
	if config.Password != "" {
		options.TLSConfig = &tls.Config{}
	}

	client := redis.NewClient(options)

	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	utils.LogInfo(fmt.Sprintf("REDIS: %v", pong), entities.InfoLog)

	return client, nil
}
