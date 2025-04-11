package controllers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
)

func (b *Base) BookingHandler(w http.ResponseWriter, r *http.Request) {

	var payload = new(entities.BookingPayload)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.ValidateBooking(payload)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return

	}

	userid, _ := strconv.Atoi(userID)
	payload.UserID = userid

	err = b.bookingService.MakeBooking(ctx, *payload)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	_ = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"msg": "created"})

}
