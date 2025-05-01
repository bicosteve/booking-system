package payments

import (
	"errors"
	"fmt"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

func CreateStripePayment(conf entities.StripeConfig, data entities.TRXPayload) (*stripe.PaymentIntent, error) {
	stripe.Key = conf.StripeSecret
	amountInCents := data.Payment.Amount * 100

	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(int64(amountInCents)),
		Currency:           stripe.String(string(stripe.CurrencyKES)),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}

	params.AddMetadata("order_id", fmt.Sprintf("order_%s", data.OrderID))
	params.AddMetadata("user_id", fmt.Sprintf("user_%d", data.UserID))

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, errors.New("stripe payment create session failed")
	}

	return pi, nil
}

func GetPaymentStatus(stripeKey, paymentId string) (*stripe.PaymentIntent, error) {
	stripe.Key = stripeKey
	params := &stripe.PaymentIntentParams{}
	result, err := paymentintent.Get(paymentId, params)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return nil, errors.New("stripe payment get session failed")
	}

	return result, nil
}
