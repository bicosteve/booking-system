package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/repo"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v72"
)

func newPaymentService(t *testing.T) (*PaymentService, sqlmock.Sqlmock, redismock.ClientMock, func()) {
	t.Helper()
	db, dbMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	rdb, cacheMock := redismock.NewClientMock()
	repository := *repo.NewDBRepository(db, rdb)
	return NewPaymentService(repository), dbMock, cacheMock, func() { db.Close() }
}

func TestPaymentService_GetActivePayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, _, cacheMock, cleanup := newPaymentService(t)
		defer cleanup()

		cacheMock.ExpectHGetAll("user:5").SetVal(map[string]string{
			"OrderId": "order-1",
			"Status":  "initial",
		})

		payment, err := svc.GetActivePayment(context.Background(), "5")
		assert.NoError(t, err)
		assert.Equal(t, "order-1", payment.OrderID)
		assert.Equal(t, "initial", payment.Status)
	})

	t.Run("error", func(t *testing.T) {
		svc, _, cacheMock, cleanup := newPaymentService(t)
		defer cleanup()

		cacheMock.ExpectHGetAll("user:5").SetErr(errors.New("redis down"))

		payment, err := svc.GetActivePayment(context.Background(), "5")
		assert.Error(t, err)
		assert.Equal(t, entities.Payment{}, payment)
	})
}

func TestPaymentService_HoldPayment(t *testing.T) {
	pi := &stripe.PaymentIntent{
		ID:            "pi_1",
		ClientSecret:  "secret",
		Description:   "desc",
		CaptureMethod: stripe.PaymentIntentCaptureMethodManual,
	}
	data := entities.TRXPayload{
		OrderID: "order-1",
		UserID:  5,
		RoomID:  10,
		Payment: entities.PaymentBody{Amount: 100},
	}

	// HoldPayment builds a map that includes time.Now() values. We match on the
	// command name + key via a custom matcher and ignore the field values, but
	// the expected arg list length must still equal the actual one, so we pass a
	// map with the same 14 keys HoldPayment writes.
	expectedFields := map[string]any{
		"OrderId": "", "UserId": "", "Amount": "", "Status": "",
		"PaymentUrl": "", "PaymentId": "", "ClientSecret": "", "TransactionId": "",
		"CustomerId": "", "RoomID": "", "Response": "", "CapturedMethod": "",
		"CreatedAt": "", "UpdatedAt": "",
	}
	hsetMatcher := func(expectedCmd, actualCmd []interface{}) error {
		if len(actualCmd) < 2 {
			return errors.New("unexpected command")
		}
		if actualCmd[0] != "hset" || actualCmd[1] != "user:5" {
			return errors.New("hset key mismatch")
		}
		return nil
	}

	t.Run("success", func(t *testing.T) {
		svc, _, cacheMock, cleanup := newPaymentService(t)
		defer cleanup()

		cacheMock.CustomMatch(hsetMatcher).ExpectHSet("user:5", expectedFields).SetVal(1)

		err := svc.HoldPayment(context.Background(), pi, data)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		svc, _, cacheMock, cleanup := newPaymentService(t)
		defer cleanup()

		cacheMock.CustomMatch(hsetMatcher).ExpectHSet("user:5", expectedFields).SetErr(errors.New("redis down"))

		err := svc.HoldPayment(context.Background(), pi, data)
		assert.Error(t, err)
	})
}

func TestPaymentService_RemovePayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, _, cacheMock, cleanup := newPaymentService(t)
		defer cleanup()

		cacheMock.ExpectDel("user:5").SetVal(1)

		err := svc.RemovePayment(context.Background(), "5")
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		svc, _, cacheMock, cleanup := newPaymentService(t)
		defer cleanup()

		cacheMock.ExpectDel("user:5").SetErr(errors.New("redis down"))

		err := svc.RemovePayment(context.Background(), "5")
		assert.Error(t, err)
	})
}

func TestPaymentService_AddPayment(t *testing.T) {
	data := &entities.TRXPayload{
		RoomID:    10,
		UserID:    5,
		OrderID:   "order-1",
		TrxID:     "trx-1",
		Reference: "ref-1",
		Status:    1,
		Payment:   entities.PaymentBody{Amount: 200},
	}

	t.Run("success", func(t *testing.T) {
		svc, dbMock, _, cleanup := newPaymentService(t)
		defer cleanup()

		dbMock.ExpectPrepare("INSERT INTO transaction").
			ExpectExec().
			WithArgs(10, 5, "order-1", "trx-1", "ref-1", int64(200), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := svc.AddPayment(context.Background(), data)
		assert.NoError(t, err)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		svc, dbMock, _, cleanup := newPaymentService(t)
		defer cleanup()

		dbMock.ExpectPrepare("INSERT INTO transaction").WillReturnError(sql.ErrConnDone)

		err := svc.AddPayment(context.Background(), data)
		assert.Error(t, err)
	})
}

func TestPaymentService_UpdatePayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, dbMock, _, cleanup := newPaymentService(t)
		defer cleanup()

		dbMock.ExpectPrepare("UPDATE transaction SET status").
			ExpectExec().
			WithArgs(1, "trx-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := svc.UpdatePayment(context.Background(), 1, "trx-1")
		assert.NoError(t, err)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		svc, dbMock, _, cleanup := newPaymentService(t)
		defer cleanup()

		dbMock.ExpectPrepare("UPDATE transaction SET status").WillReturnError(sql.ErrConnDone)

		err := svc.UpdatePayment(context.Background(), 1, "trx-1")
		assert.Error(t, err)
	})
}
