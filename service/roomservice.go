package service

import (
	"context"

	"github.com/bicosteve/booking-system/entities"
)

func (rs *RoomService) CreateRoom(ctx context.Context, rp entities.RoomPayload) error {
	err := rs.roomRepository.CreateRoom(ctx, rp)
	if err != nil {
		return err
	}

	return nil
}

func (rs *RoomService) FindARoom(ctx context.Context, roomID int) (*entities.Room, error) {
	room, err := rs.roomRepository.FindRoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	return room, nil
}

func (rs *RoomService) UpdateARoom(ctx context.Context, data entities.Room, roomID int) error {

	err := rs.roomRepository.UpdateARoom(ctx, data, roomID)

	if err != nil {
		return err
	}

	return nil
}
