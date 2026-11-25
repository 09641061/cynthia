package crypto

import (
	"strings"
	"testing"

	"go-service/internal/iam/domain"
)

func TestArgon2idHashAndVerify(t *testing.T) {
	hasher := NewArgon2idHasher()
	encoded, err := hasher.Hash("TakoduDev2026!")
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("unexpected PHC format: %s", encoded)
	}
	ok, err := hasher.Verify("TakoduDev2026!", encoded)
	if err != nil || !ok {
		t.Fatalf("Verify valid password = (%v, %v), want (true, nil)", ok, err)
	}
	ok, err = hasher.Verify("wrong-password", encoded)
	if err != nil || ok {
		t.Fatalf("Verify wrong password = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestArgon2idRejectsMalformedAndEmpty(t *testing.T) {
	hasher := NewArgon2idHasher()
	if _, err := hasher.Hash(""); err == nil {
		t.Fatal("expected error for empty password")
	}
	if _, err := hasher.Verify("pw", "not-a-phc-hash"); err == nil {
		t.Fatal("expected error for malformed hash")
	}
	badParams := []string{
		"$argon2id$v=19$m=65536,t=0,p=2$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"$argon2id$v=19$m=65536,t=3,p=0$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"$argon2id$v=19$m=8,t=3,p=2$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
		"$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHRzYWx0c2FsdA$",
	}
	for _, encoded := range badParams {
		if _, err := hasher.Verify("pw", encoded); err == nil {
			t.Fatalf("expected error for invalid params %s", encoded)
		}
	}
}

func TestGenerateSecureTokenAndHash(t *testing.T) {
	raw, hash, err := domain.GenerateSecureToken()
	if err != nil {
		t.Fatalf("GenerateSecureToken failed: %v", err)
	}
	if raw == "" || len(hash) != 64 {
		t.Fatalf("unexpected token/hash lengths: raw=%d hash=%d", len(raw), len(hash))
	}
	if got := domain.HashTokenSHA256(raw); got != hash {
		t.Fatalf("HashTokenSHA256 mismatch: got %s want %s", got, hash)
	}
}
