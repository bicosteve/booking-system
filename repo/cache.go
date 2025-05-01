package repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bicosteve/booking-system/entities"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, p *entities.Payment) error
	FindPayment(ctx context.Context, userId string) (entities.PaymentBody, error)
	RemovePayment(ctx context.Context, userId string) error
}

func (r *Repository) CreatePayment(ctx context.Context, p *entities.Payment) error {
	key := fmt.Sprintf("user:%d", p.UserID)
	err := r.cache.HSet(ctx, key, map[string]any{
		"OrderId":        p.OrderID,
		"UserId":         p.UserID,
		"Amount":         p.Amount,
		"Status":         p.Status,
		"PaymentUrl":     p.PaymentUrl,
		"PaymentId":      p.PaymentId,
		"ClientSecret":   p.ClientSecret,
		"TransactionId":  p.TransactionID,
		"CustomerId":     p.CustomerId,
		"RoomID":         p.RoomID,
		"Response":       p.Response,
		"CapturedMethod": p.CaptureMethod,
		"CreatedAt":      p.CreatedAt,
		"UpdatedAt":      p.UpdatedAt,
	}).Err()

	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) FindPayment(ctx context.Context, userId string) (*entities.Payment, error) {
	var key = fmt.Sprintf("user:%s", userId)
	var payment entities.Payment

	result, err := r.cache.HGetAll(ctx, key).Result()
	if err != nil {
		return &entities.Payment{}, err
	}

	if result == nil {
		return &entities.Payment{}, errors.New("there is no payment")
	}

	for key, value := range result {
		switch key {
		case "OrderId":
			payment.OrderID = value
		case "UserID":
			id, _ := strconv.Atoi(value)
			payment.UserID = id
		case "PaymentId":
			payment.PaymentId = value
		case "RoomID":
			room_id, _ := strconv.Atoi(value)
			payment.RoomID = room_id
		case "Amount":
			amount, _ := strconv.ParseFloat(value, 64)
			payment.Amount = amount
		case "ClientSecret":
			payment.ClientSecret = value
		case "TransactionID":
			payment.TransactionID = value
		case "CustomerId":
			custID, _ := strconv.Atoi(value)
			payment.CustomerId = custID
		case "Status":
			payment.Status = value
		case "Response":
			payment.Response = value
		case "PaymentUrl":
			payment.PaymentUrl = value
		case "CapturedMethod":
			payment.CaptureMethod = value
		case "CreatedAt":
			created, _ := time.Parse("2006-01-02 15:04:05", value)
			payment.CreatedAt = created
		case "UpdatedAt":
			updated, _ := time.Parse("2006-01-02 15:04:05", value)
			payment.CreatedAt = updated
		}

	}

	return &payment, nil
}

func (r *Repository) RemovePayment(ctx context.Context, userId string) error {
	_, err := r.cache.Del(ctx, fmt.Sprintf("user:%s", userId)).Result()
	if err != nil {
		return err
	}

	return nil
}
