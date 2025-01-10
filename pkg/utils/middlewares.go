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
				entities.MessageLogs.ErrorLog.Println("Missing Authorization header")
				ErrorJSON(w, errors.New("missing authorization header"), http.StatusUnauthorized)
				return
			}

			parts := strings.Split(header, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				entities.MessageLogs.ErrorLog.Println("Invalid authorization header")
				ErrorJSON(w, errors.New("invalid authorization header"), http.StatusUnauthorized)
				return
			}

			claims, err := verifyAuthToken(parts[1], secret)
			if err != nil {
				entities.MessageLogs.ErrorLog.Println("Invalid authorization token")
				ErrorJSON(w, errors.New("invalid authorization token"), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), entities.UsernameKeyValue, claims.Username)
			ctx = context.WithValue(ctx, entities.IsVendorKeyValue, claims.IsVendor)
			ctx = context.WithValue(ctx, entities.UseridKeyValue, claims.UserID)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

func AdminMiddlware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isVendor, ok := r.Context().Value(entities.IsVendorKeyValue).(string)
		if !ok {
			entities.MessageLogs.ErrorLog.Println("error while getting role from context")
			ErrorJSON(w, errors.New("an error occured"), http.StatusInternalServerError)
			return
		}

		if isVendor != "YES" {
			entities.MessageLogs.ErrorLog.Println("Unauthorized access for user")
			ErrorJSON(w, errors.New("unauthorized access"), http.StatusForbidden)
			return

		}

		next.ServeHTTP(w, r)

	})
}
