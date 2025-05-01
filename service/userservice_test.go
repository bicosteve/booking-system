package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/repo"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestSubmitRegistrationRequest(t *testing.T) {
	tests := []struct {
		name      string
		payload   entities.UserPayload
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful registration",
			payload: entities.UserPayload{
				Email:       "test@example.com",
				PhoneNumber: "1234567890",
				IsVendor:    "NO",
				Password:    "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.* FROM user").
					ExpectQuery().
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				mock.ExpectPrepare("INSERT INTO user").
					ExpectExec().
					WithArgs("test@example.com", "1234567890", "NO", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "user already exists",
			payload: entities.UserPayload{
				Email:       "existing@example.com",
				PhoneNumber: "1234567890",
				IsVendor:    "NO",
				Password:    "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.* FROM user").
					ExpectQuery().
					WithArgs("existing@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			wantErr: true,
			errMsg:  "user already registered",
		},
		{
			name: "database error",
			payload: entities.UserPayload{
				Email:       "test@example.com",
				PhoneNumber: "1234567890",
				IsVendor:    "NO",
				Password:    "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.* FROM user").
					ExpectQuery().
					WithArgs("test@example.com").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
			errMsg:  "sql: connection is already closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock: %v", err)
			}

			defer db.Close()

			_db, _ := redismock.NewClientMock()

			tt.setupMock(mock)
			repository := *repo.NewDBRepository(db, _db)
			service := NewUserService(repository)

			err = service.SubmitRegistrationRequest(context.Background(), tt.payload)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Equal(t, tt.errMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

		})
	}
}

func TestSubmitLoginRequest(t *testing.T) {
	mockTime := time.Now()
	tests := []struct {
		name      string
		payload   entities.UserPayload
		secret    string
		setupMock func(sqlmock.Sqlmock)
		want      string
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful login",
			payload: entities.UserPayload{
				Email:    "test@gmail.com",
				Password: "1234",
			},
			secret: "ctl2k84u3np52g1p2dbg",
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
						"1", "test@gmail.com", "0704961755", "NO",
						"$2a$10$/r5qIMP1AkNOMdr495Ff0eCdrZWyW79Q5E3RxFVgCbk0ret4j4mDa", "",
						mockTime, mockTime, mockTime,
					))
			},
			wantErr: false,
		},
		{
			name: "user not found",
			payload: entities.UserPayload{
				Email:    "nonexistent@example.com",
				Password: "1234",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.* FROM user").
					ExpectQuery().
					WithArgs("nonexistent@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			wantErr: true,
			errMsg:  "user is not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock: %v", err)
			}

			defer db.Close()

			tt.setupMock(mock)

			_db, _ := redismock.NewClientMock()

			repository := *repo.NewDBRepository(db, _db)
			service := NewUserService(repository)

			token, err := service.SubmitLoginRequest(context.Background(), tt.payload, tt.secret)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
				if tt.errMsg != "" {
					assert.Equal(t, tt.errMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

			// mockRepo.AssertExpectations(t)
		})
	}
}

func TestSubmitProfileRequest(t *testing.T) {
	tCreated, _ := time.Parse(time.RFC3339Nano, "2025-03-02T09:20:55.482005+03:00")
	tUpdated, _ := time.Parse(time.RFC3339Nano, "2025-03-02T09:20:55.482005+03:00")
	tPassword, _ := time.Parse(time.RFC3339Nano, "2025-03-02T09:20:55.482005+03:00")
	mockUser := &entities.User{
		ID:                 "1",
		Email:              "test@gmail.com",
		PhoneNumber:        "0704961755",
		IsVender:           "NO",
		Password:           "$2a$10$/r5qIMP1AkNOMdr495Ff0eCdrZWyW79Q5E3RxFVgCbk0ret4j4mDa",
		PasswordResetToken: "",
		CreatedAt:          tCreated,
		UpdatedAt:          tUpdated,
		PasswordInsertedAt: tPassword,
	}

	tests := []struct {
		name      string
		email     string
		setupMock func(mock sqlmock.Sqlmock)
		want      *entities.User
		wantErr   bool
	}{
		{
			name:  "successful profile retrieval",
			email: "test@gmail.com",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectPrepare("SELECT \\* FROM user WHERE email = ?").
					ExpectQuery().
					WithArgs("test@gmail.com").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "email", "phone_number", "isVender",
						"password", "password_reset_token",
						"created_at", "updated_at", "password_inserted_at",
					}).AddRow(
						"1", "test@gmail.com", "0704961755", "NO",
						"$2a$10$/r5qIMP1AkNOMdr495Ff0eCdrZWyW79Q5E3RxFVgCbk0ret4j4mDa", "", tCreated, tUpdated, tPassword,
					))
			},
			want:    mockUser,
			wantErr: false,
		},
		{
			name:  "profile not found",
			email: "nonexistent@example.com",
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectPrepare("SELECT \\* FROM user").
					ExpectQuery().
					WithArgs("nonexistent@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock: %v", err)
			}

			defer db.Close()

			tt.setupMock(mock)

			_db, _ := redismock.NewClientMock()

			repository := *repo.NewDBRepository(db, _db)
			service := NewUserService(repository)

			user, err := service.SubmitProfileRequest(context.Background(), tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, user)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

		})
	}
}

func TestInsertPasswordResetToken(t *testing.T) {
	mockUser := entities.User{
		ID:    "1",
		Email: "test@gmail.com",
	}

	tests := []struct {
		name      string
		user      entities.User
		setupMock func(m sqlmock.Sqlmock)
		wantErr   bool
	}{
		// {
		// 	name: "successful token insertion",
		// 	user: mockUser,
		// 	setupMock: func(m sqlmock.Sqlmock) {
		// 		q := "UPDATE user SET password_reset_token = \\?, updated_at = \\? WHERE email = \\?"
		// 		m.ExpectPrepare(q).
		// 			ExpectExec().
		// 			WithArgs("tokens", time.Now(), "test@gmail.com").
		// 			WillReturnResult(sqlmock.NewResult(1, 1))

		// 		// m.ExpectPrepare("SELECT * FROM user WHERE email = ?").
		// 		// 	ExpectQuery().
		// 		// 	WithArgs("test@gmail.com").
		// 		// 	WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// 	},
		// 	wantErr: false,
		// },
		{
			name: "database error",
			user: mockUser,
			setupMock: func(m sqlmock.Sqlmock) {
				q := "UPDATE user SET password_reset_token = \\?, updated_at = \\? WHERE email = \\?"
				m.ExpectPrepare(q).
					ExpectExec().
					WithArgs("tokens", time.Now(), "test@gmail.com").
					WillReturnError(sql.ErrConnDone)

				// m.("InsertPasswordResetToken", mock.Anything, mock.AnythingOfType("string"), "test@example.com").Return(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock: %v", err)
			}

			defer db.Close()

			_db, _ := redismock.NewClientMock()

			tt.setupMock(mock)

			repository := *repo.NewDBRepository(db, _db)
			service := NewUserService(repository)

			_, err = service.InsertPasswordResetToken(context.Background(), db, tt.user)

			if tt.wantErr {
				assert.Error(t, err)
				// assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				// assert.NotEmpty(t, token)
			}

			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

		})
	}
}
