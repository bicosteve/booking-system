package payments

import (
	"errors"
	"fmt"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
)

func CreateStripePayment(conf entities.StripeConfig, data entities.TRXPayload) (*stripe.CheckoutSession, error) {
	stripe.Key = conf.StripeSecret
	amountInCents := data.Payment.Amount * 100
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					UnitAmount: stripe.Int64(int64(amountInCents)),
					Currency:   stripe.String(data.Payment.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(data.Payment.Description),
					},
				},
				Quantity: stripe.Int64(int64(data.Days)),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(conf.SuccessURL),
		CancelURL:  stripe.String(conf.CancelURL),
	}

	params.AddMetadata("order_id", fmt.Sprintf("order_%d", data.RoomID))
	params.AddMetadata("user_id", fmt.Sprintf("user_%d", data.UserID))

	session, err := session.New(params)
	if err != nil {
		return nil, errors.New("stripe payment create session failed")
	}

	return session, nil
}

func GetPaymentStatus(stripeKey, paymentId string) (*stripe.CheckoutSession, error) {
	stripe.Key = stripeKey
	session, err := session.Get(paymentId, nil)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return nil, errors.New("stripe payment get session failed")
	}

	return session, nil
}
