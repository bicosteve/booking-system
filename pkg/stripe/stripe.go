package stripe

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

func MakePayments(stripeSecret string, data entities.TRXPayload) (*stripe.PaymentIntent, error) {

	stripe.Key = stripeSecret

	params := &stripe.PaymentIntentParams{
		Amount:       stripe.Int64(data.Payment.Amount * 100),
		Currency:     stripe.String(data.Payment.Currency),
		Customer:     stripe.String(data.Payment.Currency),
		UseStripeSDK: stripe.Bool(false),
		Description:  stripe.String(data.Payment.Description),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, err
	}

	return pi, nil
}

func MakeAPIPayments(stripeSecret string, data entities.TRXPayload) (string, error) {
	daata := url.Values{}

	daata.Set("amount", fmt.Sprintf("%d", data.Payment.Amount*100))
	daata.Set("currency", data.Payment.Currency)
	daata.Set("source", "tok_visa")
	daata.Set("description", data.Payment.Description)

	req, err := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1/charges", bytes.NewBufferString(daata.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", stripeSecret))
	req.Header.Add("Content-Type", "application/x-wwww-form-urlencoded")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	return response.Status, nil
}
