package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/edwinwalela/africastalking-go/pkg/sms"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/crypto/bcrypt"
)

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

	if data.IsVendor == "" {
		return errors.New("isVendor is required")
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

func ValidateRoom(data *entities.RoomPayload) error {
	if data.Cost == "" {
		return errors.New("room cost is required")
	}

	if data.Status == "" {
		return errors.New("room status required")
	}

	return nil
}

func ValidateBooking(data *entities.BookingPayload) error {
	if data.Days == nil {
		return errors.New("days is required")
	}

	if data.UserID == nil {
		return errors.New("user id is required")
	}

	if data.RoomID == nil {
		return errors.New("room id is required")
	}

	if data.Amount == nil {
		return errors.New("amount is required")
	}

	return nil
}

func GeneratePasswordHash(p string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return "", err
	}

	return string(hashedPassword), nil
}

func ComparePasswordWithHash(password string, hash *string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(*hash), []byte(password))
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return false

	}

	return true
}

func GenerateAuthToken(user entities.User, secret string) (string, error) {
	type claims entities.Claims
	c := &claims{
		Username:    user.Email,
		UserID:      user.ID,
		IsVendor:    user.IsVender,
		PhoneNumber: user.PhoneNumber,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return "", err
	}

	return tokenString, nil
}

func verifyAuthToken(tokenString, secret string) (*entities.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &entities.Claims{}, func(t *jwt.Token) (interface{}, error) {
		_, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			entities.MessageLogs.ErrorLog.Println("error occurent while signing token")
			return nil, fmt.Errorf("error occurent while signing token")
		}

		return []byte(secret), nil
	})

	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return &entities.Claims{}, err
	}

	if !token.Valid {
		entities.MessageLogs.ErrorLog.Println("token is invalid")
		return nil, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(*entities.Claims)
	if !ok {
		entities.MessageLogs.ErrorLog.Println("invalid claims")
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

func GenerateResetToken(userId string) (string, error) {
	// 1.  Generate a random string for the token
	// and encode the byte slice to URL-safe Base64 string
	tknBytes := make([]byte, 32)
	_, err := rand.Read(tknBytes)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return "", err
	}

	tkn := base64.URLEncoding.EncodeToString(tknBytes)

	// 2. Get  Current Time in UTC and add 10 minutes
	currentTime := time.Now().UTC()
	expireAt := currentTime.Add(10 * time.Minute)
	expireTimeInMillis := expireAt.UnixMilli()

	// 3. Combine the tokenString and expirationTimeToString to form reset token
	tknString := fmt.Sprintf("%s|%d|%s", tkn, expireTimeInMillis, userId)

	return tknString, nil

}

func IsValidResetToken(token string) (bool, string, error) {
	// 1. Split token string into 3 parts to separate randStr, timeInMillis, userID
	parts := strings.Split(token, "|")
	if len(parts) < 3 {
		entities.MessageLogs.ErrorLog.Println("invalid reset token")
		return false, "", errors.New("invalid reset token")
	}

	tokenExpirationStr := parts[1]
	userId := parts[2]

	// 2. Convert expiration time from string to int
	timeInInt, err := strconv.Atoi(tokenExpirationStr)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return false, "", err
	}

	// 3. Convert expiration time from milliseconds to time.Time & get current time
	expirationTime := time.UnixMilli(int64(timeInInt)).UTC()
	currentTime := time.Now().UTC()

	// 4. Compare current time with expiration time, return true
	if currentTime.After(expirationTime) {
		return false, "", nil
	}

	return true, userId, nil

}

func SendMail(key, from, subject, to, token string) (int, error) {
	client := sendgrid.NewSendClient(key)
	mail_from := mail.NewEmail("Booking System", from)
	mail_to := mail.NewEmail("User", to)
	plainTextContent := fmt.Sprintf("Your reset token %s. Expires in 10 minutes", token)
	html := "<h1>Hello there! From Booking System</h1>"

	message := mail.NewSingleEmail(mail_from, subject, mail_to, plainTextContent, html)

	res, err := client.Send(message)
	if err != nil {
		return 0, err
	}

	return res.StatusCode, nil
}

func SendSMS(key, username, phoneNumber, msg string) (string, error) {

	client := &sms.Client{
		ApiKey:    key,
		Username:  username,
		IsSandbox: true,
	}

	number := fmt.Sprintf("+254%s", phoneNumber)

	request := &sms.BulkRequest{
		To:            []string{number}, // can have more than one number
		Message:       msg,
		From:          username,      // app username
		BulkSMSMode:   true,          // set to true to avoid overchaging
		Enqueue:       false,         // send to a queue to dispatch later
		RetryDuration: time.Hour * 2, // retries after one hour
	}

	res, err := client.SendBulk(request)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return "", err
	}

	return res.Message, nil
}

func ValidateFilters(f entities.Filters) error {
	if f.Page < 0 || f.Page > 100 {
		return errors.New("page must be between 1 and 100")
	}

	if f.PageSize < 0 || f.PageSize > 20 {
		return errors.New("page size must be between 1 and 20")
	}

	sortSafeList := []string{"id", "cost", "created_at"}

	for _, list := range sortSafeList {
		if list == f.Sort {
			return nil
		} else {
			return errors.New("provided sort parameter is not allowed")
		}

	}

	return nil
}

func FilterRoomByID(rooms []*entities.Room, targetID string) (*entities.Room, bool) {
	for _, item := range rooms {
		if item.ID == targetID {
			return item, true
		}
	}
	return &entities.Room{}, false
}

func FilterRoomByStatus(rooms []*entities.Room, status string) ([]*entities.Room, bool) {
	var _rooms []*entities.Room
	if status != "VACANT" && status != "BOOKED" {
		return nil, false
	}

	for _, room := range rooms {
		if room.Status == status {
			_rooms = append(_rooms, room)
			return _rooms, true
		}
	}

	return nil, false
}
