// Package session manages JWT access tokens and refresh tokens.
//
// Design:
//   - Access token:  short-lived (15 min), signed HS256 JWT.
//   - Refresh token: long-lived (7 days), opaque 256-bit random token stored
//                    in an in-memory map (swap for Redis at scale).
//   - Logout:        adds JTI (JWT ID) to an in-memory blacklist with TTL cleanup.
package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken  = errors.New("session: invalid or expired token")
	ErrTokenRevoked  = errors.New("session: token has been revoked")
	ErrRefreshExpired = errors.New("session: refresh token expired or not found")
)

// Claims are embedded in the JWT payload.
type Claims struct {
	UserID   string `json:"uid"`
	Username string `json:"sub"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// TokenPair is returned after a successful login.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// refreshEntry stores a refresh token and its owner.
type refreshEntry struct {
	UserID   string
	Username string
	Role     string
	Expiry   time.Time
}

// Manager handles token issuance, validation, refresh, and revocation.
type Manager struct {
	secret         []byte
	accessExpiry   time.Duration
	refreshExpiry  time.Duration

	mu          sync.Mutex
	blacklist   map[string]time.Time  // jti → expiry (for revocation)
	refreshMap  map[string]refreshEntry // token → entry
}

// NewManager creates a token manager with the given HMAC secret and expiry settings.
func NewManager(secret string, accessExpiry, refreshExpiry time.Duration) *Manager {
	m := &Manager{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		blacklist:     make(map[string]time.Time),
		refreshMap:    make(map[string]refreshEntry),
	}
	go m.cleanupLoop()
	return m
}

// Issue generates a new access + refresh token pair for a user.
func (m *Manager) Issue(userID, username, role string) (*TokenPair, error) {
	jti, err := randomToken(16)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiry := now.Add(m.accessExpiry)

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiry),
			Issuer:    "secure-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return nil, fmt.Errorf("session: sign token: %w", err)
	}

	// Refresh token is an opaque 256-bit random string.
	rt, err := randomToken(32)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.refreshMap[rt] = refreshEntry{
		UserID:   userID,
		Username: username,
		Role:     role,
		Expiry:   now.Add(m.refreshExpiry),
	}
	m.mu.Unlock()

	return &TokenPair{
		AccessToken:  signed,
		RefreshToken: rt,
		ExpiresAt:    expiry,
	}, nil
}

// Validate parses and validates a JWT, returning its claims.
func (m *Manager) Validate(tokenStr string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("session: unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Check blacklist.
	m.mu.Lock()
	_, revoked := m.blacklist[claims.ID]
	m.mu.Unlock()
	if revoked {
		return nil, ErrTokenRevoked
	}

	return claims, nil
}

// Refresh exchanges a valid refresh token for a new token pair.
func (m *Manager) Refresh(refreshToken string) (*TokenPair, error) {
	m.mu.Lock()
	entry, ok := m.refreshMap[refreshToken]
	if ok {
		delete(m.refreshMap, refreshToken) // rotate: single-use
	}
	m.mu.Unlock()

	if !ok || time.Now().After(entry.Expiry) {
		return nil, ErrRefreshExpired
	}
	return m.Issue(entry.UserID, entry.Username, entry.Role)
}

// Revoke adds the token's JTI to the blacklist (logout).
func (m *Manager) Revoke(claims *Claims) {
	m.mu.Lock()
	m.blacklist[claims.ID] = claims.ExpiresAt.Time
	m.mu.Unlock()
}

// RevokeRefresh removes a refresh token (logout from all sessions).
func (m *Manager) RevokeRefresh(refreshToken string) {
	m.mu.Lock()
	delete(m.refreshMap, refreshToken)
	m.mu.Unlock()
}

// cleanupLoop periodically evicts expired blacklist and refresh entries.
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		m.mu.Lock()
		for jti, exp := range m.blacklist {
			if now.After(exp) {
				delete(m.blacklist, jti)
			}
		}
		for rt, e := range m.refreshMap {
			if now.After(e.Expiry) {
				delete(m.refreshMap, rt)
			}
		}
		m.mu.Unlock()
	}
}

// GeneratePublicToken is an exported wrapper around the internal CSPRNG helper.
// Used by middleware for request ID generation.
func GeneratePublicToken(byteLen int) (string, error) {
	return randomToken(byteLen)
}

func randomToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session: CSPRNG failure: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
