package service

import (
	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/repo"
)

// UserService
type UserService struct {
	repo repo.UserDBRepository
}

func NewUserService(r repo.UserDBRepository) *UserService {
	return &UserService{repo: r}
}

func (s *UserService) SubmitRegistrationRequest(data entities.UserPayload) error {
	err := s.repo.CreateUser(data)
	if err != nil {
		return err
	}

	return nil
}
