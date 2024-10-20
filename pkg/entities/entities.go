package entities

import (
	"log"
	"time"
)

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	Password    string    `json:"password"`
	IsSeller    bool      `json:"is_seller"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserPayload struct {
	Email           string `json:"email"`
	PhoneNumber     string `json:"phone_number"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type Envelope map[string]interface{}

type Message struct {
	InfoLog  *log.Logger
	ErrorLog *log.Logger
}

type JSONResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
