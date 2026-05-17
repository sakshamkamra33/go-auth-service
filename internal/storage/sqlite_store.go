package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore is a SQL-backed UserStore.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore connects to the database and runs auto-migrations.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite: failed to ping db: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("sqlite: migration failed: %w", err)
	}

	return store, nil
}

// migrate creates the users table if it doesn't exist.
func (s *SQLiteStore) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		failed_attempts INTEGER DEFAULT 0,
		locked_until DATETIME,
		last_failed_at DATETIME,
		email_verified BOOLEAN DEFAULT 0
	);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteStore) CreateUser(u *User) error {
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt

	query := `
	INSERT INTO users (
		id, username, email, password_hash, role, created_at, updated_at, 
		failed_attempts, locked_until, last_failed_at, email_verified
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		u.ID, u.Username, u.Email, u.PasswordHash, u.Role, u.CreatedAt, u.UpdatedAt,
		u.FailedAttempts, u.LockedUntil, u.LastFailedAt, u.EmailVerified,
	)

	if err != nil {
		// Catch SQLite unique constraint errors
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrUserExists
		}
		return fmt.Errorf("sqlite create user: %w", err)
	}
	return nil
}

// mapRowToUser is a helper to scan SQL rows into our Go struct
func (s *SQLiteStore) mapRowToUser(row *sql.Row) (*User, error) {
	var u User
	var lockedUntil, lastFailedAt sql.NullTime

	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
		&u.FailedAttempts, &lockedUntil, &lastFailedAt, &u.EmailVerified,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if lockedUntil.Valid {
		u.LockedUntil = lockedUntil.Time
	}
	if lastFailedAt.Valid {
		u.LastFailedAt = lastFailedAt.Time
	}

	return &u, nil
}

func (s *SQLiteStore) FindByUsername(username string) (*User, error) {
	row := s.db.QueryRow("SELECT * FROM users WHERE username = ?", username)
	return s.mapRowToUser(row)
}

func (s *SQLiteStore) FindByEmail(email string) (*User, error) {
	row := s.db.QueryRow("SELECT * FROM users WHERE email = ?", email)
	return s.mapRowToUser(row)
}

func (s *SQLiteStore) FindByID(id string) (*User, error) {
	row := s.db.QueryRow("SELECT * FROM users WHERE id = ?", id)
	return s.mapRowToUser(row)
}

func (s *SQLiteStore) UpdateUser(u *User) error {
	u.UpdatedAt = time.Now().UTC()
	query := `
	UPDATE users SET 
		password_hash = ?, role = ?, updated_at = ?, failed_attempts = ?, 
		locked_until = ?, last_failed_at = ?, email_verified = ?
	WHERE username = ?
	`
	res, err := s.db.Exec(query,
		u.PasswordHash, u.Role, u.UpdatedAt, u.FailedAttempts,
		u.LockedUntil, u.LastFailedAt, u.EmailVerified, u.Username,
	)

	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) ListUsers() ([]*User, error) {
	rows, err := s.db.Query("SELECT * FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		var lockedUntil, lastFailedAt sql.NullTime

		err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
			&u.FailedAttempts, &lockedUntil, &lastFailedAt, &u.EmailVerified,
		)
		if err != nil {
			return nil, err
		}
		if lockedUntil.Valid {
			u.LockedUntil = lockedUntil.Time
		}
		if lastFailedAt.Valid {
			u.LastFailedAt = lastFailedAt.Time
		}
		users = append(users, &u)
	}
	return users, nil
}

func (s *SQLiteStore) DeleteUser(id string) error {
	res, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
