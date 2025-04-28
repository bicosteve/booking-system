package service

import "github.com/bicosteve/booking-system/repo"

type UserService struct {
	userRepository repo.Repository
}

type RoomService struct {
	roomRepository repo.Repository
}

type BookingService struct {
	bookingRepository repo.Repository
}

type PaymentService struct {
	paymentRepository repo.Repository
}

func NewUserService(userRepository repo.Repository) *UserService {
	return &UserService{userRepository: userRepository}
}

func NewRoomService(roomRepository repo.Repository) *RoomService {
	return &RoomService{roomRepository: roomRepository}
}

func NewBookingService(bookingRepository repo.Repository) *BookingService {
	return &BookingService{bookingRepository: bookingRepository}

}

func NewPaymentService(paymentRepository repo.Repository) *PaymentService {
	return &PaymentService{paymentRepository: paymentRepository}
}
