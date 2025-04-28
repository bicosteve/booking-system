package service

import (
	"context"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stripe/stripe-go/v72"
)

func (ps PaymentService) GetActivePayment(ctx context.Context, userId string) (entities.Payment, error) {
	payment, err := ps.paymentRepository.FindPayment(ctx, userId)
	if err != nil {
		return entities.Payment{}, err
	}
	return payment, nil
}

func (ps PaymentService) StorePayment(ctx context.Context, cs *stripe.CheckoutSession, data entities.TRXPayload) error {

	payment := entities.Payment{
		UserID:     data.UserID,
		Amount:     float64(data.Payment.Amount),
		Status:     string(cs.Status),
		PaymentUrl: cs.URL,
		PaymentId:  cs.ID,
	}

	err := ps.paymentRepository.CreatePayment(ctx, &payment)
	if err != nil {
		return err
	}

	return nil
}
