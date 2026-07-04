package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func probeDown(t *testing.T, c Checker) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := c.Ping(ctx)
	assert.Error(t, err, c.Name)
}

func TestMySQLProbe_BadAddress(t *testing.T) {
	probeDown(t, MySQLProbe("root:root@tcp(127.0.0.1:1)/nodb?timeout=200ms"))
	assert.Equal(t, "mysql", MySQLProbe("").Name)
}

func TestRedisProbe_BadAddress(t *testing.T) {
	probeDown(t, RedisProbe("127.0.0.1:1"))
	assert.Equal(t, "redis", RedisProbe("").Name)
}

func TestRabbitProbe_BadAddress(t *testing.T) {
	probeDown(t, RabbitProbe("amqp://guest:guest@127.0.0.1:1/"))
	assert.Equal(t, "rabbitmq", RabbitProbe("").Name)
}

func TestKafkaProbe_BadAddress(t *testing.T) {
	probeDown(t, KafkaProbe("127.0.0.1:1"))
	assert.Equal(t, "kafka", KafkaProbe("").Name)
}
