package service

import (
	"context"

	"github.com/bicosteve/booking-system/entities"
)

func (u *UserService) SubmitMessage(ctx context.Context, data entities.SMSPayload) error {
	err := u.userRepository.AddSMSOutbox(ctx, data)
	if err != nil {
		return err
	}

	return nil
}
