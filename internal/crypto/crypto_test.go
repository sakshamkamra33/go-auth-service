package crypto_test

import (
	"strings"
	"testing"

	"github.com/sakshamkamra33/secure-auth/internal/crypto"
)

// TestHashAndVerify — round-trip correctness.
func TestHashAndVerify(t *testing.T) {
	pw := "StrongPass@123!"
	hash, err := crypto.HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	ok, err := crypto.VerifyPassword(pw, hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("expected match, got false")
	}
}

// TestVerifyWrongPassword — wrong password must not verify.
func TestVerifyWrongPassword(t *testing.T) {
	hash, _ := crypto.HashPassword("StrongPass@123!")
	ok, err := crypto.VerifyPassword("WrongPass@999!", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Fatal("wrong password should not verify")
	}
}

// TestHashUniqueness — same password must produce different hashes (salts differ).
func TestHashUniqueness(t *testing.T) {
	pw := "StrongPass@123!"
	h1, _ := crypto.HashPassword(pw)
	h2, _ := crypto.HashPassword(pw)
	if h1 == h2 {
		t.Fatal("two hashes of same password must differ (unique salts)")
	}
}

// TestPHCFormat — hash must follow $argon2id$ PHC format.
func TestPHCFormat(t *testing.T) {
	hash, _ := crypto.HashPassword("StrongPass@123!")
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected PHC format, got: %s", hash)
	}
}

// TestGenerateToken — tokens must be non-empty and unique.
func TestGenerateToken(t *testing.T) {
	t1, err := crypto.GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	t2, _ := crypto.GenerateToken(32)
	if t1 == t2 {
		t.Fatal("two tokens must differ")
	}
	if len(t1) == 0 {
		t.Fatal("token must not be empty")
	}
}

// TestPasswordStrength — table-driven validation tests.
func TestPasswordStrength(t *testing.T) {
	cases := []struct {
		pw    string
		valid bool
	}{
		{"short", false},
		{"alllowercase123!", false},      // no uppercase
		{"ALLUPPERCASE123!", false},      // no lowercase
		{"NoDigitsHere!!", false},        // no digit
		{"NoSpecialChar123", false},      // no special
		{"ValidPass@123!", true},
		{"Another$ecure1Pass", true},
	}
	for _, tc := range cases {
		err := crypto.ValidatePasswordStrength(tc.pw)
		if tc.valid && err != nil {
			t.Errorf("pw=%q: expected valid, got %v", tc.pw, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("pw=%q: expected invalid, got nil", tc.pw)
		}
	}
}
