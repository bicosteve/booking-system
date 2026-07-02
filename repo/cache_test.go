package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

func TestCreatePayment(t *testing.T) {
	now := time.Now()
	payment := &entities.Payment{
		OrderID:       "order-1",
		UserID:        5,
		Amount:        100,
		Status:        "initial",
		PaymentUrl:    "http://pay",
		PaymentId:     "pi_1",
		ClientSecret:  "secret",
		TransactionID: "trx_1",
		CustomerId:    5,
		RoomID:        10,
		Response:      "ok",
		CaptureMethod: "manual",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	expectedFields := map[string]any{
		"OrderId":        payment.OrderID,
		"UserId":         payment.UserID,
		"Amount":         payment.Amount,
		"Status":         payment.Status,
		"PaymentUrl":     payment.PaymentUrl,
		"PaymentId":      payment.PaymentId,
		"ClientSecret":   payment.ClientSecret,
		"TransactionId":  payment.TransactionID,
		"CustomerId":     payment.CustomerId,
		"RoomID":         payment.RoomID,
		"Response":       payment.Response,
		"CapturedMethod": payment.CaptureMethod,
		"CreatedAt":      payment.CreatedAt,
		"UpdatedAt":      payment.UpdatedAt,
	}

	t.Run("success", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		mock.ExpectHSet("user:5", expectedFields).SetVal(1)

		repo := &Repository{cache: client}
		err := repo.CreatePayment(context.Background(), payment)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		mock.ExpectHSet("user:5", expectedFields).SetErr(errors.New("redis down"))

		repo := &Repository{cache: client}
		err := repo.CreatePayment(context.Background(), payment)
		assert.Error(t, err)
	})
}

func TestFindPayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		mock.ExpectHGetAll("user:5").SetVal(map[string]string{
			"OrderId":   "order-1",
			"UserID":    "5",
			"PaymentId": "pi_1",
			"RoomID":    "10",
			"Amount":    "100.5",
			"Status":    "initial",
		})

		repo := &Repository{cache: client}
		payment, err := repo.FindPayment(context.Background(), "5")
		assert.NoError(t, err)
		assert.Equal(t, "order-1", payment.OrderID)
		assert.Equal(t, 5, payment.UserID)
		assert.Equal(t, "pi_1", payment.PaymentId)
		assert.Equal(t, 10, payment.RoomID)
		assert.Equal(t, 100.5, payment.Amount)
		assert.Equal(t, "initial", payment.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		mock.ExpectHGetAll("user:5").SetErr(errors.New("redis down"))

		repo := &Repository{cache: client}
		payment, err := repo.FindPayment(context.Background(), "5")
		assert.Error(t, err)
		assert.Equal(t, &entities.Payment{}, payment)
	})
}

func TestRemovePayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		mock.ExpectDel("user:5").SetVal(1)

		repo := &Repository{cache: client}
		err := repo.RemovePayment(context.Background(), "5")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		mock.ExpectDel("user:5").SetErr(errors.New("redis down"))

		repo := &Repository{cache: client}
		err := repo.RemovePayment(context.Background(), "5")
		assert.Error(t, err)
	})
}
