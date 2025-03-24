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
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.ValidateRoom(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println("error extracting userid from context")
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
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusCreated, map[string]string{"msg": "created"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}
}

func (b *Base) FindRoomHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)

	defer cancel()

	roomId := r.URL.Query().Get("room_id")

	rooms, err := b.roomService.FindRooms(ctx)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return
	}

	if roomId != "" {
		id, err := strconv.Atoi(roomId)
		if err != nil {
			utils.ErrorJSON(w, errors.New(err.Error()), http.StatusBadRequest)
			entities.MessageLogs.ErrorLog.Println(err.Error())
			return
		}

		for _, room := range rooms {
			room_id, _ := strconv.Atoi(room.ID)
			if room_id == id {
				_ = utils.DeserializeJSON(w, http.StatusOK, room)
				return
			} else {
				utils.ErrorJSON(w, errors.New("error: room id provided not found"), http.StatusNotFound)
				entities.MessageLogs.ErrorLog.Println("room not found")
				return
			}

		}

	}

	_ = utils.DeserializeJSON(w, http.StatusOK, rooms)

}

func (b *Base) UpdateARoom(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()

	roomId, err := strconv.Atoi(chi.URLParam(r, "room_id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(int)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println("error extracting userid from context")
		return
	}

	var room entities.Room
	var input struct {
		Cost   *string `json:"cost"`
		Status *string `json:"status"`
	}

	err = utils.SerializeJSON(w, r, &input)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return
	}

	if input.Cost != nil {
		room.Cost = *input.Cost
	}

	if input.Status != nil {
		room.Status = *input.Status
	}

	err = b.roomService.UpdateARoom(ctx, room, roomId, userID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return

	}

	err = utils.DeserializeJSON(w, http.StatusOK, map[string]interface{}{"data": room})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return
	}

}

func (b *Base) DeleteARoom(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)
	defer cancel()

	roomId, err := strconv.Atoi(chi.URLParam(r, "room_id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(int)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println("error extracting userid from context")
		return
	}

	err = b.roomService.DeleteARoom(ctx, roomId, userID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return
	}

	err = utils.DeserializeJSON(w, http.StatusOK, map[string]string{"msg": "Deleted"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return
	}

}
