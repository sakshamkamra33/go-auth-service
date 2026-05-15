// Package auth implements the authentication business logic.
//
// Features added in v2:
//   - Email verification on register (token stored in-memory with TTL)
//   - Password reset via email token (stateful, single-use, 15-min TTL)
//   - Audit log persistence to FileAuditStore (queryable via admin API)
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/sakshamkamra33/go-auth-service/internal/config"
	"github.com/sakshamkamra33/go-auth-service/internal/crypto"
	"github.com/sakshamkamra33/go-auth-service/internal/email"
	"github.com/sakshamkamra33/go-auth-service/internal/logger"
	"github.com/sakshamkamra33/go-auth-service/internal/session"
	"github.com/sakshamkamra33/go-auth-service/internal/storage"
)

// Service errors.
var (
	ErrInvalidCredentials = errors.New("auth: invalid username or password")
	ErrAccountLocked      = errors.New("auth: account is temporarily locked")
	ErrUserExists         = errors.New("auth: username already taken")
	ErrWeakPassword       = errors.New("auth: password too weak")
	ErrValidation         = errors.New("auth: input validation failed")
	ErrInvalidToken       = errors.New("auth: invalid or expired token")
	ErrEmailNotVerified   = errors.New("auth: email not verified")
)

// RegisterRequest carries user registration input.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest carries login credentials.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// tokenEntry is an in-memory token with expiry and owner.
type tokenEntry struct {
	username string
	expiry   time.Time
}

// Service is the auth business logic layer.
type Service struct {
	store    storage.UserStore
	audit    storage.AuditStore
	sessions *session.Manager
	mailer   *email.Mailer
	cfg      *config.Config

	mu           sync.Mutex
	verifyTokens map[string]tokenEntry // token → {username, expiry}
	resetTokens  map[string]tokenEntry // token → {username, expiry}
}

// NewService constructs an auth service with its dependencies injected.
func NewService(
	store storage.UserStore,
	audit storage.AuditStore,
	sessions *session.Manager,
	mailer *email.Mailer,
	cfg *config.Config,
) *Service {
	s := &Service{
		store:        store,
		audit:        audit,
		sessions:     sessions,
		mailer:       mailer,
		cfg:          cfg,
		verifyTokens: make(map[string]tokenEntry),
		resetTokens:  make(map[string]tokenEntry),
	}
	go s.cleanupTokens()
	return s
}

// Register validates and creates a new user account, then sends a verification email.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*storage.User, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if len(req.Username) < 3 || len(req.Username) > 32 {
		return nil, fmt.Errorf("%w: username must be 3–32 chars", ErrValidation)
	}
	if !isAlphanumericUnderscore(req.Username) {
		return nil, fmt.Errorf("%w: username may only contain letters, digits, _", ErrValidation)
	}
	if err := crypto.ValidatePasswordStrength(req.Password); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWeakPassword, err)
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("auth: hashing failed: %w", err)
	}
	uid, err := crypto.GenerateToken(16)
	if err != nil {
		return nil, err
	}

	user := &storage.User{
		ID:            uid,
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  hash,
		Role:          storage.RoleUser,
		EmailVerified: false,
	}

	if err := s.store.CreateUser(user); err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("auth: store create: %w", err)
	}

	// Generate email verification token (24h TTL).
	vToken, _ := crypto.GenerateToken(24)
	s.mu.Lock()
	s.verifyTokens[vToken] = tokenEntry{username: req.Username, expiry: time.Now().Add(24 * time.Hour)}
	s.mu.Unlock()

	// Send verification email (console in dev mode).
	s.mailer.Send(req.Email, "Verify your email", //nolint:errcheck
		fmt.Sprintf("Hi %s,\n\nVerify your email:\nPOST /api/v1/auth/verify-email\nToken: %s\n\nExpires in 24 hours.",
			req.Username, vToken))

	s.writeAudit(ctx, "REGISTER", req.Username, "SUCCESS", "")
	logger.AuditEvent(ctx, "REGISTER", req.Username, "SUCCESS")
	return user, nil
}

// Login authenticates a user and issues a JWT token pair.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*session.TokenPair, error) {
	user, err := s.store.FindByUsername(req.Username)
	if err != nil {
		crypto.HashPassword(req.Password) //nolint:errcheck — timing equalisation
		s.writeAudit(ctx, "LOGIN_FAIL", req.Username, "FAILURE", "user_not_found")
		logger.AuditEvent(ctx, "LOGIN_FAIL", req.Username, "FAILURE", slog.String("reason", "user_not_found"))
		return nil, ErrInvalidCredentials
	}

	if user.IsLocked() {
		s.writeAudit(ctx, "LOGIN_FAIL", req.Username, "FAILURE", "account_locked")
		logger.AuditEvent(ctx, "LOGIN_FAIL", req.Username, "FAILURE", slog.String("reason", "account_locked"))
		return nil, fmt.Errorf("%w until %s", ErrAccountLocked, user.LockedUntil.Format(time.RFC3339))
	}

	ok, err := crypto.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !ok {
		user.FailedAttempts++
		user.LastFailedAt = time.Now().UTC()
		if user.FailedAttempts >= s.cfg.MaxLoginAttempts {
			exp := math.Pow(2, float64(user.FailedAttempts-s.cfg.MaxLoginAttempts))
			user.LockedUntil = time.Now().Add(time.Duration(float64(s.cfg.LockoutBaseDuration) * exp)).UTC()
		}
		_ = s.store.UpdateUser(user)
		detail := fmt.Sprintf("attempt=%d", user.FailedAttempts)
		s.writeAudit(ctx, "LOGIN_FAIL", req.Username, "FAILURE", detail)
		logger.AuditEvent(ctx, "LOGIN_FAIL", req.Username, "FAILURE", slog.Int("attempt", user.FailedAttempts))
		return nil, ErrInvalidCredentials
	}

	// Success: reset lockout.
	user.FailedAttempts = 0
	user.LockedUntil = time.Time{}
	_ = s.store.UpdateUser(user)

	pair, err := s.sessions.Issue(user.ID, user.Username, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("auth: token issue: %w", err)
	}

	// Warn (not block) if email not verified.
	if !user.EmailVerified {
		logger.FromContext(ctx).Warn("login with unverified email", "username", req.Username)
	}

	s.writeAudit(ctx, "LOGIN_OK", req.Username, "SUCCESS", "")
	logger.AuditEvent(ctx, "LOGIN_OK", req.Username, "SUCCESS")
	return pair, nil
}

