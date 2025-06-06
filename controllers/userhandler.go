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

type APIResponse struct {
	Msg string
}

// RegisterAccount godoc
// @Summary Registers User
// @Description Receives user payload, validate it then send it to service
// @ID register-user
// @Tags Register
// @Accept json
// @Produce json
// @Param payload body entities.UserPayload true "Register User"
// @Success 201 {object} APIResponse "User registered"
// @Failure 400 {object} entities.JSONResponse "Bad request, validation error"
// @Failure 500 {object} entities.JSONResponse "Internal server error"
// @Router /api/user/register [post]
func (b *Base) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	var payload = new(entities.UserPayload)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	err = utils.ValidateUser(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	err = b.userService.SubmitRegistrationRequest(ctx, *payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusCreated, map[string]string{"msg": "success"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}
}

// Generate auth token godoc
// @Summary Authorize User
// @Description Receives user payload, validate it then send it to service
// @ID login user
// @Tags Login
// @Accept json
// @Produce json
// @Param  payload body entities.UserPayload true "Login User"
// @Success 200 {object} APIResponse "{"token":"xxxxxxxxxxx"}"
// @Failure 400 {object} entities.JSONResponse "Bad Request, validation error"
// @Failure 404 {object} entities.JSONResponse "Bad Request, user not found"
// @Failure 500 {object} entities.JSONResponse "Internal server error"
// @Router /api/user/login [post]
func (b *Base) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	var payload = new(entities.UserPayload)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	err = utils.ValidateLogin(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
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
			utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
			return
		}
		return
	}

	r.Header.Set("Authorization", "Bearer "+token)

	err = utils.DeserializeJSON(w, http.StatusOK, map[string]string{"token": token})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}
}

type APIUserResponse struct {
	User entities.User
}

// Get profile info godoc
// @Summary Get a  User
// @Description Returns logged in user details
// @ID user-profile
// @Tags Profile
// @Produce json
// @Success 200 {object} APIUserResponse "User retrieved successfully"
// @Failure 500 {object} entities.JSONResponse "Internal server error"
// @Router /api/user/me [get]
func (b *Base) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	userName, ok := r.Context().Value(entities.UsernameKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		utils.LogError("error extracting username from context", entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	user, err := b.userService.SubmitProfileRequest(ctx, userName)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusNotFound)
		return

	}

	err = utils.DeserializeJSON(w, http.StatusOK, map[string]any{"user": user})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}
}

// @Summary Generate Password Reset Token
// @Description Receives user payload, validate it then send it to service
// @ID reset-token
// @Tags Token
// @Accept json
// @Produce json
// @Param payload body entities.UserPayload true "Generate auth token"
// @Success 200 {object} APIUserResponse "Returns user"
// @Failure 500 {object} APIUserResponse "Internal server error"
// @Router /api/user/reset [post]
func (b *Base) GenerateResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
	defer cancel()

	userName, ok := r.Context().Value(entities.UsernameKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		utils.LogError("error extracting username from context", entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	phoneNumber, ok := r.Context().Value(entities.PhoneNumberKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		utils.LogError("error extracting phonenumber from context", entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	userID, ok := r.Context().Value(entities.UseridKeyValue).(string)
	if !ok {
		utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
		utils.LogError("error extracting userid from context", entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	// Use pointers since this payload may come with some values empty
	var payload struct {
		Email       *string `json:"email"`
		PhoneNumber *string `json:"phone_number"`
		IsVendor    *string `json:"is_vendor"`
		Password    *string `json:"password"`
	}

	err := utils.SerializeJSON(w, r, &payload)
	if err != nil {
		utils.ErrorJSON(w, errors.New(err.Error()), http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	if payload.Email == nil {
		utils.ErrorJSON(w, errors.New("bad request"), http.StatusBadRequest)
		utils.LogError("email is required field", entities.ErrorLog, http.StatusBadRequest)
		return
	}

	if *payload.Email != userName {
		utils.ErrorJSON(w, errors.New("bad request"), http.StatusBadRequest)
		utils.LogError("session username mismatch!", entities.ErrorLog, http.StatusBadRequest)
		return
	}

	user, err := b.userService.SubmitProfileRequest(ctx, userName)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	tkn, err := b.userService.InsertPasswordResetToken(ctx, b.DB, *user)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	msg := entities.SMSPayload{
		UserID:  userID,
		Message: tkn,
	}

	err = b.userService.SubmitMessage(ctx, msg)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	ujumbe := fmt.Sprintf("Your reset token is '%s' ", tkn)

	_, err = utils.SendMail(b.sengridkey, b.mailfrom, "Reset Token", *payload.Email, ujumbe)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	_, err = utils.SendSMS(b.atklng, b.appusername, phoneNumber, ujumbe)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusInternalServerError)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusCreated, map[string]interface{}{"reset_tkn": tkn})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

}

// @Summary Reset Password
// @Description Receives user payload, validate it then send it to service
// @ID reset-password
// @Tags Token
// @Accept json
// @Produce json
// @Param payload body entities.UserPayload true "Generate auth token"
// @Success 200 {object} APIUserResponse "Returns user"
// @Failure 400 {object} APIUserResponse "Internal server error"
// @Failure 500 {object} APIUserResponse "Internal server error"
// @Router /api/user/password-reset [put]
func (b *Base) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", b.contentType)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	tkn := r.URL.Query().Get("token")
	if len(tkn) < 1 {
		utils.ErrorJSON(w, errors.New("reset token is required"), http.StatusBadRequest)
		utils.LogError("reset token not provided", entities.ErrorLog, http.StatusBadRequest)
		return
	}

	var payload struct {
		Password        *string `json:"password"`
		ConfirmPassword *string `json:"confirm-password"`
	}

	err := utils.SerializeJSON(w, r, &payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	if *payload.Password != *payload.ConfirmPassword {
		utils.ErrorJSON(w, errors.New("confirm password and password  mismatch"), http.StatusBadRequest)
		utils.LogError("confirm password and password are required", entities.ErrorLog, http.StatusBadRequest)
		return
	}

	if payload.Password == nil {
		utils.ErrorJSON(w, errors.New("password  is required"), http.StatusBadRequest)
		utils.LogError("confirm password and password are required", entities.ErrorLog, http.StatusBadRequest)
		return
	}

	if payload.ConfirmPassword == nil {
		utils.ErrorJSON(w, errors.New("confirm password  is required"), http.StatusBadRequest)
		utils.LogError("confirm password and password are required", entities.ErrorLog, http.StatusBadRequest)
		return
	}

	err = b.userService.SubmitPasswordResetRequest(ctx, b.DB, payload.Password, tkn)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusCreated, map[string]interface{}{"msg": "password successfully reset"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.LogError(err.Error(), entities.ErrorLog, http.StatusBadRequest)
		return
	}

}
