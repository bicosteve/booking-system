package repo

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

func TestAddSMSOutbox(t *testing.T) {
	tests := []struct {
		name    string
		msg     entities.SMSPayload
		wantErr bool
		setup   func(mock sqlmock.Sqlmock)
	}{
		{
			name: "successful message insertion",
			msg: entities.SMSPayload{
				Message: "123456",
				UserID:  "1",
			},
			wantErr: false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO sms_outbox").
					ExpectExec().
					WithArgs("Your reset token is '123456' ", "1").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "prepare statement error",
			msg: entities.SMSPayload{
				Message: "123456",
				UserID:  "1",
			},
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO sms_outbox").
					WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name: "execution error",
			msg: entities.SMSPayload{
				Message: "123456",
				UserID:  "1",
			},
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO sms_outbox").
					ExpectExec().
					WithArgs("Your reset token is '123456' ", "1").
					WillReturnError(sql.ErrNoRows)
			},
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

			// Set up the mock expectations
			tt.setup(mock)

			// Create a new repository with the mock db
			repo := &Repository{
				db: db,
			}

			// Execute the function
			err = repo.AddSMSOutbox(context.Background(), tt.msg)

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
