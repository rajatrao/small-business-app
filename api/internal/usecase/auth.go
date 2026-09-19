package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTTL    = 7 * 24 * time.Hour
	oidcStateTTL  = 10 * time.Minute
	bcryptCost    = bcrypt.DefaultCost
	demoMinSecret = 16
)

type AuthService struct {
	users    ports.UserRepository
	oids     ports.OIDCIdentityRepository
	affinity ports.AffinityRepository
	tokens   ports.TokenIssuer
	oidc     ports.OIDCClient
	clock    ports.Clock
	webOrigin string
}

func NewAuthService(
	users ports.UserRepository,
	oids ports.OIDCIdentityRepository,
	affinity ports.AffinityRepository,
	tokens ports.TokenIssuer,
	oidc ports.OIDCClient,
	clock ports.Clock,
	webOrigin string,
) *AuthService {
	return &AuthService{
		users:     users,
		oids:      oids,
		affinity:  affinity,
		tokens:    tokens,
		oidc:      oidc,
		clock:     clock,
		webOrigin: webOrigin,
	}
}

type Session struct {
	User  domain.User
	Token string
}

func (s *AuthService) Signup(ctx context.Context, email, password, name string, role domain.Role, guestID string) (Session, error) {
	email = domain.NormalizeEmail(email)
	name = strings.TrimSpace(name)
	if email == "" || !strings.Contains(email, "@") {
		return Session{}, fmt.Errorf("%w: email", domain.ErrInvalid)
	}
	if len(password) < 8 {
		return Session{}, fmt.Errorf("%w: password must be at least 8 characters", domain.ErrInvalid)
	}
	if name == "" {
		return Session{}, fmt.Errorf("%w: name", domain.ErrInvalid)
	}
	if role == "" {
		role = domain.RoleCustomer
	}
	if role != domain.RoleCustomer && role != domain.RoleOwner {
		role = domain.RoleCustomer
	}

	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return Session{}, domain.ErrDuplicateEmail
	} else if !errors.Is(err, domain.ErrNotFound) {
		return Session{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return Session{}, err
	}
	hs := string(hash)
	u := domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: &hs,
		Name:         name,
		Role:         role,
		CreatedAt:    s.clock.Now(),
	}
	if err := s.users.Create(ctx, u); err != nil {
		return Session{}, err
	}
	if guestID != "" {
		_ = s.affinity.MergeGuestIntoUser(ctx, guestID, u.ID)
	}
	return s.issue(u)
}

func (s *AuthService) Login(ctx context.Context, email, password, guestID string) (Session, error) {
	email = domain.NormalizeEmail(email)
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Session{}, domain.ErrInvalidCredentials
		}
		return Session{}, err
	}
	if u.PasswordHash == nil || *u.PasswordHash == "" {
		return Session{}, domain.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		return Session{}, domain.ErrInvalidCredentials
	}
	if guestID != "" {
		_ = s.affinity.MergeGuestIntoUser(ctx, guestID, u.ID)
	}
	return s.issue(u)
}

func (s *AuthService) Me(ctx context.Context, principal *domain.Principal) (domain.User, error) {
	if principal == nil || principal.UserID == "" {
		return domain.User{}, domain.ErrUnauthorized
	}
	return s.users.GetByID(ctx, principal.UserID)
}

func (s *AuthService) BecomeOwner(ctx context.Context, principal *domain.Principal) (Session, error) {
	if principal == nil {
		return Session{}, domain.ErrUnauthorized
	}
	u, err := s.users.GetByID(ctx, principal.UserID)
	if err != nil {
		return Session{}, err
	}
	if u.Role == domain.RoleCustomer {
		if err := s.users.UpdateRole(ctx, u.ID, domain.RoleOwner); err != nil {
			return Session{}, err
		}
		u.Role = domain.RoleOwner
	}
	return s.issue(u)
}

func (s *AuthService) ParseToken(token string) (domain.Principal, error) {
	return s.tokens.Parse(token)
}

func (s *AuthService) issue(u domain.User) (Session, error) {
	tok, err := s.tokens.Issue(domain.Principal{UserID: u.ID, Email: u.Email, Role: u.Role}, sessionTTL)
	if err != nil {
		return Session{}, err
	}
	return Session{User: u, Token: tok}, nil
}

type OIDCStatePayload struct {
	State        string      `json:"state"`
	Nonce        string      `json:"nonce"`
	CodeVerifier string      `json:"code_verifier"`
	Role         domain.Role `json:"role"`
	Next         string      `json:"next"`
	Exp          int64       `json:"exp"`
}

