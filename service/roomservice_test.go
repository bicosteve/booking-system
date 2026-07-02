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

func newRoomService(t *testing.T) (*RoomService, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	rdb, _ := redismock.NewClientMock()
	repository := *repo.NewDBRepository(db, rdb)
	return NewRoomService(repository), mock, func() { db.Close() }
}

func TestRoomService_CreateRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("INSERT INTO room").
			ExpectExec().
			WithArgs("100", "VACANT", 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := svc.CreateRoom(context.Background(), entities.RoomPayload{Cost: "100", Status: "VACANT", Vendor: 1})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("INSERT INTO room").WillReturnError(sql.ErrConnDone)

		err := svc.CreateRoom(context.Background(), entities.RoomPayload{Cost: "100", Status: "VACANT", Vendor: 1})
		assert.Error(t, err)
	})
}

func TestRoomService_FindARoom(t *testing.T) {
	mockTime := time.Now()

	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT \\* FROM room WHERE room_id = ?").
			ExpectQuery().
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "cost", "status", "vender_id", "created_at", "updated_at"}).
				AddRow("1", 100.0, "VACANT", "2", mockTime, mockTime))

		room, err := svc.FindARoom(context.Background(), 1)
		assert.NoError(t, err)
		assert.Equal(t, "1", room.ID)
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT \\* FROM room WHERE room_id = ?").
			ExpectQuery().
			WithArgs(1).
			WillReturnError(sql.ErrNoRows)

		room, err := svc.FindARoom(context.Background(), 1)
		assert.Error(t, err)
		assert.Nil(t, room)
	})
}

func TestRoomService_FindRooms(t *testing.T) {
	mockTime := time.Now()

	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT \\* FROM room ORDER BY room_id DESC").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "cost", "status", "vender_id", "created_at", "updated_at"}).
				AddRow("1", 100.0, "VACANT", "1", mockTime, mockTime))

		rooms, err := svc.FindRooms(context.Background())
		assert.NoError(t, err)
		assert.Len(t, rooms, 1)
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("SELECT \\* FROM room ORDER BY room_id DESC").
			ExpectQuery().
			WillReturnError(sql.ErrConnDone)

		rooms, err := svc.FindRooms(context.Background())
		assert.Error(t, err)
		assert.Nil(t, rooms)
	})
}

func TestRoomService_UpdateARoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("UPDATE room SET cost").
			ExpectExec().
			WithArgs(150.0, "BOOKED", sqlmock.AnyArg(), 1, 2).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := svc.UpdateARoom(context.Background(), &entities.Room{Cost: 150.0, Status: "BOOKED"}, 1, 2)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("UPDATE room SET cost").WillReturnError(sql.ErrConnDone)

		err := svc.UpdateARoom(context.Background(), &entities.Room{}, 1, 2)
		assert.Error(t, err)
	})
}

func TestRoomService_DeleteARoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("DELETE FROM room WHERE room_id = \\? AND vender_id = \\?").
			ExpectExec().
			WithArgs(1, 2).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := svc.DeleteARoom(context.Background(), 1, 2)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		svc, mock, cleanup := newRoomService(t)
		defer cleanup()

		mock.ExpectPrepare("DELETE FROM room").WillReturnError(sql.ErrConnDone)

		err := svc.DeleteARoom(context.Background(), 1, 2)
		assert.Error(t, err)
	})
}
