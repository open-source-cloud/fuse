package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyHMAC_ValidSignature(t *testing.T) {
	body := []byte(`{"event":"push","ref":"refs/heads/main"}`)
	secret := "my-webhook-secret"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	assert.True(t, verifyHMAC(body, signature, secret))
}

func TestVerifyHMAC_InvalidSignature(t *testing.T) {
	body := []byte(`{"event":"push"}`)
	secret := "my-webhook-secret"

	assert.False(t, verifyHMAC(body, "sha256=invalid", secret))
}

func TestVerifyHMAC_EmptySignature(t *testing.T) {
	body := []byte(`{"event":"push"}`)
	secret := "my-webhook-secret"

	assert.False(t, verifyHMAC(body, "", secret))
}

func TestVerifyHMAC_WrongSecret(t *testing.T) {
	body := []byte(`{"event":"push"}`)
	correctSecret := "correct-secret"
	wrongSecret := "wrong-secret"

	mac := hmac.New(sha256.New, []byte(correctSecret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	assert.False(t, verifyHMAC(body, signature, wrongSecret))
}

func TestVerifyHMAC_EmptyBody(t *testing.T) {
	secret := "my-webhook-secret"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte{})
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	assert.True(t, verifyHMAC([]byte{}, signature, secret))
}
