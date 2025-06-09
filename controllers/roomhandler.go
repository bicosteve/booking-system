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

// Create a room godoc
// @Summary Admin user create a room
// @Description Receives room payload, validate it then send it to service
// @ID create-room
// @Tags rooms
// @Accept json
// @Produce json
// @Param  payload body entities.RoomPayload true "Create room"
// @Success 201 {object} entities.JSONResponse "{"msg":"created"}"
// @Failure 401 {object} entities.JSONResponse "Unauthorized"
// @Failure 500 {object} entities.JSONResponse "Internal server error"
// @Router /api/admin/rooms [post]
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

// Get a room godoc
// @Summary Get a room by rooms and filter by ID
// @Description Retrieve all rooms and filter using query param
// @ID  get-rooms
// @Tags rooms
// @Accept json
// @Produce json
// @Param room_id query string false "Room ID to filter"
// @Param status query string false "Room status to filter"
// @Success 200 {array} entities.Room "List of rooms (if no filter or multiple matches)"
// @Success 200 {object} entities.Room "Single room (if exact match)"
// @Failure 400 {object} entities.JSONResponse "Bad request, validation error"
// @Failure 404 {object} entities.JSONResponse "Bad request, room not found"
// @Failure 500 {object} entities.JSONResponse "Internal server error"
// @Router /api/user/rooms [get]
// @Security []
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

// Update a room godoc
// @Summary update a room
// @Description Receives room payload, validates it, then updates the room by identified room_id
// @ID update-room
// @Tags rooms
// @Accept json
// @Produce json
// @Param room_id path string true "Room ID to update"
// @Param  payload body entities.RoomPayload true "Room update payload"
// @Success 200 {object} entities.JSONResponse "Room updated successfully"
// @Failure 401 {object} entities.JSONResponse "Unauthorized"
// @Failure 404 {object} entities.JSONResponse "Room not found"
// @Failure 500 {object} entities.JSONResponse "Internal server error"
// @Router /api/admin/rooms/{room_id} [put]
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

// Delete a room godoc
// @Summary delete a room
// @Description Receives room_id and deletes the room
// @ID delete-room
// @Tags rooms
// @Accept json
// @Produce json
// @Param room_id path string true "Room ID to delete"
// @Success 200 {object} entities.JSONResponse "Room deleted successfully"
// @Failure 401 {object} entities.JSONResponse "Unauthorized"
// @Failure 404 {object} entities.JSONResponse "Room not found"
// @Failure 500 {object} entities.JSONResponse "Internal server error"
// @Router /api/admin/rooms/{room_id} [delete]
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
