package repo

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

func TestSaveTransactions(t *testing.T) {
	data := &entities.TRXPayload{
		RoomID:    10,
		UserID:    5,
		OrderID:   "order-1",
		TrxID:     "trx-1",
		Reference: "ref-1",
		Status:    1,
		Payment:   entities.PaymentBody{Amount: 200},
	}

	tests := []struct {
		name    string
		wantErr bool
		setup   func(mock sqlmock.Sqlmock)
	}{
		{
			name:    "success",
			wantErr: false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO transaction").
					ExpectExec().
					WithArgs(10, 5, "order-1", "trx-1", "ref-1", int64(200), 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:    "prepare error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO transaction").WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name:    "exec error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO transaction").
					ExpectExec().
					WithArgs(10, 5, "order-1", "trx-1", "ref-1", int64(200), 1).
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.setup(mock)
			repo := &Repository{db: db}
			err = repo.SaveTransactions(context.Background(), data)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateTransactions(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		setup   func(mock sqlmock.Sqlmock)
	}{
		{
			name:    "success",
			wantErr: false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE transaction SET status").
					ExpectExec().
					WithArgs(1, "trx-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:    "prepare error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE transaction SET status").WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name:    "exec error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE transaction SET status").
					ExpectExec().
					WithArgs(1, "trx-1").
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			tt.setup(mock)
			repo := &Repository{db: db}
			err = repo.UpdateTransactions(context.Background(), 1, "trx-1")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
