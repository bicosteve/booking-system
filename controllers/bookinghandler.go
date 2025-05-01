package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/payments"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	payload.UserID = &userid

	payDetails := entities.TRXPayload{
		RoomID:  *payload.RoomID,
		UserID:  *payload.UserID,
		OrderID: uuid.New().String(),
		Days:    *payload.Days,
		Payment: entities.PaymentBody{
			Amount:      int64(*payload.Amount),
			Currency:    "kes",
			Customer:    *payload.UserID,
			Description: fmt.Sprintf("booking_%d", payload.RoomID),
		},
	}

	stripeConf := entities.StripeConfig{
		StripeSecret: b.stripesecret,
		PubKey:       b.pubkey,
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

	// 2. If there is an active payment i.e status='initial' --> client_secret,pub_key
	if active.Status == "initial" {
		entities.MessageLogs.InfoLog.Println("There is an active payment in waiting")
		_ = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"message": "You have an active payment,confirm payment to proceed", "client_secret": active.ClientSecret, "pub_key": b.pubkey})
		return

	}

	// 3. Create Payment Session on Stripe Before Booking
	PaymentSession, err := payments.CreateStripePayment(stripeConf, payDetails)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 4. Store Payments In Redis
	err = b.paymentService.HoldPayment(ctx, PaymentSession, payDetails)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 5. Make Booking
	err = b.bookingService.MakeBooking(ctx, *payload)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 6. Publish booking payments
	err = utils.QPublishMessage(b.Broker, b.Topic[0], b.Key, payDetails)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 7. Return client_secret, pubkey, room_id
	_ = utils.DeserializeJSON(w, http.StatusCreated, map[string]any{"msg": "booking created", "pubkey": b.pubkey, "client_secret": PaymentSession.ClientSecret, "room_id": payload.RoomID})

}

func (b *Base) VerifyBookingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	booking_id, err := strconv.Atoi(chi.URLParam(r, "room_id"))
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// 1. Get authorized user
	user, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		entities.MessageLogs.ErrorLog.Println(errors.New("could not get logged in user"))
		utils.ErrorJSON(w, errors.New("could not get logged in user"), http.StatusInternalServerError)
		return
	}

	user_id, _ := strconv.Atoi(user)
	booking, err := b.bookingService.GetUserBooking(ctx, booking_id, user_id)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// 2. Do we have active payment?
	active, err := b.paymentService.GetActivePayment(ctx, user)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 3. What is the status
	if active.Status != "initial" {
		entities.MessageLogs.ErrorLog.Printf("no active payment for user -- %s\n", user)
		utils.ErrorJSON(w, errors.New("you do not have active payment"), http.StatusBadRequest)
		return
	}

	// 4. Fetch payment status from stripe
	pi, err := payments.GetPaymentStatus(b.stripesecret, active.PaymentId)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// 5. Can be used to store failed transactions
	payJSON, _ := json.Marshal(pi)
	paylogs := string(payJSON)
	entities.MessageLogs.InfoLog.Println(paylogs)

	// 6. If payment is successful, confirm booking & send sms/email
	if pi.Status != "succeeded" {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	data := entities.BookingPayload{
		Days:   &booking.Days,
		UserID: &user_id,
		RoomID: &booking.RoomID,
		Status: &entities.BookingStatusConfirmed,
	}

	err = b.bookingService.UpdateABooking(ctx, &data, booking_id)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	var status = entities.BookingStatusConfirmed

	// err = b.paymentService.UpdatePayment(ctx, status, pi.ID)
	// if err != nil {
	// 	entities.MessageLogs.ErrorLog.Println(err)
	// 	utils.ErrorJSON(w, err, http.StatusInternalServerError)
	// 	return
	// }

	trx := entities.TRXPayload{
		RoomID:    booking.RoomID,
		UserID:    user_id,
		OrderID:   active.OrderID,
		Reference: active.TransactionID,
		TrxID:     active.PaymentId,
		Status:    status,
		Payment: entities.PaymentBody{
			Amount: int64(active.Amount),
		},
	}

	err = utils.QPublishMessage(b.Broker, b.Topic[1], b.Key, trx)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	err = b.paymentService.RemovePayment(ctx, user)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	_ = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"msg": "booking success"})

}

func (b *Base) GetBookingHandler(w http.ResponseWriter, r *http.Request) {
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

	if *payload.Days < 0 {
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

	payload.UserID = &userid

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
