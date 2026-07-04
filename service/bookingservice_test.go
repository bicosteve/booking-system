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

func newBookingService(t *testing.T) (*BookingService, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	rdb, _ := redismock.NewClientMock()
	repository := *repo.NewDBRepository(db, rdb)
	return NewBookingService(repository), mock, func() { db.Close() }
}

func bsIntPtr(i int) *int { return &i }

func TestBookingService_MakeBooking(t *testing.T) {
	days, userID, roomID, status := 2, 5, 10, 0

	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectPrepare("UPDATE room")
		mock.ExpectPrepare("INSERT INTO booking")
		mock.ExpectExec("UPDATE room").WithArgs(roomID).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO booking").WithArgs(days, userID, roomID, status).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := svc.MakeBooking(context.Background(), entities.BookingPayload{
			Days: &days, UserID: &userID, RoomID: &roomID, Status: &status,
		})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		err := svc.MakeBooking(context.Background(), entities.BookingPayload{
			Days: &days, UserID: &userID, RoomID: &roomID, Status: &status,
		})
		assert.Error(t, err)
	})
}

func TestBookingService_GetUserBooking(t *testing.T) {
	mockTime := time.Now()

	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT \\* FROM booking").
			ExpectQuery().
			WithArgs(1, 2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "days", "user_id", "room_id", "created_at", "updated_at"}).
				AddRow(1, 3, 2, 1, mockTime, mockTime))

		booking, err := svc.GetUserBooking(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, 1, booking.ID)
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT \\* FROM booking").
			ExpectQuery().
			WithArgs(1, 2).
			WillReturnError(sql.ErrNoRows)

		booking, err := svc.GetUserBooking(context.Background(), 1, 2)
		assert.Error(t, err)
		assert.Nil(t, booking)
	})
}

func TestBookingService_GetUserBookings(t *testing.T) {
	mockTime := time.Now()

	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT \\* FROM booking WHERE user_id = ?").
			ExpectQuery().
			WithArgs(5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "days", "user_id", "room_id", "created_at", "updated_at"}).
				AddRow(1, 2, 5, 10, mockTime, mockTime))

		bookings, err := svc.GetUserBookings(context.Background(), 5)
		assert.NoError(t, err)
		assert.Len(t, bookings, 1)
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT \\* FROM booking WHERE user_id = ?").
			ExpectQuery().
			WithArgs(5).
			WillReturnError(sql.ErrConnDone)

		bookings, err := svc.GetUserBookings(context.Background(), 5)
		assert.Error(t, err)
		assert.Nil(t, bookings)
	})
}

func TestBookingService_GetVendoerBookings(t *testing.T) {
	mockTime := time.Now()

	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT b.booking_id").
			ExpectQuery().
			WithArgs(7).
			WillReturnRows(sqlmock.NewRows([]string{"booking_id", "days", "user_id", "room_id", "vender_id", "created_at", "updated_at"}).
				AddRow(1, 2, 5, 10, 7, mockTime, mockTime))

		bookings, err := svc.GetVendoerBookings(context.Background(), 7)
		assert.NoError(t, err)
		assert.Len(t, bookings, 1)
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT b.booking_id").WillReturnError(sql.ErrConnDone)

		bookings, err := svc.GetVendoerBookings(context.Background(), 7)
		assert.Error(t, err)
		assert.Nil(t, bookings)
	})
}

func TestBookingService_UpdateABooking(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectPrepare("UPDATE booking SET days").
			ExpectExec().
			WithArgs(4, 1, 100, 5).
			WillReturnResult(sqlmock.NewResult(0, 1))

		data := &entities.BookingPayload{Days: bsIntPtr(4), UserID: bsIntPtr(5), Status: bsIntPtr(1)}
		err := svc.UpdateABooking(context.Background(), data, 100)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectPrepare("UPDATE booking SET days").WillReturnError(sql.ErrConnDone)

		data := &entities.BookingPayload{Days: bsIntPtr(4), UserID: bsIntPtr(5), Status: bsIntPtr(1)}
		err := svc.UpdateABooking(context.Background(), data, 100)
		assert.Error(t, err)
	})
}

func TestBookingService_DeleteABooking(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectBegin()
		mock.ExpectPrepare("UPDATE room SET status = 'VACANT'")
		mock.ExpectPrepare("DELETE FROM booking WHERE booking_id = ?")
		mock.ExpectExec("UPDATE room SET status = 'VACANT'").WithArgs(10, 7).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("DELETE FROM booking WHERE booking_id = ?").WithArgs(100).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := svc.DeleteABooking(context.Background(), 100, 7, 10)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newBookingService(t)
		defer cleanup()

		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		err := svc.DeleteABooking(context.Background(), 100, 7, 10)
		assert.Error(t, err)
	})
}
