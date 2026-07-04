package utils

import "testing"

// NewRabbitMQConnection must return an error for an unreachable broker, not
// call log.Fatalf (which would os.Exit and abort the test binary).
func TestNewRabbitMQConnection_InvalidURL(t *testing.T) {
	conn, err := NewRabbitMQConnection("amqp://guest:guest@127.0.0.1:1/")
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("expected error for invalid rabbitmq url, got nil")
	}
}
