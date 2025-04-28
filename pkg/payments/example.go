package payments

import (
	"errors"
	"fmt"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)

type PaymentClient interface {
	CreatePayment(amount float64, userId, orderId int) (*stripe.CheckoutSession, error)
	GetPaymentStatus(paymentId string) (*stripe.CheckoutSession, error)
}

type payment struct {
	stripeSecretKey string
	successURL      string
	cancelURL       string
}

func NewPaymentClient(stripeSecretKey, successURL, cancelURL string) PaymentClient {
	return &payment{
		stripeSecretKey: stripeSecretKey,
		successURL:      successURL,
		cancelURL:       cancelURL,
	}
}

func (p payment) CreatePayment(amount float64, userId int, orderId int) (*stripe.CheckoutSession, error) {

	stripe.Key = p.stripeSecretKey
	amountInCents := amount * 100
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					UnitAmount: stripe.Int64(int64(amountInCents)),
					Currency:   stripe.String("kes"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Room Booking"),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(p.successURL),
		CancelURL:  stripe.String(p.cancelURL),
	}

	params.AddMetadata("order_id", fmt.Sprintf("%d", orderId))
	params.AddMetadata("user_id", fmt.Sprintf("%d", userId))

	session, err := session.New(params)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return nil, errors.New("payment create session failed")
	}

	return session, nil
}

func (p payment) GetPaymentStatus(paymentId string) (*stripe.CheckoutSession, error) {
	stripe.Key = p.stripeSecretKey
	session, err := session.Get(paymentId, nil)
	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return nil, errors.New("payment get session failed")
	}
	return session, nil
}
