package service

import (
	"context"

	"github.com/bicosteve/booking-system/entities"
)

func (s *Service) SubmitMessage(ctx context.Context, data entities.SMSPayload) error {
	err := s.repo.AddSMSOutbox(ctx, data)
	if err != nil {
		return err
	}

	return nil
}
