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
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func setupBookingBase(t *testing.T) (*Base, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	rdb, _ := redismock.NewClientMock()
	repository := *repo.NewDBRepository(db, rdb)

	base := &Base{
		bookingService: service.NewBookingService(repository),
		paymentService: service.NewPaymentService(repository),
		contentType:    "application/json",
		DB:             db,
		KafkaStatus:    0,
		RabbitMQStatus: 0,
	}
	return base, mock
}

func withURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withBookingUser(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), entities.UseridKeyValue, id))
}

func TestGetBookingHandler(t *testing.T) {
	mockTime := time.Now()
	getQuery := "SELECT * FROM booking\n\t\t\tWHERE status = 0 \n\t\t\tAND booking_id = ? AND user_id = ?\n\t\t\tORDER BY created_at DESC LIMIT 1"

	t.Run("successful get", func(t *testing.T) {
		base, mock := setupBookingBase(t)
		mock.ExpectPrepare(getQuery).
			ExpectQuery().
			WithArgs(1, 5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "days", "user_id", "room_id", "created_at", "updated_at"}).
				AddRow(1, 2, 5, 1, mockTime, mockTime))

		req := httptest.NewRequest(http.MethodGet, "/book/1", nil)
		req = withURLParam(req, "room_id", "1")
		req = withBookingUser(req, "5")
		w := httptest.NewRecorder()

		base.GetBookingHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid room id param", func(t *testing.T) {
		base, _ := setupBookingBase(t)
		req := httptest.NewRequest(http.MethodGet, "/book/abc", nil)
		req = withURLParam(req, "room_id", "abc")
		req = withBookingUser(req, "5")
		w := httptest.NewRecorder()

		base.GetBookingHandler(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetAllBookingsHandler(t *testing.T) {
	mockTime := time.Now()
	q := "SELECT * FROM booking WHERE user_id = ?"

	t.Run("success", func(t *testing.T) {
		base, mock := setupBookingBase(t)
		mock.ExpectPrepare(q).
			ExpectQuery().
			WithArgs(5).
			WillReturnRows(sqlmock.NewRows([]string{"id", "days", "user_id", "room_id", "created_at", "updated_at"}).
				AddRow(1, 2, 5, 10, mockTime, mockTime))

		req := httptest.NewRequest(http.MethodGet, "/book/all", nil)
		req = withBookingUser(req, "5")
		w := httptest.NewRecorder()

		base.GetAllBookingsHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("missing user id", func(t *testing.T) {
		base, _ := setupBookingBase(t)
		req := httptest.NewRequest(http.MethodGet, "/book/all", nil)
		w := httptest.NewRecorder()

		base.GetAllBookingsHandler(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetAllAdminBookingsHandler(t *testing.T) {
	mockTime := time.Now()
	q := "SELECT b.booking_id, b.days, b.user_id, b.room_id, r.vender_id,\n\t\t\t\tb.created_at, b.updated_at\n\t\t\tFROM booking b JOIN room r ON b.room_id = r.room_id\n\t\t\tWHERE r.vender_id = ?"

	t.Run("success", func(t *testing.T) {
		base, mock := setupBookingBase(t)
		mock.ExpectPrepare(q).
			ExpectQuery().
			WithArgs(7).
			WillReturnRows(sqlmock.NewRows([]string{"booking_id", "days", "user_id", "room_id", "vender_id", "created_at", "updated_at"}).
				AddRow(1, 2, 5, 10, 7, mockTime, mockTime))

		req := httptest.NewRequest(http.MethodGet, "/admin/book/all", nil)
		req = withBookingUser(req, "7")
		w := httptest.NewRecorder()

		base.GetAllAdminBookingsHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpdateBookingHandler(t *testing.T) {
	updateQuery := "UPDATE booking SET days = ?, status = ?, updated_at = NOW()\n\t\t  WHERE booking_id = ? AND user_id = ?"

	t.Run("successful update", func(t *testing.T) {
		base, mock := setupBookingBase(t)
		mock.ExpectPrepare(updateQuery).
			ExpectExec().
			WithArgs(3, sqlmock.AnyArg(), 100, 5).
			WillReturnResult(sqlmock.NewResult(0, 1))

		days := 3
		payload, _ := json.Marshal(entities.BookingPayload{Days: &days})
		req := httptest.NewRequest(http.MethodPut, "/book/100", bytes.NewBuffer(payload))
		req = withURLParam(req, "booking_id", "100")
		req = withBookingUser(req, "5")
		w := httptest.NewRecorder()

		base.UpdateBooking(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid booking id", func(t *testing.T) {
		base, _ := setupBookingBase(t)
		days := 3
		payload, _ := json.Marshal(entities.BookingPayload{Days: &days})
		req := httptest.NewRequest(http.MethodPut, "/book/abc", bytes.NewBuffer(payload))
		req = withURLParam(req, "booking_id", "abc")
		req = withBookingUser(req, "5")
		w := httptest.NewRecorder()

		base.UpdateBooking(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteBookingHandler(t *testing.T) {
	roomUpdate := "UPDATE room SET status = 'VACANT'\n\t\t\t\t\tWHERE room_id = ? and vender_id = ?"
	bookingDelete := "DELETE FROM booking WHERE booking_id = ?"

	newReq := func(bookingID, roomID string) *http.Request {
		req := httptest.NewRequest(http.MethodDelete, "/admin/book/"+bookingID+"/"+roomID, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("booking_id", bookingID)
		rctx.URLParams.Add("room_id", roomID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		return req.WithContext(context.WithValue(req.Context(), entities.UseridKeyValue, "7"))
	}

	t.Run("successful delete", func(t *testing.T) {
		base, mock := setupBookingBase(t)
		mock.ExpectBegin()
		mock.ExpectPrepare(roomUpdate)
		mock.ExpectPrepare(bookingDelete)
		mock.ExpectExec(roomUpdate).WithArgs(10, 7).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(bookingDelete).WithArgs(100).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		req := newReq("100", "10")
		w := httptest.NewRecorder()

		base.DeleteBooking(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid booking id", func(t *testing.T) {
		base, _ := setupBookingBase(t)
		req := newReq("abc", "10")
		w := httptest.NewRecorder()

		base.DeleteBooking(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCreateBookingHandler_ValidationError(t *testing.T) {
	// Missing required fields should fail validation before any Stripe interaction.
	base, _ := setupBookingBase(t)

	payload, _ := json.Marshal(entities.BookingPayload{})
	req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewBuffer(payload))
	req = withBookingUser(req, "5")
	w := httptest.NewRecorder()

	base.CreateBookingHandler(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifyBookingHandler_InvalidParam(t *testing.T) {
	base, _ := setupBookingBase(t)

	req := httptest.NewRequest(http.MethodGet, "/verify/abc", nil)
	req = withURLParam(req, "room_id", "abc")
	req = withBookingUser(req, "5")
	w := httptest.NewRecorder()

	base.VerifyBookingHandler(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
