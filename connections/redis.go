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

func redisOptions(cfg entities.RedisConfig) *redis.Options {
	var tlsConfig *tls.Config
	if cfg.TLS {
		tlsConfig = &tls.Config{ServerName: cfg.Address}
	}
	return &redis.Options{
		Addr:         cfg.Address + ":" + cfg.Port,
		Password:     cfg.Password,
		DB:           cfg.Database,
		ClientName:   cfg.Name,
		PoolSize:     100,
		PoolTimeout:  time.Second * 20,
		MinIdleConns: 32,
		TLSConfig:    tlsConfig,
	}
}

func NewRedisDB(ctx context.Context, cfg entities.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(redisOptions(cfg))
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	utils.LogInfo(fmt.Sprintf("REDIS: %v", pong), entities.InfoLog)

	return client, nil
}
