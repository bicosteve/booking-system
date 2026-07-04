package controllers

import (
	"context"
	"errors"

	"github.com/bicosteve/booking-system/pkg/health"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// defaultLiveCheckers builds checkers from Base's persistent handles. Disabled
// message brokers are reported as "disabled".
func (b *Base) defaultLiveCheckers() []health.Checker {
	var cs []health.Checker

	cs = append(cs, health.Checker{
		Name: "mysql",
		Ping: func(ctx context.Context) error {
			if b.DB == nil {
				return errors.New("mysql client not initialized")
			}
			return b.DB.PingContext(ctx)
		},
	})

	cs = append(cs, health.Checker{
		Name: "redis",
		Ping: func(ctx context.Context) error {
			if b.Redis == nil {
				return errors.New("redis client not initialized")
			}
			return b.Redis.Ping(ctx).Err()
		},
	})

	if b.RabbitMQStatus == 1 {
		cs = append(cs, health.RabbitProbe(b.rabbitURL, utils.RabbitTLSConfig(b.rabbitCfg)))
	} else {
		cs = append(cs, health.Checker{Name: "rabbitmq", Disabled: true})
	}

	if b.KafkaStatus == 1 && b.KafkaProducer != nil {
		cs = append(cs, health.Checker{
			Name: "kafka",
			Ping: func(ctx context.Context) error {
				ac, err := kafka.NewAdminClientFromProducer(b.KafkaProducer)
				if err != nil {
					return err
				}
				defer ac.Close()
				_, err = ac.GetMetadata(nil, true, 1000)
				return err
			},
		})
	} else {
		cs = append(cs, health.Checker{Name: "kafka", Disabled: true})
	}

	return cs
}

// healthCheckers returns the provider-derived checkers (tests), or the live
// handle checkers (production).
func (b *Base) healthCheckers() []health.Checker {
	if b.checkersProvider != nil {
		return b.checkersProvider()
	}
	return b.defaultLiveCheckers()
}
