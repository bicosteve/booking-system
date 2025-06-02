package utils

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/bicosteve/booking-system/entities"
)

func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")

			if len(header) == 0 {
				LogError("missing authorization header", entities.ErrorLog, http.StatusUnauthorized)
				ErrorJSON(w, errors.New("missing authorization header"), http.StatusUnauthorized)
				return
			}

			parts := strings.Split(header, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				LogError("Invalid authorization header", entities.ErrorLog, http.StatusUnauthorized)
				ErrorJSON(w, errors.New("invalid authorization header"), http.StatusUnauthorized)
				return
			}

			claims, err := verifyAuthToken(parts[1], secret)
			if err != nil {
				LogError("Invalid authorization token", entities.ErrorLog)
				ErrorJSON(w, errors.New("invalid authorization token"), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), entities.UsernameKeyValue, claims.Username)
			ctx = context.WithValue(ctx, entities.IsVendorKeyValue, claims.IsVendor)
			ctx = context.WithValue(ctx, entities.UseridKeyValue, claims.UserID)
			ctx = context.WithValue(ctx, entities.PhoneNumberKeyValue, claims.PhoneNumber)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

func AdminMiddlware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isVendor, ok := r.Context().Value(entities.IsVendorKeyValue).(string)
		if !ok {
			LogError("error getting role from context", entities.ErrorLog)
			ErrorJSON(w, errors.New("an error occured"), http.StatusInternalServerError)
			return
		}

		if isVendor != "YES" {
			LogError("unauthorized access for user", entities.ErrorLog)
			ErrorJSON(w, errors.New("unauthorized access"), http.StatusForbidden)
			return

		}

		next.ServeHTTP(w, r)

	})
}
