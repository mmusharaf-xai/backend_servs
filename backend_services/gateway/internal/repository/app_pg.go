package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/pagination"
)

const defaultAppsLimit = pagination.DefaultLimit

type AppPG struct {
	pool *pgxpool.Pool
}

func NewAppPG(pool *pgxpool.Pool) *AppPG {
	return &AppPG{pool: pool}
}

func (r *AppPG) List(ctx context.Context, q, cursor string, limit int) ([]*domain.App, string, error) {
	if limit <= 0 {
		limit = defaultAppsLimit
	}
	limit = pagination.NormalizeLimit(limit)

	var (
		conds []string
		args  []any
	)

	if q != "" {
		args = append(args, "%"+q+"%")
		idx := len(args)
		conds = append(conds, fmt.Sprintf("(name ILIKE $%d OR tagline ILIKE $%d OR category ILIKE $%d)", idx, idx, idx))
	}

	if cursor != "" {
		sortOrder, id, err := decodeAppCursor(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", err)
		}
		args = append(args, sortOrder, id)
		conds = append(conds, fmt.Sprintf("(sort_order, id) > ($%d, $%d)", len(args)-1, len(args)))
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	// Fetch one extra row to detect whether another page exists.
	args = append(args, limit+1)

	query := fmt.Sprintf(`
		SELECT id, slug, name, tagline, description, icon, category, status, sort_order, created_at, updated_at
		FROM apps
		%s
		ORDER BY sort_order, id
		LIMIT $%d
	`, where, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list apps: %w", err)
	}
	defer rows.Close()

	var apps []*domain.App
	for rows.Next() {
		var a domain.App
		if err := rows.Scan(&a.ID, &a.Slug, &a.Name, &a.Tagline, &a.Description, &a.Icon,
			&a.Category, &a.Status, &a.SortOrder, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, "", fmt.Errorf("scan app: %w", err)
		}
		apps = append(apps, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate apps: %w", err)
	}

	nextCursor := ""
	if len(apps) > limit {
		last := apps[limit-1]
		apps = apps[:limit]
		nextCursor = encodeAppCursor(last.SortOrder, last.ID)
	}

	return apps, nextCursor, nil
}

func (r *AppPG) GetBySlug(ctx context.Context, slug string) (*domain.App, error) {
	var a domain.App
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, tagline, description, icon, category, status, sort_order, created_at, updated_at
		FROM apps WHERE slug = $1
	`, slug).Scan(&a.ID, &a.Slug, &a.Name, &a.Tagline, &a.Description, &a.Icon,
		&a.Category, &a.Status, &a.SortOrder, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get app by slug: %w", err)
	}
	return &a, nil
}

// encodeAppCursor produces an opaque cursor from the keyset (sort_order, id).
func encodeAppCursor(sortOrder int, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d:%s", sortOrder, id)))
}

func decodeAppCursor(cursor string) (int, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, "", err
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("malformed cursor")
	}
	sortOrder, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", err
	}
	return sortOrder, parts[1], nil
}
