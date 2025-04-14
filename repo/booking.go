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
	UpdateABooking(ctx context.Context, data *entities.BookingPayload, bookingID int) error
	DeleteABooking(ctx context.Context, bookingID, vendorID, roomID int) error
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
		_ = tx.Rollback()
		return err
	}

	defer updateRoomSTM.Close()

	insertQuery := `INSERT INTO booking(days,user_id,room_id,created_at, updated_at)
			VALUES (?, ?, ?, NOW(), NOW())`

	insertRoomSTM, err := tx.PrepareContext(ctx, insertQuery)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	defer insertRoomSTM.Close()

	updateResult, err := updateRoomSTM.ExecContext(ctx, data.RoomID)
	if err != nil {
		_ = tx.Rollback()
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
		_ = tx.Rollback()
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
	q := `SELECT b.booking_id, b.days, b.user_id, b.room_id, r.vender_id,
				b.created_at, b.updated_at
			FROM booking b JOIN room r ON b.room_id = r.room_id 
			WHERE r.vender_id = ?`

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
		err = rows.Scan(&booking.ID, &booking.Days, &booking.UserID, &booking.RoomID, &booking.VenderID, &booking.CreatedAt, &booking.UpdateAt)

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

func (r *Repository) DeleteABooking(ctx context.Context, bookingID, vendorID, roomID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	room_query := `UPDATE room SET status = 'VACANT' 
					WHERE room_id = ? and vender_id = ?`

	booking_query := `DELETE FROM booking WHERE booking_id = ?`
	room_stmt, err := r.db.PrepareContext(ctx, room_query)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer room_stmt.Close()

	booking_stmt, err := r.db.PrepareContext(ctx, booking_query)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer booking_stmt.Close()

	_, err = room_stmt.ExecContext(ctx, roomID, vendorID)
	if err != nil {
		_ = tx.Rollback()
		return nil
	}

	_, err = booking_stmt.ExecContext(ctx, bookingID)
	if err != nil {
		_ = tx.Rollback()
		return nil
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil
}
