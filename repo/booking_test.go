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

func bkIntPtr(i int) *int { return &i }

func TestCreateABooking(t *testing.T) {
	days, userID, roomID, status := 2, 5, 10, 0

	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectPrepare("UPDATE room")
		mock.ExpectPrepare("INSERT INTO booking")
		mock.ExpectExec("UPDATE room").
			WithArgs(roomID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO booking").
			WithArgs(days, userID, roomID, status).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		repo := &Repository{db: db}
		data := entities.BookingPayload{
			Days:   &days,
			UserID: &userID,
			RoomID: &roomID,
			Status: &status,
		}
		err = repo.CreateABooking(context.Background(), data)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("begin error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		repo := &Repository{db: db}
		data := entities.BookingPayload{Days: &days, UserID: &userID, RoomID: &roomID, Status: &status}
		err = repo.CreateABooking(context.Background(), data)
		assert.Error(t, err)
	})

	t.Run("no room updated rolls back", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectPrepare("UPDATE room")
		mock.ExpectPrepare("INSERT INTO booking")
		mock.ExpectExec("UPDATE room").
			WithArgs(roomID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // no rows affected
		mock.ExpectRollback()

		repo := &Repository{db: db}
		data := entities.BookingPayload{Days: &days, UserID: &userID, RoomID: &roomID, Status: &status}
		err = repo.CreateABooking(context.Background(), data)
		assert.Error(t, err)
	})
}

func TestGetABooking(t *testing.T) {
	mockTime := time.Now()

	t.Run("found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT \\* FROM booking").
			ExpectQuery().
			WithArgs(1, 2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "days", "user_id", "room_id", "created_at", "updated_at"}).
				AddRow(1, 3, 2, 1, mockTime, mockTime))

		repo := &Repository{db: db}
		booking, err := repo.GetABooking(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, 1, booking.ID)
		assert.Equal(t, 3, booking.Days)
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT \\* FROM booking").
			ExpectQuery().
			WithArgs(1, 2).
			WillReturnError(sql.ErrNoRows)

		repo := &Repository{db: db}
		booking, err := repo.GetABooking(context.Background(), 1, 2)
		assert.Error(t, err)
		assert.Nil(t, booking)
	})
}

func TestGetUserBookings(t *testing.T) {
	mockTime := time.Now()

	t.Run("returns bookings", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT \\* FROM booking WHERE user_id = ?").
			ExpectQuery().
			WithArgs(5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "days", "user_id", "room_id", "created_at", "updated_at"}).
				AddRow(1, 2, 5, 10, mockTime, mockTime).
				AddRow(2, 3, 5, 11, mockTime, mockTime))

		repo := &Repository{db: db}
		bookings, err := repo.GetUserBookings(context.Background(), 5)
		assert.NoError(t, err)
		assert.Len(t, bookings, 2)
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT \\* FROM booking WHERE user_id = ?").
			ExpectQuery().
			WithArgs(5).
			WillReturnError(sql.ErrConnDone)

		repo := &Repository{db: db}
		bookings, err := repo.GetUserBookings(context.Background(), 5)
		assert.Error(t, err)
		assert.Nil(t, bookings)
	})
}

func TestGetVendorBookings(t *testing.T) {
	mockTime := time.Now()

	t.Run("returns bookings", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT b.booking_id").
			ExpectQuery().
			WithArgs(7).
			WillReturnRows(sqlmock.NewRows([]string{"booking_id", "days", "user_id", "room_id", "vender_id", "created_at", "updated_at"}).
				AddRow(1, 2, 5, 10, 7, mockTime, mockTime))

		repo := &Repository{db: db}
		bookings, err := repo.GetVendorBookings(context.Background(), 7)
		assert.NoError(t, err)
		assert.Len(t, bookings, 1)
		assert.Equal(t, 7, bookings[0].VenderID)
	})

	t.Run("prepare error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT b.booking_id").WillReturnError(sql.ErrConnDone)

		repo := &Repository{db: db}
		bookings, err := repo.GetVendorBookings(context.Background(), 7)
		assert.Error(t, err)
		assert.Nil(t, bookings)
	})
}

func TestUpdateABooking(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		setup   func(mock sqlmock.Sqlmock)
	}{
		{
			name:    "success",
			wantErr: false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE booking SET days").
					ExpectExec().
					WithArgs(4, 1, 100, 5).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:    "prepare error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE booking SET days").WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name:    "exec error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE booking SET days").
					ExpectExec().
					WithArgs(4, 1, 100, 5).
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
			data := &entities.BookingPayload{
				Days:   bkIntPtr(4),
				UserID: bkIntPtr(5),
				Status: bkIntPtr(1),
			}
			err = repo.UpdateABooking(context.Background(), data, 100)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDeleteABooking(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectPrepare("UPDATE room SET status = 'VACANT'")
		mock.ExpectPrepare("DELETE FROM booking WHERE booking_id = ?")
		mock.ExpectExec("UPDATE room SET status = 'VACANT'").
			WithArgs(10, 7).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("DELETE FROM booking WHERE booking_id = ?").
			WithArgs(100).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		repo := &Repository{db: db}
		err = repo.DeleteABooking(context.Background(), 100, 7, 10)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("begin error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		repo := &Repository{db: db}
		err = repo.DeleteABooking(context.Background(), 100, 7, 10)
		assert.Error(t, err)
	})

	t.Run("exec error rolls back", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectPrepare("UPDATE room SET status = 'VACANT'")
		mock.ExpectPrepare("DELETE FROM booking WHERE booking_id = ?")
		mock.ExpectExec("UPDATE room SET status = 'VACANT'").
			WithArgs(10, 7).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		repo := &Repository{db: db}
		err = repo.DeleteABooking(context.Background(), 100, 7, 10)
		assert.Error(t, err)
	})
}
