package service

import "github.com/bicosteve/booking-system/repo"

type UserService struct {
	userRepository repo.Repository
}

type RoomService struct {
	roomRepository repo.Repository
}

func NewUserService(userRepository repo.Repository) *UserService {
	return &UserService{userRepository: userRepository}
}

func NewRoomService(roomRepository repo.Repository) *RoomService {
	return &RoomService{roomRepository: roomRepository}
}
