package service

import (
	"context"
	"database/sql"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/repo"
)

func SubmitMessage(ctx context.Context, d *sql.DB, data entities.SMSPayload) error {
	err := repo.AddSMSOutbox(ctx, d, data)
	if err != nil {
		return err
	}

	return nil
}
