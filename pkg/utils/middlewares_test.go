package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

func contextWithVendor(r *http.Request, val string) context.Context {
	return context.WithValue(r.Context(), entities.IsVendorKeyValue, val)
}

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret"

	// a downstream handler that records the context values it received
	makeNext := func(captured *entities.Claims) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			captured.Username, _ = r.Context().Value(entities.UsernameKeyValue).(string)
			captured.IsVendor, _ = r.Context().Value(entities.IsVendorKeyValue).(string)
			captured.UserID, _ = r.Context().Value(entities.UseridKeyValue).(string)
			captured.PhoneNumber, _ = r.Context().Value(entities.PhoneNumberKeyValue).(string)
			w.WriteHeader(http.StatusOK)
		})
	}

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		var captured entities.Claims
		AuthMiddleware(secret)(makeNext(&captured)).ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Token abc")
		w := httptest.NewRecorder()

		var captured entities.Claims
		AuthMiddleware(secret)(makeNext(&captured)).ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer not.a.valid.token")
		w := httptest.NewRecorder()

		var captured entities.Claims
		AuthMiddleware(secret)(makeNext(&captured)).ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid token sets context", func(t *testing.T) {
		user := entities.User{
			ID:          "5",
			Email:       "user@example.com",
			IsVender:    "YES",
			PhoneNumber: "0700000000",
		}
		token, err := GenerateAuthToken(user, secret)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		var captured entities.Claims
		AuthMiddleware(secret)(makeNext(&captured)).ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "user@example.com", captured.Username)
		assert.Equal(t, "YES", captured.IsVendor)
		assert.Equal(t, "5", captured.UserID)
		assert.Equal(t, "0700000000", captured.PhoneNumber)
	})
}

func TestAdminMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("no role in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		AdminMiddlware(next).ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("non vendor forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := contextWithVendor(req, "NO")
		w := httptest.NewRecorder()

		AdminMiddlware(next).ServeHTTP(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("vendor allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := contextWithVendor(req, "YES")
		w := httptest.NewRecorder()

		AdminMiddlware(next).ServeHTTP(w, req.WithContext(ctx))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
