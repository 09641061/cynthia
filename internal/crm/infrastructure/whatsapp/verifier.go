package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Verifier validates Meta WhatsApp webhook verification challenges and HMAC-SHA256 signatures.
type Verifier struct {
	verifyToken string
	appSecret   string
}

// NewVerifier creates a Meta WhatsApp webhook verifier.
func NewVerifier(cfg Config) *Verifier {
	return &Verifier{
		verifyToken: strings.TrimSpace(cfg.VerifyToken),
		appSecret:   strings.TrimSpace(cfg.AppSecret),
	}
}

// VerifyChallenge validates Meta's webhook subscription handshake.
func (v *Verifier) VerifyChallenge(mode, token, challenge string) (string, bool) {
	if v.verifyToken == "" || mode != "subscribe" || challenge == "" {
		return "", false
	}
	if !hmac.Equal([]byte(token), []byte(v.verifyToken)) {
		return "", false
	}
	return challenge, true
}

// VerifySignature verifies the X-Hub-Signature-256 header against the raw request body.
func (v *Verifier) VerifySignature(signatureHeader string, body []byte) bool {
	if v.appSecret == "" {
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
	mac := hmac.New(sha256.New, []byte(v.appSecret))
	mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(provided, expected)
}
