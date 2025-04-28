package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/payments"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/go-chi/chi/v5"
)

func (b *Base) CreateBookingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	var payload = new(entities.BookingPayload)

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

	payDetails := entities.TRXPayload{
		RoomID: payload.RoomID,
		UserID: userid,
		Days:   payload.Days,
		Payment: entities.PaymentBody{
			Amount:      int64(payload.Amount),
			Currency:    "kes",
			Customer:    payload.UserID,
			Description: fmt.Sprintf("booking_%d", payload.RoomID),
		},
	}

	stripeConf := entities.StripeConfig{
		StripeSecret: b.stripesecret,
		Publishable:  b.pp_clientid,
		SuccessURL:   b.successURL,
		CancelURL:    b.cancelURL,
	}

	// 1. Check if there is an active payment session or create new payment session
	active, err := b.paymentService.GetActivePayment(ctx, userID)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 1b. If there is a payment in process return bad request
	if active.Status == "initial" {
		entities.MessageLogs.ErrorLog.Println("There is an active payment in waiting")
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return

	}

	// 2. Create Payment Session on Stripe Before Booking
	session, err := payments.CreateStripePayment(stripeConf, payDetails)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 3. Store Payments In Redis
	err = b.paymentService.StorePayment(ctx, session, payDetails)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 4. Make Booking
	err = b.bookingService.MakeBooking(ctx, *payload)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 5. Publish booking payments
	err = utils.QPublishMessage(b.Broker, b.Topic, b.Key, payDetails)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	_ = utils.DeserializeJSON(w, http.StatusCreated, map[string]any{"msg": "booking created", "result": session, "payment_url": session.URL})

}

func (b *Base) GetBookingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bookingID, err := strconv.Atoi(chi.URLParam(r, "booking_id"))
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

	book, err := b.bookingService.GetUserBooking(ctx, bookingID, userid)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	_ = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"data": book})

}

func (b *Base) GetAllHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bookingID, err := strconv.Atoi(chi.URLParam(r, "room_id"))
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		entities.MessageLogs.ErrorLog.Println(errors.New("an error occured"))
		utils.ErrorJSON(w, errors.New("an error occured"), http.StatusInternalServerError)
		return

	}
	userid, _ := strconv.Atoi(userID)

	book, err := b.bookingService.GetUserBooking(ctx, bookingID, userid)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	_ = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"data": book})

}

func (b *Base) GetAllBookingsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		entities.MessageLogs.ErrorLog.Println(errors.New("an error occured"))
		utils.ErrorJSON(w, errors.New("an error occured"), http.StatusInternalServerError)
		return

	}
	userid, _ := strconv.Atoi(userID)

	bookings, err := b.bookingService.GetUserBookings(ctx, userid)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	_ = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"data": bookings})

}

func (b *Base) GetAllAdminBookingsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		entities.MessageLogs.ErrorLog.Println(errors.New("an error occured"))
		utils.ErrorJSON(w, errors.New("an error occured"), http.StatusInternalServerError)
		return

	}
	userid, _ := strconv.Atoi(userID)

	bookings, err := b.bookingService.GetVendoerBookings(ctx, userid)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	_ = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"data": bookings})

}

func (b *Base) UpdateBooking(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bookingID, err := strconv.Atoi(chi.URLParam(r, "booking_id"))
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	var payload = new(entities.BookingPayload)

	err = utils.SerializeJSON(w, r, payload)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	if payload.Days < 0 {
		entities.MessageLogs.ErrorLog.Println("Days in payload cannot be empty")
		utils.ErrorJSON(w, errors.New("days in payload cannot be empty"), http.StatusBadRequest)
		return

	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		entities.MessageLogs.ErrorLog.Println(errors.New("an error occured"))
		utils.ErrorJSON(w, errors.New("an error occured"), http.StatusInternalServerError)
		return

	}
	userid, _ := strconv.Atoi(userID)

	payload.UserID = userid

	err = b.bookingService.UpdateABooking(ctx, payload, bookingID)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	_ = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"msg": payload})

}

func (b *Base) DeleteBooking(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	bookingID, err := strconv.Atoi(chi.URLParam(r, "booking_id"))
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	roomID, err := strconv.Atoi(chi.URLParam(r, "room_id"))
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		entities.MessageLogs.ErrorLog.Println(errors.New("an error occured"))
		utils.ErrorJSON(w, errors.New("an error occured"), http.StatusInternalServerError)
		return
	}

	user_id, _ := strconv.Atoi(userID)

	err = b.bookingService.DeleteABooking(ctx, bookingID, user_id, roomID)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	_ = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"msg": "success"})
}
