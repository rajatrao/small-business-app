package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rajat/localdiscovery/internal/domain"
)

func newAuth(t *testing.T) (*AuthService, *fakeUsers) {
	t.Helper()
	users := newFakeUsers()
	svc := NewAuthService(users, newFakeOIDCIdent(), &fakeAffinity{}, fakeTokens{}, unconfiguredOIDC{}, fakeClock{t: time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)}, "http://localhost:5173")
	return svc, users
}

func TestSignupAndLogin(t *testing.T) {
	svc, _ := newAuth(t)
	ctx := context.Background()

	sess, err := svc.Signup(ctx, "Maya@Demo.local", "demo1234", "Maya Neighbor", domain.RoleCustomer, "guest-1")
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if sess.User.Email != "maya@demo.local" {
		t.Fatalf("email normalized: %s", sess.User.Email)
	}
	if sess.User.Role != domain.RoleCustomer {
		t.Fatalf("role: %s", sess.User.Role)
	}
	if sess.Token == "" {
		t.Fatal("expected token")
	}

	_, err = svc.Signup(ctx, "maya@demo.local", "demo1234", "Maya", domain.RoleCustomer, "")
	if !errors.Is(err, domain.ErrDuplicateEmail) {
		t.Fatalf("duplicate: %v", err)
	}

	_, err = svc.Login(ctx, "maya@demo.local", "wrong-password", "")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("bad password: %v", err)
	}

	got, err := svc.Login(ctx, "MAYA@demo.local", "demo1234", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if got.User.ID != sess.User.ID {
		t.Fatalf("id mismatch")
	}
}

func TestSignupOwnerRole(t *testing.T) {
	svc, _ := newAuth(t)
	sess, err := svc.Signup(context.Background(), "owner@demo.local", "demo1234", "Owen", domain.RoleOwner, "")
	if err != nil {
		t.Fatal(err)
	}
	if sess.User.Role != domain.RoleOwner {
		t.Fatalf("want OWNER got %s", sess.User.Role)
	}
}

func TestSignupRejectsShortPassword(t *testing.T) {
	svc, _ := newAuth(t)
	_, err := svc.Signup(context.Background(), "a@b.co", "short", "A", domain.RoleCustomer, "")
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestStartOIDCUnconfigured(t *testing.T) {
	svc, _ := newAuth(t)
	_, _, err := svc.StartOIDC(context.Background(), domain.RoleCustomer, "/", "http://localhost:8080/auth/oidc/callback")
	if !errors.Is(err, domain.ErrOIDCNotConfigured) {
		t.Fatalf("got %v", err)
	}
}

func TestBecomeOwner(t *testing.T) {
	svc, _ := newAuth(t)
	sess, err := svc.Signup(context.Background(), "c@demo.local", "demo1234", "Cee", domain.RoleCustomer, "")
	if err != nil {
		t.Fatal(err)
	}
	p := &domain.Principal{UserID: sess.User.ID, Role: domain.RoleCustomer}
	out, err := svc.BecomeOwner(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if out.User.Role != domain.RoleOwner {
		t.Fatalf("got %s", out.User.Role)
	}
	if out.Token == "" {
		t.Fatal("expected refreshed token")
	}
}

func TestMeUnauthorized(t *testing.T) {
	svc, _ := newAuth(t)
	_, err := svc.Me(context.Background(), nil)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}
