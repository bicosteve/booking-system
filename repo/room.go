package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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
		if err == sql.ErrNoRows {
			return nil, errors.New("room with that id was not found")
		}
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

func (r *Repository) AllRooms(ctx context.Context) ([]*entities.Room, error) {
	q := `SELECT * FROM room ORDER BY room_id DESC`
	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var rooms []*entities.Room
	for rows.Next() {
		var room entities.Room
		err = rows.Scan(&room.ID, &room.Cost, &room.Status, &room.VenderId, &room.CreateAt, &room.UpdatedAt)
		if err != nil {
			return nil, err
		}

		rooms = append(rooms, &room)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %v", err)
	}

	return rooms, nil
}

func (r *Repository) UpdateARoom(ctx context.Context, room entities.Room, roomId, venderID int) error {
	q := `
		UPDATE room SET cost = ?, status = ?, updated_at = ? WHERE room_id = ? AND vender_id = ?
	`
	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	args := []interface{}{room.Cost, room.Status, time.Now(), room.ID, roomId}

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) DeleteARoom(ctx context.Context, roomId, userId int) error {

	q := `DELETE room WHERE room_id = ? AND vendor_id = ?`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, q)

	if err != nil {
		return err
	}

	return nil
}
