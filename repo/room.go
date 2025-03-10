package repo

import (
	"context"

	"github.com/bicosteve/booking-system/entities"
)

type RoomRepository interface {
	CreateRoom(ctx context.Context, room entities.RoomPayload) error
	FindRoomByID(ctx context.Context, roomID int) (*entities.Room, error)
	UpdateARoom(ctx context.Context, room entities.Room, roomID int) error
	DeleteARoom(ctx context.Context, roomID int) error
}

func (r *Repository) CreateRoom(ctx context.Context, room entities.RoomPayload) error {
	q := `
		INSERT INTO room(cost, status, vender_id, created_at, updated_at) 
		VALUES (?,?,?,NOW(),NOW())
	`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	args := []interface{}{room.Cost, room.Status, room.Vendor}

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) FindRoomByID(ctx context.Context, roomID int) (*entities.Room, error) {

	q := `SELECT * FROM room WHERE room_id = ?`

	stmt, err := r.db.PrepareContext(ctx, q)

	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	var room entities.Room

	row := stmt.QueryRowContext(ctx, roomID)

	err = row.Scan(&room.ID, &room.Cost, &room.Status, &room.VenderId, &room.CreateAt, &room.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &room, nil
}

func (r *Repository) UpdateARoom(ctx context.Context, room entities.Room, roomId int) error {
	q := `
		UPDATE room SET cost = ?, status = ? WHERE room_id = ? AND vender_id = ?
	`
	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	args := []interface{}{room.Cost, room.Status, room.ID, roomId}

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return err
	}

	return nil
}
