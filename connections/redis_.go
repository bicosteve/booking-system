package connections

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/bicosteve/booking-system/entities"
)

func TestNewRedisDB(t *testing.T) {
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

	ctx := context.Background()

	redisDB, err := NewRedisDB(ctx, config)
	if err != nil {
		t.Fatalf("NewRedisDB() returned an error : %v", err)
	}

	if redisDB.ClientID(ctx) == nil {
		t.Error("expected redis client to be initialized but got nil")
	}

}
