package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

func testCAPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"}, IsCA: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestRabbitTLSConfig_Disabled(t *testing.T) {
	assert.Nil(t, RabbitTLSConfig(entities.RabbitMQConfig{}))
}

func TestRabbitTLSConfig_NoCA(t *testing.T) {
	cfg := RabbitTLSConfig(entities.RabbitMQConfig{TLS: true, Host: "b.example.com"})
	if assert.NotNil(t, cfg) {
		assert.Equal(t, "b.example.com", cfg.ServerName)
		assert.Nil(t, cfg.RootCAs, "no CA => system roots expected")
	}
}

func TestRabbitTLSConfig_WithCaPem(t *testing.T) {
	cfg := RabbitTLSConfig(entities.RabbitMQConfig{TLS: true, Host: "b.example.com", CaPem: testCAPEM(t)})
	if assert.NotNil(t, cfg) {
		assert.NotNil(t, cfg.RootCAs, "RootCAs populated from valid PEM")
		assert.Equal(t, "b.example.com", cfg.ServerName)
	}
}
