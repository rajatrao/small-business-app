package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
)

type VoteRepo struct{ pool *pgxpool.Pool }

func NewVoteRepo(pool *pgxpool.Pool) *VoteRepo { return &VoteRepo{pool: pool} }

func (r *VoteRepo) Toggle(ctx context.Context, userID string, storeID, productID *string) (bool, error) {
	var existing string
	var err error
	if storeID != nil {
		err = r.pool.QueryRow(ctx, `SELECT id FROM votes WHERE user_id=$1 AND store_id=$2`, userID, *storeID).Scan(&existing)
	} else {
		err = r.pool.QueryRow(ctx, `SELECT id FROM votes WHERE user_id=$1 AND product_id=$2`, userID, *productID).Scan(&existing)
	}
	if err == nil && existing != "" {
		_, err = r.pool.Exec(ctx, `DELETE FROM votes WHERE id=$1`, existing)
		return false, err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO votes (id, user_id, store_id, product_id, created_at)
		VALUES ($1,$2,$3,$4, now())`, uuid.NewString(), userID, storeID, productID)
	return true, err
}

func (r *VoteRepo) HasVoted(ctx context.Context, userID string, storeID, productID *string) (bool, error) {
	var ok bool
	var err error
	if storeID != nil {
		err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM votes WHERE user_id=$1 AND store_id=$2)`, userID, *storeID).Scan(&ok)
	} else if productID != nil {
		err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM votes WHERE user_id=$1 AND product_id=$2)`, userID, *productID).Scan(&ok)
	}
	return ok, err
}

func (r *VoteRepo) CountByStoreIDs(ctx context.Context, ids []string, since *time.Time) (map[string]int, error) {
	out := map[string]int{}
	if len(ids) == 0 {
		return out, nil
	}
	q := `SELECT store_id, COUNT(*) FROM votes WHERE store_id::text = ANY($1)`
	args := []any{ids}
	if since != nil {
		q += ` AND created_at >= $2`
		args = append(args, *since)
	}
	q += ` GROUP BY store_id`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

func (r *VoteRepo) CountByProductIDs(ctx context.Context, ids []string, since *time.Time) (map[string]int, error) {
	out := map[string]int{}
	if len(ids) == 0 {
		return out, nil
	}
	q := `SELECT product_id, COUNT(*) FROM votes WHERE product_id::text = ANY($1)`
	args := []any{ids}
	if since != nil {
		q += ` AND created_at >= $2`
		args = append(args, *since)
	}
	q += ` GROUP BY product_id`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

type ReviewRepo struct{ pool *pgxpool.Pool }

func NewReviewRepo(pool *pgxpool.Pool) *ReviewRepo { return &ReviewRepo{pool: pool} }

func (r *ReviewRepo) Create(ctx context.Context, rv domain.Review) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reviews (id, user_id, store_id, product_id, rating, body, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		rv.ID, rv.UserID, rv.StoreID, rv.ProductID, rv.Rating, rv.Body, rv.CreatedAt)
	return err
}

func (r *ReviewRepo) ListByStore(ctx context.Context, storeID string) ([]domain.Review, error) {
	rows, err := r.pool.Query(ctx, reviewSelect+` WHERE r.store_id=$1 ORDER BY r.created_at DESC`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectReviews(rows)
}

func (r *ReviewRepo) ListByProduct(ctx context.Context, productID string) ([]domain.Review, error) {
	rows, err := r.pool.Query(ctx, reviewSelect+` WHERE r.product_id=$1 ORDER BY r.created_at DESC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectReviews(rows)
}

const reviewSelect = `
SELECT r.id, r.user_id, r.store_id, r.product_id, r.rating, r.body, r.created_at,
       u.name,
       COALESCE(s.name,''), COALESCE(s.slug,''), COALESCE(s.category,''), COALESCE(s.city,''),
       COALESCE(p.name,''), COALESCE(p.slug,'')
FROM reviews r
JOIN users u ON u.id = r.user_id
LEFT JOIN stores s ON s.id = COALESCE(r.store_id, (SELECT store_id FROM products WHERE id = r.product_id))
LEFT JOIN products p ON p.id = r.product_id`

func collectReviews(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]domain.Review, error) {
	var out []domain.Review
	for rows.Next() {
		var rv domain.Review
		if err := rows.Scan(
			&rv.ID, &rv.UserID, &rv.StoreID, &rv.ProductID, &rv.Rating, &rv.Body, &rv.CreatedAt,
			&rv.AuthorName, &rv.StoreName, &rv.StoreSlug, &rv.StoreCategory, &rv.StoreCity,
			&rv.ProductName, &rv.ProductSlug,
		); err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

func (r *ReviewRepo) StatsByStoreIDs(ctx context.Context, ids []string, since *time.Time) (map[string]domain.ReviewStats, error) {
	out := map[string]domain.ReviewStats{}
	if len(ids) == 0 {
		return out, nil
	}
	q := `SELECT store_id, COUNT(*), COALESCE(SUM(rating),0) FROM reviews WHERE store_id::text = ANY($1)`
	args := []any{ids}
	if since != nil {
		q += ` AND created_at >= $2`
		args = append(args, *since)
	}
	q += ` GROUP BY store_id`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var st domain.ReviewStats
		if err := rows.Scan(&id, &st.Count, &st.RatingSum); err != nil {
			return nil, err
		}
		out[id] = st
	}
	return out, rows.Err()
}

func (r *ReviewRepo) StatsByProductIDs(ctx context.Context, ids []string, since *time.Time) (map[string]domain.ReviewStats, error) {
	out := map[string]domain.ReviewStats{}
	if len(ids) == 0 {
		return out, nil
	}
	q := `SELECT product_id, COUNT(*), COALESCE(SUM(rating),0) FROM reviews WHERE product_id::text = ANY($1)`
	args := []any{ids}
	if since != nil {
		q += ` AND created_at >= $2`
		args = append(args, *since)
	}
	q += ` GROUP BY product_id`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var st domain.ReviewStats
		if err := rows.Scan(&id, &st.Count, &st.RatingSum); err != nil {
			return nil, err
		}
		out[id] = st
	}
	return out, rows.Err()
}

func (r *ReviewRepo) ListRecent(ctx context.Context, city, category string, since *time.Time, limit int) ([]domain.Review, error) {
	if limit <= 0 {
		limit = 20
	}
	q := reviewSelect + ` WHERE lower(s.city)=lower($1)`
	args := []any{city}
	n := 2
	if category != "" {
		q += ` AND s.category=$` + itoa(n)
		args = append(args, category)
		n++
	}
	if since != nil {
		q += ` AND r.created_at >= $` + itoa(n)
		args = append(args, *since)
		n++
	}
	q += ` ORDER BY r.created_at DESC LIMIT $` + itoa(n)
	args = append(args, limit)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectReviews(rows)
}

func itoa(n int) string {
	if n < 0 {
		return "0"
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

var _ ports.VoteRepository = (*VoteRepo)(nil)
var _ ports.ReviewRepository = (*ReviewRepo)(nil)
