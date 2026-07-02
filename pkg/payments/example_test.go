package payments

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPaymentClient(t *testing.T) {
	client := NewPaymentClient("sk_test", "https://success", "https://cancel")
	assert.NotNil(t, client)

	p, ok := client.(*payment)
	assert.True(t, ok)
	assert.Equal(t, "sk_test", p.stripeSecretKey)
	assert.Equal(t, "https://success", p.successURL)
	assert.Equal(t, "https://cancel", p.cancelURL)
}

func TestPaymentClient_CreatePayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cleanup := withMockStripeBackend(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"pi_abc","object":"payment_intent","client_secret":"pi_abc_secret"}`))
		})
		defer cleanup()

		client := NewPaymentClient("sk_test", "s", "c")
		pi, err := client.CreatePayment(50, 1, 2)
		assert.NoError(t, err)
		assert.Equal(t, "pi_abc", pi.ID)
	})

	t.Run("error", func(t *testing.T) {
		cleanup := withMockStripeBackend(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"message":"bad","type":"invalid_request_error"}}`))
		})
		defer cleanup()

		client := NewPaymentClient("sk_test", "s", "c")
		pi, err := client.CreatePayment(50, 1, 2)
		assert.Error(t, err)
		assert.Nil(t, pi)
		assert.EqualError(t, err, "payment create intent failed")
	})
}

func TestPaymentClient_GetPaymentStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cleanup := withMockStripeBackend(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"pi_get","object":"payment_intent","status":"succeeded"}`))
		})
		defer cleanup()

		client := NewPaymentClient("sk_test", "s", "c")
		pi, err := client.GetPaymentStatus("pi_get")
		assert.NoError(t, err)
		assert.Equal(t, "pi_get", pi.ID)
	})

	t.Run("error", func(t *testing.T) {
		cleanup := withMockStripeBackend(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":{"message":"missing","type":"invalid_request_error"}}`))
		})
		defer cleanup()

		client := NewPaymentClient("sk_test", "s", "c")
		pi, err := client.GetPaymentStatus("pi_missing")
		assert.Error(t, err)
		assert.Nil(t, pi)
		assert.EqualError(t, err, "payment get paymentintent failed")
	})
}
