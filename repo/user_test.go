package repo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name    string
		user    entities.UserPayload
		wantErr bool
		setup   func(mock sqlmock.Sqlmock)
	}{
		{
			name: "successful user creation",
			user: entities.UserPayload{
				Email:       "test@example.com",
				PhoneNumber: "1234567890",
				IsVendor:    "false",
				Password:    "password123",
			},
			wantErr: false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO user").
					ExpectExec().
					WithArgs("test@example.com", "1234567890", "false", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "prepare statement error",
			user: entities.UserPayload{
				Email:       "test@example.com",
				PhoneNumber: "1234567890",
				IsVendor:    "false",
				Password:    "password123",
			},
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO user").
					WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name: "execution error",
			user: entities.UserPayload{
				Email:       "test@example.com",
				PhoneNumber: "1234567890",
				IsVendor:    "false",
				Password:    "password123",
			},
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO user").
					ExpectExec().
					WithArgs("test@example.com", "1234567890", "false", sqlmock.AnyArg()).
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock database: %v", err)
			}
			defer db.Close()

			tt.setup(mock)

			repo := &Repository{db: db}
			err = repo.CreateUser(context.Background(), tt.user)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestFindUserByEmail(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		wantExists bool
		wantErr    bool
		setup      func(mock sqlmock.Sqlmock)
	}{
		{
			name:       "user exists",
			email:      "test@example.com",
			wantExists: true,
			wantErr:    false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.*FROM user").
					ExpectQuery().
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
		},
		{
			name:       "user does not exist",
			email:      "nonexistent@example.com",
			wantExists: false,
			wantErr:    false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.*FROM user").
					ExpectQuery().
					WithArgs("nonexistent@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
		},
		{
			name:       "prepare statement error",
			email:      "test@example.com",
			wantExists: false,
			wantErr:    true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT COUNT.*FROM user").
					WillReturnError(sql.ErrConnDone)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock database: %v", err)
			}
			defer db.Close()

			tt.setup(mock)

			repo := &Repository{db: db}
			exists, err := repo.FindUserByEmail(context.Background(), tt.email)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestFindAProfile(t *testing.T) {
	mockTime := time.Now()
	tests := []struct {
		name    string
		email   string
		want    *entities.User
		wantErr bool
		setup   func(mock sqlmock.Sqlmock)
	}{
		{
			name:  "profile found",
			email: "test@example.com",
			want: &entities.User{
				ID:                 "1",
				Email:              "test@example.com",
				PhoneNumber:        "1234567890",
				IsVender:           "false",
				Password:           "hashedpassword",
				PasswordResetToken: "",
				CreatedAt:          mockTime,
				UpdatedAt:          mockTime,
				PasswordInsertedAt: mockTime,
			},
			wantErr: false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT \\* FROM user").
					ExpectQuery().
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "email", "phone_number", "isVender",
						"password", "password_reset_token",
						"created_at", "updated_at", "password_inserted_at",
					}).AddRow(
						"1", "test@example.com", "1234567890", "false",
						"hashedpassword", "",
						mockTime, mockTime, mockTime,
					))
			},
		},
		{
			name:    "profile not found",
			email:   "nonexistent@example.com",
			want:    nil,
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("SELECT \\* FROM user").
					ExpectQuery().
					WithArgs("nonexistent@example.com").
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock database: %v", err)
			}
			defer db.Close()

			tt.setup(mock)

			repo := &Repository{db: db}
			got, err := repo.FindAProfile(context.Background(), tt.email)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInsertPasswordResetToken(t *testing.T) {
	tests := []struct {
		name       string
		resetToken string
		email      string
		wantErr    bool
		setup      func(mock sqlmock.Sqlmock)
	}{
		{
			name:       "successful token insertion",
			resetToken: "reset_token_123",
			email:      "test@example.com",
			wantErr:    false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE user SET password_reset_token").
					ExpectExec().
					WithArgs("reset_token_123", sqlmock.AnyArg(), "test@example.com").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:       "prepare statement error",
			resetToken: "reset_token_123",
			email:      "test@example.com",
			wantErr:    true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE user SET password_reset_token").
					WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name:       "execution error",
			resetToken: "reset_token_123",
			email:      "test@example.com",
			wantErr:    true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE user SET password_reset_token").
					ExpectExec().
					WithArgs("reset_token_123", sqlmock.AnyArg(), "test@example.com").
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock database: %v", err)
			}
			defer db.Close()

			tt.setup(mock)

			repo := &Repository{db: db}
			err = repo.InsertPasswordResetToken(context.Background(), tt.resetToken, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestUpdatePassword(t *testing.T) {
	tests := []struct {
		name        string
		newPassword string
		userID      int
		wantErr     bool
		setup       func(mock sqlmock.Sqlmock)
	}{
		{
			name:        "successful password update",
			newPassword: "newpassword123",
			userID:      1,
			wantErr:     false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE user SET hashed_password").
					ExpectExec().
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:        "prepare statement error",
			newPassword: "newpassword123",
			userID:      1,
			wantErr:     true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE user SET hashed_password").
					WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name:        "execution error",
			newPassword: "newpassword123",
			userID:      1,
			wantErr:     true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE user SET hashed_password").
					ExpectExec().
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock database: %v", err)
			}
			defer db.Close()

			tt.setup(mock)

			repo := &Repository{db: db}
			err = repo.UpdatePassword(context.Background(), &tt.newPassword, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
