package service

import (
	"context"

	"github.com/bicosteve/booking-system/entities"
)

func (b *BookingService) MakeBooking(ctx context.Context, data entities.BookingPayload) error {

	err := b.bookingRepository.CreateABooking(ctx, data)
	if err != nil {
		return err
	}

	return nil
}

func (b *BookingService) GetUserBooking(ctx context.Context, roomID, userID int) (*entities.Booking, error) {
	booking, err := b.bookingRepository.GetABooking(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}

	return booking, nil
}

func (b *BookingService) GetUserBookings(ctx context.Context, userID int) ([]*entities.Booking, error) {
	bookings, err := b.bookingRepository.GetUserBookings(ctx, userID)
	if err != nil {
		return nil, err
	}

	return bookings, nil
}

func (b *BookingService) GetVendoerBookings(ctx context.Context, userID int) ([]*entities.Booking, error) {
	bookings, err := b.bookingRepository.GetVendorBookings(ctx, userID)
	if err != nil {
		return nil, err
	}

	return bookings, nil
}

func (b *BookingService) UpdateABooking(ctx context.Context, data *entities.BookingPayload, bookingID int) error {
	err := b.bookingRepository.UpdateABooking(ctx, data, bookingID)
	if err != nil {
		return err
	}

	return nil
}

func (b *BookingService) DeleteABooking(ctx context.Context, bookingID, vendorID, roomID int) error {
	err := b.bookingRepository.DeleteABooking(ctx, bookingID, vendorID, roomID)
	if err != nil {
		return err
	}
	return nil
}
