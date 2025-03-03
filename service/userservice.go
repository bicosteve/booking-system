package service

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/bicosteve/booking-system/repo"
)

type UserService struct {
	userRepository repo.Repository
}

func NewUserService(userRepository repo.Repository) *UserService {
	return &UserService{userRepository: userRepository}
}

func (s *UserService) SubmitRegistrationRequest(ctx context.Context, data entities.UserPayload) error {
	isAvailable, err := s.userRepository.FindUserByEmail(ctx, data.Email)
	if err != nil {
		return err
	}

	if isAvailable {
		return errors.New("user already registered")
	}

	err = s.userRepository.CreateUser(ctx, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) SubmitLoginRequest(ctx context.Context, data entities.UserPayload, secret string) (string, error) {
	isAvailable, err := s.userRepository.FindUserByEmail(ctx, data.Email)
	if err != nil {
		return "", err
	}

	if !isAvailable {
		return "", errors.New("user is not available")
	}

	user, err := s.userRepository.FindAProfile(ctx, data.Email)
	if err != nil {
		return "", err
	}

	isValid := utils.ComparePasswordWithHash(data.Password, &user.Password)
	if !isValid {
		return "", err
	}

	token, err := utils.GenerateAuthToken(*user, secret)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) SubmitProfileRequest(ctx context.Context, email string) (*entities.User, error) {

	user, err := s.userRepository.FindAProfile(ctx, email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) InsertPasswordResetToken(ctx context.Context, d *sql.DB, user entities.User) (string, error) {

	resetToken, err := utils.GenerateResetToken(user.ID)
	if err != nil {
		return "", err
	}

	err = s.userRepository.InsertPasswordResetToken(ctx, resetToken, user.Email)
	if err != nil {
		return "", err
	}

	return resetToken, nil
}

func (s *UserService) SubmitPasswordResetRequest(ctx context.Context, d *sql.DB, password *string, tkn string) error {

	isValid, id, err := utils.IsValidResetToken(tkn)
	if err != nil {
		return err
	}

	if !isValid {
		return errors.New("reset token has expired")
	}

	userId, _ := strconv.Atoi(id)

	err = s.userRepository.UpdatePassword(ctx, password, userId)

	if err != nil {
		return err
	}

	return nil
}
