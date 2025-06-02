package controllers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/go-chi/chi/v5"
)

func (b *Base) CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)

	var payload = new(entities.RoomPayload)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)

	defer cancel()

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	err = utils.ValidateRoom(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		utils.LogError("error extracting userid from context", entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	user_id, _ := strconv.Atoi(userID)

	p := entities.RoomPayload{
		Cost:   payload.Cost,
		Status: payload.Status,
		Vendor: user_id,
	}

	err = b.roomService.CreateRoom(ctx, p)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusCreated, map[string]string{"msg": "created"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}
}

func (b *Base) FindRoomHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)

	defer cancel()

	roomId := r.URL.Query().Get("room_id")
	status := r.URL.Query().Get("status")

	rooms, err := b.roomService.FindRooms(ctx)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	if roomId != "" {
		room, found := utils.FilterRoomByID(rooms, roomId)
		if found {
			_ = utils.DeserializeJSON(w, http.StatusOK, room)
			return
		}

		utils.ErrorJSON(w, errors.New("error: room id provided not found"), http.StatusNotFound)
		utils.LogError("room not found", entities.ErrorLog, http.StatusNotFound)
		return

	}

	if status != "" {
		rooms, found := utils.FilterRoomByStatus(rooms, status)
		if found {
			_ = utils.DeserializeJSON(w, http.StatusOK, rooms)
			return
		}

		utils.ErrorJSON(w, errors.New("error: room status provided not found"), http.StatusNotFound)
		utils.LogError("room not found", entities.ErrorLog, http.StatusNotFound)
		return

	}

	_ = utils.DeserializeJSON(w, http.StatusOK, rooms)

}

func (b *Base) UpdateARoom(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()

	roomId, err := strconv.Atoi(chi.URLParam(r, "room_id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	userId, _ := strconv.Atoi(userID)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		utils.LogError("error extracting userid from context", entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	var room entities.Room
	var input struct {
		Cost   *float64 `json:"cost"`
		Status *string  `json:"status"`
	}

	err = utils.SerializeJSON(w, r, &input)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	if input.Cost != nil {
		room.Cost = *input.Cost
	}

	if input.Status != nil {
		room.Status = *input.Status
	}

	err = b.roomService.UpdateARoom(ctx, &room, roomId, userId)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return

	}

	err = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"msg": "room updated"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}

}

func (b *Base) DeleteARoom(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()

	roomId, err := strconv.Atoi(chi.URLParam(r, "room_id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		utils.LogError("error extracting userid from context", entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	id, _ := strconv.Atoi(userID)
	err = b.roomService.DeleteARoom(ctx, roomId, id)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusOK, map[string]string{"msg": "Deleted"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}

}
