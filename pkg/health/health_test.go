package health

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheck_AllUp(t *testing.T) {
	checkers := []Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return nil }},
		{Name: "rabbitmq", Ping: func(context.Context) error { return nil }},
		{Name: "kafka", Ping: func(context.Context) error { return nil }},
	}
	r := Check(context.Background(), checkers)
	assert.Equal(t, "healthy", r.Status)
	for _, c := range r.Checks {
		assert.Equal(t, "up", c.Status, c.Name)
		assert.Empty(t, c.Error)
	}
}

func TestCheck_OneDown(t *testing.T) {
	checkers := []Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return errors.New("redis down") }},
		{Name: "rabbitmq", Ping: func(context.Context) error { return nil }},
		{Name: "kafka", Ping: func(context.Context) error { return nil }},
	}
	r := Check(context.Background(), checkers)
	assert.Equal(t, "unhealthy", r.Status)
	var redisResult *Result
	for i := range r.Checks {
		if r.Checks[i].Name == "redis" {
			redisResult = &r.Checks[i]
		}
	}
	if assert.NotNil(t, redisResult) {
		assert.Equal(t, "down", redisResult.Status)
		assert.Contains(t, redisResult.Error, "redis down")
	}
}

func TestCheck_Disabled(t *testing.T) {
	called := false
	checkers := []Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return nil }},
		{Name: "rabbitmq", Disabled: true, Ping: func(context.Context) error { called = true; return nil }},
		{Name: "kafka", Disabled: true, Ping: func(context.Context) error { called = true; return nil }},
	}
	r := Check(context.Background(), checkers)
	assert.Equal(t, "healthy", r.Status)
	assert.False(t, called, "disabled checkers must not be executed")
	for _, c := range r.Checks {
		if c.Name == "rabbitmq" || c.Name == "kafka" {
			assert.Equal(t, "disabled", c.Status, c.Name)
		} else {
			assert.Equal(t, "up", c.Status, c.Name)
		}
	}
}

func TestCheck_NilPing(t *testing.T) {
	checkers := []Checker{{Name: "mysql", Ping: nil}}
	r := Check(context.Background(), checkers)
	assert.Equal(t, "unhealthy", r.Status)
	assert.Equal(t, "down", r.Checks[0].Status)
	assert.Contains(t, r.Checks[0].Error, "ping function not configured")
}

func TestAwait_SuccessAfterRetries(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	c := Checker{Name: "mysql", Ping: func(context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls < 3 {
			return errors.New("not ready")
		}
		return nil
	}}
	err := Await(context.Background(), []Checker{c}, 10*time.Millisecond, 2*time.Second)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, calls, 3)
}

func TestAwait_Timeout(t *testing.T) {
	c := Checker{Name: "mysql", Ping: func(context.Context) error { return errors.New("nope") }}
	err := Await(context.Background(), []Checker{c}, 10*time.Millisecond, 50*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not ready after")
	assert.Contains(t, err.Error(), "mysql")
}

func TestAwait_CancelledContext(t *testing.T) {
	c := Checker{Name: "mysql", Ping: func(context.Context) error { return errors.New("nope") }}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Await(ctx, []Checker{c}, 10*time.Millisecond, 200*time.Millisecond)
	assert.Error(t, err)
}

func TestAwait_EmptyCheckers(t *testing.T) {
	err := Await(context.Background(), nil, 10*time.Millisecond, 100*time.Millisecond)
	assert.NoError(t, err)
}
