package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testWebhookSecret = "test-webhook-secret" //nolint:gosec // test constant

func TestVerifyHMAC_ValidSignature(t *testing.T) {
	body := []byte(`{"event":"push","ref":"refs/heads/main"}`)

	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	assert.True(t, verifyHMAC(body, signature, testWebhookSecret))
}

func TestVerifyHMAC_InvalidSignature(t *testing.T) {
	body := []byte(`{"event":"push"}`)

	assert.False(t, verifyHMAC(body, "sha256=invalid", testWebhookSecret))
}

func TestVerifyHMAC_EmptySignature(t *testing.T) {
	body := []byte(`{"event":"push"}`)

	assert.False(t, verifyHMAC(body, "", testWebhookSecret))
}

func TestVerifyHMAC_WrongSecret(t *testing.T) {
	body := []byte(`{"event":"push"}`)
	correctSecret := "correct-secret" //nolint:gosec // test constant

	mac := hmac.New(sha256.New, []byte(correctSecret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	assert.False(t, verifyHMAC(body, signature, testWebhookSecret))
}

func TestVerifyHMAC_EmptyBody(t *testing.T) {
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write([]byte{})
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	assert.True(t, verifyHMAC([]byte{}, signature, testWebhookSecret))
}
