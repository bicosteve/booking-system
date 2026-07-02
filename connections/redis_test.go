package connections

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

// NewRedisDB only enables TLS when a password is configured. A local/dev Redis
// (no password) connects over plaintext, so a connection to a plaintext miniredis
// server should succeed.

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
	// When a password is set, TLS is enabled. A plaintext miniredis cannot
	// complete the TLS handshake, so Ping fails.
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
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := NewRedisDB(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, client)
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
