package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
)

type AffinityRepo struct{ pool *pgxpool.Pool }

func NewAffinityRepo(pool *pgxpool.Pool) *AffinityRepo { return &AffinityRepo{pool: pool} }

func (r *AffinityRepo) Bump(ctx context.Context, userID, guestID, category string, delta float64) error {
	if userID != "" {
		return r.ensureUserBump(ctx, userID, category, delta)
	}
	if guestID == "" {
		return nil
	}
	return r.ensureGuestBump(ctx, guestID, category, delta)
}

func (r *AffinityRepo) ensureUserBump(ctx context.Context, userID, category string, delta float64) error {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT id FROM category_affinities WHERE user_id=$1::uuid AND category=$2`, userID, category).Scan(&id)
	if err == nil {
		_, err = r.pool.Exec(ctx, `UPDATE category_affinities SET score = score + $2, updated_at = now() WHERE id=$1`, id, delta)
		return err
	}
	if err != pgx.ErrNoRows && !isNoRows(err) {
		// try insert
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO category_affinities (id, user_id, guest_id, category, score, updated_at)
		VALUES (gen_random_uuid(), $1::uuid, NULL, $2, $3, now())`, userID, category, delta)
	if isUnique(err) {
		_, err = r.pool.Exec(ctx, `
			UPDATE category_affinities SET score = score + $3, updated_at = now()
			WHERE user_id=$1::uuid AND category=$2`, userID, category, delta)
	}
	return err
}

func (r *AffinityRepo) ensureGuestBump(ctx context.Context, guestID, category string, delta float64) error {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT id FROM category_affinities WHERE guest_id=$1 AND category=$2`, guestID, category).Scan(&id)
	if err == nil {
		_, err = r.pool.Exec(ctx, `UPDATE category_affinities SET score = score + $2, updated_at = now() WHERE id=$1`, id, delta)
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO category_affinities (id, user_id, guest_id, category, score, updated_at)
		VALUES (gen_random_uuid(), NULL, $1, $2, $3, now())`, guestID, category, delta)
	if isUnique(err) {
		_, err = r.pool.Exec(ctx, `
			UPDATE category_affinities SET score = score + $3, updated_at = now()
			WHERE guest_id=$1 AND category=$2`, guestID, category, delta)
	}
	return err
}

func isNoRows(err error) bool {
	return err != nil && (err == pgx.ErrNoRows || strings.Contains(err.Error(), "no rows"))
}

func (r *AffinityRepo) List(ctx context.Context, userID, guestID string) ([]domain.CategoryAffinity, error) {
	if userID == "" && guestID == "" {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, guest_id, category, score
		FROM category_affinities
		WHERE ($1 <> '' AND user_id::text = $1) OR ($2 <> '' AND guest_id = $2)`, userID, guestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CategoryAffinity
	for rows.Next() {
		var a domain.CategoryAffinity
		if err := rows.Scan(&a.ID, &a.UserID, &a.GuestID, &a.Category, &a.Score); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AffinityRepo) MergeGuestIntoUser(ctx context.Context, guestID, userID string) error {
	if guestID == "" || userID == "" {
		return nil
	}
	rows, err := r.pool.Query(ctx, `SELECT category, score FROM category_affinities WHERE guest_id=$1`, guestID)
	if err != nil {
		return err
	}
	type row struct {
		cat   string
		score float64
	}
	var pending []row
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.cat, &x.score); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, x)
	}
	rows.Close()
	for _, x := range pending {
		if err := r.ensureUserBump(ctx, userID, x.cat, x.score); err != nil {
			return err
		}
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM category_affinities WHERE guest_id=$1`, guestID)
	return err
}

type ChatRepo struct{ pool *pgxpool.Pool }

func NewChatRepo(pool *pgxpool.Pool) *ChatRepo { return &ChatRepo{pool: pool} }

func (r *ChatRepo) CreateThread(ctx context.Context, t domain.ChatThread) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO chat_threads (id, store_id, product_id, created_at) VALUES ($1,$2,$3,$4)`,
		t.ID, t.StoreID, t.ProductID, t.CreatedAt)
	return err
}

func (r *ChatRepo) GetThread(ctx context.Context, id string) (domain.ChatThread, error) {
	var t domain.ChatThread
	err := r.pool.QueryRow(ctx, `SELECT id, store_id, product_id, created_at FROM chat_threads WHERE id=$1`, id).
		Scan(&t.ID, &t.StoreID, &t.ProductID, &t.CreatedAt)
	if err != nil {
		return domain.ChatThread{}, domain.ErrNotFound
	}
	return t, nil
}

func (r *ChatRepo) AddMessage(ctx context.Context, m domain.ChatMessage) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO chat_messages (id, thread_id, role, body, created_at) VALUES ($1,$2,$3,$4,$5)`,
		m.ID, m.ThreadID, m.Role, m.Body, m.CreatedAt)
	return err
}

