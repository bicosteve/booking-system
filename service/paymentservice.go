package service

import (
	"context"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stripe/stripe-go/v72"
)

func (ps PaymentService) GetActivePayment(ctx context.Context, userId string) (entities.Payment, error) {
	payment, err := ps.paymentRepository.FindPayment(ctx, userId)
	if err != nil {
		return entities.Payment{}, err
	}
	return *payment, nil
}

func (ps PaymentService) HoldPayment(ctx context.Context, pi *stripe.PaymentIntent, data entities.TRXPayload) error {
	payment := entities.Payment{
		OrderID:       data.OrderID,
		UserID:        data.UserID,
		PaymentId:     pi.ID,
		Amount:        float64(data.Payment.Amount),
		ClientSecret:  pi.ClientSecret,
		TransactionID: pi.ID,
		CustomerId:    data.UserID,
		RoomID:        data.RoomID,
		Status:        "initial",
		Response:      pi.Description,
		CaptureMethod: string(pi.CaptureMethod),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := ps.paymentRepository.CreatePayment(ctx, &payment)
	if err != nil {
		return err
	}

	return nil
}

func (ps PaymentService) RemovePayment(ctx context.Context, userId string) error {
	err := ps.paymentRepository.RemovePayment(ctx, userId)
	if err != nil {
		return err
	}
	return nil
}

func (ps PaymentService) AddPayment(ctx context.Context, data *entities.TRXPayload) error {

	err := ps.paymentRepository.SaveTransactions(ctx, data)
	if err != nil {
		return err
	}

	return nil
}

func (ps PaymentService) UpdatePayment(ctx context.Context, status int, trx_id string) error {
	err := ps.paymentRepository.UpdateTransactions(ctx, status, trx_id)
	if err != nil {
		return err
	}
	return nil
}
