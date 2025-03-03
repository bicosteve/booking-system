package service

import (
	"context"
	"database/sql"

	"github.com/bicosteve/booking-system/entities"
)

func (u *UserService) SubmitMessage(ctx context.Context, d *sql.DB, data entities.SMSPayload) error {
	err := u.userRepository.AddSMSOutbox(ctx, data)
	if err != nil {
		return err
	}

	return nil
}
