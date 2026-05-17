// Package api — HTTP handlers (v2): adds forgot/reset password, email verify,
// audit log endpoint, and admin delete user.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/sakshamkamra33/go-auth-service/internal/auth"
	"github.com/sakshamkamra33/go-auth-service/internal/config"
	"github.com/sakshamkamra33/go-auth-service/internal/middleware"
	"github.com/sakshamkamra33/go-auth-service/internal/session"
	"github.com/sakshamkamra33/go-auth-service/internal/storage"
)

// Handler holds all HTTP handler dependencies.
type Handler struct {
	authSvc  *auth.Service
	sessions *session.Manager
	store    storage.UserStore
	audit    storage.AuditStore
	cfg      *config.Config
}

// NewHandler constructs the handler with injected dependencies.
func NewHandler(svc *auth.Service, sm *session.Manager, store storage.UserStore, audit storage.AuditStore, cfg *config.Config) *Handler {
	return &Handler{authSvc: svc, sessions: sm, store: store, audit: audit, cfg: cfg}
}

// NewRouter builds and returns the fully wired chi router.
func NewRouter(h *Handler, rl *middleware.RateLimiter) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.CORS)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)
	r.Use(chimiddleware.Recoverer)
	r.Use(rl.Handler)

	// Health check endpoint
	r.Get("/health", h.Health)

	// Root path welcome message (fixes the 404 issue)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Welcome to the Go Auth Service API! 🚀", "version": "1.0"}`))
	})

	r.Route("/api/v1/auth", func(r chi.Router) {
		// Public
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.RefreshToken)
		r.Post("/forgot-password", h.ForgotPassword)
		r.Post("/reset-password", h.ResetPassword)
		r.Post("/verify-email", h.VerifyEmail)

		// Protected (JWT required)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(h.sessions))
			r.Get("/me", h.Me)
			r.Post("/logout", h.Logout)
		})
	})

	// Admin: JWT + admin role
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(middleware.Authenticate(h.sessions))
		r.Use(middleware.RequireRole("admin"))
		r.Get("/users", h.ListUsers)
		r.Delete("/users/{userID}", h.DeleteUser)
		r.Get("/audit", h.AuditLogs)
	})

	return r
}

// --- Handlers ---

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"status": "ok", "service": "secure-auth"})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
	if !decode(w, r, &req) {
		return
	}
	user, err := h.authSvc.Register(r.Context(), req)
	if err != nil {
		respondErr(w, err)
		return
	}
	respond(w, http.StatusCreated, map[string]any{
		"message":        "user registered — check your email to verify your account",
		"id":             user.ID,
		"username":       user.Username,
		"email_verified": user.EmailVerified,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if !decode(w, r, &req) {
		return
	}
	pair, err := h.authSvc.Login(r.Context(), req)
	if err != nil {
		respondErr(w, err)
		return
	}
	respond(w, http.StatusOK, pair)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	h.authSvc.Logout(r.Context(), claims, body.RefreshToken)
	respond(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decode(w, r, &body) {
		return
	}
	pair, err := h.sessions.Refresh(body.RefreshToken)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired refresh token"}`, http.StatusUnauthorized)
		return
	}
	respond(w, http.StatusOK, pair)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	user, err := h.store.FindByID(claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"role":           user.Role,
		"email_verified": user.EmailVerified,
		"created_at":     user.CreatedAt,
	})
}

// --- New: Email Verification ---

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := h.authSvc.VerifyEmail(r.Context(), body.Token); err != nil {
		http.Error(w, `{"error":"invalid or expired verification token"}`, http.StatusBadRequest)
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "email verified successfully"})
}

// --- New: Password Reset ---

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &body) {
		return
	}
	// Always return 200 — never reveal if email exists.
	h.authSvc.ForgotPassword(r.Context(), body.Email) //nolint:errcheck
	respond(w, http.StatusOK, map[string]string{
		"message": "if that email is registered, a reset link has been sent",
	})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := h.authSvc.ResetPassword(r.Context(), body.Token, body.NewPassword); err != nil {
		respondErr(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "password reset successfully"})
}

// --- Admin: Users ---

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListUsers()
	if err != nil {
		http.Error(w, `{"error":"could not list users"}`, http.StatusInternalServerError)
		return
	}
	type safeUser struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		Email         string `json:"email"`
		Role          string `json:"role"`
		EmailVerified bool   `json:"email_verified"`
		Locked        bool   `json:"locked"`
	}
	safe := make([]safeUser, 0, len(users))
	for _, u := range users {
		safe = append(safe, safeUser{
			ID: u.ID, Username: u.Username, Email: u.Email,
			Role: string(u.Role), EmailVerified: u.EmailVerified, Locked: u.IsLocked(),
		})
	}
	respond(w, http.StatusOK, map[string]any{"users": safe, "total": len(safe)})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		http.Error(w, `{"error":"missing userID"}`, http.StatusBadRequest)
		return
	}
	if err := h.store.DeleteUser(userID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"delete failed"}`, http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

// --- Admin: Audit Logs ---

func (h *Handler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit == 0 {
		limit = 50
	}

	entries, err := h.audit.List(limit, offset)
	if err != nil {
		http.Error(w, `{"error":"could not read audit log"}`, http.StatusInternalServerError)
		return
	}
	total, _ := h.audit.Count()
	respond(w, http.StatusOK, map[string]any{
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// --- helpers ---

func respond(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload) //nolint:errcheck
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return false
	}
	return true
}

func respondErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		http.Error(w, `{"error":"invalid username or password"}`, http.StatusUnauthorized)
	case errors.Is(err, auth.ErrAccountLocked):
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusTooManyRequests)
	case errors.Is(err, auth.ErrUserExists):
		http.Error(w, `{"error":"username already taken"}`, http.StatusConflict)
	case errors.Is(err, auth.ErrWeakPassword):
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
	case errors.Is(err, auth.ErrValidation):
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
	case errors.Is(err, auth.ErrInvalidToken):
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusBadRequest)
	default:
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}
}
