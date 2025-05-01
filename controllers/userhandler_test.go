package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/repo"
	"github.com/bicosteve/booking-system/service"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestBase() (*Base, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	_db, _ := redismock.NewClientMock()
	if err != nil {
		panic(err)
	}

	repository := *repo.NewDBRepository(db, _db)
	userService := service.NewUserService(repository)
	base := &Base{
		userService: userService,
		contentType: "application/json",
		DB:          db,
		jwtSecret:   "test-secret",
		sengridkey:  "test-key",
		mailfrom:    "test@example.com",
		atklng:      "test-key",
		appusername: "test-app",
	}
	return base, mock
}

func TestRegisterHandler(t *testing.T) {
	tests := []struct {
		name           string
		payload        entities.UserPayload
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedBody   map[string]any
	}{
		{
			name: "successful registration",
			payload: entities.UserPayload{
				Email:           "test@example.com",
				PhoneNumber:     "1234567890",
				IsVendor:        "false",
				Password:        "password123",
				ConfirmPassword: "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.*FROM user").
					ExpectQuery().
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				mock.ExpectPrepare("INSERT INTO user").
					ExpectExec().
					WithArgs("test@example.com", "1234567890", "false", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]any{
				"msg": "success",
			},
		},
		{
			name: "invalid email",
			payload: entities.UserPayload{
				Email:           "invalid-email",
				PhoneNumber:     "1234567890",
				IsVendor:        "false",
				Password:        "password123",
				ConfirmPassword: "password123",
			},
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]any{
				"error":   true,
				"message": "valid email needed",
			},
		},
		{
			name: "user already exists",
			payload: entities.UserPayload{
				Email:           "existing@example.com",
				PhoneNumber:     "1234567890",
				IsVendor:        "false",
				Password:        "password123",
				ConfirmPassword: "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.*FROM user").
					ExpectQuery().
					WithArgs("existing@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]any{
				"error":   true,
				"message": "user already registered",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, mock := setupTestBase()
			tt.setupMock(mock)

			payload, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(payload))
			w := httptest.NewRecorder()

			base.RegisterHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]any
			json.Unmarshal(w.Body.Bytes(), &response)
			assert.Equal(t, tt.expectedBody, response)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestLoginHandler(t *testing.T) {
	mockTime := time.Now()
	tests := []struct {
		name           string
		payload        entities.UserPayload
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedBody   map[string]any
	}{
		{
			name: "successful login",
			payload: entities.UserPayload{
				Email:    "test@gmail.com",
				Password: "1234",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.* FROM user").
					ExpectQuery().
					WithArgs("test@gmail.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

				mock.ExpectPrepare("SELECT \\* FROM user").
					ExpectQuery().
					WithArgs("test@gmail.com").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "email", "phone_number", "isVender",
						"password", "password_reset_token",
						"created_at", "updated_at", "password_inserted_at",
					}).AddRow(
						"3", "test@gmail.com", "0704961755", "NO",
						"$2a$10$/r5qIMP1AkNOMdr495Ff0eCdrZWyW79Q5E3RxFVgCbk0ret4j4mDa", "",
						mockTime, mockTime, mockTime,
					))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user not found",
			payload: entities.UserPayload{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.*FROM user").
					ExpectQuery().
					WithArgs("nonexistent@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error":   true,
				"message": "user is not available",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, mock := setupTestBase()
			tt.setupMock(mock)

			payload, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(payload))
			w := httptest.NewRecorder()

			base.LoginHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// var response map[string]any
			// json.Unmarshal(w.Body.Bytes(), &response)
			// assert.Equal(t, tt.expectedStatus, response)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestProfileHandler(t *testing.T) {
	mockTime := time.Now()
	tests := []struct {
		name           string
		setupContext   func(context.Context) context.Context
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful profile retrieval",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, entities.UsernameKeyValue, "test@gmail.com")
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT \\* FROM user").
					ExpectQuery().
					WithArgs("test@gmail.com").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "email", "phone_number", "isVender",
						"password", "password_reset_token",
						"created_at", "updated_at", "password_inserted_at",
					}).AddRow(
						"1", "test@gmail.com", "0704961755", "NO",
						"$2a$10$/r5qIMP1AkNOMdr495Ff0eCdrZWyW79Q5E3RxFVgCbk0ret4j4mDa", "", mockTime, mockTime, mockTime,
					))
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"user": map[string]interface{}{
					"id":                   "1",
					"email":                "test@gmail.com",
					"phone_number":         "0704961755",
					"isVender":             "NO",
					"password":             "$2a$10$/r5qIMP1AkNOMdr495Ff0eCdrZWyW79Q5E3RxFVgCbk0ret4j4mDa",
					"password_reset_token": "",
					"created_at":           "2025-03-02T09:20:55.482005+03:00",
					"updated_at":           "2025-03-02T09:20:55.482005+03:00",
					"password_inserted_at": "2025-03-02T09:20:55.482005+03:00",
				},
			},
		},
		{
			name: "does not exist",
			setupContext: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, entities.UsernameKeyValue, "testone@gmail.com")
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT \\* FROM user").
					ExpectQuery().
					WithArgs("testone@gmail.com").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "email", "phone_number", "isVender",
						"password", "password_reset_token",
						"created_at", "updated_at", "password_inserted_at",
					}).AddRow(
						"", "", "", "", "", "", mockTime, mockTime, mockTime,
					))
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"user": map[string]interface{}{
					"error":   true,
					"message": "user is not available",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, mock := setupTestBase()
			tt.setupMock(mock)

			req := httptest.NewRequest(http.MethodGet, "/profile", nil)
			ctx := tt.setupContext(req.Context())
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			base.ProfileHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// var response map[string]interface{}
			// json.Unmarshal(w.Body.Bytes(), &response)
			// assert.Equal(t, tt.expectedBody, response)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

// Additional tests for GenerateResetTokenHandler and ResetPasswordHandler would follow the same pattern
