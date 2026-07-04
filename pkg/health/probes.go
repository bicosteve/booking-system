package health

import (
	"context"
	"crypto/tls"
	"database/sql"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
)

// MySQLProbe opens a throwaway *sql.DB against dsn and pings it.
func MySQLProbe(dsn string) Checker {
	return Checker{
		Name: "mysql",
		Ping: func(ctx context.Context) error {
			db, err := sql.Open("mysql", dsn)
			if err != nil {
				return err
			}
			defer db.Close()
			return db.PingContext(ctx)
		},
	}
}

// RedisProbe creates a throwaway redis client against addr (host:port) and pings it.
func RedisProbe(addr string) Checker {
	return Checker{
		Name: "redis",
		Ping: func(ctx context.Context) error {
			cli := redis.NewClient(&redis.Options{Addr: addr})
			defer cli.Close()
			return cli.Ping(ctx).Err()
		},
	}
}

// RabbitProbe dials the provided amqp URL once and closes the connection.
// When tlsConfig != nil it uses amqp.DialTLS, else amqp.Dial. Honors ctx
// cancellation so it never blocks longer than the caller's deadline.
func RabbitProbe(url string, tlsConfig *tls.Config) Checker {
	return Checker{
		Name: "rabbitmq",
		Ping: func(ctx context.Context) error {
			done := make(chan error, 1)
			go func() {
				var conn *amqp.Connection
				var err error
				if tlsConfig != nil {
					conn, err = amqp.DialTLS(url, tlsConfig)
				} else {
					conn, err = amqp.Dial(url)
				}
				if err != nil {
					done <- err
					return
				}
				done <- conn.Close()
			}()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-done:
				return err
			}
		},
	}
}

// KafkaProbe uses an ephemeral AdminClient built from cm to fetch cluster metadata,
// a real broker round-trip. The metadata timeout is capped by ctx's deadline.
func KafkaProbe(cm *kafka.ConfigMap) Checker {
	return Checker{
		Name: "kafka",
		Ping: func(ctx context.Context) error {
			ac, err := kafka.NewAdminClient(cm)
			if err != nil {
				return err
			}
			defer ac.Close()
			timeoutMs := 1000
			if dl, ok := ctx.Deadline(); ok {
				if d := time.Until(dl).Milliseconds(); d > 0 && d < int64(timeoutMs) {
					timeoutMs = int(d)
				}
			}
			_, err = ac.GetMetadata(nil, true, timeoutMs)
			return err
		},
	}
}
