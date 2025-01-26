package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/edwinwalela/africastalking-go/pkg/sms"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/crypto/bcrypt"
)

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
	tknBytes := make([]byte, 32)
	_, err := rand.Read(tknBytes)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err.Error())
		return "", err
	}

	// encode the byte slice to URL-safe Base64 string
	tknString := base64.URLEncoding.EncodeToString(tknBytes)

	// Generate the current timestamp and turn it into a string
	expirationTime := time.Now().UTC().Add(10 * time.Minute)
	expirationTimeToString := expirationTime.Format(time.RFC3339)

	// Combine the tokenString and expirationTimeToString to form reset token
	tkn := fmt.Sprintf("%s_%s_%s", tknString, expirationTimeToString, userId)

	return tkn, nil
}

func IsValidResetToken(token string) (bool, string, error) {
	parts := strings.Split(token, "_")
	if len(parts) < 3 {
		entities.MessageLogs.ErrorLog.Println("invalid reset token string")
		return false, "", errors.New("token string is invalid")
	}

	tokenExpirationStr := parts[1]
	expirationTime, err := time.Parse(time.RFC3339, tokenExpirationStr)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return false, "", errors.New(err.Error())
	}

	userId := parts[2]

	return expirationTime.Before(time.Now().UTC()), userId, nil
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
