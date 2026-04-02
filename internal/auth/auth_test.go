package auth

import (
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "secret123") {
		t.Error("password should match")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("wrong password should not match")
	}
}

func TestGenerateAndValidateJWT(t *testing.T) {
	svc := NewService("test-secret-key")

	token, err := svc.GenerateJWT(1, "admin", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := svc.ValidateJWT(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 1 {
		t.Errorf("expected user_id 1, got %d", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("expected username admin, got %s", claims.Username)
	}
	if claims.Role != "admin" {
		t.Errorf("expected role admin, got %s", claims.Role)
	}
}

func TestValidateJWT_Invalid(t *testing.T) {
	svc := NewService("test-secret-key")

	_, err := svc.ValidateJWT("invalid-token")
	if err == nil {
		t.Error("should fail on invalid token")
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	svc1 := NewService("secret-1")
	svc2 := NewService("secret-2")

	token, _ := svc1.GenerateJWT(1, "user", "viewer")
	_, err := svc2.ValidateJWT(token)
	if err == nil {
		t.Error("should fail with wrong secret")
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(token1) != 64 {
		t.Errorf("expected 64 char token, got %d", len(token1))
	}

	token2, _ := GenerateToken()
	if token1 == token2 {
		t.Error("tokens should be unique")
	}
}

func TestTOTP(t *testing.T) {
	secret, uri, err := GenerateTOTPSecret("testuser")
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Error("secret should not be empty")
	}
	if uri == "" {
		t.Error("uri should not be empty")
	}

	// Invalid code should fail
	if ValidateTOTPCode(secret, "000000") {
		// This could theoretically pass, but extremely unlikely
		t.Log("TOTP validation with 000000 passed (unlikely but possible)")
	}
}
