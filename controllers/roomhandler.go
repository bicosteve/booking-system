package controllers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
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

func (b *Base) FindRoomAHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*3)

	defer cancel()

	roomId := r.URL.Query().Get("room_id")
	if roomId == "" {
		utils.ErrorJSON(w, errors.New("room ID is required"), http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println("room ID is empty")
		return
	}

	id, err := strconv.Atoi(roomId)
	if err != nil {
		utils.ErrorJSON(w, errors.New("provided id is invalid"), http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println("provided id is invalid")
		return
	}

	room, err := b.roomService.FindARoom(ctx, id)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return
	}

	err = utils.DeserializeJSON(w, http.StatusOK, room)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return

	}

}
