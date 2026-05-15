// Package storage — audit log store (JSONL file).
// Each line is one JSON-encoded AuditEntry.
// The admin endpoint reads these back with pagination.
package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEntry records a security event (login, register, logout, etc.)
type AuditEntry struct {
	Timestamp time.Time `json:"ts"`
	Event     string    `json:"event"`
	Username  string    `json:"username"`
	Status    string    `json:"status"`
	IP        string    `json:"ip,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Detail    string    `json:"detail,omitempty"`
}

// AuditStore persists and retrieves audit log entries.
type AuditStore interface {
	Append(entry AuditEntry) error
	List(limit, offset int) ([]*AuditEntry, error)
	Count() (int, error)
}

// FileAuditStore is a JSONL-file backed AuditStore.
type FileAuditStore struct {
	mu   sync.Mutex
	path string
}

// NewFileAuditStore creates a store that appends to <dir>/audit.jsonl.
func NewFileAuditStore(dir string) (*FileAuditStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("audit: mkdir: %w", err)
	}
	return &FileAuditStore{path: filepath.Join(dir, "audit.jsonl")}, nil
}

func (a *FileAuditStore) Append(entry AuditEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	f, err := os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("audit: open: %w", err)
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, string(data))
	return err
}

func (a *FileAuditStore) List(limit, offset int) ([]*AuditEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f, err := os.Open(a.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*AuditEntry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []*AuditEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e AuditEntry
		if json.Unmarshal(scanner.Bytes(), &e) == nil {
			all = append(all, &e)
		}
	}
	if offset >= len(all) {
		return []*AuditEntry{}, nil
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	return all, scanner.Err()
}

func (a *FileAuditStore) Count() (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f, err := os.Open(a.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()
	n := 0
	s := bufio.NewScanner(f)
	for s.Scan() {
		n++
	}
	return n, s.Err()
}
