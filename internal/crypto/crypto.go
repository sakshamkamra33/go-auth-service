// Package crypto provides cryptographic primitives for the auth service.
//
// Key design decisions:
//   - Argon2id (not bcrypt/SHA-256): memory-hard, GPU-resistant, PHC winner 2015.
//   - CSPRNG via crypto/rand: cryptographically secure, unlike math/rand.
//   - Constant-time comparison via crypto/subtle: prevents timing side-channels.
//   - PHC string format: self-describing, future-proof if params need rotation.
package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

// Argon2id tuning — OWASP recommended minimums (2024):
//   memory=64MiB, iterations=3, parallelism=4  → ~300ms on a modern server.
const (
	memory      uint32 = 64 * 1024 // 64 MiB
	iterations  uint32 = 3
	parallelism uint8  = 4
	saltLen            = 16
	keyLen      uint32 = 32
	phcVersion         = 19 // argon2.Version
)

var (
	ErrInvalidHash         = errors.New("crypto: invalid argon2 hash format")
	ErrIncompatibleVersion = errors.New("crypto: incompatible argon2 version")
	ErrWeakPassword        = errors.New("crypto: password does not meet strength requirements")
)

// HashPassword hashes password with Argon2id and returns a PHC string.
// The PHC string embeds all parameters so VerifyPassword is self-contained.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("crypto: CSPRNG failure: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		phcVersion, memory, iterations, parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword compares a plaintext password against a stored PHC hash.
// Always runs in constant time — safe against timing attacks.
func VerifyPassword(password, encoded string) (bool, error) {
	salt, storedHash, p, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	candidate := argon2.IDKey([]byte(password), salt, p.t, p.m, p.p, p.kl)
	return subtle.ConstantTimeCompare(storedHash, candidate) == 1, nil
}

// GenerateToken returns a URL-safe, base64-encoded cryptographically secure token.
// byteLen controls entropy: 32 → 256 bits (recommended for session tokens).
func GenerateToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto: CSPRNG failure: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidatePasswordStrength enforces NIST SP 800-63B policy:
// 12–128 chars, must contain uppercase, lowercase, digit, and special char.
func ValidatePasswordStrength(pw string) error {
	if l := len(pw); l < 12 {
		return fmt.Errorf("%w: need ≥12 chars (got %d)", ErrWeakPassword, l)
	}
	if len(pw) > 128 {
		return fmt.Errorf("%w: max 128 chars", ErrWeakPassword)
	}
	var up, lo, di, sp bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			up = true
		case unicode.IsLower(r):
			lo = true
		case unicode.IsDigit(r):
			di = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			sp = true
		}
	}
	var missing []string
	if !up {
		missing = append(missing, "uppercase")
	}
	if !lo {
		missing = append(missing, "lowercase")
	}
	if !di {
		missing = append(missing, "digit")
	}
	if !sp {
		missing = append(missing, "special char")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing %s", ErrWeakPassword, strings.Join(missing, ", "))
	}
	return nil
}

// --- internal helpers ---

type phcParams struct{ m, t uint32; p uint8; kl uint32 }

func decodeHash(enc string) (salt, hash []byte, p *phcParams, err error) {
	parts := strings.Split(enc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, nil, ErrInvalidHash
	}
	var ver int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &ver); err != nil || ver != phcVersion {
		return nil, nil, nil, ErrIncompatibleVersion
	}
	p = &phcParams{}
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.m, &p.t, &p.p); err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	if hash, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	p.kl = uint32(len(hash))
	return salt, hash, p, nil
}
