package repo

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bicosteve/booking-system/entities"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, p *entities.Payment) error
	FindPayment(ctx context.Context, userId string) (entities.PaymentBody, error)
}

func (r *Repository) CreatePayment(ctx context.Context, p *entities.Payment) error {
	key := fmt.Sprintf("user:%d", p.UserID)
	err := r.cache.HSet(ctx, key, map[string]any{
		"UserId":     p.UserID,
		"Amount":     p.Amount,
		"Status":     p.Status,
		"PaymentUrl": p.PaymentUrl,
		"PaymentId":  p.PaymentId,
	}).Err()

	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) FindPayment(ctx context.Context, userId string) (entities.Payment, error) {
	key := fmt.Sprintf("user:%s", userId)
	res, err := r.cache.HGetAll(ctx, key).Result()
	if err != nil {
		return entities.Payment{}, err
	}

	var payment entities.Payment

	for key, value := range res {
		switch key {
		case "UserId":
			user, _ := strconv.Atoi(value)
			payment.UserID = user
		case "Amount":
			amount, _ := strconv.ParseFloat(value, 64)
			payment.Amount = amount
		case "Status":
			payment.Status = value
		case "PaymentUrl":

			payment.PaymentUrl = value
		case "PaymentId":
			payment.PaymentId = value

		}
	}

	return payment, nil
}
