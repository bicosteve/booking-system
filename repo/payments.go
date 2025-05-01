package repo

import (
	"context"

	"github.com/bicosteve/booking-system/entities"
)

type PayRepository interface {
	SaveTransactions(ctx context.Context, data *entities.TRXPayload) error
	UpdateTransactions(ctx context.Context, data *entities.TRXPayload) error
}

func (r *Repository) SaveTransactions(ctx context.Context, data *entities.TRXPayload) error {

	q := `INSERT INTO transaction(room_id,user_id,order_id, trx_id,reference,amount,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,NOW(),NOW())`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	args := []interface{}{data.RoomID, data.UserID, data.OrderID, data.TrxID, data.Reference, data.Payment.Amount, data.Status}

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) UpdateTransactions(ctx context.Context, status int, trx_id string) error {

	q := `UPDATE transaction SET status = ?, updated_at = NOW() WHERE trx_id = ?`
	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, status, trx_id)
	if err != nil {
		return err
	}

	return nil
}
