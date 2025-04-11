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
