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

func TestCreateRoom(t *testing.T) {
	tests := []struct {
		name    string
		room    entities.RoomPayload
		wantErr bool
		setup   func(mock sqlmock.Sqlmock)
	}{
		{
			name:    "successful create",
			room:    entities.RoomPayload{Cost: "100", Status: "VACANT", Vendor: 1},
			wantErr: false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO room").
					ExpectExec().
					WithArgs("100", "VACANT", 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:    "prepare error",
			room:    entities.RoomPayload{Cost: "100", Status: "VACANT", Vendor: 1},
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO room").WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name:    "exec error",
			room:    entities.RoomPayload{Cost: "100", Status: "VACANT", Vendor: 1},
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO room").
					ExpectExec().
					WithArgs("100", "VACANT", 1).
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
			err = repo.CreateRoom(context.Background(), tt.room)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFindRoomByID(t *testing.T) {
	mockTime := time.Now()

	t.Run("found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT \\* FROM room WHERE room_id = ?").
			ExpectQuery().
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "cost", "status", "vender_id", "created_at", "updated_at"}).
				AddRow("1", 100.0, "VACANT", "2", mockTime, mockTime))

		repo := &Repository{db: db}
		room, err := repo.FindRoomByID(context.Background(), 1)
		assert.NoError(t, err)
		assert.Equal(t, "1", room.ID)
		assert.Equal(t, 100.0, room.Cost)
		assert.Equal(t, "VACANT", room.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT \\* FROM room WHERE room_id = ?").
			ExpectQuery().
			WithArgs(99).
			WillReturnError(sql.ErrNoRows)

		repo := &Repository{db: db}
		room, err := repo.FindRoomByID(context.Background(), 99)
		assert.Error(t, err)
		assert.Nil(t, room)
	})
}

func TestAllRooms(t *testing.T) {
	mockTime := time.Now()

	t.Run("returns rooms", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT \\* FROM room ORDER BY room_id DESC").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "cost", "status", "vender_id", "created_at", "updated_at"}).
				AddRow("2", 200.0, "BOOKED", "1", mockTime, mockTime).
				AddRow("1", 100.0, "VACANT", "1", mockTime, mockTime))

		repo := &Repository{db: db}
		rooms, err := repo.AllRooms(context.Background())
		assert.NoError(t, err)
		assert.Len(t, rooms, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		mock.ExpectPrepare("SELECT \\* FROM room ORDER BY room_id DESC").
			ExpectQuery().
			WillReturnError(sql.ErrConnDone)

		repo := &Repository{db: db}
		rooms, err := repo.AllRooms(context.Background())
		assert.Error(t, err)
		assert.Nil(t, rooms)
	})

	t.Run("scan error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		// too few columns -> scan fails
		mock.ExpectPrepare("SELECT \\* FROM room ORDER BY room_id DESC").
			ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

		repo := &Repository{db: db}
		rooms, err := repo.AllRooms(context.Background())
		assert.Error(t, err)
		assert.Nil(t, rooms)
	})
}

func TestUpdateARoom(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		setup   func(mock sqlmock.Sqlmock)
	}{
		{
			name:    "success",
			wantErr: false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE room SET cost").
					ExpectExec().
					WithArgs(150.0, "BOOKED", sqlmock.AnyArg(), 1, 2).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:    "prepare error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE room SET cost").WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name:    "exec error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE room SET cost").
					ExpectExec().
					WithArgs(150.0, "BOOKED", sqlmock.AnyArg(), 1, 2).
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
			data := &entities.Room{Cost: 150.0, Status: "BOOKED"}
			err = repo.UpdateARoom(context.Background(), data, 1, 2)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDeleteARoom(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		setup   func(mock sqlmock.Sqlmock)
	}{
		{
			name:    "success",
			wantErr: false,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("DELETE FROM room WHERE room_id = \\? AND vender_id = \\?").
					ExpectExec().
					WithArgs(1, 2).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:    "prepare error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("DELETE FROM room").WillReturnError(sql.ErrConnDone)
			},
		},
		{
			name:    "exec error",
			wantErr: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("DELETE FROM room").
					ExpectExec().
					WithArgs(1, 2).
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
			err = repo.DeleteARoom(context.Background(), 1, 2)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
