package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
)

type StoreRepo struct{ pool *pgxpool.Pool }

func NewStoreRepo(pool *pgxpool.Pool) *StoreRepo { return &StoreRepo{pool: pool} }

const storeCols = `SELECT id, owner_id, slug, name, description, category, city, phone, address, created_at FROM stores`

func (r *StoreRepo) Create(ctx context.Context, s domain.Store) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO stores (id, owner_id, slug, name, description, category, city, phone, address, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		s.ID, s.OwnerID, s.Slug, s.Name, s.Description, s.Category, s.City, s.Phone, s.Address, s.CreatedAt)
	return err
}

func (r *StoreRepo) Update(ctx context.Context, s domain.Store) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE stores SET name=$2, description=$3, category=$4, city=$5, phone=$6, address=$7
		WHERE id=$1`, s.ID, s.Name, s.Description, s.Category, s.City, s.Phone, s.Address)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *StoreRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM stores WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *StoreRepo) GetByID(ctx context.Context, id string) (domain.Store, error) {
	return scanStore(r.pool.QueryRow(ctx, storeCols+` WHERE id=$1`, id))
}

func (r *StoreRepo) GetBySlug(ctx context.Context, slug string) (domain.Store, error) {
	return scanStore(r.pool.QueryRow(ctx, storeCols+` WHERE slug=$1`, slug))
}

func (r *StoreRepo) ListByCity(ctx context.Context, city, category string) ([]domain.Store, error) {
	q := storeCols + ` WHERE lower(city)=lower($1)`
	args := []any{city}
	if category != "" {
		q += ` AND category=$2`
		args = append(args, category)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectStores(rows)
}

func (r *StoreRepo) ListByOwner(ctx context.Context, ownerID string) ([]domain.Store, error) {
	rows, err := r.pool.Query(ctx, storeCols+` WHERE owner_id=$1 ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectStores(rows)
}

func (r *StoreRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM stores WHERE slug=$1)`, slug).Scan(&ok)
	return ok, err
}

func scanStore(row pgx.Row) (domain.Store, error) {
	var s domain.Store
	err := row.Scan(&s.ID, &s.OwnerID, &s.Slug, &s.Name, &s.Description, &s.Category, &s.City, &s.Phone, &s.Address, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Store{}, domain.ErrNotFound
	}
	return s, err
}

func collectStores(rows pgx.Rows) ([]domain.Store, error) {
	var out []domain.Store
	for rows.Next() {
		var s domain.Store
		if err := rows.Scan(&s.ID, &s.OwnerID, &s.Slug, &s.Name, &s.Description, &s.Category, &s.City, &s.Phone, &s.Address, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type ProductRepo struct{ pool *pgxpool.Pool }

func NewProductRepo(pool *pgxpool.Pool) *ProductRepo { return &ProductRepo{pool: pool} }

const productCols = `SELECT id, store_id, slug, name, description, price_cents, created_at FROM products`

func (r *ProductRepo) Create(ctx context.Context, p domain.Product) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO products (id, store_id, slug, name, description, price_cents, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		p.ID, p.StoreID, p.Slug, p.Name, p.Description, p.PriceCents, p.CreatedAt)
	return err
}

func (r *ProductRepo) Update(ctx context.Context, p domain.Product) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE products SET name=$2, description=$3, price_cents=$4 WHERE id=$1`,
		p.ID, p.Name, p.Description, p.PriceCents)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id string) (domain.Product, error) {
	return scanProduct(r.pool.QueryRow(ctx, productCols+` WHERE id=$1`, id))
}

func (r *ProductRepo) GetBySlug(ctx context.Context, slug string) (domain.Product, error) {
	return scanProduct(r.pool.QueryRow(ctx, productCols+` WHERE slug=$1`, slug))
}

func (r *ProductRepo) ListByStore(ctx context.Context, storeID string) ([]domain.Product, error) {
	rows, err := r.pool.Query(ctx, productCols+` WHERE store_id=$1 ORDER BY created_at DESC`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectProducts(rows)
}

func (r *ProductRepo) ListByCity(ctx context.Context, city, category string) ([]domain.Product, error) {
	q := productCols + ` p WHERE EXISTS (
		SELECT 1 FROM stores s WHERE s.id=p.store_id AND lower(s.city)=lower($1)`
	args := []any{city}
	if category != "" {
		q += ` AND s.category=$2`
		args = append(args, category)
	}
	q += `) ORDER BY p.created_at DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectProducts(rows)
}

func (r *ProductRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE slug=$1)`, slug).Scan(&ok)
	return ok, err
}

func scanProduct(row pgx.Row) (domain.Product, error) {
	var p domain.Product
	err := row.Scan(&p.ID, &p.StoreID, &p.Slug, &p.Name, &p.Description, &p.PriceCents, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Product{}, domain.ErrNotFound
	}
	return p, err
}

func collectProducts(rows pgx.Rows) ([]domain.Product, error) {
	var out []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.StoreID, &p.Slug, &p.Name, &p.Description, &p.PriceCents, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type PhotoRepo struct{ pool *pgxpool.Pool }

func NewPhotoRepo(pool *pgxpool.Pool) *PhotoRepo { return &PhotoRepo{pool: pool} }

func (r *PhotoRepo) Create(ctx context.Context, p domain.Photo) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO photos (id, store_id, product_id, url, sort_order, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		p.ID, p.StoreID, p.ProductID, p.URL, p.SortOrder, p.CreatedAt)
	return err
}

func (r *PhotoRepo) ListByStoreIDs(ctx context.Context, ids []string) (map[string][]domain.Photo, error) {
	out := map[string][]domain.Photo{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, store_id, product_id, url, sort_order, created_at
		FROM photos WHERE store_id::text = ANY($1) AND product_id IS NULL
		ORDER BY sort_order, created_at`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		p, sid, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		if sid != "" {
			out[sid] = append(out[sid], p)
		}
	}
	return out, rows.Err()
}

func (r *PhotoRepo) ListByProductIDs(ctx context.Context, ids []string) (map[string][]domain.Photo, error) {
	out := map[string][]domain.Photo{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, store_id, product_id, url, sort_order, created_at
		FROM photos WHERE product_id::text = ANY($1)
		ORDER BY sort_order, created_at`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p domain.Photo
		if err := rows.Scan(&p.ID, &p.StoreID, &p.ProductID, &p.URL, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		if p.ProductID != nil {
			out[*p.ProductID] = append(out[*p.ProductID], p)
		}
	}
	return out, rows.Err()
}

func scanPhoto(rows pgx.Rows) (domain.Photo, string, error) {
	var p domain.Photo
	err := rows.Scan(&p.ID, &p.StoreID, &p.ProductID, &p.URL, &p.SortOrder, &p.CreatedAt)
	sid := ""
	if p.StoreID != nil {
		sid = *p.StoreID
	}
	return p, sid, err
}

var _ ports.StoreRepository = (*StoreRepo)(nil)
var _ ports.ProductRepository = (*ProductRepo)(nil)
var _ ports.PhotoRepository = (*PhotoRepo)(nil)