func (s *AuthService) StartOIDC(ctx context.Context, role domain.Role, next, redirectURI string) (authURL string, stateCookie string, err error) {
	if s.oidc == nil || !s.oidc.Configured() {
		return "", "", domain.ErrOIDCNotConfigured
	}
	if role != domain.RoleOwner {
		role = domain.RoleCustomer
	}
	start, err := s.oidc.Start(ctx, ports.OIDCStartParams{
		Role:        role,
		Next:        next,
		RedirectURI: redirectURI,
	})
	if err != nil {
		return "", "", err
	}
	payload := OIDCStatePayload{
		State:        start.State,
		Nonce:        start.Nonce,
		CodeVerifier: start.CodeVerifier,
		Role:         role,
		Next:         sanitizeNext(next, s.webOrigin),
		Exp:          s.clock.Now().Add(oidcStateTTL).Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	return start.AuthURL, base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *AuthService) HandleOIDCCallback(ctx context.Context, code, state, stateCookie, redirectURI, guestID string) (Session, string, error) {
	if s.oidc == nil || !s.oidc.Configured() {
		return Session{}, "", domain.ErrOIDCNotConfigured
	}
	raw, err := base64.RawURLEncoding.DecodeString(stateCookie)
	if err != nil {
		return Session{}, "", fmt.Errorf("%w: oidc state", domain.ErrInvalid)
	}
	var payload OIDCStatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Session{}, "", fmt.Errorf("%w: oidc state", domain.ErrInvalid)
	}
	if payload.State != state || payload.Exp < s.clock.Now().Unix() {
		return Session{}, "", fmt.Errorf("%w: oidc state mismatch", domain.ErrInvalid)
	}
	claims, err := s.oidc.Exchange(ctx, code, payload.CodeVerifier, redirectURI, payload.Nonce)
	if err != nil {
		return Session{}, "", err
	}
	if claims.Email == "" {
		return Session{}, "", fmt.Errorf("%w: oidc email missing", domain.ErrInvalid)
	}
	u, err := s.findOrCreateOIDCUser(ctx, claims, payload.Role)
	if err != nil {
		return Session{}, "", err
	}
	if guestID != "" {
		_ = s.affinity.MergeGuestIntoUser(ctx, guestID, u.ID)
	}
	sess, err := s.issue(u)
	if err != nil {
		return Session{}, "", err
	}
	next := payload.Next
	if next == "" {
		if u.Role == domain.RoleOwner {
			next = "/dashboard"
		} else {
			next = "/"
		}
	}
	return sess, next, nil
}

func (s *AuthService) findOrCreateOIDCUser(ctx context.Context, claims ports.OIDCClaims, role domain.Role) (domain.User, error) {
	ident, err := s.oids.GetByIssuerSubject(ctx, claims.Issuer, claims.Subject)
	if err == nil {
		return s.users.GetByID(ctx, ident.UserID)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, err
	}

	email := domain.NormalizeEmail(claims.Email)
	existing, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		if err := s.oids.Create(ctx, domain.OIDCIdentity{
			ID:        uuid.NewString(),
			UserID:    existing.ID,
			Issuer:    claims.Issuer,
			Subject:   claims.Subject,
			CreatedAt: s.clock.Now(),
		}); err != nil {
			return domain.User{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, err
	}

	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	if role != domain.RoleOwner {
		role = domain.RoleCustomer
	}
	u := domain.User{
		ID:        uuid.NewString(),
		Email:     email,
		Name:      name,
		Role:      role,
		CreatedAt: s.clock.Now(),
	}
	if err := s.users.Create(ctx, u); err != nil {
		return domain.User{}, err
	}
	if err := s.oids.Create(ctx, domain.OIDCIdentity{
		ID:        uuid.NewString(),
		UserID:    u.ID,
		Issuer:    claims.Issuer,
		Subject:   claims.Subject,
		CreatedAt: s.clock.Now(),
	}); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

func sanitizeNext(next, webOrigin string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return ""
	}
	if strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") {
		return next
	}
	if webOrigin != "" {
		if u, err := url.Parse(next); err == nil {
			if ou, err2 := url.Parse(webOrigin); err2 == nil && u.Host == ou.Host {
				if u.Path == "" {
					return "/"
				}
				return u.Path
			}
		}
	}
	return ""
}

func RandomGuestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}
