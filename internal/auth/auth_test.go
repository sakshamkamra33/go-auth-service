package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sakshamkamra33/secure-auth/internal/auth"
	"github.com/sakshamkamra33/secure-auth/internal/config"
	"github.com/sakshamkamra33/secure-auth/internal/session"
	"github.com/sakshamkamra33/secure-auth/internal/storage"
)

// --- In-memory fake store for tests (no file I/O) ---

type fakeStore struct {
	users map[string]*storage.User
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: make(map[string]*storage.User)}
}

func (f *fakeStore) CreateUser(u *storage.User) error {
	if _, ok := f.users[u.Username]; ok {
		return storage.ErrUserExists
	}
	f.users[u.Username] = u
	return nil
}
func (f *fakeStore) FindByUsername(name string) (*storage.User, error) {
	u, ok := f.users[name]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *u
	return &cp, nil
}
func (f *fakeStore) FindByEmail(email string) (*storage.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, storage.ErrNotFound
}
func (f *fakeStore) FindByID(id string) (*storage.User, error)  { return nil, nil }
func (f *fakeStore) UpdateUser(u *storage.User) error           { f.users[u.Username] = u; return nil }
func (f *fakeStore) ListUsers() ([]*storage.User, error)        { return nil, nil }
func (f *fakeStore) DeleteUser(id string) error                 { return nil }

// --- Test helpers ---

func newService() *auth.Service {
	cfg := &config.Config{
		MaxLoginAttempts:    3,
		LockoutBaseDuration: 30 * time.Second,
		JWTSecret:           "test-secret-32-bytes-long-pad!!",
		AccessTokenExpiry:   15 * time.Minute,
		RefreshTokenExpiry:  7 * 24 * time.Hour,
	}
	mgr := session.NewManager(cfg.JWTSecret, cfg.AccessTokenExpiry, cfg.RefreshTokenExpiry)
	return auth.NewService(newFakeStore(), nil, mgr, nil, cfg)
}

const validPassword = "StrongPass@123!"

// TestRegisterAndLogin — happy path.
func TestRegisterAndLogin(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	_, err := svc.Register(ctx, auth.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: validPassword,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	pair, err := svc.Login(ctx, auth.LoginRequest{
		Username: "alice",
		Password: validPassword,
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("access token should not be empty")
	}
}

// TestDuplicateRegister — second registration with same username must fail.
func TestDuplicateRegister(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	req := auth.RegisterRequest{Username: "bob", Email: "b@b.com", Password: validPassword}

	svc.Register(ctx, req) //nolint:errcheck
	_, err := svc.Register(ctx, req)
	if !errors.Is(err, auth.ErrUserExists) {
		t.Fatalf("want ErrUserExists, got %v", err)
	}
}

// TestWrongPassword — must return ErrInvalidCredentials (not expose "user not found").
func TestWrongPassword(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	svc.Register(ctx, auth.RegisterRequest{Username: "carol", Email: "c@c.com", Password: validPassword}) //nolint:errcheck

	_, err := svc.Login(ctx, auth.LoginRequest{Username: "carol", Password: "WrongPass@999!"})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

// TestAccountLockout — after MaxLoginAttempts failures account locks.
func TestAccountLockout(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	svc.Register(ctx, auth.RegisterRequest{Username: "dave", Email: "d@d.com", Password: validPassword}) //nolint:errcheck

	for i := 0; i < 3; i++ {
		svc.Login(ctx, auth.LoginRequest{Username: "dave", Password: "Wrong@Pass1!"}) //nolint:errcheck
	}

	_, err := svc.Login(ctx, auth.LoginRequest{Username: "dave", Password: validPassword})
	if !errors.Is(err, auth.ErrAccountLocked) {
		t.Fatalf("want ErrAccountLocked after %d failures, got %v", 3, err)
	}
}

// TestWeakPasswordRejection — weak passwords must be rejected at registration.
func TestWeakPasswordRejection(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	_, err := svc.Register(ctx, auth.RegisterRequest{Username: "eve", Email: "e@e.com", Password: "weak"})
	if !errors.Is(err, auth.ErrWeakPassword) {
		t.Fatalf("want ErrWeakPassword, got %v", err)
	}
}
