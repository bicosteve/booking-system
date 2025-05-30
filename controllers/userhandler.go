package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	_ "github.com/swaggo/http-swagger/v2"
)

// @Summary Registers User
// @Description **Receives user payload, validate it then send it to service
// @ID register user
// @Tags Register
// @Accept application/json
// @Produce application/json
// @Success 201 {object} booking.system "User registered"
// @Failure 400 {object} booking.system "Invalid payload"
// @Failure 500 {object} booking.system "Internal server error"
// @Router /api/user/register
func (b *Base) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	var payload = new(entities.UserPayload)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.ValidateUser(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = b.userService.SubmitRegistrationRequest(ctx, *payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusCreated, map[string]string{"msg": "success"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}
}

// @Summary Login User
// @Description **Receives user payload, validate it then send it to service
// @ID login user
// @Tags Login
// @Accept application/json
// @Produce application/json
// @Success 200 {object} booking.system "Generate token"
// @Failure 404 {object} booking.system "User not found"
// @Failure 500 {object} booking.system "Internal server error"
// @Router /api/user/login
func (b *Base) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	var payload = new(entities.UserPayload)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.ValidateLogin(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	token, err := b.userService.SubmitLoginRequest(ctx, *payload, b.jwtSecret)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// if token len is 0, meaning passwords do not match
	if len(token) < 1 {
		err = utils.DeserializeJSON(w, http.StatusBadRequest, map[string]string{"msg": "password does not match username"})
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			entities.MessageLogs.ErrorLog.Println(err)
			return
		}
		return
	}

	r.Header.Set("Authorization", "Bearer "+token)

	err = utils.DeserializeJSON(w, http.StatusOK, map[string]string{"token": token})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}
}

// @Summary Get a  User
// @Description **Receives user payload, validate it then send it to service
// @ID user profile
// @Tags Profile
// @Accept application/json
// @Produce application/json
// @Success 200 {object} booking.system "Returns user"
// @Failure 500 {object} booking.system "Internal server error"
// @Router /api/user/me
func (b *Base) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	userName, ok := r.Context().Value(entities.UsernameKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println("error extracting username from context")
		return
	}

	user, err := b.userService.SubmitProfileRequest(ctx, userName)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		entities.MessageLogs.ErrorLog.Println(err)
		return

	}

	err = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"user": user})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}
}

// @Summary Generate Password Reset Token
// @Description **Receives user payload, validate it then send it to service
// @ID reset token
// @Tags Token
// @Accept application/json
// @Produce application/json
// @Success 200 {object} booking.system "Returns user"
// @Failure 500 {object} booking.system "Internal server error"
// @Router /api/user/reset
func (b *Base) GenerateResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	userName, ok := r.Context().Value(entities.UsernameKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println("error extracting username from context")
		return
	}

	phoneNumber, ok := r.Context().Value(entities.PhoneNumberKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println("error extracting phonenumber from context")
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println("error extracting userid from context")
		return
	}

	// Payload can come with or without some values but
	var payload struct {
		Email       *string `json:"email"`
		PhoneNumber *string `json:"phone_number"`
		IsVendor    *string `json:"is_vendor"`
		Password    *string `json:"password"`
	}

	err := utils.SerializeJSON(w, r, &payload)
	if err != nil {
		utils.ErrorJSON(w, errors.New(err.Error()), http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	if payload.Email == nil {
		utils.ErrorJSON(w, errors.New("bad request"), http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println("email is required field")
		return
	}

	if *payload.Email != userName {
		utils.ErrorJSON(w, errors.New("bad request"), http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println("session useraname mismatch!")
		return
	}

	user, err := b.userService.SubmitProfileRequest(ctx, userName)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	tkn, err := b.userService.InsertPasswordResetToken(ctx, b.DB, *user)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	msg := entities.SMSPayload{
		UserID:  userID,
		Message: tkn,
	}

	err = b.userService.SubmitMessage(ctx, msg)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	ujumbe := fmt.Sprintf("Your reset token is '%s' ", tkn)

	_, err = utils.SendMail(b.sengridkey, b.mailfrom, "Reset Token", *payload.Email, ujumbe)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	_, err = utils.SendSMS(b.atklng, b.appusername, phoneNumber, ujumbe)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusCreated, map[string]interface{}{"reset_tkn": tkn})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

}

// @Summary Reset Password
// @Description **Receives user payload, validate it then send it to service
// @ID reset token
// @Tags Token
// @Accept application/json
// @Produce application/json
// @Success 200 {object} booking.system "Returns success"
// @Failure 500 {object} booking.system "Internal server error"
// @Router /api/user/password-reset?token={token}
func (b *Base) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	tkn := r.URL.Query().Get("token")
	if len(tkn) < 1 {
		utils.ErrorJSON(w, errors.New("reset token is required"), http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println("reset token is not provided in query param")
		return
	}

	var payload struct {
		Password        *string `json:"password"`
		ConfirmPassword *string `json:"confirm-password"`
	}

	err := utils.SerializeJSON(w, r, &payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return
	}

	if *payload.Password != *payload.ConfirmPassword {
		utils.ErrorJSON(w, errors.New("confirm password and password  mismatch"), http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(errors.New("confirm password and password  are required"))
		return
	}

	if payload.Password == nil {
		utils.ErrorJSON(w, errors.New("password  is required"), http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(errors.New("confirm password and password  are required"))
		return
	}

	if payload.ConfirmPassword == nil {
		utils.ErrorJSON(w, errors.New("confirm password  is required"), http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(errors.New("confirm password and password  are required"))
		return
	}

	err = b.userService.SubmitPasswordResetRequest(ctx, b.DB, payload.Password, tkn)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(errors.New(err.Error()))
		return
	}

	err = utils.DeserializeJSON(w, http.StatusCreated, map[string]interface{}{"msg": "password successfully reset"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

}
