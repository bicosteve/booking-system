package controllers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/bicosteve/booking-system/service"
)

func (b *Base) RegisterHandler(s *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		err = s.SubmitRegistrationRequest(ctx, *payload)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusInternalServerError)
			entities.MessageLogs.ErrorLog.Println(err)
			return
		}

		err = utils.DeserializeJSON(w, http.StatusOK, map[string]string{"msg": "success"})
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			entities.MessageLogs.ErrorLog.Println(err)
			return
		}
	}

}

func (b *Base) LoginHandler(s *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		token, err := s.SubmitLoginRequest(ctx, *payload, b.jwtSecret)
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

}

func (b *Base) ProfileHandler(s *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", b.contentType)
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
		defer cancel()

		userName, ok := r.Context().Value(entities.UsernameKeyValue).(string)
		if !ok {
			utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
			entities.MessageLogs.ErrorLog.Println("error extracting username from context")
			return
		}

		user, err := s.SubmitProfileRequest(ctx, userName)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusInternalServerError)
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

}

func (b *Base) GenerateResetTokenHandler(s *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", b.contentType)
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
		defer cancel()

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

		userName, ok := r.Context().Value(entities.UsernameKeyValue).(string)
		if !ok {
			utils.ErrorJSON(w, errors.New("internal server error"), http.StatusInternalServerError)
			entities.MessageLogs.ErrorLog.Println("error extracting username from context")
			return
		}

		if *payload.Email != userName {
			utils.ErrorJSON(w, errors.New("bad request"), http.StatusBadRequest)
			entities.MessageLogs.ErrorLog.Println("session useraname mismatch!")
			return
		}

		user, err := s.SubmitProfileRequest(ctx, userName)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusInternalServerError)
			entities.MessageLogs.ErrorLog.Println(err)
			return
		}

		tkn, err := s.InsertPasswordResetToken(ctx, *user)
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
}

func (b *Base) ResetPasswordHandler(s *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			Password        *string
			ConfirmPassword *string
		}

		err := utils.SerializeJSON(w, r, &payload)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			entities.MessageLogs.ErrorLog.Println(err.Error())
			return
		}

		if payload.Password != payload.ConfirmPassword {
			utils.ErrorJSON(w, errors.New("confirm password and password  mismatch"), http.StatusBadRequest)
			entities.MessageLogs.ErrorLog.Println(errors.New("confirm password and password  are required"))
			return
		}

		if payload.Password == nil {
			utils.ErrorJSON(w, errors.New("password  are required"), http.StatusBadRequest)
			entities.MessageLogs.ErrorLog.Println(errors.New("confirm password and password  are required"))
			return
		}

		if payload.ConfirmPassword == nil {
			utils.ErrorJSON(w, errors.New("password  are required"), http.StatusBadRequest)
			entities.MessageLogs.ErrorLog.Println(errors.New("confirm password and password  are required"))
			return
		}

		err = s.SubmitPasswordResetRequest(ctx, payload.Password, tkn)
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
}