func (r *ChatRepo) ListMessages(ctx context.Context, threadID string, limit int) ([]domain.ChatMessage, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, thread_id, role, body, created_at FROM (
			SELECT id, thread_id, role, body, created_at FROM chat_messages
			WHERE thread_id=$1 ORDER BY created_at DESC LIMIT $2
		) t ORDER BY created_at ASC`, threadID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ChatMessage
	for rows.Next() {
		var m domain.ChatMessage
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Role, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type SearchRepo struct {
	pool *pgxpool.Pool
}

func NewSearchRepo(pool *pgxpool.Pool, _ bool) *SearchRepo {
	return &SearchRepo{pool: pool}
}

func (r *SearchRepo) SearchStores(ctx context.Context, q, city string, limit int) ([]domain.Store, error) {
	q = strings.TrimSpace(q)
	if limit <= 0 {
		limit = 20
	}
	stores, err := r.searchStoresTrgm(ctx, q, city, limit)
	if err == nil {
		return stores, nil
	}
	like := "%" + q + "%"
	sql := storeCols + ` WHERE (name ILIKE $1 OR description ILIKE $1 OR category ILIKE $1)`
	args := []any{like}
	if city != "" {
		sql += ` AND lower(city)=lower($2)`
		args = append(args, city)
		sql += ` ORDER BY created_at DESC LIMIT $3`
		args = append(args, limit)
	} else {
		sql += ` ORDER BY created_at DESC LIMIT $2`
		args = append(args, limit)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectStores(rows)
}

func (r *SearchRepo) searchStoresTrgm(ctx context.Context, q, city string, limit int) ([]domain.Store, error) {
	sql := storeCols + ` WHERE (name % $1 OR name ILIKE '%'||$1||'%' OR description ILIKE '%'||$1||'%')`
	args := []any{q}
	if city != "" {
		sql += ` AND lower(city)=lower($2)`
		args = append(args, city)
		sql += ` ORDER BY similarity(name, $1) DESC LIMIT $3`
		args = append(args, limit)
	} else {
		sql += ` ORDER BY similarity(name, $1) DESC LIMIT $2`
		args = append(args, limit)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectStores(rows)
}

func (r *SearchRepo) SearchProducts(ctx context.Context, q, city string, limit int) ([]domain.Product, error) {
	q = strings.TrimSpace(q)
	if limit <= 0 {
		limit = 20
	}
	prods, err := r.searchProductsTrgm(ctx, q, city, limit)
	if err == nil {
		return prods, nil
	}
	like := "%" + q + "%"
	sql := `SELECT p.id, p.store_id, p.slug, p.name, p.description, p.price_cents, p.created_at
		FROM products p JOIN stores s ON s.id = p.store_id
		WHERE (p.name ILIKE $1 OR p.description ILIKE $1)`
	args := []any{like}
	if city != "" {
		sql += ` AND lower(s.city)=lower($2) ORDER BY p.created_at DESC LIMIT $3`
		args = append(args, city, limit)
	} else {
		sql += ` ORDER BY p.created_at DESC LIMIT $2`
		args = append(args, limit)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectProducts(rows)
}

func (r *SearchRepo) searchProductsTrgm(ctx context.Context, q, city string, limit int) ([]domain.Product, error) {
	sql := `SELECT p.id, p.store_id, p.slug, p.name, p.description, p.price_cents, p.created_at
		FROM products p JOIN stores s ON s.id = p.store_id
		WHERE (p.name % $1 OR p.name ILIKE '%'||$1||'%' OR p.description ILIKE '%'||$1||'%')`
	args := []any{q}
	if city != "" {
		sql += ` AND lower(s.city)=lower($2) ORDER BY similarity(p.name, $1) DESC LIMIT $3`
		args = append(args, city, limit)
	} else {
		sql += ` ORDER BY similarity(p.name, $1) DESC LIMIT $2`
		args = append(args, limit)
	}
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectProducts(rows)
}

var _ ports.AffinityRepository = (*AffinityRepo)(nil)
var _ ports.ChatRepository = (*ChatRepo)(nil)
var _ ports.SearchRepository = (*SearchRepo)(nil)
