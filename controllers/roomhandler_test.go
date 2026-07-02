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

func setupRoomBase(t *testing.T) (*Base, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	rdb, _ := redismock.NewClientMock()
	repository := *repo.NewDBRepository(db, rdb)

	base := &Base{
		roomService: service.NewRoomService(repository),
		contentType: "application/json",
		DB:          db,
	}
	return base, mock
}

// withUserID injects the user id into the request context as the handlers expect.
func withUserID(r *http.Request, id string) *http.Request {
	ctx := context.WithValue(r.Context(), entities.UseridKeyValue, id)
	return r.WithContext(ctx)
}

func TestCreateRoomHandler(t *testing.T) {
	insertQuery := "\n\t\tINSERT INTO room(cost, status, vender_id, created_at, updated_at)\n\t\tVALUES (?,?,?,NOW(),NOW())\n\t"

	t.Run("successful create", func(t *testing.T) {
		base, mock := setupRoomBase(t)
		mock.ExpectPrepare(insertQuery).
			ExpectExec().
			WithArgs("100", "VACANT", 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		payload, _ := json.Marshal(entities.RoomPayload{Cost: "100", Status: "VACANT"})
		req := httptest.NewRequest(http.MethodPost, "/rooms", bytes.NewBuffer(payload))
		req = withUserID(req, "1")
		w := httptest.NewRecorder()

		base.CreateRoomHandler(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("validation error - missing cost", func(t *testing.T) {
		base, _ := setupRoomBase(t)

		payload, _ := json.Marshal(entities.RoomPayload{Status: "VACANT"})
		req := httptest.NewRequest(http.MethodPost, "/rooms", bytes.NewBuffer(payload))
		req = withUserID(req, "1")
		w := httptest.NewRecorder()

		base.CreateRoomHandler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing user id in context", func(t *testing.T) {
		base, _ := setupRoomBase(t)

		payload, _ := json.Marshal(entities.RoomPayload{Cost: "100", Status: "VACANT"})
		req := httptest.NewRequest(http.MethodPost, "/rooms", bytes.NewBuffer(payload))
		w := httptest.NewRecorder()

		base.CreateRoomHandler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestFindRoomHandler(t *testing.T) {
	mockTime := time.Now()
	allRoomsQuery := "SELECT * FROM room ORDER BY room_id DESC"

	roomRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "cost", "status", "vender_id", "created_at", "updated_at"}).
			AddRow("1", 100.0, "VACANT", "2", mockTime, mockTime).
			AddRow("2", 200.0, "BOOKED", "2", mockTime, mockTime)
	}

	t.Run("all rooms - no filter", func(t *testing.T) {
		base, mock := setupRoomBase(t)
		mock.ExpectPrepare(allRoomsQuery).ExpectQuery().WillReturnRows(roomRows())

		req := httptest.NewRequest(http.MethodGet, "/rooms", nil)
		w := httptest.NewRecorder()

		base.FindRoomHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filter by id found", func(t *testing.T) {
		base, mock := setupRoomBase(t)
		mock.ExpectPrepare(allRoomsQuery).ExpectQuery().WillReturnRows(roomRows())

		req := httptest.NewRequest(http.MethodGet, "/rooms?room_id=1", nil)
		w := httptest.NewRecorder()

		base.FindRoomHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by id not found", func(t *testing.T) {
		base, mock := setupRoomBase(t)
		mock.ExpectPrepare(allRoomsQuery).ExpectQuery().WillReturnRows(roomRows())

		req := httptest.NewRequest(http.MethodGet, "/rooms?room_id=99", nil)
		w := httptest.NewRecorder()

		base.FindRoomHandler(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("filter by status found", func(t *testing.T) {
		base, mock := setupRoomBase(t)
		mock.ExpectPrepare(allRoomsQuery).ExpectQuery().WillReturnRows(roomRows())

		req := httptest.NewRequest(http.MethodGet, "/rooms?status=VACANT", nil)
		w := httptest.NewRecorder()

		base.FindRoomHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("filter by invalid status", func(t *testing.T) {
		base, mock := setupRoomBase(t)
		mock.ExpectPrepare(allRoomsQuery).ExpectQuery().WillReturnRows(roomRows())

		req := httptest.NewRequest(http.MethodGet, "/rooms?status=UNKNOWN", nil)
		w := httptest.NewRecorder()

		base.FindRoomHandler(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdateARoomHandler(t *testing.T) {
	updateQuery := "\n\t\tUPDATE room SET cost = ?, status = ?, updated_at = ? WHERE room_id = ? AND vender_id = ?\n\t"

	newReq := func(body string, roomID string) *http.Request {
		req := httptest.NewRequest(http.MethodPut, "/rooms/"+roomID, bytes.NewBufferString(body))
		req = withUserID(req, "2")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("room_id", roomID)
		return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	t.Run("successful update", func(t *testing.T) {
		base, mock := setupRoomBase(t)
		mock.ExpectPrepare(updateQuery).
			ExpectExec().
			WithArgs(150.0, "BOOKED", sqlmock.AnyArg(), 1, 2).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// preserve user id in context alongside chi route context
		req := newReq(`{"cost":150,"status":"BOOKED"}`, "1")
		req = req.WithContext(context.WithValue(req.Context(), entities.UseridKeyValue, "2"))
		w := httptest.NewRecorder()

		base.UpdateARoom(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid room id", func(t *testing.T) {
		base, _ := setupRoomBase(t)
		req := newReq(`{"cost":150}`, "abc")
		req = req.WithContext(context.WithValue(req.Context(), entities.UseridKeyValue, "2"))
		w := httptest.NewRecorder()

		base.UpdateARoom(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteARoomHandler(t *testing.T) {
	deleteQuery := "DELETE FROM room WHERE room_id = ? AND vender_id = ?"

	newReq := func(roomID string) *http.Request {
		req := httptest.NewRequest(http.MethodDelete, "/rooms/"+roomID, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("room_id", roomID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		return req.WithContext(context.WithValue(req.Context(), entities.UseridKeyValue, "2"))
	}

	t.Run("successful delete", func(t *testing.T) {
		base, mock := setupRoomBase(t)
		mock.ExpectPrepare(deleteQuery).
			ExpectExec().
			WithArgs(1, 2).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req := newReq("1")
		w := httptest.NewRecorder()

		base.DeleteARoom(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid room id", func(t *testing.T) {
		base, _ := setupRoomBase(t)
		req := newReq("abc")
		w := httptest.NewRecorder()

		base.DeleteARoom(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
