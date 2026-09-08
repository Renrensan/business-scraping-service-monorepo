package utils

import (
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "SecretPassword123!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !CheckPasswordHash(password, hash) {
		t.Errorf("Password check failed for correct password")
	}

	if CheckPasswordHash("WrongPassword", hash) {
		t.Errorf("Password check succeeded for wrong password")
	}
}

func TestGenerateRandomToken(t *testing.T) {
	token, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if len(token) != 64 { // 32 bytes hex encoded = 64 chars
		t.Errorf("Expected token length 64, got %d", len(token))
	}
}
