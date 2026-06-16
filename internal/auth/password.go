package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns the bcrypt hash of a plaintext password. The plaintext
// is never logged or returned anywhere.
func HashPassword(plaintext string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ComparePassword reports whether plaintext matches the stored bcrypt hash. It
// uses bcrypt's constant-time comparison and returns false on any mismatch.
func ComparePassword(hash, plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}