// Logout revokes the access token and optional refresh token.
func (s *Service) Logout(ctx context.Context, claims *session.Claims, refreshToken string) {
	s.sessions.Revoke(claims)
	if refreshToken != "" {
		s.sessions.RevokeRefresh(refreshToken)
	}
	s.writeAudit(ctx, "LOGOUT", claims.Username, "SUCCESS", "")
	logger.AuditEvent(ctx, "LOGOUT", claims.Username, "SUCCESS")
}

// VerifyEmail validates a verification token and marks the user's email as verified.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	s.mu.Lock()
	entry, ok := s.verifyTokens[token]
	if ok {
		delete(s.verifyTokens, token) // single-use
	}
	s.mu.Unlock()

	if !ok || time.Now().After(entry.expiry) {
		return ErrInvalidToken
	}

	user, err := s.store.FindByUsername(entry.username)
	if err != nil {
		return ErrInvalidToken
	}
	user.EmailVerified = true
	if err := s.store.UpdateUser(user); err != nil {
		return fmt.Errorf("auth: update user: %w", err)
	}
	s.writeAudit(ctx, "EMAIL_VERIFIED", entry.username, "SUCCESS", "")
	logger.AuditEvent(ctx, "EMAIL_VERIFIED", entry.username, "SUCCESS")
	return nil
}

// ForgotPassword generates a password reset token and sends it via email.
// Always returns nil — never reveals whether an email exists (anti-enumeration).
func (s *Service) ForgotPassword(ctx context.Context, emailAddr string) error {
	user, err := s.store.FindByEmail(emailAddr)
	if err != nil {
		// Don't reveal user doesn't exist — just return success silently.
		return nil
	}

	token, err := crypto.GenerateToken(32)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.resetTokens[token] = tokenEntry{username: user.Username, expiry: time.Now().Add(15 * time.Minute)}
	s.mu.Unlock()

	s.mailer.Send(emailAddr, "Reset your password", //nolint:errcheck
		fmt.Sprintf("Hi %s,\n\nReset your password:\nPOST /api/v1/auth/reset-password\nBody: {\"token\":\"%s\",\"new_password\":\"...\"}\n\nExpires in 15 minutes.",
			user.Username, token))

	s.writeAudit(ctx, "PASSWORD_RESET_REQUESTED", user.Username, "SUCCESS", "")
	return nil
}

// ResetPassword validates a reset token and sets a new password.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	s.mu.Lock()
	entry, ok := s.resetTokens[token]
	if ok {
		delete(s.resetTokens, token) // single-use
	}
	s.mu.Unlock()

	if !ok || time.Now().After(entry.expiry) {
		return ErrInvalidToken
	}
	if err := crypto.ValidatePasswordStrength(newPassword); err != nil {
		return fmt.Errorf("%w: %v", ErrWeakPassword, err)
	}

	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user, err := s.store.FindByUsername(entry.username)
	if err != nil {
		return ErrInvalidToken
	}
	user.PasswordHash = hash
	user.FailedAttempts = 0
	user.LockedUntil = time.Time{}

	if err := s.store.UpdateUser(user); err != nil {
		return fmt.Errorf("auth: update user: %w", err)
	}
	s.writeAudit(ctx, "PASSWORD_RESET", entry.username, "SUCCESS", "")
	logger.AuditEvent(ctx, "PASSWORD_RESET", entry.username, "SUCCESS")
	return nil
}

// --- helpers ---

func (s *Service) writeAudit(ctx context.Context, event, username, status, detail string) {
	if s.audit == nil {
		return
	}
	rid, _ := ctx.Value(struct{ k string }{"request_id"}).(string)
	s.audit.Append(storage.AuditEntry{ //nolint:errcheck
		Event:     event,
		Username:  username,
		Status:    status,
		Detail:    detail,
		RequestID: rid,
	})
}

func (s *Service) cleanupTokens() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for t, e := range s.verifyTokens {
			if now.After(e.expiry) {
				delete(s.verifyTokens, t)
			}
		}
		for t, e := range s.resetTokens {
			if now.After(e.expiry) {
				delete(s.resetTokens, t)
			}
		}
		s.mu.Unlock()
	}
}

func isAlphanumericUnderscore(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}
