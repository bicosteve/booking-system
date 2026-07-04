package utils

import (
	"testing"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

func TestKafkaConfigMap_Plaintext(t *testing.T) {
	cm := KafkaConfigMap(entities.KakfaConfig{Broker: "localhost:19092"})
	v, _ := cm.Get("bootstrap.servers", "")
	assert.Equal(t, "localhost:19092", v)
	v, _ = cm.Get("security.protocol", "")
	assert.Equal(t, "", v, "plaintext map must not set security.protocol")
}

func TestKafkaConfigMap_SASL_SSL(t *testing.T) {
	cm := KafkaConfigMap(entities.KakfaConfig{
		Broker: "a.example:9092", SecurityProtocol: "SASL_SSL",
		SaslUsername: "u", SaslPassword: "p", CaPem: "PEM",
	})
	v, _ := cm.Get("security.protocol", "")
	assert.Equal(t, "SASL_SSL", v)
	v, _ = cm.Get("sasl.mechanisms", "")
	assert.Equal(t, "SCRAM-SHA-256", v, "default mechanism")
	v, _ = cm.Get("sasl.username", "")
	assert.Equal(t, "u", v)
	v, _ = cm.Get("sasl.password", "")
	assert.Equal(t, "p", v)
	v, _ = cm.Get("ssl.ca.pem", "")
	assert.Equal(t, "PEM", v)
	v, _ = cm.Get("ssl.ca.location", "")
	assert.Equal(t, "", v, "ca.location not set when only CaPem provided")
}

func TestKafkaConfigMap_CaLocationPrecedence(t *testing.T) {
	cm := KafkaConfigMap(entities.KakfaConfig{
		Broker: "a", SecurityProtocol: "SASL_SSL",
		CaPem: "P", CaLocation: "/etc/ca.pem",
	})
	v, _ := cm.Get("ssl.ca.pem", "")
	assert.Equal(t, "", v, "CaLocation takes precedence; ca.pem not set")
	v, _ = cm.Get("ssl.ca.location", "")
	assert.Equal(t, "/etc/ca.pem", v)
}

func TestKafkaConfigMap_CustomMechanism(t *testing.T) {
	cm := KafkaConfigMap(entities.KakfaConfig{Broker: "a", SecurityProtocol: "SASL_SSL", SaslMechanism: "SCRAM-SHA-512"})
	v, _ := cm.Get("sasl.mechanisms", "")
	assert.Equal(t, "SCRAM-SHA-512", v)
}
