package repo

import (
	"context"
	"fmt"

	"github.com/bicosteve/booking-system/entities"
)

type BookingRepository interface {
	CreateABooking(ctx context.Context, data entities.BookingPayload) error
	GetABooking(ctx context.Context, bookingID, userId int) (*entities.Booking, error)
	GetUserBookings(ctx context.Context, userID int) ([]*entities.Booking, error)
	GetVendorBookings(ctx context.Context, vendorID int) ([]*entities.Booking, error)
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

func (r *Repository) GetABooking(ctx context.Context, bookingID, userId int) (*entities.Booking, error) {
	q := `SELECT * FROM booking WHERE booking_id = ? AND user_id = ?`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	var booking entities.Booking

	row := stmt.QueryRowContext(ctx, bookingID, userId)

	err = row.Scan(&booking.ID, &booking.Days, &booking.UserID, &booking.RoomID, &booking.CreatedAt, &booking.UpdateAt)
	if err != nil {
		return nil, err
	}

	return &booking, nil
}

func (r *Repository) GetUserBookings(ctx context.Context, userID int) ([]*entities.Booking, error) {

	q := `SELECT * FROM booking WHERE user_id = ?`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var bookings []*entities.Booking

	for rows.Next() {
		var booking entities.Booking
		err = rows.Scan(&booking.ID, &booking.Days, &booking.UserID, &booking.RoomID, &booking.CreatedAt, &booking.UpdateAt)

		if err != nil {
			return nil, err
		}

		bookings = append(bookings, &booking)

	}

	return bookings, nil
}

func (r *Repository) GetVendorBookings(ctx context.Context, vendorID int) ([]*entities.Booking, error) {
	q := `SELECT * FROM booking INNER JOIN room 
			WHERE booking.room_id = room.room_id 
			AND room.vender_id = user_id = ?`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, vendorID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var bookings []*entities.Booking

	for rows.Next() {
		var booking entities.Booking
		err = rows.Scan(&booking.ID, &booking.Days, &booking.UserID, &booking.RoomID, &booking.CreatedAt, &booking.UpdateAt)

		if err != nil {
			return nil, err
		}

		bookings = append(bookings, &booking)

	}

	return bookings, nil

}

func (r *Repository) UpdateABooking(ctx context.Context, data *entities.BookingPayload, bookingID int) error {

	q := `UPDATE booking SET days = ?, updated_at = NOW() 
			WHERE booking_id = ? AND user_id = ?`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	args := []interface{}{data.Days, bookingID, data.UserID}

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) DeleteABooking(ctx context.Context, bookingID, userID, roomID int) error {

	q := `DELETE FROM booking WHERE booking_id = ? AND user_id = ?`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, bookingID, userID)
	if err != nil {
		return nil
	}

	return nil
}
