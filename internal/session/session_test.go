package session_test

import (
	"testing"
	"time"

	"github.com/sakshamkamra33/secure-auth/internal/session"
)

func newManager() *session.Manager {
	return session.NewManager("test-secret-32-bytes-long-pad!!", 15*time.Minute, 7*24*time.Hour)
}

func TestIssueAndValidate(t *testing.T) {
	mgr := newManager()
	pair, err := mgr.Issue("uid1", "alice", "user")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	claims, err := mgr.Validate(pair.AccessToken)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if claims.Username != "alice" {
		t.Errorf("want username=alice, got %s", claims.Username)
	}
}

func TestRevoke(t *testing.T) {
	mgr := newManager()
	pair, _ := mgr.Issue("uid1", "alice", "user")
	claims, _ := mgr.Validate(pair.AccessToken)

	mgr.Revoke(claims)

	_, err := mgr.Validate(pair.AccessToken)
	if err == nil {
		t.Fatal("revoked token should not validate")
	}
}

func TestRefreshRotation(t *testing.T) {
	mgr := newManager()
	pair1, _ := mgr.Issue("uid1", "alice", "user")

	pair2, err := mgr.Refresh(pair1.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if pair2.AccessToken == pair1.AccessToken {
		t.Fatal("refreshed access token should differ")
	}

	// Old refresh token must be invalidated (single-use).
	_, err = mgr.Refresh(pair1.RefreshToken)
	if err == nil {
		t.Fatal("old refresh token should be invalidated after rotation")
	}
}

func TestInvalidToken(t *testing.T) {
	mgr := newManager()
	_, err := mgr.Validate("not.a.valid.jwt")
	if err == nil {
		t.Fatal("invalid token should fail validation")
	}
}
