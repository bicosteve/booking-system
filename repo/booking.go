package repo

import (
	"context"
	"fmt"

	"github.com/bicosteve/booking-system/entities"
)

type BookingRepository interface {
	CreateABooking(ctx context.Context, data entities.BookingPayload) error
	UpdateABooking(ctx context.Context, data entities.Booking, bookingId, userId int) error
	DeleteABooking(ctx context.Context, bookingId, userId int) error
}

func (r *Repository) CreateABooking(ctx context.Context, data entities.BookingPayload) error {

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	updateQuery := `UPDATE room 
		SET status = 'BOOKED', updated_at = NOW() WHERE room_id = ?`

	updateRoomSTM, err := tx.PrepareContext(ctx, updateQuery)
	if err != nil {
		return err
	}

	defer updateRoomSTM.Close()

	insertQuery := `INSERT INTO booking(days,user_id,room_id,created_at, updated_at)
			VALUES (?, ?, ?, NOW(), NOW())`

	insertRoomSTM, err := tx.PrepareContext(ctx, insertQuery)
	if err != nil {
		return err
	}

	defer insertRoomSTM.Close()

	updateResult, err := updateRoomSTM.ExecContext(ctx, data.RoomID)
	if err != nil {
		return err
	}

	roomsAffected, err := updateResult.RowsAffected()
	if err != nil {
		return err
	}

	if roomsAffected < 1 {
		return fmt.Errorf("no room for room id %d or room not found", data.RoomID)
	}

	args := []any{data.Days, data.UserID, data.RoomID}

	insertResult, err := insertRoomSTM.ExecContext(ctx, args...)
	if err != nil {
		return err
	}

	bookingsAffected, err := insertResult.RowsAffected()
	if err != nil {
		return err
	}

	if bookingsAffected < 1 {
		return fmt.Errorf("no booking done for user %d and room %d", data.UserID, data.RoomID)
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) UpdateABooking() error {
	return nil
}

func (r *Repository) DeleteABooking() error {
	return nil
}
