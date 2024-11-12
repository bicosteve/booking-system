package entities

import (
	"errors"
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

type SerializedUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var ErrNoRecord = errors.New("models: no matching record found")
var ErrDuplicateEmail = errors.New("models: user already exists")
var ErrorInvalidCredentials = errors.New("models: incorrect password or email")
var ErrorDBConnection = errors.New("db: could not connect db becacuse ")
var ErrorDBPing = errors.New("db: could not ping db because ")
var SuccessDBPing = "db: successfully connected to db"
