package jwt

import (
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
)

type Issuer struct {
	secret []byte
}

func NewIssuer(secret string) *Issuer {
	return &Issuer{secret: []byte(secret)}
}

type claims struct {
	Role  string `json:"role"`
	Email string `json:"email"`
	jwtv5.RegisteredClaims
}

func (i *Issuer) Issue(p domain.Principal, ttl time.Duration) (string, error) {
	now := time.Now()
	c := claims{
		Role:  string(p.Role),
		Email: p.Email,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Subject:   p.UserID,
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, c)
	return t.SignedString(i.secret)
}

func (i *Issuer) Parse(token string) (domain.Principal, error) {
	t, err := jwtv5.ParseWithClaims(token, &claims{}, func(t *jwtv5.Token) (any, error) {
		if t.Method != jwtv5.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return i.secret, nil
	})
	if err != nil || !t.Valid {
		return domain.Principal{}, domain.ErrUnauthorized
	}
	c, ok := t.Claims.(*claims)
	if !ok || c.Subject == "" {
		return domain.Principal{}, domain.ErrUnauthorized
	}
	return domain.Principal{UserID: c.Subject, Email: c.Email, Role: domain.Role(c.Role)}, nil
}

var _ ports.TokenIssuer = (*Issuer)(nil)
