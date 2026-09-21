package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
)

type UserRepo struct{ pool *pgxpool.Pool }

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

func (r *UserRepo) Create(ctx context.Context, u domain.User) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name, role, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		u.ID, u.Email, u.PasswordHash, u.Name, string(u.Role), u.CreatedAt)
	if err != nil {
		if isUnique(err) {
			return domain.ErrDuplicateEmail
		}
		return err
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (domain.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx, userCols+` WHERE id=$1`, id))
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx, userCols+` WHERE email=$1`, email))
}

func (r *UserRepo) UpdateRole(ctx context.Context, id string, role domain.Role) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET role=$2 WHERE id=$1`, id, string(role))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const userCols = `SELECT id, email, password_hash, name, role, created_at FROM users`

func (r *UserRepo) scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	var role string
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	u.Role = domain.Role(role)
	return u, nil
}

type OIDCIdentRepo struct{ pool *pgxpool.Pool }

func NewOIDCIdentRepo(pool *pgxpool.Pool) *OIDCIdentRepo { return &OIDCIdentRepo{pool: pool} }

func (r *OIDCIdentRepo) GetByIssuerSubject(ctx context.Context, issuer, subject string) (domain.OIDCIdentity, error) {
	var i domain.OIDCIdentity
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, issuer, subject, created_at
		FROM oidc_identities WHERE issuer=$1 AND subject=$2`, issuer, subject,
	).Scan(&i.ID, &i.UserID, &i.Issuer, &i.Subject, &i.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OIDCIdentity{}, domain.ErrNotFound
	}
	return i, err
}

func (r *OIDCIdentRepo) Create(ctx context.Context, ident domain.OIDCIdentity) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO oidc_identities (id, user_id, issuer, subject, created_at)
		VALUES ($1,$2,$3,$4,$5)`,
		ident.ID, ident.UserID, ident.Issuer, ident.Subject, ident.CreatedAt)
	return err
}

func isUnique(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "unique"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}

var _ ports.UserRepository = (*UserRepo)(nil)
var _ ports.OIDCIdentityRepository = (*OIDCIdentRepo)(nil)

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func fmtID(v any) string { return fmt.Sprint(v) }
