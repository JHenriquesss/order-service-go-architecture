package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordProducesVerifiableBcryptHash(t *testing.T) {
	const plaintext = "s3cret-pass"
	hash, err := HashPassword(plaintext)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == plaintext || strings.Contains(hash, plaintext) {
		t.Fatal("hash must not contain or equal the plaintext password")
	}
	if !ComparePassword(hash, plaintext) {
		t.Fatal("ComparePassword should accept the correct password")
	}
}

func TestComparePasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if ComparePassword(hash, "wrong-password") {
		t.Fatal("ComparePassword should reject an incorrect password")
	}
	if ComparePassword("not-a-hash", "correct-horse") {
		t.Fatal("ComparePassword should reject an invalid hash")
	}
}

func TestHashPasswordReturnsErrorForOverlongInput(t *testing.T) {
	// bcrypt rejects passwords longer than 72 bytes; the error path must surface.
	long := strings.Repeat("x", 73)
	if _, err := HashPassword(long); err == nil {
		t.Fatal("expected error for password longer than 72 bytes")
	}
}
