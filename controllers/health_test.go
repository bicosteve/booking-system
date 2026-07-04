package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bicosteve/booking-system/pkg/health"
	"github.com/stretchr/testify/assert"
)

func newHealthBase(checkers []health.Checker) *Base {
	return &Base{checkersProvider: func() []health.Checker { return checkers }}
}

func TestHealthCheck_AllUp(t *testing.T) {
	base := newHealthBase([]health.Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return nil }},
		{Name: "rabbitmq", Ping: func(context.Context) error { return nil }},
		{Name: "kafka", Ping: func(context.Context) error { return nil }},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/health/test", nil)
	w := httptest.NewRecorder()
	base.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var rep health.Report
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&rep))
	assert.Equal(t, "healthy", rep.Status)
	assert.Len(t, rep.Checks, 4)
}

func TestHealthCheck_OneDown(t *testing.T) {
	base := newHealthBase([]health.Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return errors.New("redis down") }},
		{Name: "rabbitmq", Ping: func(context.Context) error { return nil }},
		{Name: "kafka", Ping: func(context.Context) error { return nil }},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/health/test", nil)
	w := httptest.NewRecorder()
	base.HealthCheck(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var rep health.Report
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&rep))
	assert.Equal(t, "unhealthy", rep.Status)
	var redisResult *health.Result
	for i := range rep.Checks {
		if rep.Checks[i].Name == "redis" {
			redisResult = &rep.Checks[i]
		}
	}
	if assert.NotNil(t, redisResult) {
		assert.Equal(t, "down", redisResult.Status)
	}
}

func TestHealthCheck_DisabledDeps(t *testing.T) {
	base := newHealthBase([]health.Checker{
		{Name: "mysql", Ping: func(context.Context) error { return nil }},
		{Name: "redis", Ping: func(context.Context) error { return nil }},
		{Name: "rabbitmq", Disabled: true},
		{Name: "kafka", Disabled: true},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/health/test", nil)
	w := httptest.NewRecorder()
	base.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var rep health.Report
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&rep))
	assert.Equal(t, "healthy", rep.Status)
	for _, c := range rep.Checks {
		if c.Name == "rabbitmq" || c.Name == "kafka" {
			assert.Equal(t, "disabled", c.Status, c.Name)
		}
	}
}
