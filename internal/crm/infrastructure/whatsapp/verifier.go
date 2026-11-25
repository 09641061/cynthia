package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

// Verifier validates Meta WhatsApp webhook verification challenges and HMAC-SHA256 signatures.
type Verifier struct{}

// NewVerifier creates a Meta WhatsApp webhook verifier.
func NewVerifier() *Verifier { return &Verifier{} }

// VerifyChallenge validates Meta's webhook subscription handshake.
func (v *Verifier) VerifyChallenge(mode, token, challenge string) (string, bool) {
	expected := strings.TrimSpace(os.Getenv("WHATSAPP_VERIFY_TOKEN"))
	if expected == "" || mode != "subscribe" || challenge == "" {
		return "", false
	}
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return "", false
	}
	return challenge, true
}

// VerifySignature verifies the X-Hub-Signature-256 header against the raw request body.
func (v *Verifier) VerifySignature(signatureHeader string, body []byte) bool {
	secret := strings.TrimSpace(os.Getenv("WHATSAPP_APP_SECRET"))
	if secret == "" {
		return false
	}
	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	providedHex := strings.TrimPrefix(signatureHeader, prefix)
	provided, err := hex.DecodeString(providedHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(provided, expected)
}
