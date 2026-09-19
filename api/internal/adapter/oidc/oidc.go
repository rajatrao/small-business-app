package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
	"golang.org/x/oauth2"
)

type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type Client struct {
	cfg      Config
	mu       sync.Mutex
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
}

func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Configured() bool {
	return c.cfg.Issuer != "" && c.cfg.ClientID != "" && c.cfg.RedirectURI != ""
}

func (c *Client) ensure(ctx context.Context) error {
	if !c.Configured() {
		return domain.ErrOIDCNotConfigured
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.provider != nil {
		return nil
	}
	p, err := oidc.NewProvider(ctx, c.cfg.Issuer)
	if err != nil {
		return fmt.Errorf("oidc discover: %w", err)
	}
	c.provider = p
	c.verifier = p.Verifier(&oidc.Config{ClientID: c.cfg.ClientID})
	return nil
}

func (c *Client) oauth(redirectURI string) oauth2.Config {
	if redirectURI == "" {
		redirectURI = c.cfg.RedirectURI
	}
	return oauth2.Config{
		ClientID:     c.cfg.ClientID,
		ClientSecret: c.cfg.ClientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     c.provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
}

func (c *Client) Start(ctx context.Context, p ports.OIDCStartParams) (ports.OIDCStartResult, error) {
	if err := c.ensure(ctx); err != nil {
		return ports.OIDCStartResult{}, err
	}
	state, err := randomB64(24)
	if err != nil {
		return ports.OIDCStartResult{}, err
	}
	nonce, err := randomB64(24)
	if err != nil {
		return ports.OIDCStartResult{}, err
	}
	verifier, err := randomB64(32)
	if err != nil {
		return ports.OIDCStartResult{}, err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	oc := c.oauth(p.RedirectURI)
	authURL := oc.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oidc.Nonce(nonce),
	)
	return ports.OIDCStartResult{
		AuthURL:      authURL,
		State:        state,
		Nonce:        nonce,
		CodeVerifier: verifier,
	}, nil
}

func (c *Client) Exchange(ctx context.Context, code, codeVerifier, redirectURI, nonce string) (ports.OIDCClaims, error) {
	if err := c.ensure(ctx); err != nil {
		return ports.OIDCClaims{}, err
	}
	oc := c.oauth(redirectURI)
	tok, err := oc.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return ports.OIDCClaims{}, fmt.Errorf("oidc exchange: %w", err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok || raw == "" {
		return ports.OIDCClaims{}, fmt.Errorf("%w: missing id_token", domain.ErrInvalid)
	}
	idt, err := c.verifier.Verify(ctx, raw)
	if err != nil {
		return ports.OIDCClaims{}, fmt.Errorf("id_token: %w", err)
	}
	if nonce != "" && idt.Nonce != nonce {
		return ports.OIDCClaims{}, fmt.Errorf("%w: nonce", domain.ErrInvalid)
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idt.Claims(&claims); err != nil {
		return ports.OIDCClaims{}, err
	}
	if claims.Email == "" {
		// userinfo fallback
		uinfo := c.provider.UserInfoEndpoint()
		if uinfo != "" {
			ui, err := c.provider.UserInfo(ctx, oauth2.StaticTokenSource(tok))
			if err == nil {
				_ = ui.Claims(&claims)
				if claims.Email == "" {
					claims.Email = ui.Email
				}
			}
		}
	}
	return ports.OIDCClaims{
		Issuer:        idt.Issuer,
		Subject:       idt.Subject,
		Email:         strings.TrimSpace(claims.Email),
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		Nonce:         idt.Nonce,
	}, nil
}

func randomB64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

var _ ports.OIDCClient = (*Client)(nil)
