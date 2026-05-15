// Package storage — JSON file implementation of UserStore.
//
// Concurrency safety: a sync.RWMutex guards all reads/writes so concurrent
// HTTP requests cannot corrupt the data file — a race condition present in
// the original C project.
//
// Atomicity: writes go to a temp file then os.Rename (atomic on POSIX and
// Windows NTFS), preventing partial-write corruption on crash.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Sentinel errors — callers check with errors.Is.
var (
	ErrNotFound  = errors.New("storage: user not found")
	ErrUserExists = errors.New("storage: username already taken")
)

// JSONStore is a thread-safe, file-backed UserStore.
// Good for demos / single-node deployments; swap for SQLite/Postgres at scale.
type JSONStore struct {
	mu      sync.RWMutex
	path    string
	users   map[string]*User // keyed by username
}

// db is the on-disk schema.
type db struct {
	Users map[string]*User `json:"users"`
}

// NewJSONStore loads (or creates) the JSON database at the given path.
func NewJSONStore(dir string) (*JSONStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("storage: mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, "users.json")
	s := &JSONStore{path: path, users: make(map[string]*User)}
	if err := s.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *JSONStore) CreateUser(u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[u.Username]; ok {
		return ErrUserExists
	}
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt
	s.users[u.Username] = u
	return s.flush()
}

func (s *JSONStore) FindByUsername(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[username]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *JSONStore) FindByEmail(email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (s *JSONStore) FindByID(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (s *JSONStore) UpdateUser(u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[u.Username]; !ok {
		return ErrNotFound
	}
	u.UpdatedAt = time.Now().UTC()
	s.users[u.Username] = u
	return s.flush()
}

func (s *JSONStore) ListUsers() ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		cp := *u
		out = append(out, &cp)
	}
	return out, nil
}

func (s *JSONStore) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for uname, u := range s.users {
		if u.ID == id {
			delete(s.users, uname)
			return s.flush()
		}
	}
	return ErrNotFound
}

// --- persistence helpers ---

func (s *JSONStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var d db
	if err := json.Unmarshal(data, &d); err != nil {
		return fmt.Errorf("storage: corrupt DB: %w", err)
	}
	if d.Users != nil {
		s.users = d.Users
	}
	return nil
}

// flush writes atomically: temp file → os.Rename.
// Prevents partial-write corruption on crash.
func (s *JSONStore) flush() error {
	d := db{Users: s.users}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: marshal: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("storage: write temp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("storage: atomic rename: %w", err)
	}
	return nil
}
