package connections

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

// NewRedisDB enables TLS only when cfg.TLS is set. A local/dev Redis (TLS off)
// connects over plaintext, so a connection to a plaintext miniredis server
// should succeed.

func TestNewRedisDB_PlaintextServerSucceeds(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start mini redis: %v", err)
	}
	defer mr.Close()

	config := entities.RedisConfig{
		Address:  mr.Host(),
		Port:     mr.Port(),
		Password: "",
		Database: 0,
		Name:     "test-redis",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := NewRedisDB(ctx, config)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	if client != nil {
		assert.NotNil(t, client.ClientID(ctx))
		_ = client.Close()
	}
}

func TestNewRedisDB_TLSAgainstPlaintextFails(t *testing.T) {
	// With TLS explicitly enabled, a plaintext miniredis cannot complete the
	// TLS handshake, so Ping fails.
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start mini redis: %v", err)
	}
	defer mr.Close()
	mr.RequireAuth("somepassword")

	config := entities.RedisConfig{
		Address:  mr.Host(),
		Port:     mr.Port(),
		Password: "somepassword",
		Database: 0,
		Name:     "test-redis",
		TLS:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := NewRedisDB(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestRedisOptions_NoTLS(t *testing.T) {
	opts := redisOptions(entities.RedisConfig{Address: "127.0.0.1", Port: "6379", Password: "x", Name: "n"})
	assert.Nil(t, opts.TLSConfig)
	assert.Equal(t, "127.0.0.1:6379", opts.Addr)
	assert.Equal(t, "x", opts.Password)
	assert.Equal(t, "n", opts.ClientName)
}

func TestRedisOptions_TLS(t *testing.T) {
	opts := redisOptions(entities.RedisConfig{Address: "us1-x.upstash.io", Port: "6379", TLS: true})
	if assert.NotNil(t, opts.TLSConfig) {
		assert.Equal(t, "us1-x.upstash.io", opts.TLSConfig.ServerName)
	}
}

func TestNewRedisDB_UnreachableHost(t *testing.T) {
	config := entities.RedisConfig{
		Address:  "127.0.0.1",
		Port:     "1",
		Password: "",
		Database: 0,
		Name:     "test-redis",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := NewRedisDB(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, client)
}
