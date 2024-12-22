package utils

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/bicosteve/booking-system/pkg/entities"
)

// Serialize the incoming Json payload
func SerializeJSON(w http.ResponseWriter, r *http.Request, data any) error {

	maxBytes := 1048576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	r.Close = true

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&data)
	if err != nil {
		return err
	}

	err = decoder.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("invalid JSON format value")
	}

	return nil
}

// Deserialize the outgoing data from server to json format
func DeserializeJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {

	out, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value

		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}

	payload := entities.JSONResponse{
		Error:   true,
		Message: err.Error(),
	}

	err = DeserializeJSON(w, statusCode, payload)
	if err != nil {
		return err
	}

	return nil
}

func ValidateUser(data *entities.UserPayload) error {

	if data.Email == "" {
		return errors.New("email is required")
	}

	if !entities.EmailRegex.MatchString(data.Email) {
		return errors.New("valid email needed")
	}

	if data.PhoneNumber == "" {
		return errors.New("phone number is required")
	}

	if data.Password == "" {
		return errors.New("password is required")
	}

	if data.ConfirmPassword == "" {
		return errors.New("confirm password is required")
	}

	if strings.Compare(data.Password, data.ConfirmPassword) != 0 {
		return errors.New("password and confirm password is must match")
	}

	return nil
}

func ValidateLogin(data *entities.UserPayload) error {

	if data.Email == "" {
		return errors.New("email is required")
	}

	if !entities.EmailRegex.MatchString(data.Email) {
		return errors.New("valid email needed")
	}

	if data.Password == "" {
		return errors.New("password is required")
	}

	return nil
}
