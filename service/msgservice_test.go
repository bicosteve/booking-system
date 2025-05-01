package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/repo"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestSubmitMessage(t *testing.T) {
	tests := []struct {
		name      string
		message   entities.SMSPayload
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "successful message submission",
			message: entities.SMSPayload{
				UserID:  "1",
				Message: "test message",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO sms_outbox").
					ExpectExec().
					WithArgs("Your reset token is 'test message' ", "1").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			message: entities.SMSPayload{
				UserID:  "1",
				Message: "test message",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO sms_outbox").
					ExpectExec().
					WithArgs("Your reset token is 'test message' ", "1").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
		{
			name: "prepare statement error",
			message: entities.SMSPayload{
				UserID:  "1",
				Message: "test message",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO sms_outbox").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new mock database
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock database: %v", err)
			}
			defer db.Close()

			_db, _ := redismock.NewClientMock()

			// Set up the mock expectations
			tt.setupMock(mock)

			// Create a new repository and service
			repository := *repo.NewDBRepository(db, _db)
			service := NewUserService(repository)

			// Execute the function
			err = service.SubmitMessage(context.Background(), tt.message)

			// Check if the error matches our expectation
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Ensure all expectations were met
			err = mock.ExpectationsWereMet()
			if err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
