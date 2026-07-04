package payments

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v72"
)

// withMockStripeBackend points the stripe SDK at a local httptest server so the
// package functions can be exercised without contacting the real Stripe API.
func withMockStripeBackend(t *testing.T, handler http.HandlerFunc) func() {
	t.Helper()
	srv := httptest.NewServer(handler)

	backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		URL: stripe.String(srv.URL),
	})
	stripe.SetBackend(stripe.APIBackend, backend)

	return func() {
		srv.Close()
		// reset backend so other tests are unaffected
		stripe.SetBackend(stripe.APIBackend, nil)
	}
}

func TestCreateStripePayment_Success(t *testing.T) {
	cleanup := withMockStripeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"pi_123","object":"payment_intent","client_secret":"pi_123_secret","status":"requires_payment_method"}`))
	})
	defer cleanup()

	conf := entities.StripeConfig{StripeSecret: "sk_test_dummy"}
	data := entities.TRXPayload{
		OrderID: "order-1",
		UserID:  7,
		Payment: entities.PaymentBody{Amount: 100},
	}

	pi, err := CreateStripePayment(conf, data)
	assert.NoError(t, err)
	assert.NotNil(t, pi)
	assert.Equal(t, "pi_123", pi.ID)
	assert.Equal(t, "pi_123_secret", pi.ClientSecret)
}

func TestCreateStripePayment_Error(t *testing.T) {
	cleanup := withMockStripeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"invalid amount","type":"invalid_request_error"}}`))
	})
	defer cleanup()

	conf := entities.StripeConfig{StripeSecret: "sk_test_dummy"}
	data := entities.TRXPayload{Payment: entities.PaymentBody{Amount: -1}}

	pi, err := CreateStripePayment(conf, data)
	assert.Error(t, err)
	assert.Nil(t, pi)
	assert.EqualError(t, err, "stripe payment create session failed")
}

func TestGetPaymentStatus_Success(t *testing.T) {
	cleanup := withMockStripeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"pi_999","object":"payment_intent","status":"succeeded"}`))
	})
	defer cleanup()

	pi, err := GetPaymentStatus("sk_test_dummy", "pi_999")
	assert.NoError(t, err)
	assert.NotNil(t, pi)
	assert.Equal(t, "pi_999", pi.ID)
	assert.Equal(t, stripe.PaymentIntentStatus("succeeded"), pi.Status)
}

func TestGetPaymentStatus_Error(t *testing.T) {
	cleanup := withMockStripeBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"no such payment_intent","type":"invalid_request_error"}}`))
	})
	defer cleanup()

	pi, err := GetPaymentStatus("sk_test_dummy", "pi_missing")
	assert.Error(t, err)
	assert.Nil(t, pi)
	assert.EqualError(t, err, "stripe payment get session failed")
}
