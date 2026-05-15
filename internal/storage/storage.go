// Package storage defines the UserStore interface and domain models.
// Concrete implementations (JSON file, SQLite, Postgres) satisfy this interface,
// letting the auth layer stay storage-agnostic — easy to swap in interviews.
package storage

import "time"

// Role controls access levels within the system.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User is the domain model persisted by a UserStore.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"` // Argon2id PHC string
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Lockout state (exponential backoff).
	FailedAttempts  int       `json:"failed_attempts"`
	LockedUntil     time.Time `json:"locked_until"`
	LastFailedAt    time.Time `json:"last_failed_at"`

	// Email verification state.
	EmailVerified bool `json:"email_verified"`
}

// IsLocked reports whether the account is currently locked out.
func (u *User) IsLocked() bool {
	return time.Now().Before(u.LockedUntil)
}


// UserStore is the storage abstraction for user data.
// Any backend (file, SQLite, Postgres) that satisfies this interface
// can be plugged in without touching the auth business logic.
type UserStore interface {
	// CreateUser persists a new user. Returns ErrUserExists if username taken.
	CreateUser(user *User) error

	// FindByUsername retrieves a user by username. Returns ErrNotFound if missing.
	FindByUsername(username string) (*User, error)

	// FindByEmail retrieves a user by email address.
	FindByEmail(email string) (*User, error)

	// FindByID retrieves a user by ID.
	FindByID(id string) (*User, error)

	// UpdateUser persists changes to an existing user (lockout state, etc.).
	UpdateUser(user *User) error

	// ListUsers returns all users (admin endpoint).
	ListUsers() ([]*User, error)

	// DeleteUser removes a user by ID.
	DeleteUser(id string) error
}
