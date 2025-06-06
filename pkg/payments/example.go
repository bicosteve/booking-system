package payments

import (
	"errors"
	"fmt"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

type PaymentClient interface {
	CreatePayment(amount float64, userId, orderId int) (*stripe.PaymentIntent, error)
	GetPaymentStatus(paymentId string) (*stripe.PaymentIntent, error)
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

func (p payment) CreatePayment(amount float64, userId int, orderId int) (*stripe.PaymentIntent, error) {

	stripe.Key = p.stripeSecretKey
	amountInCents := amount * 100
	_ = amountInCents

	// Customize the checkout page
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(int64(amountInCents)),
		Currency:           stripe.String(string(stripe.CurrencyKES)),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}

	/*params := &stripe.CheckoutSessionParams{
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
	session, err := session.New(params) */
	params.AddMetadata("order_id", fmt.Sprintf("%d", orderId))
	params.AddMetadata("user_id", fmt.Sprintf("%d", userId))

	pi, err := paymentintent.New(params)
	if err != nil {
		utils.LogError(err.Error(), entities.ErrorLog)
		return nil, errors.New("payment create intent failed")
	}

	return pi, nil
}

func (p payment) GetPaymentStatus(paymentId string) (*stripe.PaymentIntent, error) {
	stripe.Key = p.stripeSecretKey
	params := &stripe.PaymentIntentParams{}
	result, err := paymentintent.Get(paymentId, params)
	if err != nil {
		utils.LogError(err.Error(), entities.ErrorLog)
		return nil, errors.New("payment get paymentintent failed")
	}
	return result, nil
}
